// Package daemon keeps a session with the keyboard: it connects over USB or
// Bluetooth, sends a heartbeat every 100 ms (the app's cadence, carrying the
// active profile), reconnects after unplug/sleep, and publishes status.
package daemon

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"clvx/internal/device"
	cproto "clvx/internal/proto"
)

const (
	HeartbeatPeriod = 100 * time.Millisecond
	ExchangeTimeout = 400 * time.Millisecond
	MaxFailures     = 3
	RetryDelay      = 2 * time.Second
)

// Status is the snapshot exposed over the socket.
type Status struct {
	Connected     bool      `json:"connected"`
	Transport     string    `json:"transport,omitempty"`
	BatteryPct    uint64    `json:"batteryPercent"`
	Charging      bool      `json:"charging"`
	PowerMode     uint64    `json:"powerMode"`
	BacklightMode uint64    `json:"backlightMode"`
	ActiveProfile uint32    `json:"activeProfile"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	Error         string    `json:"error,omitempty"`
}

type Daemon struct {
	mode string

	mu     sync.Mutex
	status Status
	subs   map[chan Status]struct{}
}

func New(transportMode string) *Daemon {
	return &Daemon{mode: transportMode, subs: map[chan Status]struct{}{}}
}

// Snapshot returns the current status.
func (d *Daemon) Snapshot() Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.status
}

// SetProfile changes the active profile sent in every heartbeat.
func (d *Daemon) SetProfile(p uint32) {
	d.update(func(s *Status) { s.ActiveProfile = p })
}

// Subscribe returns a channel of status changes and a cancel func.
func (d *Daemon) Subscribe() (<-chan Status, func()) {
	ch := make(chan Status, 8)
	d.mu.Lock()
	d.subs[ch] = struct{}{}
	d.mu.Unlock()
	return ch, func() {
		d.mu.Lock()
		delete(d.subs, ch)
		d.mu.Unlock()
	}
}

func (d *Daemon) update(fn func(*Status)) {
	d.mu.Lock()
	fn(&d.status)
	snap := d.status
	for ch := range d.subs {
		select {
		case ch <- snap:
		default: // slow subscriber: drop the update, they get the next one
		}
	}
	d.mu.Unlock()
}

// Run blocks until ctx is cancelled, keeping a session alive.
func (d *Daemon) Run(ctx context.Context) {
	for ctx.Err() == nil {
		t, err := device.Open(d.mode)
		if err != nil {
			d.update(func(s *Status) { s.Connected = false; s.Transport = ""; s.Error = err.Error() })
			sleep(ctx, RetryDelay)
			continue
		}
		log.Printf("connected via %s", t.Name())
		d.update(func(s *Status) { s.Connected = true; s.Transport = t.Name(); s.Error = "" })
		err = d.heartbeats(ctx, t)
		t.Close()
		d.update(func(s *Status) { s.Connected = false; s.Transport = ""; s.Error = errString(err) })
		log.Printf("disconnected: %v", err)
		sleep(ctx, RetryDelay)
	}
}

func (d *Daemon) heartbeats(ctx context.Context, t device.Transport) error {
	failures := 0
	next := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		profile := d.Snapshot().ActiveProfile
		reply, err := t.Exchange(cproto.EncodeHeartBeat(profile), ExchangeTimeout)
		if err == nil {
			err = d.handleReply(reply)
		}
		if err != nil {
			failures++
			if failures >= MaxFailures {
				return err
			}
		} else {
			failures = 0
		}
		next = next.Add(HeartbeatPeriod)
		if wait := time.Until(next); wait > 0 {
			sleep(ctx, wait)
		} else {
			next = time.Now()
		}
	}
}

func (d *Daemon) handleReply(reply []byte) error {
	resp, err := cproto.DecodeResponse(reply)
	if err != nil {
		return err
	}
	if resp.Field != 6 {
		return errors.New("unexpected reply to heartbeat")
	}
	h, err := cproto.DecodeHeartbeat(resp.Body)
	if err != nil {
		return err
	}
	d.update(func(s *Status) {
		s.BatteryPct = h.BatteryLevel
		s.Charging = h.Charging
		s.PowerMode = h.PowerMode
		s.BacklightMode = h.BacklightMode
		s.LastHeartbeat = time.Now()
		s.Error = ""
	})
	return nil
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
