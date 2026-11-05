package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

// Captured from get_profile_settings over the daemon's BLE connection on
// 2026-11-05. Keeping the fixture in the test makes the byte-exact regression
// independent of the keyboard's current settings.
const liveProfileBase64 = "EjQaGAoQCgIKABICCgAaAgoAIgIKABIECgIKACIYChAKAgoAEgIKABoCCgAiAgoAEgQKAgoAGkwKBggBEAEYARJCGgQQA2AAIjoQA3o2ChkKBQgBEOkBEgQIABAAGgQIABAAIgQIABAAEhkKBQgBEOoBEgQIABAAGgQIABAAIgQIABAAImUKHAoJCgcIABBkGP8BEg0SBwgAEAAY/wEYASADOAASHgoLCgkI/wEQ9AEYswESDRIHCAAQ/wEYABgBIAI4ABoSCgoKCAj/ARD/ARgAEgIYATgeIhEKCQoHCGQQABj/ARICGAE4HirUAQrRAZIBGyoZCgUIABDjARIECAAQGRoECAAQACIECAAQAJoBGyoZCgUIABDjARIECAAQCBoECAAQACIECAAQAKIBGyoZCgUIARC2ARIECAAQABoECAAQACIECAAQAKoBGyoZCgUIARDNARIECAAQABoECAAQACIECAAQALIBGyoZCgUIARC1ARIECAAQABoECAAQACIECAAQALoBGyoZCgUIARDiARIECAAQABoECAAQACIECAAQAMIBGioYCgQIABBGEgQIABAAGgQIABAAIgQIABAA"

func liveProfileBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(liveProfileBase64)
	if err != nil {
		t.Fatalf("base64.DecodeString(live profile) error = %v", err)
	}
	return raw
}

func liveProfileThroughJSON(t *testing.T) *ProfileSettings {
	t.Helper()
	p, err := DecodeProfile(liveProfileBytes(t))
	if err != nil {
		t.Fatalf("DecodeProfile(live profile) error = %v", err)
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal(live profile) error = %v", err)
	}
	var roundTrip ProfileSettings
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal(live profile) error = %v", err)
	}
	return &roundTrip
}

func mutateLiveProfile(t *testing.T, mutate func(*ProfileSettings)) ([]byte, []byte, *ProfileSettings) {
	t.Helper()
	original := liveProfileBytes(t)
	p := liveProfileThroughJSON(t)
	mutate(p)
	encoded, err := p.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() mutation error = %v", err)
	}
	decoded, err := DecodeProfile(encoded)
	if err != nil {
		t.Fatalf("DecodeProfile(mutated profile) error = %v", err)
	}
	return original, encoded, decoded
}

func assertSingleByteChange(t *testing.T, original, encoded []byte, wantOld, wantNew byte) {
	t.Helper()
	if len(encoded) != len(original) {
		t.Fatalf("ProfileSettings.Bytes() mutated length = %d, want %d", len(encoded), len(original))
	}
	differences := 0
	for i := range original {
		if original[i] == encoded[i] {
			continue
		}
		differences++
		if original[i] != wantOld || encoded[i] != wantNew {
			t.Fatalf("ProfileSettings.Bytes() byte %d = %02x -> %02x, want %02x -> %02x", i, original[i], encoded[i], wantOld, wantNew)
		}
	}
	if differences != 1 {
		t.Fatalf("ProfileSettings.Bytes() changed %d bytes, want exactly 1\n got %x\nwant %x", differences, encoded, original)
	}
}

func assertProfileRestored(t *testing.T, p *ProfileSettings, restore func(), original []byte) {
	t.Helper()
	restore()
	encoded, err := p.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() restore error = %v", err)
	}
	if !bytes.Equal(encoded, original) {
		t.Fatalf("ProfileSettings.Bytes() restored profile mismatch\n got %x\nwant %x", encoded, original)
	}
}

func TestProfileLiveCaptureRoundTripByteExact(t *testing.T) {
	original := liveProfileBytes(t)
	decoded, err := DecodeProfile(original)
	if err != nil {
		t.Fatalf("DecodeProfile(live profile) error = %v", err)
	}
	first, err := decoded.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() first round-trip error = %v", err)
	}
	if !bytes.Equal(first, original) {
		t.Fatalf("ProfileSettings.Bytes() first round-trip mismatch\n got %x\nwant %x", first, original)
	}
	redecoded, err := DecodeProfile(first)
	if err != nil {
		t.Fatalf("DecodeProfile(first round-trip) error = %v", err)
	}
	second, err := redecoded.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() second round-trip error = %v", err)
	}
	if !bytes.Equal(second, original) {
		t.Fatalf("ProfileSettings.Bytes() second round-trip mismatch\n got %x\nwant %x", second, original)
	}
	throughJSON := liveProfileThroughJSON(t)
	fromRPC, err := throughJSON.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() JSON round-trip error = %v", err)
	}
	if !bytes.Equal(fromRPC, original) {
		t.Fatalf("ProfileSettings.Bytes() JSON round-trip mismatch\n got %x\nwant %x", fromRPC, original)
	}
}

