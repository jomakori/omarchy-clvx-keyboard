package proto

import (
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Response status values for GET_PROFILE_SETTINGS.
const (
	GetProfileStatusOK = iota
	GetProfileStatusUnknownError
	GetProfileStatusUnsupportedID
	GetProfileStatusNotFound
)

// Response status values for SET_PROFILE_SETTINGS.
const (
	SetProfileStatusOK = iota
	SetProfileStatusUnknownError
	SetProfileStatusValidationError
	SetProfileStatusUnsupportedID
)

// EmptyAction represents the empty touchpad/nothing/asGlobal/aurora messages.
type EmptyAction struct {
	RawBytes
}

// ProfileSettings is the single working profile. The decoded fields are views;
// Raw is authoritative when the profile is written back.
type ProfileSettings struct {
	RawBytes
	ID        uint32             `json:"id"`
	Gestures  *GestureSettings   `json:"gestures,omitempty"`
	TouchZone *TouchZoneSettings `json:"touchZone,omitempty"`
	Backlight *BacklightSettings `json:"backlight,omitempty"`
	Keyboard  *KeyboardSettings  `json:"keyboard,omitempty"`
}

type profileSettingsJSON struct {
	ID        uint32             `json:"id"`
	Gestures  *GestureSettings   `json:"gestures,omitempty"`
	TouchZone *TouchZoneSettings `json:"touchZone,omitempty"`
	Backlight *BacklightSettings `json:"backlight,omitempty"`
	Keyboard  *KeyboardSettings  `json:"keyboard,omitempty"`
	Raw       []byte             `json:"raw,omitempty"`
}

var (
	_ json.Marshaler   = ProfileSettings{}
	_ json.Unmarshaler = (*ProfileSettings)(nil)
)

// MarshalJSON includes raw bytes so the standalone profile RPC can perform an
// exact no-op round-trip without exposing ProfileRaw in get_settings JSON.
func (p ProfileSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(profileSettingsJSON{
		ID:        p.ID,
		Gestures:  p.Gestures,
		TouchZone: p.TouchZone,
		Backlight: p.Backlight,
		Keyboard:  p.Keyboard,
		Raw:       p.Raw,
	})
}

// UnmarshalJSON restores both the decoded profile views and their raw bytes.
func (p *ProfileSettings) UnmarshalJSON(data []byte) error {
	var v profileSettingsJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	p.ID, p.Gestures, p.TouchZone, p.Backlight, p.Keyboard = v.ID, v.Gestures, v.TouchZone, v.Backlight, v.Keyboard
	p.RawBytes = rawBytes(v.Raw)
	return nil
}

// DecodeProfile parses the working profile while preserving its original bytes.
func DecodeProfile(pb []byte) (*ProfileSettings, error) {
	p := &ProfileSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		switch num {
		case 1:
			if typ == protowire.VarintType {
				if v, ok := fieldVarint(body); ok {
					p.ID = uint32(v)
				}
			}
		case 2:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					p.Gestures, err = DecodeGestures(b)
					return err
				})
			}
		case 3:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					p.TouchZone, err = DecodeTouchZone(b)
					return err
				})
			}
		case 4:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					p.Backlight, err = DecodeBacklight(b)
					return err
				})
			}
		case 5:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) error {
					var err error
					p.Keyboard, err = DecodeKeyboard(b)
					return err
				})
			}
		}
		return nil
	})
	return p, err
}

// Bytes merges typed edits into the original profile. Unchanged subtrees are
// emitted verbatim so fields unknown to this decoder survive every write.
func (p *ProfileSettings) Bytes() ([]byte, error) {
	if p == nil || p.Raw == nil {
		return nil, errors.New("proto: profile has no raw bytes")
	}
	if p.ID != 0 {
		return nil, fmt.Errorf("proto: unsupported profile id %d", p.ID)
	}
	original, err := DecodeProfile(p.Raw)
	if err != nil {
		return nil, fmt.Errorf("proto: decode original profile: %w", err)
	}
	if original.ID != 0 {
		return nil, fmt.Errorf("proto: unsupported original profile id %d", original.ID)
	}
	// A raw-only request remains valid for transport tooling.
	if p.Gestures == nil && p.TouchZone == nil && p.Backlight == nil && p.Keyboard == nil {
		return append([]byte(nil), p.Raw...), nil
	}
	inheritRawBytes(p, original)
	unchanged, err := decodedEqual(p, original)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return append([]byte(nil), p.Raw...), nil
	}
	encoded, err := encodeProfile(p, original)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// EncodeGetProfileSettings builds a request for the selected profile.
