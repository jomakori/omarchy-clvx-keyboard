package proto

import "google.golang.org/protobuf/encoding/protowire"

// BacklightSettings contains independent keyboard, touchpad, and slider zones.
// Every message retains Raw so writes can round-trip newer firmware fields
// without understanding them.
type BacklightSettings struct {
	RawBytes
	Keyboard    *KeyboardBacklight `json:"keyboard,omitempty"`
	Touchpad    *TouchpadBacklight `json:"touchpad,omitempty"`
	LeftSlider  *SliderBacklight   `json:"leftSlider,omitempty"`
	RightSlider *SliderBacklight   `json:"rightSlider,omitempty"`
}

// BacklightZoneEffects contains the effects shared by every backlight zone.
type BacklightZoneEffects struct {
	SolidColor   *SolidColor `json:"solidColor,omitempty"`
	Breathing    *Breathing  `json:"breathing,omitempty"`
	ColorCycle   *ColorCycle `json:"colorCycle,omitempty"`
	ColorWave    *ColorWave  `json:"colorWave,omitempty"`
	Transparency *uint32     `json:"transparency,omitempty"`
}

// KeyboardBacklight contains the keyboard zone's lighting settings.
type KeyboardBacklight struct {
	RawBytes
	BacklightZoneEffects
	InteractiveAnimation *InteractiveAnimation `json:"interactiveAnimation,omitempty"`
	Aurora               *EmptyAction          `json:"aurora,omitempty"`
}

// TouchpadBacklight contains the touchpad zone's lighting settings.
type TouchpadBacklight struct {
	RawBytes
	BacklightZoneEffects
	InteractiveAnimation *TouchpadInteractiveAnimation `json:"interactiveAnimation,omitempty"`
	Aurora               *EmptyAction                  `json:"aurora,omitempty"`
}

// SliderBacklight contains one slider zone's lighting settings.
type SliderBacklight struct {
	RawBytes
	BacklightZoneEffects
	InteractiveAnimation *SliderInteractiveAnimation `json:"interactiveAnimation,omitempty"`
}

// SolidColor is a static-color lighting effect.
type SolidColor struct {
	RawBytes
	Color *Color `json:"color,omitempty"`
}

// Color is an RGB color with optional components.
type Color struct {
	RawBytes
	Red   *uint32 `json:"red,omitempty"`
	Green *uint32 `json:"green,omitempty"`
	Blue  *uint32 `json:"blue,omitempty"`
}

// Breathing is a periodic fading-color lighting effect.
type Breathing struct {
	RawBytes
	Color  *Color  `json:"color,omitempty"`
	Period *uint32 `json:"period,omitempty"`
}

// ColorCycle cycles through a configurable line of colors.
type ColorCycle struct {
	RawBytes
	ColorLinePicker *ColorLinePicker `json:"colorLinePicker,omitempty"`
	Period          *uint32          `json:"period,omitempty"`
}

// ColorWave animates a configurable line of colors in a direction.
type ColorWave struct {
	RawBytes
	ColorLinePicker *ColorLinePicker `json:"colorLinePicker,omitempty"`
	Period          *uint32          `json:"period,omitempty"`
	Direction       *uint32          `json:"direction,omitempty"`
	Length          *uint32          `json:"length,omitempty"`
}

// ColorLinePicker describes the markers used by cycle and wave effects.
type ColorLinePicker struct {
	RawBytes
	MarkersNumber uint32    `json:"markersNumber"`
	MarkersArray  []*Marker `json:"markersArray,omitempty"`
}

// Marker is a positioned color in a color line.
type Marker struct {
	RawBytes
	Color        *Color  `json:"color,omitempty"`
	Transparency *uint32 `json:"transparency,omitempty"`
	Position     *uint32 `json:"position,omitempty"`
}

