package proto

import (
	"bytes"
	"encoding/hex"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

// Captured from a live CLVX over BLE: GET_SETTINGS reply, AppSettings.global.
// Field mapping verified against app-settings.ts GlobalSettings.encode.
const capturedGlobalHex = "10011801200128003005380140004800600170017801800101880100a0011ea801ac02b00101c00100c80100d00100d80100"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad fixture hex: %v", err)
	}
	return b
}

func TestDecodeGlobalCaptured(t *testing.T) {
	g, err := DecodeGlobal(mustHex(t, capturedGlobalHex))
	if err != nil {
		t.Fatalf("DecodeGlobal: %v", err)
	}
	checks := []struct {
		name string
		got  *bool
		want bool
	}{
		{"tap1fEnable", g.Tap1fEnable, true},
		{"tap2fEnable", g.Tap2fEnable, true},
		{"holdEnable", g.HoldEnable, true},
		{"swapClickButtons", g.SwapClickButtons, false},
		{"newbieModeEnable", g.NewbieModeEnable, true},
		{"touchActivationAfterLiftOff", g.TouchActivationAfterLiftOff, false},
		{"fnLock", g.FnLock, false},
		{"batterySavingModeEnable", g.BatterySavingModeEnable, true},
		{"keySuppressorEnable", g.KeySuppressorEnable, true},
		{"holdDelayOnBorderEnable", g.HoldDelayOnBorderEnable, true},
		{"swapFnCtrl", g.SwapFnCtrl, false},
		{"autoUsbSwitchEnable", g.AutoUsbSwitchEnable, true},
		{"typeOnlyLongSwitchEnable", g.TypeOnlyLongSwitchEnable, false},
		{"ecoBrightness", g.EcoBrightness, false},
	}
	for _, c := range checks {
		if c.got == nil {
			t.Errorf("%s: not decoded", c.name)
			continue
		}
		if *c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, *c.got, c.want)
		}
	}
	nums := []struct {
		name string
		got  *uint32
		want uint32
	}{
		{"currentAILevel", g.CurrentAILevel, 5},
		{"idleBacklightTime", g.IdleBacklightTime, 30},
		{"backlightTime", g.BacklightTime, 300},
		{"brightness", g.Brightness, 0},
	}
	for _, c := range nums {
		if c.got == nil {
			t.Errorf("%s: not decoded", c.name)
			continue
		}
		if *c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, *c.got, c.want)
		}
	}
	if g.DominantHand == nil || *g.DominantHand != "RIGHT" {
		t.Errorf("dominantHand = %v, want RIGHT", g.DominantHand)
	}
	if g.Tap1fMouseButton == nil || *g.Tap1fMouseButton != "LEFT_BUTTON" {
		t.Errorf("tap1fMouseButton = %v, want LEFT_BUTTON", g.Tap1fMouseButton)
	}
}

func TestGlobalRoundTrip(t *testing.T) {
	want := mustHex(t, capturedGlobalHex)
	g, err := DecodeGlobal(want)
	if err != nil {
		t.Fatalf("DecodeGlobal: %v", err)
	}
	got, err := g.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("re-encode mismatch\n got %s\nwant %s", hex.EncodeToString(got), capturedGlobalHex)
	}
}

func TestGlobalMergeOnlyTouchesPatchFields(t *testing.T) {
	g, err := DecodeGlobal(mustHex(t, capturedGlobalHex))
	if err != nil {
		t.Fatalf("DecodeGlobal: %v", err)
	}
	fnLock := false
	brightness := uint32(42)
	g.Merge(&Global{FnLock: &fnLock, Brightness: &brightness})

	if g.FnLock == nil || *g.FnLock {
		t.Errorf("fnLock not patched")
	}
	if g.Brightness == nil || *g.Brightness != 42 {
		t.Errorf("brightness not patched")
	}
	// Untouched fields keep the keyboard's values.
	if g.BacklightTime == nil || *g.BacklightTime != 300 {
		t.Errorf("backlightTime lost: %v", g.BacklightTime)
	}
	if g.DominantHand == nil || *g.DominantHand != "RIGHT" {
		t.Errorf("dominantHand lost: %v", g.DominantHand)
	}
	if g.Tap1fEnable == nil || !*g.Tap1fEnable {
		t.Errorf("tap1fEnable lost: %v", g.Tap1fEnable)
	}
}

func TestGlobalEncodeRejectsUnknownEnum(t *testing.T) {
	name := "SIDEWAYS"
	if _, err := (&Global{DominantHand: &name}).Encode(); err == nil {
		t.Fatal("expected error for unknown enum")
	}
}