func EncodeGetProfileSettings(profileID uint32) []byte {
	var body []byte
	if profileID != 0 {
		body = protowire.AppendTag(body, 1, protowire.VarintType)
		body = protowire.AppendVarint(body, uint64(profileID))
	}
	return encodeEnvelope(ReqGetProfileSettings, body)
}

// EncodeSetProfileSettings builds a request containing an exact raw profile.
func EncodeSetProfileSettings(profileRaw []byte) []byte {
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.BytesType)
	body = protowire.AppendBytes(body, profileRaw)
	return encodeEnvelope(ReqSetProfileSettings, body)
}

// ParseProfileResponse validates and decodes a get or set profile response.
func ParseProfileResponse(r Response) (*ProfileSettings, uint32, error) {
	settingsField := protowire.Number(2)
	switch r.Field {
	case 4: // SET_PROFILE_SETTINGS: settings is field 3 on validation errors.
		settingsField = 3
	case 8: // GET_PROFILE_SETTINGS
	default:
		return nil, 0, fmt.Errorf("proto: response field %d is not a profile reply", r.Field)
	}
	var status uint64
	var raw []byte
	err := walkFields(r.Body, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.VarintType {
			v, err := varintValue(body)
			if err != nil {
				return err
			}
			status = v
		} else if num == settingsField && typ == protowire.BytesType {
			b, err := bytesValue(body)
			if err != nil {
				return err
			}
			raw = b
		}
		return nil
	})
	if err != nil || raw == nil {
		return nil, uint32(status), err
	}
	p, err := DecodeProfile(raw)
	return p, uint32(status), err
}

// GestureSettings and related types mirror app-settings.ts. Empty oneof
// alternatives are retained as {} in JSON, so the selected action is visible.
type GestureSettings struct {
	RawBytes
	ThreeFinger *Gesture `json:"threeFinger,omitempty"`
	FourFinger  *Gesture `json:"fourFinger,omitempty"`
}

// Gesture contains the swipe and tap alternatives for a gesture.
type Gesture struct {
	RawBytes
	Swipe *Swipe `json:"swipe,omitempty"`
	Tap   *Tap   `json:"tap,omitempty"`
}

// Swipe maps each swipe direction to an action.
type Swipe struct {
	RawBytes
	Up    *SwipeAction `json:"Up,omitempty"`
	Down  *SwipeAction `json:"Down,omitempty"`
	Left  *SwipeAction `json:"Left,omitempty"`
	Right *SwipeAction `json:"Right,omitempty"`
}

// SwipeAction is the selected action for one swipe direction.
type SwipeAction struct {
	RawBytes
	Touchpad           *EmptyAction   `json:"touchpad,omitempty"`
	CustomShortcut     *SwipeShortcut `json:"customShortcut,omitempty"`
	Custom             *uint32        `json:"custom,omitempty"`
	Nothing            *EmptyAction   `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction   `json:"asGlobal,omitempty"`
	PredefinedShortcut *SwipeShortcut `json:"predefinedShortcut,omitempty"`
}

// SwipeShortcut contains a swipe shortcut and its continuation settings.
type SwipeShortcut struct {
	RawBytes
	Direct      *KeyCombination `json:"direct,omitempty"`
	Sensitivity *uint32         `json:"sensitivity,omitempty"`
	Continues   *bool           `json:"continues,omitempty"`
	Opposite    *KeyCombination `json:"opposite,omitempty"`
}

// Tap contains the action assigned to a tap gesture.
type Tap struct {
	RawBytes
	Action *TapAction `json:"action,omitempty"`
}

// TapAction is the selected action for a tap gesture.
type TapAction struct {
	RawBytes
	Touchpad           *EmptyAction    `json:"touchpad,omitempty"`
	CustomShortcut     *KeyCombination `json:"customShortcut,omitempty"`
	Custom             *uint32         `json:"custom,omitempty"`
	Nothing            *EmptyAction    `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction    `json:"asGlobal,omitempty"`
	PredefinedShortcut *KeyCombination `json:"predefinedShortcut,omitempty"`
}