// InteractiveAnimation configures the keyboard's touch-reactive lighting.
type InteractiveAnimation struct {
	RawBytes
	Color    *Color  `json:"color,omitempty"`
	Enable   *bool   `json:"enable,omitempty"`
	Duration *uint32 `json:"duration,omitempty"`
}

// TouchpadInteractiveAnimation configures the touchpad's touch-reactive lighting.
type TouchpadInteractiveAnimation struct {
	RawBytes
	Color  *Color  `json:"color,omitempty"`
	Enable *bool   `json:"enable,omitempty"`
	Trace  *uint32 `json:"trace,omitempty"`
}

// SliderInteractiveAnimation configures a slider's touch-reactive lighting.
type SliderInteractiveAnimation struct {
	RawBytes
	Enable *bool `json:"enable,omitempty"`
}

// DecodeBacklight parses backlight settings while preserving the original bytes.
func DecodeBacklight(pb []byte) (*BacklightSettings, error) {
	b := &BacklightSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1:
			return decodeChild(body, func(pb []byte) error {
				var err error
				b.Keyboard, err = decodeKeyboardBacklight(pb)
				return err
			})
		case 2:
			return decodeChild(body, func(pb []byte) error {
				var err error
				b.Touchpad, err = decodeTouchpadBacklight(pb)
				return err
			})
		case 3:
			return decodeChild(body, func(pb []byte) error {
				var err error
				b.LeftSlider, err = decodeSliderBacklight(pb)
				return err
			})
		case 4:
			return decodeChild(body, func(pb []byte) error {
				var err error
				b.RightSlider, err = decodeSliderBacklight(pb)
				return err
			})
		}
		return nil
	})
	return b, err
}

// decodeZoneEffects handles fields shared by all zones. Other fields are
// dispatched to the zone-specific callback; unknown fields are ignored.
func decodeZoneEffects(pb []byte, effects *BacklightZoneEffects, special func(protowire.Number, protowire.Type, []byte) error) error {
	return walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		switch num {
		case 1:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					effects.SolidColor, err = decodeSolidColor(b)
					return err
				})
			}
		case 4:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					effects.Breathing, err = decodeBreathing(b)
					return err
				})
			}
		case 5:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					effects.ColorCycle, err = decodeColorCycle(b)
					return err
				})
			}
		case 6:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					effects.ColorWave, err = decodeColorWave(b)
					return err
				})
			}
		case 7:
			if typ == protowire.VarintType {
				if v, ok := fieldVarint(body); ok {
					effects.Transparency = uint32Ptr(v)
				}
			}
		default:
			return special(num, typ, body)
		}
		return nil
	})
}

func decodeKeyboardBacklight(pb []byte) (*KeyboardBacklight, error) {
	k := &KeyboardBacklight{RawBytes: rawBytes(pb)}
	err := decodeZoneEffects(pb, &k.BacklightZoneEffects, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				k.InteractiveAnimation, err = decodeInteractiveAnimation(b)
				return err
			})
		}
		if num == 8 && typ == protowire.BytesType {
			e, err := decodeEmpty(body)
			k.Aurora = e
			return err
		}
		return nil
	})
	return k, err
}

func decodeTouchpadBacklight(pb []byte) (*TouchpadBacklight, error) {
	t := &TouchpadBacklight{RawBytes: rawBytes(pb)}
	err := decodeZoneEffects(pb, &t.BacklightZoneEffects, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				t.InteractiveAnimation, err = decodeTouchpadInteractive(b)
				return err
			})
		}
		if num == 8 && typ == protowire.BytesType {
			e, err := decodeEmpty(body)
			t.Aurora = e
			return err
		}
		return nil
	})
	return t, err
}

func decodeSliderBacklight(pb []byte) (*SliderBacklight, error) {
	s := &SliderBacklight{RawBytes: rawBytes(pb)}
	err := decodeZoneEffects(pb, &s.BacklightZoneEffects, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				s.InteractiveAnimation, err = decodeSliderInteractive(b)
				return err
			})
		}
		return nil
	})
	return s, err
}

