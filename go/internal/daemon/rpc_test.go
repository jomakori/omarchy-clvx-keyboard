package daemon

import (
	"bytes"
	"errors"
	"testing"
	"time"

	cproto "clvx/internal/proto"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestSetProfileSettingsReturnsMergedViewWithoutReadback(t *testing.T) {
	d := connectedTestDaemon()
	touchpad := testVarint(1, 0)
	touchZone := testBytes(1, touchpad)
	raw := testBytes(3, touchZone)
	profile, err := cproto.DecodeProfile(raw)
	if err != nil {
		t.Fatalf("DecodeProfile() error = %v", err)
	}
	enabled := true
	profile.TouchZone.Touchpad.Enable = &enabled
	wantRaw, err := profile.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() error = %v", err)
	}

	var applied *cproto.ProfileSettings
	done := make(chan error, 1)
	go func() {
		var err error
		applied, err = d.SetProfileSettings(profile)
		done <- err
	}()

	req := nextPending(t, d)
	if want := cproto.EncodeSetProfileSettings(wantRaw); !bytes.Equal(req.pb, want) {
		t.Errorf("SET_PROFILE_SETTINGS request = %x, want %x", req.pb, want)
	}
	req.reply <- pendingReply{body: testResponse(4, nil)}
	waitWithoutReadback(t, d, done)

	if applied.TouchZone == nil || applied.TouchZone.Touchpad == nil ||
		applied.TouchZone.Touchpad.Enable == nil || !*applied.TouchZone.Touchpad.Enable {
		t.Fatalf("SetProfileSettings() = %+v, want merged touchpad enable", applied)
	}
	if !bytes.Equal(applied.Raw, wantRaw) {
		t.Fatalf("SetProfileSettings() raw = %x, want merged bytes %x", applied.Raw, wantRaw)
	}
	roundTrip, err := applied.Bytes()
	if err != nil {
		t.Fatalf("applied ProfileSettings.Bytes() error = %v", err)
	}
	if !bytes.Equal(roundTrip, wantRaw) {
		t.Fatalf("applied round-trip = %x, want %x", roundTrip, wantRaw)
	}
}

func TestSetGlobalReturnsMergedViewWithoutReadback(t *testing.T) {
	d := connectedTestDaemon()
	global := append(testVarint(9, 0), testVarint(21, 300)...)
	profileRaw := testBytes(3, testBytes(1, testVarint(1, 1)))
	app := append(testBytes(1, global), testBytes(2, profileRaw)...)
	app = append(app, testVarint(3, 7)...)
	enabled := true
	patch := &cproto.Global{FnLock: &enabled}
	want, err := cproto.DecodeSettings(app)
	if err != nil {
		t.Fatalf("DecodeSettings() error = %v", err)
	}
	want.Global.Merge(patch)
	wantApp, err := want.Encode()
	if err != nil {
		t.Fatalf("Settings.Encode() error = %v", err)
	}

	var applied *cproto.Settings
	done := make(chan error, 1)
	go func() {
		var err error
		applied, err = d.SetGlobal(patch)
		done <- err
	}()

	req := nextPending(t, d)
	if want := cproto.EncodeGetSettings(); !bytes.Equal(req.pb, want) {
		t.Errorf("first SetGlobal request = %x, want GET_SETTINGS %x", req.pb, want)
	}
	getBody := testBytes(2, app)
	req.reply <- pendingReply{body: testResponse(2, getBody)}

	req = nextPending(t, d)
	if want := cproto.EncodeSetSettings(wantApp); !bytes.Equal(req.pb, want) {
		t.Errorf("second SetGlobal request = %x, want SET_SETTINGS %x", req.pb, want)
	}
	req.reply <- pendingReply{body: testResponse(3, nil)}
	waitWithoutReadback(t, d, done)

	if applied.Global == nil || applied.Global.FnLock == nil || !*applied.Global.FnLock {
		t.Fatalf("SetGlobal() = %+v, want merged fnLock", applied)
	}
	if applied.Global.BacklightTime == nil || *applied.Global.BacklightTime != 300 {
		t.Fatalf("SetGlobal() backlightTime = %v, want unchanged 300", applied.Global.BacklightTime)
	}
	roundTrip, err := applied.Encode()
	if err != nil {
		t.Fatalf("applied Settings.Encode() error = %v", err)
	}
	if !bytes.Equal(roundTrip, wantApp) {
		t.Fatalf("applied round-trip = %x, want %x", roundTrip, wantApp)
	}
}

func connectedTestDaemon() *Daemon {
	d := New("test")
	d.status.Connected = true
	return d
}

func nextPending(t *testing.T, d *Daemon) pendingRequest {
	t.Helper()
	select {
	case req := <-d.requests:
		return req
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for device exchange")
		return pendingRequest{}
	}
}

func waitWithoutReadback(t *testing.T, d *Daemon, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("write error = %v", err)
		}
	case req := <-d.requests:
		req.reply <- pendingReply{err: errors.New("unexpected read-back")}
		<-done
		t.Fatal("write issued an unexpected read-back exchange")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for write result")
	}
}

func testResponse(field protowire.Number, body []byte) []byte {
	return testBytes(field, body)
}

func testBytes(field protowire.Number, body []byte) []byte {
	var out []byte
	out = protowire.AppendTag(out, field, protowire.BytesType)
	return protowire.AppendBytes(out, body)
}

func testVarint(field protowire.Number, value uint64) []byte {
	var out []byte
	out = protowire.AppendTag(out, field, protowire.VarintType)
	return protowire.AppendVarint(out, value)
}
