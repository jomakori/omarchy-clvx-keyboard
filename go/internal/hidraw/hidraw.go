// Package hidraw is the USB transport: Linux hidraw nodes for the CLVX vendor
// interface (VID 0x36F7, bus 0x0003). The keyboard exposes several hidraw
// nodes; the vendor one is found by probing candidates with GetDeviceInfo.
package hidraw

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clvx/internal/frame"

	"golang.org/x/sys/unix"
)

// Candidates returns hidraw device nodes whose HID_ID matches a USB CLVX device.
func Candidates() ([]string, error) {
	nodes, err := filepath.Glob("/sys/class/hidraw/hidraw*")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range nodes {
		f, err := os.Open(filepath.Join(n, "device", "uevent"))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "HID_ID=") && strings.HasPrefix(line, "HID_ID=0003:000036F7:") {
				out = append(out, "/dev/"+filepath.Base(n))
			}
		}
		f.Close()
	}
	return out, nil
}

// Device is one open hidraw node.
type Device struct {
	f    *os.File
	path string
}

func Open(path string) (*Device, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("hidraw %s: %w", path, err)
	}
	return &Device{f: f, path: path}, nil
}

func (d *Device) Path() string { return d.path }

// Drain discards any input reports already queued (stale replies).
func (d *Device) Drain() {
	for {
		if n, _ := unix.Poll([]unix.PollFd{{Fd: int32(d.f.Fd()), Events: unix.POLLIN}}, 0); n <= 0 {
			return
		}
		buf := make([]byte, frame.USBReportLen)
		if _, err := d.f.Read(buf); err != nil {
			return
		}
	}
}
func (d *Device) Close() { _ = d.f.Close() }

// Exchange writes one protobuf request as output reports and reads input
// reports until the terminator arrives, or the timeout expires.
func (d *Device) Exchange(pb []byte, timeout time.Duration) ([]byte, error) {
	d.Drain()
	for _, r := range frame.USBReports(pb) {
		if _, err := d.f.Write(r); err != nil {
			return nil, fmt.Errorf("hidraw write: %w", err)
		}
	}
	deadline := time.Now().Add(timeout)
	var acc []byte
	for {
		left := time.Until(deadline)
		if left <= 0 {
			return nil, errors.New("hidraw: timeout waiting for reply")
		}
		report, err := d.readReport(left)
		if err != nil {
			return nil, err
		}
		if report == nil {
			continue
		}
		var out []byte
		acc, out, err = frame.USBDecodeInput(acc, report)
		if err == frame.ErrNoTerminator {
			continue
		}
		if err != nil {
			return nil, err
		}
		return out, nil
	}
}

func (d *Device) readReport(timeout time.Duration) ([]byte, error) {
	fds := []unix.PollFd{{Fd: int32(d.f.Fd()), Events: unix.POLLIN}}
	var n int
	var err error
	for {
		n, err = unix.Poll(fds, int(timeout.Milliseconds()))
		if err != unix.EINTR {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("hidraw: timeout waiting for reply")
	}
	buf := make([]byte, frame.USBReportLen)
	m, err := d.f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:m], nil
}