func TestProfileMutateRightSliderSensitivity(t *testing.T) {
	original, _, decoded := mutateLiveProfile(t, func(p *ProfileSettings) {
		p.TouchZone.Slider.Right.Sensitivity = uint32Ptr(4)
	})
	encoded := decoded.Raw
	assertSingleByteChange(t, original, encoded, 3, 4)
	if got := *decoded.TouchZone.Slider.Right.Sensitivity; got != 4 {
		t.Fatalf("DecodeProfile() right sensitivity = %d, want 4", got)
	}
	if got := *decoded.TouchZone.Slider.Left.Sensitivity; got != 3 {
		t.Fatalf("DecodeProfile() left sensitivity = %d, want unchanged 3", got)
	}
	assertProfileRestored(t, decoded, func() {
		decoded.TouchZone.Slider.Right.Sensitivity = uint32Ptr(3)
	}, original)
}

func TestProfileMutateFKeyVariant(t *testing.T) {
	original, _, decoded := mutateLiveProfile(t, func(p *ProfileSettings) {
		action := p.Keyboard.FKey.F6
		action.CustomShortcut = action.PredefinedShortcut
		action.PredefinedShortcut = nil
	})
	encoded := decoded.Raw
	assertSingleByteChange(t, original, encoded, 0x2a, 0x0a)
	action := decoded.Keyboard.FKey.F6
	if action.CustomShortcut == nil || action.PredefinedShortcut != nil {
		t.Fatalf("DecodeProfile() F6 action = %+v, want only customShortcut", action)
	}
	assertProfileRestored(t, decoded, func() {
		action := decoded.Keyboard.FKey.F6
		action.PredefinedShortcut = action.CustomShortcut
		action.CustomShortcut = nil
	}, original)
}

func TestProfileMutateZoneTransparency(t *testing.T) {
	original, _, decoded := mutateLiveProfile(t, func(p *ProfileSettings) {
		p.Backlight.LeftSlider.Transparency = uint32Ptr(31)
	})
	encoded := decoded.Raw
	assertSingleByteChange(t, original, encoded, 30, 31)
	if got := *decoded.Backlight.LeftSlider.Transparency; got != 31 {
		t.Fatalf("DecodeProfile() left slider transparency = %d, want 31", got)
	}
	if got := *decoded.Backlight.RightSlider.Transparency; got != 30 {
		t.Fatalf("DecodeProfile() right slider transparency = %d, want unchanged 30", got)
	}
	assertProfileRestored(t, decoded, func() {
		decoded.Backlight.LeftSlider.Transparency = uint32Ptr(30)
	}, original)
}

func TestProfileMutateBacklightColor(t *testing.T) {
	original, _, decoded := mutateLiveProfile(t, func(p *ProfileSettings) {
		p.Backlight.Keyboard.SolidColor.Color.Red = uint32Ptr(1)
	})
	encoded := decoded.Raw
	assertSingleByteChange(t, original, encoded, 0, 1)
	if got := *decoded.Backlight.Keyboard.SolidColor.Color.Red; got != 1 {
		t.Fatalf("DecodeProfile() keyboard red = %d, want 1", got)
	}
	if got := *decoded.Backlight.Keyboard.SolidColor.Color.Green; got != 100 {
		t.Fatalf("DecodeProfile() keyboard green = %d, want unchanged 100", got)
	}
	assertProfileRestored(t, decoded, func() {
		decoded.Backlight.Keyboard.SolidColor.Color.Red = uint32Ptr(0)
	}, original)
}

func TestProfileMutationRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*ProfileSettings)
		wantField string
	}{
		{
			name: "sensitivity below range",
			mutate: func(p *ProfileSettings) {
				p.TouchZone.Slider.Right.Sensitivity = uint32Ptr(0)
			},
			wantField: "touchZone.slider.right.sensitivity",
		},
		{
			name: "sensitivity above range",
			mutate: func(p *ProfileSettings) {
				p.TouchZone.Slider.Right.Sensitivity = uint32Ptr(6)
			},
			wantField: "touchZone.slider.right.sensitivity",
		},
		{
			name: "zone transparency above range",
			mutate: func(p *ProfileSettings) {
				p.Backlight.LeftSlider.Transparency = uint32Ptr(101)
			},
			wantField: "backlight.leftSlider.transparency",
		},
		{
			name: "marker transparency above range",
			mutate: func(p *ProfileSettings) {
				p.Backlight.Keyboard.SolidColor = nil
				p.Backlight.Keyboard.ColorCycle = &ColorCycle{
					ColorLinePicker: &ColorLinePicker{
						MarkersNumber: 1,
						MarkersArray:  []*Marker{{Transparency: uint32Ptr(101)}},
					},
				}
			},
			wantField: "backlight.keyboard.colorCycle.colorLinePicker.markersArray[0].transparency",
		},
		{
			name: "RGB above range",
			mutate: func(p *ProfileSettings) {
				p.Backlight.Keyboard.SolidColor.Color.Red = uint32Ptr(256)
			},
			wantField: "backlight.keyboard.solidColor.color.red",
		},
		{
			name: "slider has two variants",
			mutate: func(p *ProfileSettings) {
				p.TouchZone.Slider.Right.Nothing = &EmptyAction{}
			},
			wantField: "touchZone.slider.right has 2 action variants",
		},
		{
			name: "slider has no variant",
			mutate: func(p *ProfileSettings) {
				p.TouchZone.Slider.Right.PredefinedShortcut = nil
			},
			wantField: "touchZone.slider.right has 0 action variants",
		},
		{
			name: "key action has two variants",
			mutate: func(p *ProfileSettings) {
				p.Keyboard.FKey.F6.Nothing = &EmptyAction{}
			},
			wantField: "keyboard.fKey.F6 has 2 action variants",
		},
		{
			name: "key action has no variant",
			mutate: func(p *ProfileSettings) {
				p.Keyboard.FKey.F6.PredefinedShortcut = nil
			},
			wantField: "keyboard.fKey.F6 has 0 action variants",
		},
		{
			name: "zone has two variants",
			mutate: func(p *ProfileSettings) {
				p.Backlight.Keyboard.Breathing = &Breathing{}
			},
			wantField: "backlight.keyboard has 2 lighting effect variants",
		},
		{
			name: "zone has no variant",
			mutate: func(p *ProfileSettings) {
				p.Backlight.Keyboard.SolidColor = nil
			},
			wantField: "backlight.keyboard has 0 lighting effect variants",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := liveProfileThroughJSON(t)
			tt.mutate(p)
			encoded, err := p.Bytes()
			if err == nil {
				t.Fatalf("ProfileSettings.Bytes(%s) error = nil, want rejection", tt.name)
			}
			if encoded != nil {
				t.Fatalf("ProfileSettings.Bytes(%s) = %x on error, want no partial bytes", tt.name, encoded)
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("ProfileSettings.Bytes(%s) error = %q, want field %q", tt.name, err, tt.wantField)
			}
		})
	}
}

func TestProfileMutationRejectsIllegalFKeyField(t *testing.T) {
	p, err := DecodeProfile(liveProfileBytes(t))
	if err != nil {
		t.Fatalf("DecodeProfile(live profile) error = %v", err)
	}
	p.Keyboard.FKey.Raw = append(p.Keyboard.FKey.Raw, testVarintField(25, 1)...)
	action := p.Keyboard.FKey.F6
	action.CustomShortcut = action.PredefinedShortcut
	action.PredefinedShortcut = nil
	encoded, err := p.Bytes()
	if err == nil {
		t.Fatal("ProfileSettings.Bytes(F-key field 25) error = nil, want rejection")
	}
	if encoded != nil {
		t.Fatalf("ProfileSettings.Bytes(F-key field 25) = %x on error, want no partial bytes", encoded)
	}
	if !strings.Contains(err.Error(), "keyboard.fKey contains field 25") {
		t.Fatalf("ProfileSettings.Bytes(F-key field 25) error = %q, want offending field", err)
	}
}

func TestProfileMutationPreservesUnknownSubfields(t *testing.T) {
	p, err := DecodeProfile(liveProfileBytes(t))
	if err != nil {
		t.Fatalf("DecodeProfile(live profile) error = %v", err)
	}
	color := p.Backlight.Keyboard.SolidColor.Color
	color.Raw = append(color.Raw, testVarintField(99, 7)...)
	color.Red = uint32Ptr(1)
	encoded, err := p.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes() with unknown color field error = %v", err)
	}
	decoded, err := DecodeProfile(encoded)
	if err != nil {
		t.Fatalf("DecodeProfile(mutated profile) error = %v", err)
	}
	found := false
	err = walkFields(decoded.Backlight.Keyboard.SolidColor.Color.Raw, func(num protowire.Number, _ protowire.Type, body []byte) error {
		if num == 99 {
			value, ok := fieldVarint(body)
			found = ok && value == 7
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walkFields(mutated color) error = %v", err)
	}
	if !found {
		t.Fatal("ProfileSettings.Bytes() unknown color field 99 = missing, want value 7")
	}
}
