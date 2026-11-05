package proto

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func encodeBacklight(b *BacklightSettings, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(b, b.Raw, DecodeBacklight, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return b.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, b.Keyboard, path+".keyboard", encodeKeyboardBacklight)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, b.Touchpad, path+".touchpad", encodeTouchpadBacklight)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 3, b.LeftSlider, path+".leftSlider", encodeSliderBacklight)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, b.RightSlider, path+".rightSlider", encodeSliderBacklight)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, b.Raw, path, 1, 2, 3, 4)
}

func encodeKeyboardBacklight(k *KeyboardBacklight, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(k, k.Raw, decodeKeyboardBacklight, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return k.Raw, nil
	}
	if err := validateZoneEffects(&k.BacklightZoneEffects, k.Aurora, path); err != nil {
		return nil, err
	}
	var out []byte
	out, err = appendMessageField(out, 1, k.SolidColor, path+".solidColor", encodeSolidColor)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, k.InteractiveAnimation, path+".interactiveAnimation", encodeInteractiveAnimation)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, k.Breathing, path+".breathing", encodeBreathing)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 5, k.ColorCycle, path+".colorCycle", encodeColorCycle)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 6, k.ColorWave, path+".colorWave", encodeColorWave)
	if err != nil {
		return nil, err
	}
	if k.Transparency != nil {
		out = appendVarintField(out, 7, k.Transparency)
	}
	out, err = appendMessageField(out, 8, k.Aurora, path+".aurora", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, k.Raw, path, 1, 2, 4, 5, 6, 7, 8)
}

func encodeTouchpadBacklight(t *TouchpadBacklight, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(t, t.Raw, decodeTouchpadBacklight, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return t.Raw, nil
	}
	if err := validateZoneEffects(&t.BacklightZoneEffects, t.Aurora, path); err != nil {
		return nil, err
	}
	var out []byte
	out, err = appendMessageField(out, 1, t.SolidColor, path+".solidColor", encodeSolidColor)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, t.InteractiveAnimation, path+".interactiveAnimation", encodeTouchpadInteractiveAnimation)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, t.Breathing, path+".breathing", encodeBreathing)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 5, t.ColorCycle, path+".colorCycle", encodeColorCycle)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 6, t.ColorWave, path+".colorWave", encodeColorWave)
	if err != nil {
		return nil, err
	}
	if t.Transparency != nil {
		out = appendVarintField(out, 7, t.Transparency)
	}
	out, err = appendMessageField(out, 8, t.Aurora, path+".aurora", encodeEmptyAction)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, t.Raw, path, 1, 2, 4, 5, 6, 7, 8)
}

func encodeSliderBacklight(s *SliderBacklight, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(s, s.Raw, decodeSliderBacklight, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return s.Raw, nil
	}
	if err := validateZoneEffects(&s.BacklightZoneEffects, nil, path); err != nil {
		return nil, err
	}
	var out []byte
	out, err = appendMessageField(out, 1, s.SolidColor, path+".solidColor", encodeSolidColor)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 2, s.InteractiveAnimation, path+".interactiveAnimation", encodeSliderInteractiveAnimation)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 4, s.Breathing, path+".breathing", encodeBreathing)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 5, s.ColorCycle, path+".colorCycle", encodeColorCycle)
	if err != nil {
		return nil, err
	}
	out, err = appendMessageField(out, 6, s.ColorWave, path+".colorWave", encodeColorWave)
	if err != nil {
		return nil, err
	}
	if s.Transparency != nil {
		out = appendVarintField(out, 7, s.Transparency)
	}
	return appendUnknownFields(out, s.Raw, path, 1, 2, 4, 5, 6, 7)
}

func validateZoneEffects(effects *BacklightZoneEffects, aurora *EmptyAction, path string) error {
	variants := variantCount(
		effects.SolidColor != nil,
		effects.Breathing != nil,
		effects.ColorCycle != nil,
		effects.ColorWave != nil,
		aurora != nil,
	)
	if variants != 1 {
		return fmt.Errorf("proto: %s has %d lighting effect variants, want exactly one", path, variants)
	}
	return validateTransparency(effects.Transparency, path+".transparency")
}

func validateTransparency(value *uint32, path string) error {
	if value != nil && *value > 100 {
		return fmt.Errorf("proto: %s = %d, want a value from 0 through 100", path, *value)
	}
	return nil
}

func encodeSolidColor(s *SolidColor, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(s, s.Raw, decodeSolidColor, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return s.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, s.Color, path+".color", encodeColor)
	if err != nil {
		return nil, err
	}
	return appendUnknownFields(out, s.Raw, path, 1)
}

