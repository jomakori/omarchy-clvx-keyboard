package proto

import (
	"bytes"
	"encoding/json"
	"testing"
)

func actionColorProfileRaw(t *testing.T, keyboardRGB, touchpadRGB [3]uint32, animations bool) []byte {
	t.Helper()
	color := func(rgb [3]uint32) []byte {
		return testMessage(
			testVarintField(1, uint64(rgb[0])),
			testVarintField(2, uint64(rgb[1])),
			testVarintField(3, uint64(rgb[2])),
		)
	}
	solid := func(rgb [3]uint32) []byte {
		return testBytesField(1, testBytesField(1, color(rgb)))
	}
	keyboard := solid([3]uint32{9, 8, 7})
	touchpad := solid([3]uint32{6, 5, 4})
	if animations {
		keyboardAnimation := testMessage(
			testBytesField(2, color(keyboardRGB)),
			testVarintField(3, 0),
			testVarintField(4, 4),
		)
		touchpadAnimation := testMessage(
			testBytesField(2, color(touchpadRGB)),
			testVarintField(3, 1),
			testVarintField(4, 3),
		)
		keyboard = testMessage(keyboard, testBytesField(2, keyboardAnimation))
		touchpad = testMessage(touchpad, testBytesField(2, touchpadAnimation))
	}
	leftSlider := testMessage(
		solid([3]uint32{1, 2, 3}),
		testBytesField(2, testVarintField(3, 1)),
		testVarintField(7, 40),
	)
	rightSlider := testMessage(
		solid([3]uint32{3, 2, 1}),
		testBytesField(2, testVarintField(3, 0)),
		testVarintField(7, 50),
	)
	backlight := testMessage(
		testBytesField(1, keyboard),
		testBytesField(2, touchpad),
		testBytesField(3, leftSlider),
		testBytesField(4, rightSlider),
	)
	leftAction := testMessage(testVarintField(2, 4), testVarintField(12, 0))
	rightAction := testMessage(testVarintField(2, 2), testBytesField(13, nil))
	touchZone := testBytesField(2, testMessage(
		testBytesField(3, leftAction),
		testBytesField(4, rightAction),
	))
	gestures := testMessage(testVarintField(90, 1), testBytesField(91, []byte{2, 3, 4}))
	return testMessage(
		testBytesField(2, gestures),
		testBytesField(3, touchZone),
		testBytesField(4, backlight),
	)
}

func actionColorProfileThroughJSON(t *testing.T, raw []byte) *ProfileSettings {
	t.Helper()
	p, err := DecodeProfile(raw)
	if err != nil {
		t.Fatalf("DecodeProfile(action color fixture) error = %v", err)
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal(action color fixture) error = %v", err)
	}
	var roundTrip ProfileSettings
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal(action color fixture) error = %v", err)
	}
	return &roundTrip
}

