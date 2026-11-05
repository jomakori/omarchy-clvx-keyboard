package proto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/encoding/protowire"
)

var rawBytesType = reflect.TypeOf(RawBytes{})

// inheritRawBytes restores the nested raw views omitted by profile JSON. The
// top-level raw profile is decoded before a daemon write, so every unchanged
// subtree can still be emitted verbatim after a JSON round-trip.
func inheritRawBytes(current, original any) {
	inheritRawValue(reflect.ValueOf(current), reflect.ValueOf(original))
}

func inheritRawValue(current, original reflect.Value) {
	if !current.IsValid() || !original.IsValid() || current.Type() != original.Type() {
		return
	}
	switch current.Kind() {
	case reflect.Pointer:
		if current.IsNil() || original.IsNil() {
			return
		}
		inheritRawValue(current.Elem(), original.Elem())
	case reflect.Struct:
		if current.Type() == rawBytesType {
			currentRaw := current.FieldByName("Raw")
			originalRaw := original.FieldByName("Raw")
			if currentRaw.CanSet() && currentRaw.IsNil() && !originalRaw.IsNil() {
				currentRaw.Set(originalRaw)
			}
			return
		}
		for i := range current.NumField() {
			inheritRawValue(current.Field(i), original.Field(i))
		}
	case reflect.Slice:
		if current.IsNil() || original.IsNil() {
			return
		}
		n := min(current.Len(), original.Len())
		for i := range n {
			inheritRawValue(current.Index(i), original.Index(i))
		}
	}
}

func decodedEqual(current, original any) (bool, error) {
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("proto: marshal current decoded settings: %w", err)
	}
	originalJSON, err := json.Marshal(original)
	if err != nil {
		return false, fmt.Errorf("proto: marshal original decoded settings: %w", err)
	}
	return bytes.Equal(currentJSON, originalJSON), nil
}

func originalUnchanged[T any](current *T, raw []byte, decode func([]byte) (*T, error), path string) (bool, error) {
	if raw == nil {
		return false, nil
	}
	original, err := decode(raw)
	if err != nil {
		return false, fmt.Errorf("proto: decode original %s: %w", path, err)
	}
	unchanged, err := decodedEqual(current, original)
	if err != nil {
		return false, err
	}
	return unchanged, nil
}

func appendVarintField(out []byte, num protowire.Number, value *uint32) []byte {
	if value == nil {
		return out
	}
	out = protowire.AppendTag(out, num, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(*value))
}

func appendBoolField(out []byte, num protowire.Number, value *bool) []byte {
	if value == nil {
		return out
	}
	var encoded uint64
	if *value {
		encoded = 1
	}
	out = protowire.AppendTag(out, num, protowire.VarintType)
	return protowire.AppendVarint(out, encoded)
}

func appendBytesField(out []byte, num protowire.Number, body []byte) []byte {
	out = protowire.AppendTag(out, num, protowire.BytesType)
	return protowire.AppendBytes(out, body)
}

func appendMessageField[T any](out []byte, num protowire.Number, value *T, path string, encode func(*T, string) ([]byte, error)) ([]byte, error) {
	if value == nil {
		return out, nil
	}
	body, err := encode(value, path)
	if err != nil {
		return nil, err
	}
	return appendBytesField(out, num, body), nil
}

// appendUnknownFields retains fields the typed decoder did not claim. It
// copies their complete wire representation, including non-canonical tags and
// values, rather than decoding and re-encoding them.
func appendUnknownFields(out, raw []byte, path string, known ...protowire.Number) ([]byte, error) {
	for len(raw) > 0 {
		field := raw
		num, typ, n := protowire.ConsumeTag(raw)
		if n < 0 {
			return nil, fmt.Errorf("proto: %s: parse original field tag: %w", path, protowire.ParseError(n))
		}
		raw = raw[n:]
		m := protowire.ConsumeFieldValue(num, typ, raw)
		if m < 0 {
			return nil, fmt.Errorf("proto: %s: parse original field %d: %w", path, num, protowire.ParseError(m))
		}
		isKnown := false
		for _, candidate := range known {
			if num == candidate {
				isKnown = true
				break
			}
		}
		if !isKnown {
			out = append(out, field[:n+m]...)
		}
		raw = raw[m:]
	}
	return out, nil
}

func encodeProfile(p, original *ProfileSettings) ([]byte, error) {
	var out []byte
	var err error
	// Gesture settings are decode-only. Always retain the device original,
	// including when another profile subtree is edited.
	if original.Gestures != nil {
		out = appendBytesField(out, 2, original.Gestures.Raw)
	}
	out, err = appendMessageField(out, 3, p.TouchZone, "touchZone", encodeTouchZone)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, p.Backlight, "backlight", encodeBacklight)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 5, p.Keyboard, "keyboard", encodeKeyboard)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, p.Raw, "profile", 1, 2, 3, 4, 5)
}

func encodeTouchZone(t *TouchZoneSettings, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(t, t.Raw, DecodeTouchZone, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return t.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, t.Touchpad, path+".touchpad", encodeTouchpadSettings)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, t.Slider, path+".slider", encodeSliderSettings)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, t.Raw, path, 1, 2)
}