// KeyCombination contains up to four simultaneously pressed keys.
type KeyCombination struct {
	RawBytes
	Key1 *Key `json:"key1,omitempty"`
	Key2 *Key `json:"key2,omitempty"`
	Key3 *Key `json:"key3,omitempty"`
	Key4 *Key `json:"key4,omitempty"`
}

// Key identifies a key by protocol type and code.
type Key struct {
	RawBytes
	Type *uint32 `json:"type,omitempty"`
	Code *uint32 `json:"code,omitempty"`
}

// DecodeGestures parses three-finger and four-finger gesture settings.
func DecodeGestures(pb []byte) (*GestureSettings, error) {
	g := &GestureSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || (num != 3 && num != 4) {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			v, err := decodeGesture(b)
			if num == 3 {
				g.ThreeFinger = v
			} else {
				g.FourFinger = v
			}
			return err
		})
	})
	return g, err
}

func decodeGesture(pb []byte) (*Gesture, error) {
	g := &Gesture{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1:
			return decodeChild(body, func(b []byte) error {
				var err error
				g.Swipe, err = decodeSwipe(b)
				return err
			})
		case 2:
			return decodeChild(body, func(b []byte) error {
				var err error
				g.Tap, err = decodeTap(b)
				return err
			})
		}
		return nil
	})
	return g, err
}

func decodeSwipe(pb []byte) (*Swipe, error) {
	s := &Swipe{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || num < 1 || num > 4 {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			a, err := decodeSwipeAction(b)
			switch num {
			case 1:
				s.Up = a
			case 2:
				s.Down = a
			case 3:
				s.Left = a
			case 4:
				s.Right = a
			}
			return err
		})
	})
	return s, err
}

func decodeSwipeAction(pb []byte) (*SwipeAction, error) {
	a := &SwipeAction{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 3 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				a.Custom = uint32Ptr(v)
			}
			return nil
		}
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1, 4, 5:
			e, err := decodeEmpty(body)
			if num == 1 {
				a.Touchpad = e
			} else if num == 4 {
				a.Nothing = e
			} else {
				a.AsGlobal = e
			}
			return err
		case 2, 6:
			return decodeChild(body, func(b []byte) error {
				s, err := decodeSwipeShortcut(b)
				if num == 2 {
					a.CustomShortcut = s
				} else {
					a.PredefinedShortcut = s
				}
				return err
			})
		}
		return nil
	})
	return a, err
}

func decodeSwipeShortcut(pb []byte) (*SwipeShortcut, error) {
	s := &SwipeShortcut{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		switch num {
		case 1, 4:
			if typ != protowire.BytesType {
				return nil
			}
			return decodeChild(body, func(b []byte) error {
				k, err := DecodeKeyCombination(b)
				if num == 1 {
					s.Direct = k
				} else {
					s.Opposite = k
				}
				return err
			})
		case 2:
			if typ == protowire.VarintType {
				if v, ok := fieldVarint(body); ok {
					s.Sensitivity = uint32Ptr(v)
				}
			}
		case 3:
			if typ == protowire.VarintType {
				if v, ok := fieldVarint(body); ok {
					s.Continues = boolPtr(v)
				}
			}
		}
		return nil
	})
	return s, err
}

func decodeTap(pb []byte) (*Tap, error) {
	t := &Tap{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				t.Action, err = decodeTapAction(b)
				return err
			})
		}
		return nil
	})
	return t, err
}

func decodeTapAction(pb []byte) (*TapAction, error) {
	a := &TapAction{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 3 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				a.Custom = uint32Ptr(v)
			}
			return nil
		}
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1, 4, 5:
			e, err := decodeEmpty(body)
			if num == 1 {
				a.Touchpad = e
			} else if num == 4 {
				a.Nothing = e
			} else {
				a.AsGlobal = e
			}
			return err
		case 2, 6:
			return decodeChild(body, func(b []byte) error {
				k, err := DecodeKeyCombination(b)
				if num == 2 {
					a.CustomShortcut = k
				} else {
					a.PredefinedShortcut = k
				}
				return err
			})
		}
		return nil
	})
	return a, err
}