func TestProfileBytesPinsInteractiveAnimationColors(t *testing.T) {
	keyboardOrange := [3]uint32{255, 128, 0}
	touchpadBlue := [3]uint32{0, 0, 255}
	keyboardBlue := [3]uint32{0, 0, 255}
	touchpadGreen := [3]uint32{0, 255, 0}
	wrongRaw := actionColorProfileRaw(t, keyboardOrange, touchpadBlue, true)
	pinnedRaw := actionColorProfileRaw(t, keyboardBlue, touchpadGreen, true)
	tests := []struct {
		name     string
		settings func(*testing.T) *ProfileSettings
	}{
		{
			name: "typed edit",
			settings: func(t *testing.T) *ProfileSettings {
				return actionColorProfileThroughJSON(t, wrongRaw)
			},
		},
		{
			name: "whole subtree replace",
			settings: func(t *testing.T) *ProfileSettings {
				p := actionColorProfileThroughJSON(t, pinnedRaw)
				replacement, err := DecodeProfile(wrongRaw)
				if err != nil {
					t.Fatalf("DecodeProfile(wrong subtree) error = %v", err)
				}
				p.Backlight = replacement.Backlight
				return p
			},
		},
		{
			name: "raw restore",
			settings: func(t *testing.T) *ProfileSettings {
				return &ProfileSettings{RawBytes: rawBytes(wrongRaw)}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.settings(t).Bytes()
			if err != nil {
				t.Fatalf("ProfileSettings.Bytes(%s) error = %v", tt.name, err)
			}
			got, err := DecodeProfile(encoded)
			if err != nil {
				t.Fatalf("DecodeProfile(%s encoded profile) error = %v", tt.name, err)
			}
			assertActionColor(t, tt.name+" keyboard", got.Backlight.Keyboard.InteractiveAnimation.Color, keyboardBlue)
			assertActionColor(t, tt.name+" touchpad", got.Backlight.Touchpad.InteractiveAnimation.Color, touchpadGreen)
			if got := *got.Backlight.Keyboard.InteractiveAnimation.Enable; got {
				t.Errorf("ProfileSettings.Bytes(%s) keyboard enable = %t, want false", tt.name, got)
			}
			if got := *got.Backlight.Keyboard.InteractiveAnimation.Duration; got != 4 {
				t.Errorf("ProfileSettings.Bytes(%s) keyboard duration = %d, want 4", tt.name, got)
			}
			if got := *got.Backlight.Touchpad.InteractiveAnimation.Enable; !got {
				t.Errorf("ProfileSettings.Bytes(%s) touchpad enable = %t, want true", tt.name, got)
			}
			if got := *got.Backlight.Touchpad.InteractiveAnimation.Trace; got != 3 {
				t.Errorf("ProfileSettings.Bytes(%s) touchpad trace = %d, want 3", tt.name, got)
			}
			want, err := DecodeProfile(wrongRaw)
			if err != nil {
				t.Fatalf("DecodeProfile(%s original profile) error = %v", tt.name, err)
			}
			assertRawEqual(t, tt.name+" gestures", got.Gestures.Raw, want.Gestures.Raw)
			assertRawEqual(t, tt.name+" sliders", got.TouchZone.Slider.Raw, want.TouchZone.Slider.Raw)
			assertRawEqual(t, tt.name+" left slider backlight", got.Backlight.LeftSlider.Raw, want.Backlight.LeftSlider.Raw)
			assertRawEqual(t, tt.name+" right slider backlight", got.Backlight.RightSlider.Raw, want.Backlight.RightSlider.Raw)
		})
	}
}

func TestProfileBytesDoesNotCreateInteractiveAnimations(t *testing.T) {
	raw := actionColorProfileRaw(t, [3]uint32{}, [3]uint32{}, false)
	p := actionColorProfileThroughJSON(t, raw)
	p.Backlight.Keyboard.SolidColor.Color.Red = uint32Ptr(10)
	encoded, err := p.Bytes()
	if err != nil {
		t.Fatalf("ProfileSettings.Bytes(profile without animations) error = %v", err)
	}
	got, err := DecodeProfile(encoded)
	if err != nil {
		t.Fatalf("DecodeProfile(profile without animations) error = %v", err)
	}
	if got.Backlight.Keyboard.InteractiveAnimation != nil {
		t.Errorf("ProfileSettings.Bytes(profile without animations) keyboard animation = %+v, want nil", got.Backlight.Keyboard.InteractiveAnimation)
	}
	if got.Backlight.Touchpad.InteractiveAnimation != nil {
		t.Errorf("ProfileSettings.Bytes(profile without animations) touchpad animation = %+v, want nil", got.Backlight.Touchpad.InteractiveAnimation)
	}
}

func assertActionColor(t *testing.T, name string, got *Color, want [3]uint32) {
	t.Helper()
	if got == nil || got.Red == nil || got.Green == nil || got.Blue == nil {
		t.Fatalf("DecodeProfile(%s) color = %+v, want RGB %v", name, got, want)
	}
	gotRGB := [3]uint32{*got.Red, *got.Green, *got.Blue}
	if gotRGB != want {
		t.Errorf("DecodeProfile(%s) color = %v, want %v", name, gotRGB, want)
	}
}

func assertRawEqual(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("ProfileSettings.Bytes(%s) raw = %x, want unchanged %x", name, got, want)
	}
}