func encodeTouchpadSettings(t *TouchpadSettings, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(t, t.Raw, decodeTouchpadSettings, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return t.Raw, nil
	}
	out := appendBoolField(nil, 1, t.Enable)
	out = appendBoolField(out, 2, t.OneFingerEnable)
	out = appendBoolField(out, 3, t.TwoFingerEnable)
	return appendUnknownFields(out, t.Raw, path, 1, 2, 3)
}

func encodeSliderSettings(s *SliderSettings, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(s, s.Raw, decodeSliderSettings, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return s.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 3, s.Left, path+".left", encodeSlider)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, s.Right, path+".right", encodeSlider)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, s.Raw, path, 3, 4)
}

func encodeSlider(s *Slider, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(s, s.Raw, decodeSlider, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return s.Raw, nil
	}
	if s.Sensitivity != nil && (*s.Sensitivity < 1 || *s.Sensitivity > 5) {
		return nil, fmt.Errorf("proto: %s.sensitivity = %d, want a value from 1 through 5", path, *s.Sensitivity)
	}
	variants := variantCount(
		s.CustomShortcut != nil,
		s.Custom != nil,
		s.Nothing != nil,
		s.AsGlobal != nil,
		s.PredefinedShortcut != nil,
	)
	if variants != 1 {
		return nil, fmt.Errorf("proto: %s has %d action variants, want exactly one", path, variants)
	}
	out := appendVarintField(nil, 2, s.Sensitivity)
	out, err = appendMessageField(out, 11, s.CustomShortcut, path+".customShortcut", encodeSliderShortcut)
	if err != nil {
		return nil, err
	}
	out = appendVarintField(out, 12, s.Custom)
	out, err = appendMessageField(out, 13, s.Nothing, path+".nothing", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 14, s.AsGlobal, path+".asGlobal", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 15, s.PredefinedShortcut, path+".predefinedShortcut", encodeSliderShortcut)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, s.Raw, path, 2, 11, 12, 13, 14, 15)
}

func encodeSliderShortcut(s *SliderShortcut, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(s, s.Raw, decodeSliderShortcut, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return s.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, s.Inc, path+".inc", encodeKeyCombination)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, s.Dec, path+".dec", encodeKeyCombination)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, s.Raw, path, 1, 2)
}

func encodeKeyboard(k *KeyboardSettings, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(k, k.Raw, DecodeKeyboard, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return k.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, k.FKey, path+".fKey", encodeFKeys)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, k.Raw, path, 1)
}

func encodeFKeys(f *FKeySettings, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(f, f.Raw, decodeFKeys, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return f.Raw, nil
	}
	if err := validateFKeyFields(f.Raw, path); err != nil {
		return nil, err
	}
	actions := []*KeyAction{f.F1, f.F2, f.F3, f.F4, f.F5, f.F6, f.F7, f.F8, f.F9, f.F10, f.F11, f.F12}
	var out []byte
	for i, action := range actions {
		fieldPath := fmt.Sprintf("%s.F%d", path, i+1)
		out, err = appendMessageField(out, protowire.Number(13+i), action, fieldPath, encodeKeyAction)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func validateFKeyFields(raw []byte, path string) error {
	return walkFields(raw, func(num protowire.Number, _ protowire.Type, _ []byte) error {
		if num < 13 || num > 24 {
			return fmt.Errorf("proto: %s contains field %d, want only F-key fields 13 through 24", path, num)
		}
		return nil
	})
}

func encodeKeyAction(a *KeyAction, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(a, a.Raw, decodeKeyAction, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return a.Raw, nil
	}
	variants := variantCount(
		a.CustomShortcut != nil,
		a.Custom != nil,
		a.Nothing != nil,
		a.AsGlobal != nil,
		a.PredefinedShortcut != nil,
	)
	if variants != 1 {
		return nil, fmt.Errorf("proto: %s has %d action variants, want exactly one", path, variants)
	}
	var out []byte
	out, err = appendMessageField(out, 1, a.CustomShortcut, path+".customShortcut", encodeKeyCombination)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, a.Custom, path+".custom", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 3, a.Nothing, path+".nothing", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, a.AsGlobal, path+".asGlobal", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 5, a.PredefinedShortcut, path+".predefinedShortcut", encodeKeyCombination)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, a.Raw, path, 1, 2, 3, 4, 5)
}

func encodeKeyCombination(k *KeyCombination, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(k, k.Raw, DecodeKeyCombination, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return k.Raw, nil
	}
	keys := []*Key{k.Key1, k.Key2, k.Key3, k.Key4}
	var out []byte
	for i, key := range keys {
		fieldPath := fmt.Sprintf("%s.key%d", path, i+1)
		out, err = appendMessageField(out, protowire.Number(i+1), key, fieldPath, encodeKey)
		if err != nil {
			return nil, err
		}
	}
	return appendUnknownFields(out, k.Raw, path, 1, 2, 3, 4)
}

func encodeKey(k *Key, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(k, k.Raw, decodeKey, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return k.Raw, nil
	}
	out := appendVarintField(nil, 1, k.Type)
	out = appendVarintField(out, 2, k.Code)
	return appendUnknownFields(out, k.Raw, path, 1, 2)
}

func encodeEmptyAction(e *EmptyAction, _ string) ([]byte, error) {
	if e.Raw != nil {
		return e.Raw, nil
	}
	return nil, nil
}

func variantCount(selected ...bool) int {
	var count int
	for _, present := range selected {
		if present {
			count++
		}
	}
	return count
}