func decodeSolidColor(pb []byte) (*SolidColor, error) {
	s := &SolidColor{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				s.Color, err = DecodeColor(b)
				return err
			})
		}
		return nil
	})
	return s, err
}

// DecodeColor parses an RGB color while preserving the original bytes.
func DecodeColor(pb []byte) (*Color, error) {
	c := &Color{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType || num < 1 || num > 3 {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		switch num {
		case 1:
			c.Red = uint32Ptr(v)
		case 2:
			c.Green = uint32Ptr(v)
		case 3:
			c.Blue = uint32Ptr(v)
		}
		return nil
	})
	return c, err
}

func decodeBreathing(pb []byte) (*Breathing, error) {
	b := &Breathing{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(pb []byte) error {
				var err error
				b.Color, err = DecodeColor(pb)
				return err
			})
		}
		if num == 2 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				b.Period = uint32Ptr(v)
			}
		}
		return nil
	})
	return b, err
}

func decodeColorCycle(pb []byte) (*ColorCycle, error) {
	c := &ColorCycle{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				c.ColorLinePicker, err = decodeColorLinePicker(b)
				return err
			})
		}
		if num == 2 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				c.Period = uint32Ptr(v)
			}
		}
		return nil
	})
	return c, err
}

func decodeColorWave(pb []byte) (*ColorWave, error) {
	c := &ColorWave{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				c.ColorLinePicker, err = decodeColorLinePicker(b)
				return err
			})
		}
		if typ == protowire.VarintType {
			v, ok := fieldVarint(body)
			if !ok {
				return nil
			}
			switch num {
			case 2:
				c.Period = uint32Ptr(v)
			case 3:
				c.Direction = uint32Ptr(v)
			case 4:
				c.Length = uint32Ptr(v)
			}
		}
		return nil
	})
	return c, err
}

func decodeColorLinePicker(pb []byte) (*ColorLinePicker, error) {
	p := &ColorLinePicker{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				p.MarkersNumber = uint32(v)
			}
		}
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				m, err := decodeMarker(b)
				p.MarkersArray = append(p.MarkersArray, m)
				return err
			})
		}
		return nil
	})
	return p, err
}

func decodeMarker(pb []byte) (*Marker, error) {
	m := &Marker{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				m.Color, err = DecodeColor(b)
				return err
			})
		}
		if typ == protowire.VarintType {
			v, ok := fieldVarint(body)
			if !ok {
				return nil
			}
			if num == 2 {
				m.Transparency = uint32Ptr(v)
			} else if num == 3 {
				m.Position = uint32Ptr(v)
			}
		}
		return nil
	})
	return m, err
}

func decodeInteractiveAnimation(pb []byte) (*InteractiveAnimation, error) {
	a := &InteractiveAnimation{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				a.Color, err = DecodeColor(b)
				return err
			})
		}
		if typ == protowire.VarintType {
			v, ok := fieldVarint(body)
			if !ok {
				return nil
			}
			if num == 3 {
				a.Enable = boolPtr(v)
			} else if num == 4 {
				a.Duration = uint32Ptr(v)
			}
		}
		return nil
	})
	return a, err
}

func decodeTouchpadInteractive(pb []byte) (*TouchpadInteractiveAnimation, error) {
	a := &TouchpadInteractiveAnimation{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				a.Color, err = DecodeColor(b)
				return err
			})
		}
		if typ == protowire.VarintType {
			v, ok := fieldVarint(body)
			if !ok {
				return nil
			}
			if num == 3 {
				a.Enable = boolPtr(v)
			} else if num == 4 {
				a.Trace = uint32Ptr(v)
			}
		}
		return nil
	})
	return a, err
}

func decodeSliderInteractive(pb []byte) (*SliderInteractiveAnimation, error) {
	a := &SliderInteractiveAnimation{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 3 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				a.Enable = boolPtr(v)
			}
		}
		return nil
	})
	return a, err
}
