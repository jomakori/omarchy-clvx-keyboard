package proto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	GetProfileStatusOK = iota
	GetProfileStatusUnknownError
	GetProfileStatusUnsupportedID
	GetProfileStatusNotFound
)

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

// The standalone profile RPC includes raw bytes so clients can perform an
// exact no-op round-trip without exposing ProfileRaw in get_settings JSON.
func (p ProfileSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(profileSettingsJSON{p.ID, p.Gestures, p.TouchZone, p.Backlight, p.Keyboard, p.Raw})
}

func (p *ProfileSettings) UnmarshalJSON(data []byte) error {
	var v profileSettingsJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	p.ID, p.Gestures, p.TouchZone, p.Backlight, p.Keyboard = v.ID, v.Gestures, v.TouchZone, v.Backlight, v.Keyboard
	p.RawBytes = rawBytes(v.Raw)
	return nil
}

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
				return decodeChild(body, func(b []byte) (err error) { p.Gestures, err = DecodeGestures(b); return })
			}
		case 3:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) (err error) { p.TouchZone, err = DecodeTouchZone(b); return })
			}
		case 4:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) (err error) { p.Backlight, err = DecodeBacklight(b); return })
			}
		case 5:
			if typ == protowire.BytesType {
				return decodeChild(body, func(b []byte) (err error) { p.Keyboard, err = DecodeKeyboard(b); return })
			}
		}
		return nil
	})
	return p, err
}

// Bytes returns the original profile bytes for a raw-preserving write. Typed
// profile mutation is intentionally not exposed until all nested encoders exist.
func (p *ProfileSettings) Bytes() ([]byte, error) {
	if p == nil || p.Raw == nil {
		return nil, errors.New("proto: profile has no raw bytes")
	}
	if p.ID != 0 {
		return nil, fmt.Errorf("proto: unsupported profile id %d", p.ID)
	}
	// If decoded views were supplied, reject edits rather than silently writing
	// stale raw bytes. A raw-only request is also valid for transport tooling.
	if p.Gestures != nil || p.TouchZone != nil || p.Backlight != nil || p.Keyboard != nil {
		original, err := DecodeProfile(p.Raw)
		if err != nil {
			return nil, err
		}
		got, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		want, err := json.Marshal(original)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(got, want) {
			return nil, errors.New("proto: typed profile edits are not supported; preserve raw and decoded fields exactly")
		}
	}
	return append([]byte(nil), p.Raw...), nil
}

func EncodeGetProfileSettings(profileID uint32) []byte {
	var body []byte
	if profileID != 0 {
		body = protowire.AppendTag(body, 1, protowire.VarintType)
		body = protowire.AppendVarint(body, uint64(profileID))
	}
	return encodeEnvelope(ReqGetProfileSettings, body)
}

func EncodeSetProfileSettings(profileRaw []byte) []byte {
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.BytesType)
	body = protowire.AppendBytes(body, profileRaw)
	return encodeEnvelope(ReqSetProfileSettings, body)
}

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

type Gesture struct {
	RawBytes
	Swipe *Swipe `json:"swipe,omitempty"`
	Tap   *Tap   `json:"tap,omitempty"`
}

type Swipe struct {
	RawBytes
	Up    *SwipeAction `json:"Up,omitempty"`
	Down  *SwipeAction `json:"Down,omitempty"`
	Left  *SwipeAction `json:"Left,omitempty"`
	Right *SwipeAction `json:"Right,omitempty"`
}

type SwipeAction struct {
	RawBytes
	Touchpad           *EmptyAction   `json:"touchpad,omitempty"`
	CustomShortcut     *SwipeShortcut `json:"customShortcut,omitempty"`
	Custom             *uint32        `json:"custom,omitempty"`
	Nothing            *EmptyAction   `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction   `json:"asGlobal,omitempty"`
	PredefinedShortcut *SwipeShortcut `json:"predefinedShortcut,omitempty"`
}

type SwipeShortcut struct {
	RawBytes
	Direct      *KeyCombination `json:"direct,omitempty"`
	Sensitivity *uint32         `json:"sensitivity,omitempty"`
	Continues   *bool           `json:"continues,omitempty"`
	Opposite    *KeyCombination `json:"opposite,omitempty"`
}

type Tap struct {
	RawBytes
	Action *TapAction `json:"action,omitempty"`
}

type TapAction struct {
	RawBytes
	Touchpad           *EmptyAction    `json:"touchpad,omitempty"`
	CustomShortcut     *KeyCombination `json:"customShortcut,omitempty"`
	Custom             *uint32         `json:"custom,omitempty"`
	Nothing            *EmptyAction    `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction    `json:"asGlobal,omitempty"`
	PredefinedShortcut *KeyCombination `json:"predefinedShortcut,omitempty"`
}

type KeyCombination struct {
	RawBytes
	Key1 *Key `json:"key1,omitempty"`
	Key2 *Key `json:"key2,omitempty"`
	Key3 *Key `json:"key3,omitempty"`
	Key4 *Key `json:"key4,omitempty"`
}

type Key struct {
	RawBytes
	Type *uint32 `json:"type,omitempty"`
	Code *uint32 `json:"code,omitempty"`
}

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
			return decodeChild(body, func(b []byte) (err error) { g.Swipe, err = decodeSwipe(b); return })
		case 2:
			return decodeChild(body, func(b []byte) (err error) { g.Tap, err = decodeTap(b); return })
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
			return decodeChild(body, func(b []byte) (err error) { t.Action, err = decodeTapAction(b); return })
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

type TouchpadSettings struct {
	RawBytes
	Enable          *bool `json:"enable,omitempty"`
	OneFingerEnable *bool `json:"oneFingerEnable,omitempty"`
	TwoFingerEnable *bool `json:"twoFingerEnable,omitempty"`
}

type SliderSettings struct {
	RawBytes
	Left  *Slider `json:"left,omitempty"`
	Right *Slider `json:"right,omitempty"`
}

type Slider struct {
	RawBytes
	Sensitivity        *uint32         `json:"sensitivity,omitempty"`
	CustomShortcut     *SliderShortcut `json:"customShortcut,omitempty"`
	Custom             *uint32         `json:"custom,omitempty"`
	Nothing            *EmptyAction    `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction    `json:"asGlobal,omitempty"`
	PredefinedShortcut *SliderShortcut `json:"predefinedShortcut,omitempty"`
}

type SliderShortcut struct {
	RawBytes
	Inc *KeyCombination `json:"inc,omitempty"`
	Dec *KeyCombination `json:"dec,omitempty"`
}

func DecodeTouchZone(pb []byte) (*TouchZoneSettings, error) {
	t := &TouchZoneSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1:
			return decodeChild(body, func(b []byte) (err error) { t.Touchpad, err = decodeTouchpadSettings(b); return })
		case 2:
			return decodeChild(body, func(b []byte) (err error) { t.Slider, err = decodeSliderSettings(b); return })
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

type KeyAction struct {
	RawBytes
	CustomShortcut     *KeyCombination `json:"customShortcut,omitempty"`
	Custom             *EmptyAction    `json:"custom,omitempty"`
	Nothing            *EmptyAction    `json:"nothing,omitempty"`
	AsGlobal           *EmptyAction    `json:"asGlobal,omitempty"`
	PredefinedShortcut *KeyCombination `json:"predefinedShortcut,omitempty"`
}

func DecodeKeyboard(pb []byte) (*KeyboardSettings, error) {
	k := &KeyboardSettings{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) (err error) { k.FKey, err = decodeFKeys(b); return })
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