func TestSettingsPreservesProfileBytes(t *testing.T) {
	// AppSettings{global, globalProfile=raw, counter=7}
	profile := []byte{0x08, 0x01, 0x12, 0x00, 0x1a, 0x00, 0x22, 0x00}
	var app []byte
	app = protowire.AppendTag(app, appGlobal, protowire.BytesType)
	app = protowire.AppendBytes(app, mustHex(t, capturedGlobalHex))
	app = protowire.AppendTag(app, appGlobalProfile, protowire.BytesType)
	app = protowire.AppendBytes(app, profile)
	app = protowire.AppendTag(app, appCounter, protowire.VarintType)
	app = protowire.AppendVarint(app, 7)

	s, err := DecodeSettings(app)
	if err != nil {
		t.Fatalf("DecodeSettings: %v", err)
	}
	if !bytes.Equal(s.ProfileRaw, profile) {
		t.Fatalf("profile bytes not preserved: %s", hex.EncodeToString(s.ProfileRaw))
	}
	if s.Counter == nil || *s.Counter != 7 {
		t.Fatalf("counter = %v, want 7", s.Counter)
	}
	reencoded, err := s.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(reencoded, app) {
		t.Fatalf("AppSettings round-trip mismatch\n got %s\nwant %s",
			hex.EncodeToString(reencoded), hex.EncodeToString(app))
	}
}

func TestSetSettingsEnvelope(t *testing.T) {
	app := []byte{0x0a, 0x02, 0x10, 0x01} // global{tap1fEnable:true}
	req := EncodeSetSettings(app)

	// Request: field 1 varint type=1 (SET_SETTINGS), field 3 bytes setSettings.
	got, err := DecodeRequestForTest(req)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if got.reqType != ReqSetSettings {
		t.Errorf("type = %d, want %d", got.reqType, ReqSetSettings)
	}
	if got.field != reqField[ReqSetSettings] {
		t.Errorf("oneof field = %d, want %d", got.field, reqField[ReqSetSettings])
	}
	// SetSettingsRequest.settings is field 1 and must carry AppSettings verbatim.
	if !bytes.Contains(got.body, app) {
		t.Errorf("payload does not carry AppSettings: %s", hex.EncodeToString(got.body))
	}
}

func TestParseSettingsResponse(t *testing.T) {
	app := []byte{0x0a, 0x02, 0x10, 0x01}
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.VarintType) // status OK
	body = protowire.AppendVarint(body, SettingsStatusOK)
	body = protowire.AppendTag(body, 2, protowire.BytesType)
	body = protowire.AppendBytes(body, app)

	s, status, err := ParseSettingsResponse(Response{Type: 0, Field: reqField[ReqGetSettings], Body: body})
	if err != nil {
		t.Fatalf("ParseSettingsResponse: %v", err)
	}
	if status != SettingsStatusOK {
		t.Errorf("status = %d, want %d", status, SettingsStatusOK)
	}
	if s.Global == nil || s.Global.Tap1fEnable == nil || !*s.Global.Tap1fEnable {
		t.Errorf("global not decoded: %+v", s.Global)
	}
}

func TestParseSettingsResponseValidationErrorCarriesSettings(t *testing.T) {
	app := []byte{0x0a, 0x02, 0x10, 0x00}
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.VarintType)
	body = protowire.AppendVarint(body, SettingsStatusValidationError)
	body = protowire.AppendTag(body, 2, protowire.BytesType)
	body = protowire.AppendBytes(body, app)

	_, status, err := ParseSettingsResponse(Response{Field: reqField[ReqSetSettings], Body: body})
	if err != nil {
		t.Fatalf("ParseSettingsResponse: %v", err)
	}
	if status != SettingsStatusValidationError {
		t.Errorf("status = %d, want %d", status, SettingsStatusValidationError)
	}
}

func TestParseSettingsResponseStatusOnly(t *testing.T) {
	// A successful SET_SETTINGS replies with the status and no settings body.
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.VarintType)
	body = protowire.AppendVarint(body, SettingsStatusOK)

	s, status, err := ParseSettingsResponse(Response{Field: reqField[ReqSetSettings], Body: body})
	if err != nil {
		t.Fatalf("ParseSettingsResponse: %v", err)
	}
	if s != nil {
		t.Errorf("settings = %+v, want nil for a status-only reply", s)
	}
	if status != SettingsStatusOK {
		t.Errorf("status = %d, want %d", status, SettingsStatusOK)
	}
}

type decodedEnvelope struct {
	reqType uint32
	field   protowire.Number
	body    []byte
}

// DecodeRequestForTest mirrors DecodeResponse for the Request envelope so the
// test can assert the bytes we put on the wire.
func DecodeRequestForTest(pb []byte) (decodedEnvelope, error) {
	var out decodedEnvelope
	for len(pb) > 0 {
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return out, protowire.ParseError(n)
		}
		pb = pb[n:]
		if typ == protowire.VarintType {
			v, m := protowire.ConsumeVarint(pb)
			if m < 0 {
				return out, protowire.ParseError(m)
			}
			if num == 1 {
				out.reqType = uint32(v)
			}
			pb = pb[m:]
			continue
		}
		m := protowire.ConsumeFieldValue(num, typ, pb)
		if m < 0 {
			return out, protowire.ParseError(m)
		}
		v, err := bytesValue(pb[:m])
		if err != nil {
			return out, err
		}
		out.field = num
		out.body = v
		pb = pb[m:]
	}
	return out, nil
}
