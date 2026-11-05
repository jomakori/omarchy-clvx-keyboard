// Package device hides the transport: USB (hidraw) when a CLVX is attached over
// USB, otherwise Bluetooth (GATT). Both use the same protobuf envelope.
package device

import (
	"errors"
	"fmt"
	"time"

	"clvx/internal/ble"
	"clvx/internal/hidraw"
	cproto "clvx/internal/proto"
)

// Transport exchanges one protobuf request for one protobuf reply.
type Transport interface {
	Name() string
	Exchange(pb []byte, timeout time.Duration) ([]byte, error)
	Close()
}

type usbTransport struct{ dev *hidraw.Device }

func (u usbTransport) Name() string { return "usb:" + u.dev.Path() }
func (u usbTransport) Exchange(pb []byte, t time.Duration) ([]byte, error) {
	return u.dev.Exchange(pb, t)
}
func (u usbTransport) Close() { u.dev.Close() }

type bleTransport struct{ conn *ble.Conn }

func (b bleTransport) Name() string { return "ble" }
func (b bleTransport) Exchange(pb []byte, t time.Duration) ([]byte, error) {
	return b.conn.WaitTimeout(pb, t)
}
func (b bleTransport) Close() { b.conn.Close() }

// Open selects a transport: "usb", "ble", or "auto" (USB first, then BLE).
func Open(mode string) (Transport, error) {
	var usbErr error
	if mode == "usb" || mode == "auto" {
		t, err := openUSB()
		if err == nil {
			return t, nil
		}
		if mode == "usb" {
			return nil, err
		}
		usbErr = err
	}
	if mode == "ble" || mode == "auto" {
		conn, err := ble.Open()
		if err != nil {
			if usbErr != nil {
				return nil, fmt.Errorf("%v; %v", usbErr, err)
			}
			return nil, err
		}
		return bleTransport{conn}, nil
	}
	return nil, fmt.Errorf("unknown transport %q", mode)
}

// openUSB probes each hidraw candidate with GetDeviceInfo and keeps the one
// that answers; non-vendor collections never produce a decodable reply.
func openUSB() (Transport, error) {
	nodes, err := hidraw.Candidates()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, errors.New("usb: no CLVX hidraw nodes")
	}
	req, err := cproto.EncodeEmptyRequest(cproto.ReqGetDeviceInfo)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, path := range nodes {
		dev, err := hidraw.Open(path)
		if err != nil {
			lastErr = err
			continue
		}
		reply, err := dev.Exchange(req, time.Second)
		if err != nil {
			lastErr = err
			dev.Close()
			continue
		}
		if _, err := cproto.DecodeResponse(reply); err != nil {
			lastErr = err
			dev.Close()
			continue
		}
		// Reopen for the real session: one exchange per open handle is the
		// proven pattern on this firmware.
		dev.Close()
		fresh, err := hidraw.Open(path)
		if err != nil {
			return nil, err
		}
		return usbTransport{fresh}, nil
	}
	return nil, fmt.Errorf("usb: no vendor interface answered (last error: %v)", lastErr)
}