func encodeColor(c *Color, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(c, c.Raw, DecodeColor, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return c.Raw, nil
	}
	for _, component := range []struct {
		name  string
		value *uint32
	}{
		{name: "red", value: c.Red},
		{name: "green", value: c.Green},
		{name: "blue", value: c.Blue},
	} {
		if component.value != nil && *component.value > 255 {
			return nil, fmt.Errorf("proto: %s.%s = %d, want a value from 0 through 255", path, component.name, *component.value)
		}
	}
	out := appendVarintField(nil, 1, c.Red)
	out = appendVarintField(out, 2, c.Green)
	out = appendVarintField(out, 3, c.Blue)
	return appendUnknownFields(out, c.Raw, path, 1, 2, 3)
}

func encodeBreathing(b *Breathing, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(b, b.Raw, decodeBreathing, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return b.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, b.Color, path+".color", encodeColor)
	if err != nil {
		return nil, err
	}
	out = appendVarintField(out, 2, b.Period)
	return appendUnknownFields(out, b.Raw, path, 1, 2)
}

func encodeColorCycle(c *ColorCycle, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(c, c.Raw, decodeColorCycle, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return c.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, c.ColorLinePicker, path+".colorLinePicker", encodeColorLinePicker)
	if err != nil {
		return nil, err
	}
	out = appendVarintField(out, 2, c.Period)
	return appendUnknownFields(out, c.Raw, path, 1, 2)
}

func encodeColorWave(c *ColorWave, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(c, c.Raw, decodeColorWave, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return c.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 1, c.ColorLinePicker, path+".colorLinePicker", encodeColorLinePicker)
	if err != nil {
		return nil, err
	}
	out = appendVarintField(out, 2, c.Period)
	out = appendVarintField(out, 3, c.Direction)
	out = appendVarintField(out, 4, c.Length)
	return appendUnknownFields(out, c.Raw, path, 1, 2, 3, 4)
}

func encodeColorLinePicker(p *ColorLinePicker, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(p, p.Raw, decodeColorLinePicker, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return p.Raw, nil
	}
	out := protowire.AppendTag(nil, 1, protowire.VarintType)
	out = protowire.AppendVarint(out, uint64(p.MarkersNumber))
	for i, marker := range p.MarkersArray {
		fieldPath := fmt.Sprintf("%s.markersArray[%d]", path, i)
		out, err = appendMessageField(out, 2, marker, fieldPath, encodeMarker)
		if err != nil {
			return nil, err
		}
	}
	return appendUnknownFields(out, p.Raw, path, 1, 2)
}

func encodeMarker(m *Marker, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(m, m.Raw, decodeMarker, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return m.Raw, nil
	}
	if err := validateTransparency(m.Transparency, path+".transparency"); err != nil {
		return nil, err
	}
	var out []byte
	out, err = appendMessageField(out, 1, m.Color, path+".color", encodeColor)
	if err != nil {
		return nil, err
	}
	out = appendVarintField(out, 2, m.Transparency)
	out = appendVarintField(out, 3, m.Position)
	return appendUnknownFields(out, m.Raw, path, 1, 2, 3)
}

func encodeInteractiveAnimation(a *InteractiveAnimation, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(a, a.Raw, decodeInteractiveAnimation, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return a.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 2, a.Color, path+".color", encodeColor)
	if err != nil {
		return nil, err
	}
	out = appendBoolField(out, 3, a.Enable)
	out = appendVarintField(out, 4, a.Duration)
	return appendUnknownFields(out, a.Raw, path, 2, 3, 4)
}

func encodeTouchpadInteractiveAnimation(a *TouchpadInteractiveAnimation, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(a, a.Raw, decodeTouchpadInteractive, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return a.Raw, nil
	}
	var out []byte
	out, err = appendMessageField(out, 2, a.Color, path+".color", encodeColor)
	if err != nil {
		return nil, err
	}
	out = appendBoolField(out, 3, a.Enable)
	out = appendVarintField(out, 4, a.Trace)
	return appendUnknownFields(out, a.Raw, path, 2, 3, 4)
}

func encodeSliderInteractiveAnimation(a *SliderInteractiveAnimation, path string) ([]byte, error) {
	unchanged, err := originalUnchanged(a, a.Raw, decodeSliderInteractive, path)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return a.Raw, nil
	}
	out := appendBoolField(nil, 3, a.Enable)
	return appendUnknownFields(out, a.Raw, path, 3)
}