// DecodeKeyCombination parses a key combination while preserving raw bytes.
func DecodeKeyCombination(pb []byte) (*KeyCombination, error) {
	k := &KeyCombination{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || num < 1 || num > 4 {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			key, err := decodeKey(b)
			switch num {
			case 1:
				k.Key1 = key
			case 2:
				k.Key2 = key
			case 3:
				k.Key3 = key
			case 4:
				k.Key4 = key
			}
			return err
		})
	})
	return k, err
}

func decodeKey(pb []byte) (*Key, error) {
	k := &Key{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		if num == 1 {
			k.Type = uint32Ptr(v)
		} else if num == 2 {
			k.Code = uint32Ptr(v)
		}
		return nil
	})
	return k, err
}

func decodeEmpty(body []byte) (*EmptyAction, error) {
	var e *EmptyAction
	err := decodeChild(body, func(b []byte) error {
		e = &EmptyAction{RawBytes: rawBytes(b)}
		return walkFields(b, func(protowire.Number, protowire.Type, []byte) error { return nil })
	})
	return e, err
}

// TouchZoneSettings contains touchpad enables and the two slider actions.
type TouchZoneSettings struct {
	RawBytes
	Touchpad *TouchpadSettings `json:"touchpad,omitempty"`
	Slider   *SliderSettings   `json:"slider,omitempty"`
}

// TouchpadSettings contains the touchpad's gesture enable flags.
type TouchpadSettings struct {
	RawBytes
	Enable          *bool `json:"enable,omitempty"`
	OneFingerEnable *bool `json:"oneFingerEnable,omitempty"`
	TwoFingerEnable *bool `json:"twoFingerEnable,omitempty"`
}

// SliderSettings contains the left and right slider configurations.
type SliderSettings struct {
	RawBytes
	Left  *Slider `json:"left,omitempty"`
	Right *Slider `json:"right,omitempty"`
}

