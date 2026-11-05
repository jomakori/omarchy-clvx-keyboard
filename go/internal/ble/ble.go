// Package ble talks to the CLVX vendor GATT characteristic through BlueZ (D-Bus).
// The service/characteristic are discovered by UUID, not hardcoded per device.
package ble

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"clvx/internal/frame"

	"github.com/godbus/dbus/v5"
)

const (
	ServiceUUID = "d0bf1500-c402-424a-80b0-bc7aeced077e"
	CharUUID    = "d0bf0001-c402-424a-80b0-bc7aeced077e"
	bluezName   = "org.bluez"
	charIface   = "org.bluez.GattCharacteristic1"
)

// Conn is one persistent BLE session. BlueZ requires the same bus connection
// that calls StartNotify to stay open for the lifetime of notifications.
type Conn struct {
	bus      *dbus.Conn
	charPath dbus.ObjectPath
	sigs     chan *dbus.Signal
}

// Open finds a connected CLVX device with the vendor characteristic and
// enables notifications.
func Open() (*Conn, error) {
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("ble: system bus: %w", err)
	}
	path, err := findChar(bus)
	if err != nil {
		return nil, err
	}
	c := &Conn{bus: bus, charPath: path, sigs: make(chan *dbus.Signal, 64)}
	bus.Signal(c.sigs)
	if err := bus.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchObjectPath(path),
	); err != nil {
		return nil, err
	}
	if call := c.obj().Call(charIface+".StartNotify", 0); call.Err != nil {
		return nil, fmt.Errorf("ble: StartNotify: %w", call.Err)
	}
	return c, nil
}

func (c *Conn) Close() {
	_ = c.obj().Call(charIface+".StopNotify", 0).Err
	c.bus.RemoveSignal(c.sigs)
}

func (c *Conn) obj() dbus.BusObject { return c.bus.Object(bluezName, c.charPath) }

// Path of the discovered characteristic (useful for logs).
func (c *Conn) Path() string { return string(c.charPath) }

// Exchange sends one protobuf request and returns the decoded protobuf reply.
func (c *Conn) Exchange(ctx context.Context, pb []byte) ([]byte, error) {
	drain(c.sigs)
	for _, chunk := range frame.BLEWrites(pb) {
		b := make([]byte, len(chunk))
		copy(b, chunk)
		if call := c.obj().Call(charIface+".WriteValue", 0, b, map[string]dbus.Variant{}); call.Err != nil {
			return nil, fmt.Errorf("ble: WriteValue: %w", call.Err)
		}
	}
	var acc []byte
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case s := <-c.sigs:
			if s.Path != c.charPath || s.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" || len(s.Body) < 2 {
				continue
			}
			changed, _ := s.Body[1].(map[string]dbus.Variant)
			v, ok := changed["Value"]
			if !ok {
				continue
			}
			acc = append(acc, v.Value().([]byte)...)
			if bytes.IndexByte(acc, frame.Terminator) >= 0 {
				return frame.BLEDecode(acc)
			}
		}
	}
}

func drain(ch chan *dbus.Signal) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// WaitTimeout is a convenience wrapper applying a deadline to Exchange.
func (c *Conn) WaitTimeout(pb []byte, d time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return c.Exchange(ctx, pb)
}

func findChar(bus *dbus.Conn) (dbus.ObjectPath, error) {
	var objs map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := bus.Object(bluezName, "/").Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&objs)
	if err != nil {
		return "", fmt.Errorf("ble: GetManagedObjects: %w", err)
	}
	for path, ifaces := range objs {
		props, ok := ifaces[charIface]
		if !ok {
			continue
		}
		uuid, _ := props["UUID"].Value().(string)
		if !strings.EqualFold(uuid, CharUUID) {
			continue
		}
		// Ensure the owning service is the vendor service.
		svcPath := dbus.ObjectPath(path[:strings.LastIndex(string(path), "/")])
		if svcProps, ok := objs[svcPath]["org.bluez.GattService1"]; ok {
			if u, _ := svcProps["UUID"].Value().(string); !strings.EqualFold(u, ServiceUUID) {
				continue
			}
		}
		return path, nil
	}
	return "", errors.New("ble: CLVX vendor characteristic not found — is the keyboard connected over Bluetooth?")
}
