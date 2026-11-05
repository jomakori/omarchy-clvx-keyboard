package proto

import (
	"bytes"
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func testBytesField(num protowire.Number, body []byte) []byte {
	var out []byte
	out = protowire.AppendTag(out, num, protowire.BytesType)
	return protowire.AppendBytes(out, body)
}

func testVarintField(num protowire.Number, value uint64) []byte {
	var out []byte
	out = protowire.AppendTag(out, num, protowire.VarintType)
	return protowire.AppendVarint(out, value)
}

func testMessage(fields ...[]byte) []byte { return bytes.Join(fields, nil) }

func TestBacklightUnknownFieldsAndRawRoundTrip(t *testing.T) {
	color := testMessage(testVarintField(1, 255), testVarintField(2, 12), testVarintField(3, 3), testVarintField(99, 7))
	solid := testBytesField(1, color)
	interactive := testMessage(testBytesField(2, color), testVarintField(3, 1), testVarintField(4, 5))
	marker := testMessage(testBytesField(1, color), testVarintField(2, 50), testVarintField(3, 75))
	picker := testMessage(testVarintField(1, 1), testBytesField(2, marker), testBytesField(2, marker))
	wave := testMessage(testBytesField(1, picker), testVarintField(2, 900), testVarintField(3, 180), testVarintField(4, 400))
	keyboard := testMessage(testBytesField(1, solid), testBytesField(2, interactive), testBytesField(6, wave), testBytesField(8, nil), testBytesField(99, []byte{1, 2, 3}))
	touchpadInteractive := testMessage(testBytesField(2, color), testVarintField(3, 0), testVarintField(4, 2))
	touchpad := testMessage(testBytesField(2, touchpadInteractive), testVarintField(7, 42))
	raw := testMessage(testBytesField(1, keyboard), testBytesField(2, touchpad), testVarintField(99, 123))

	b, err := DecodeBacklight(raw)
	if err != nil {
		t.Fatalf("DecodeBacklight: %v", err)
	}
	if !bytes.Equal(b.Raw, raw) {
		t.Fatal("raw backlight bytes changed")
	}
	if b.Keyboard == nil || b.Keyboard.SolidColor == nil || b.Keyboard.SolidColor.Color == nil || *b.Keyboard.SolidColor.Color.Red != 255 {
		t.Fatalf("keyboard solid color not decoded: %+v", b.Keyboard)
	}
	if b.Keyboard.InteractiveAnimation == nil || b.Keyboard.InteractiveAnimation.Duration == nil || *b.Keyboard.InteractiveAnimation.Duration != 5 {
		t.Fatalf("keyboard animation not decoded: %+v", b.Keyboard.InteractiveAnimation)
	}
	if b.Keyboard.ColorWave == nil || len(b.Keyboard.ColorWave.ColorLinePicker.MarkersArray) != 2 || *b.Keyboard.ColorWave.Direction != 180 {
		t.Fatalf("color wave not decoded: %+v", b.Keyboard.ColorWave)
	}
	if b.Keyboard.Aurora == nil {
		t.Fatal("aurora not decoded")
	}
	if b.Touchpad == nil || b.Touchpad.InteractiveAnimation == nil || *b.Touchpad.InteractiveAnimation.Trace != 2 || *b.Touchpad.Transparency != 42 {
		t.Fatalf("touchpad backlight not decoded: %+v", b.Touchpad)
	}
}

func TestSettingsTypedCustomizationDoesNotChangeWriteBytes(t *testing.T) {
	// Profile fixture includes every customization group and unknown fields at
	// multiple levels. The exact AppSettings bytes must survive SetGlobal's
	// read-modify-write path, independently of the decoded JSON views.
	profile := testMessage(
		testVarintField(1, 0),
		testBytesField(2, testBytesField(3, testBytesField(1, testMessage(testBytesField(1, testBytesField(4, nil)), testVarintField(99, 1))))),
		testBytesField(3, testBytesField(1, testVarintField(1, 1))),
		testBytesField(4, testBytesField(2, testVarintField(7, 20))),
		testBytesField(5, testBytesField(1, testBytesField(24, testBytesField(4, nil)))),
		testVarintField(99, 1234),
	)
	app := testMessage(testBytesField(appGlobal, mustHex(t, capturedGlobalHex)), testBytesField(appGlobalProfile, profile), testVarintField(appCounter, 9))
	s, err := DecodeSettings(app)
	if err != nil {
		t.Fatalf("DecodeSettings: %v", err)
	}
	if s.Gestures == nil || s.TouchZone == nil || s.Backlight == nil || s.Keyboard == nil {
		t.Fatalf("missing customization views: %+v", s)
	}
	if s.Keyboard.FKey == nil || s.Keyboard.FKey.F12 == nil || s.Keyboard.FKey.F12.AsGlobal == nil {
		t.Fatalf("F12 action not decoded: %+v", s.Keyboard)
	}
	got, err := s.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(got, app) {
		t.Fatalf("settings write changed profile fixture\n got %x\nwant %x", got, app)
	}
}

func TestGestureTouchZoneAndFRowFixtures(t *testing.T) {
	key := testMessage(testVarintField(1, 0), testVarintField(2, 4))
	combination := testBytesField(1, key)
	shortcut := testMessage(testBytesField(1, combination), testVarintField(2, 7), testVarintField(3, 1), testBytesField(4, combination))
	up := testBytesField(2, shortcut)
	tap := testBytesField(1, testBytesField(4, nil))
	gesture := testMessage(testBytesField(1, testBytesField(1, up)), testBytesField(2, tap))
	gesturesRaw := testMessage(testBytesField(3, gesture), testBytesField(4, nil), testVarintField(90, 1))
	g, err := DecodeGestures(gesturesRaw)
	if err != nil {
		t.Fatal(err)
	}
	if g.ThreeFinger.Swipe.Up.CustomShortcut.Direct.Key1.Code == nil || *g.ThreeFinger.Swipe.Up.CustomShortcut.Direct.Key1.Code != 4 {
		t.Fatalf("gesture shortcut not decoded: %+v", g.ThreeFinger)
	}
	if g.ThreeFinger.Tap.Action.Nothing == nil || !bytes.Equal(g.Raw, gesturesRaw) {
		t.Fatal("tap action or raw gestures lost")
	}

	sliderShortcut := testMessage(testBytesField(1, combination), testBytesField(2, combination))
	left := testMessage(testVarintField(2, 4), testBytesField(11, sliderShortcut), testVarintField(12, 0))
	zonesRaw := testMessage(testBytesField(1, testMessage(testVarintField(1, 1), testVarintField(2, 0), testVarintField(3, 1))), testBytesField(2, testMessage(testBytesField(3, left), testBytesField(4, testBytesField(13, nil)))))
	zones, err := DecodeTouchZone(zonesRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !*zones.Touchpad.Enable || *zones.Touchpad.OneFingerEnable || *zones.Slider.Left.Sensitivity != 4 || zones.Slider.Left.CustomShortcut.Inc.Key1 == nil || zones.Slider.Right.Nothing == nil {
		t.Fatalf("touch zones not decoded: %+v", zones)
	}
}

func TestProfileEnvelopeAndResponses(t *testing.T) {
	req, err := DecodeRequestForTest(EncodeGetProfileSettings(0))
	if err != nil {
		t.Fatal(err)
	}
	if req.reqType != ReqGetProfileSettings || req.field != 7 || len(req.body) != 0 {
		t.Fatalf("bad get profile envelope: %+v", req)
	}
	profileRaw := testMessage(testBytesField(4, testBytesField(3, testVarintField(7, 8))))
	req, err = DecodeRequestForTest(EncodeSetProfileSettings(profileRaw))
	if err != nil {
		t.Fatal(err)
	}
	if req.reqType != ReqSetProfileSettings || req.field != 4 || !bytes.Equal(req.body, testBytesField(1, profileRaw)) {
		t.Fatalf("bad set profile envelope: %+v", req)
	}

	getBody := testMessage(testVarintField(1, GetProfileStatusOK), testBytesField(2, profileRaw))
	p, status, err := ParseProfileResponse(Response{Type: 6, Field: 8, Body: getBody})
	if err != nil || status != 0 || p.Backlight.LeftSlider == nil || !bytes.Equal(p.Raw, profileRaw) {
		t.Fatalf("get profile response: p=%+v status=%d err=%v", p, status, err)
	}
	setBody := testMessage(testVarintField(1, SetProfileStatusValidationError), testBytesField(3, profileRaw))
	p, status, err = ParseProfileResponse(Response{Type: 5, Field: 4, Body: setBody})
	if err != nil || status != SetProfileStatusValidationError || p == nil {
		t.Fatalf("set profile response: p=%+v status=%d err=%v", p, status, err)
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip ProfileSettings
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	raw, err := roundTrip.Bytes()
	if err != nil || !bytes.Equal(raw, profileRaw) {
		t.Fatalf("profile JSON/raw round-trip: %x, %v", raw, err)
	}
	roundTrip.Backlight.LeftSlider = nil
	editedRaw, err := roundTrip.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() after typed edit: %v", err)
	}
	edited, err := DecodeProfile(editedRaw)
	if err != nil {
		t.Fatalf("DecodeProfile() after typed edit: %v", err)
	}
	if edited.Backlight == nil || edited.Backlight.LeftSlider != nil {
		t.Fatalf("DecodeProfile() left slider = %+v, want removed", edited.Backlight)
	}
}