// Slider contains one slider's sensitivity and selected action.
type Slider struct {
	RawBytes
	Sensitivity        *uint32         `json:"sensitivity,omitempty"`
	CustomShortcut     *SliderShortcut `json:"customShortcut,omitempty"`
	Custom             *uint32         `json:"custom,omitempty"`
	Nothing            *EmptyAction    `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction    `json:"asGlobal,omitempty"`
	PredefinedShortcut *SliderShortcut `json:"predefinedShortcut,omitempty"`
}

// SliderShortcut contains the increment and decrement key combinations.
type SliderShortcut struct {
	RawBytes
	Inc *KeyCombination `json:"inc,omitempty"`
	Dec *KeyCombination `json:"dec,omitempty"`
}

// DecodeTouchZone parses touchpad and slider settings.
func DecodeTouchZone(pb []byte) (*TouchZoneSettings, error) {
	t := &TouchZoneSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1:
			return decodeChild(body, func(b []byte) error {
				var err error
				t.Touchpad, err = decodeTouchpadSettings(b)
				return err
			})
		case 2:
			return decodeChild(body, func(b []byte) error {
				var err error
				t.Slider, err = decodeSliderSettings(b)
				return err
			})
		}
		return nil
	})
	return t, err
}

func decodeTouchpadSettings(pb []byte) (*TouchpadSettings, error) {
	t := &TouchpadSettings{RawBytes: rawBytes(pb)}
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
			t.Enable = boolPtr(v)
		case 2:
			t.OneFingerEnable = boolPtr(v)
		case 3:
			t.TwoFingerEnable = boolPtr(v)
		}
		return nil
	})
	return t, err
}

func decodeSliderSettings(pb []byte) (*SliderSettings, error) {
	s := &SliderSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || (num != 3 && num != 4) {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			v, err := decodeSlider(b)
			if num == 3 {
				s.Left = v
			} else {
				s.Right = v
			}
			return err
		})
	})
	return s, err
}

func decodeSlider(pb []byte) (*Slider, error) {
	s := &Slider{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 2 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				s.Sensitivity = uint32Ptr(v)
			}
			return nil
		}
		if num == 12 && typ == protowire.VarintType {
			if v, ok := fieldVarint(body); ok {
				s.Custom = uint32Ptr(v)
			}
			return nil
		}
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 11, 15:
			return decodeChild(body, func(b []byte) error {
				v, err := decodeSliderShortcut(b)
				if num == 11 {
					s.CustomShortcut = v
				} else {
					s.PredefinedShortcut = v
				}
				return err
			})
		case 13, 14:
			e, err := decodeEmpty(body)
			if num == 13 {
				s.Nothing = e
			} else {
				s.AsGlobal = e
			}
			return err
		}
		return nil
	})
	return s, err
}

func decodeSliderShortcut(pb []byte) (*SliderShortcut, error) {
	s := &SliderShortcut{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || (num != 1 && num != 2) {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			k, err := DecodeKeyCombination(b)
			if num == 1 {
				s.Inc = k
			} else {
				s.Dec = k
			}
			return err
		})
	})
	return s, err
}

// KeyboardSettings maps F1..F12 to actions (protobuf fields 13..24).
type KeyboardSettings struct {
	RawBytes
	FKey *FKeySettings `json:"fKey,omitempty"`
}

// FKeySettings maps the twelve function keys to their actions.
type FKeySettings struct {
	RawBytes
	F1  *KeyAction `json:"F1,omitempty"`
	F2  *KeyAction `json:"F2,omitempty"`
	F3  *KeyAction `json:"F3,omitempty"`
	F4  *KeyAction `json:"F4,omitempty"`
	F5  *KeyAction `json:"F5,omitempty"`
	F6  *KeyAction `json:"F6,omitempty"`
	F7  *KeyAction `json:"F7,omitempty"`
	F8  *KeyAction `json:"F8,omitempty"`
	F9  *KeyAction `json:"F9,omitempty"`
	F10 *KeyAction `json:"F10,omitempty"`
	F11 *KeyAction `json:"F11,omitempty"`
	F12 *KeyAction `json:"F12,omitempty"`
}

// KeyAction is the selected action for a function key.
type KeyAction struct {
	RawBytes
	CustomShortcut     *KeyCombination `json:"customShortcut,omitempty"`
	Custom             *EmptyAction    `json:"custom,omitempty"`
	Nothing            *EmptyAction    `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction    `json:"asGlobal,omitempty"`
	PredefinedShortcut *KeyCombination `json:"predefinedShortcut,omitempty"`
}

// DecodeKeyboard parses function-key settings while preserving raw bytes.
func DecodeKeyboard(pb []byte) (*KeyboardSettings, error) {
	k := &KeyboardSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				var err error
				k.FKey, err = decodeFKeys(b)
				return err
			})
		}
		return nil
	})
	return k, err
}

func decodeFKeys(pb []byte) (*FKeySettings, error) {
	f := &FKeySettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || num < 13 || num > 24 {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			a, err := decodeKeyAction(b)
			switch num {
			case 13:
				f.F1 = a
			case 14:
				f.F2 = a
			case 15:
				f.F3 = a
			case 16:
				f.F4 = a
			case 17:
				f.F5 = a
			case 18:
				f.F6 = a
			case 19:
				f.F7 = a
			case 20:
				f.F8 = a
			case 21:
				f.F9 = a
			case 22:
				f.F10 = a
			case 23:
				f.F11 = a
			case 24:
				f.F12 = a
			}
			return err
		})
	})
	return f, err
}

func decodeKeyAction(pb []byte) (*KeyAction, error) {
	a := &KeyAction{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1, 5:
			return decodeChild(body, func(b []byte) error {
				k, err := DecodeKeyCombination(b)
				if num == 1 {
					a.CustomShortcut = k
				} else {
					a.PredefinedShortcut = k
				}
				return err
			})
		case 2, 3, 4:
			e, err := decodeEmpty(body)
			if num == 2 {
				a.Custom = e
			} else if num == 3 {
				a.Nothing = e
			} else {
				a.AsGlobal = e
			}
			return err
		}
		return nil
	})
	return a, err
}
