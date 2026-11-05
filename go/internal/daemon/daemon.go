// Package daemon keeps a session with the keyboard: it connects over USB or
// Bluetooth, sends a heartbeat every 100 ms (the app's cadence), reconnects
// after unplug/sleep, and publishes status.
package daemon

import (
	"context"
	"errors"
	"fmt"
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

// Status is the snapshot exposed over the socket. ActiveProfile is deliberately
// omitted: profile switching is unsupported and heartbeats always use profile 0.
type Status struct {
	Connected     bool      `json:"connected"`
	Transport     string    `json:"transport,omitempty"`
	BatteryPct    uint64    `json:"batteryPercent"`
	Charging      bool      `json:"charging"`
	PowerMode     uint64    `json:"powerMode"`
	BacklightMode uint64    `json:"backlightMode"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	Error         string    `json:"error,omitempty"`
}

type Daemon struct {
	mode string

	mu     sync.Mutex
	status Status
	subs   map[chan Status]struct{}

	requests chan pendingRequest
}

// pendingRequest is a one-shot protobuf exchange handed to the heartbeat
// loop, which owns the transport.
type pendingRequest struct {
	pb    []byte
	reply chan pendingReply
}

type pendingReply struct {
	body []byte
	err  error
}

func New(transportMode string) *Daemon {
	return &Daemon{
		mode:     transportMode,
		subs:     map[chan Status]struct{}{},
		requests: make(chan pendingRequest, 4),
	}
}

// Snapshot returns the current status.
func (d *Daemon) Snapshot() Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.status
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
		d.servePending(t)
		// Keep the heartbeat pinned to activeProfile 0; no switching API is exposed.
		reply, err := t.Exchange(cproto.EncodeHeartBeat(0), ExchangeTimeout)
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
		wait := time.Until(next)
		if wait <= 0 {
			next = time.Now()
			continue
		}
		// Keep the heartbeat cadence while still answering RPCs promptly.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-d.requests:
			d.runPending(t, req)
		case <-time.After(wait):
		}
	}
}

// servePending drains queued exchanges without blocking the heartbeat.
func (d *Daemon) servePending(t device.Transport) {
	for {
		select {
		case req := <-d.requests:
			d.runPending(t, req)
		default:
			return
		}
	}
}

func (d *Daemon) runPending(t device.Transport, req pendingRequest) {
	body, err := t.Exchange(req.pb, ExchangeTimeout)
	req.reply <- pendingReply{body: body, err: err}
}

// exchange hands one request to the heartbeat loop, which owns the transport,
// and waits for the device's reply.
func (d *Daemon) exchange(pb []byte) ([]byte, error) {
	if !d.Snapshot().Connected {
		return nil, errors.New("clvxd: keyboard is not connected")
	}
	req := pendingRequest{pb: pb, reply: make(chan pendingReply, 1)}
	select {
	case d.requests <- req:
	case <-time.After(ExchangeTimeout):
		return nil, errors.New("clvxd: device is busy")
	}
	select {
	case r := <-req.reply:
		return r.body, r.err
	case <-time.After(3 * ExchangeTimeout):
		return nil, errors.New("clvxd: device did not answer")
	}
}

// GetSettings reads AppSettings from the keyboard.
func (d *Daemon) GetSettings() (*cproto.Settings, error) {
	reply, err := d.exchange(cproto.EncodeGetSettings())
	if err != nil {
		return nil, err
	}
	resp, err := cproto.DecodeResponse(reply)
	if err != nil {
		return nil, err
	}
	if resp.Type == cproto.RespBadRequest {
		return nil, errors.New("clvxd: GET_SETTINGS rejected with BAD_REQUEST")
	}
	settings, status, err := cproto.ParseSettingsResponse(resp)
	if err != nil {
		return nil, err
	}
	if status != cproto.SettingsStatusOK {
		return nil, fmt.Errorf("clvxd: GET_SETTINGS status %d", status)
	}
	return settings, nil
}

// SetGlobal applies a partial GlobalSettings patch: current settings are read,
// the patch is merged over them, and the full AppSettings (including the
// untouched per-profile bytes) is written back.
func (d *Daemon) SetGlobal(patch *cproto.Global) (*cproto.Settings, error) {
	current, err := d.GetSettings()
	if err != nil {
		return nil, err
	}
	current.Global.Merge(patch)
	app, err := current.Encode()
	if err != nil {
		return nil, err
	}
	reply, err := d.exchange(cproto.EncodeSetSettings(app))
	if err != nil {
		return nil, err
	}
	resp, err := cproto.DecodeResponse(reply)
	if err != nil {
		return nil, err
	}
	if resp.Type == cproto.RespBadRequest {
		return nil, errors.New("clvxd: SET_SETTINGS rejected with BAD_REQUEST")
	}
	applied, status, err := cproto.ParseSettingsResponse(resp)
	if err != nil {
		return nil, err
	}
	if status != cproto.SettingsStatusOK {
		return applied, fmt.Errorf("clvxd: SET_SETTINGS status %d", status)
	}
	// A successful write only carries the status, so read back what the
	// keyboard actually stored (it may normalise ranges).
	return d.GetSettings()
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
