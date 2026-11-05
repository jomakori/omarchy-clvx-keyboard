// Command clvx talks to a Clevetura CLVX keyboard over USB or Bluetooth.
//
//	clvx [--transport auto|usb|ble] <command>
//
// Commands: info, settings, defaults, stats, heartbeat [activeProfile], raw <type>
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"clvx/internal/device"
	cproto "clvx/internal/proto"
)

var commands = map[string]uint32{
	"info":     cproto.ReqGetDeviceInfo,
	"settings": cproto.ReqGetSettings,
	"defaults": cproto.ReqGetDefaultSettings,
	"stats":    cproto.ReqGetStatistics,
}

func main() {
	transport := flag.String("transport", "auto", "auto, usb or ble")
	flag.Parse()
	if err := run(*transport, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "clvx:", err)
		os.Exit(1)
	}
}

func run(transport string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: clvx [--transport auto|usb|ble] <info|settings|defaults|stats|heartbeat [n]|raw <type>>")
	}
	pb, err := encodeCommand(args)
	if err != nil {
		return err
	}
	t, err := device.Open(transport)
	if err != nil {
		return err
	}
	defer t.Close()
	fmt.Fprintf(os.Stderr, "transport: %s\n", t.Name())

	reply, err := t.Exchange(pb, 3*time.Second)
	if err != nil {
		return err
	}
	resp, err := cproto.DecodeResponse(reply)
	if err != nil {
		return err
	}
	if resp.Type == cproto.RespBadRequest {
		return fmt.Errorf("device replied BAD_REQUEST (request %s)", hex.EncodeToString(pb))
	}
	if resp.Field == 6 {
		return printHeartbeat(resp.Body)
	}
	fields, err := cproto.Fields(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("response type=%d field=%d (%d bytes)\n", resp.Type, resp.Field, len(resp.Body))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case uint64:
			fmt.Printf("  field %d varint %d\n", f.Num, v)
		case []byte:
			fmt.Printf("  field %d len %d %s\n", f.Num, len(v), hex.EncodeToString(v))
		}
	}
	return nil
}

func encodeCommand(args []string) ([]byte, error) {
	switch cmd := args[0]; cmd {
	case "heartbeat":
		var profile uint64
		if len(args) > 1 {
			p, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return nil, err
			}
			profile = p
		}
		return cproto.EncodeHeartBeat(uint32(profile)), nil
	case "raw":
		if len(args) < 2 {
			return nil, fmt.Errorf("raw needs a request type")
		}
		t, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			return nil, err
		}
		return cproto.EncodeEmptyRequest(uint32(t))
	default:
		t, ok := commands[cmd]
		if !ok {
			return nil, fmt.Errorf("unknown command %q", cmd)
		}
		return cproto.EncodeEmptyRequest(t)
	}
}

func printHeartbeat(body []byte) error {
	h, err := cproto.DecodeHeartbeat(body)
	if err != nil {
		return err
	}
	fmt.Printf("battery=%d%% charging=%v powerMode=%d backlightMode=%d appSettingsCounter=%d\n",
		h.BatteryLevel, h.Charging, h.PowerMode, h.BacklightMode, h.AppSettingsCount)
	return nil
}
