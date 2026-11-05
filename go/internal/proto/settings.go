// Typed AppSettings / GlobalSettings codec. Field numbers come from the
// vendor ts-proto encoders in app-settings.ts (GlobalSettings.encode) and
// app-protocol.ts (Request/Response/SetSettingsRequest/GetSettingsResponse).
//
// Only the settings the daemon can safely read-modify-write today are typed:
// GlobalSettings in full, plus the raw ProfileSettings bytes so a write
// never drops gestures, touch zones, or per-profile backlight.
package proto

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// GlobalSettings field numbers (app-settings.ts GlobalSettings.encode).
const (
	gTap1fEnable                 protowire.Number = 2
	gTap2fEnable                 protowire.Number = 3
	gHoldEnable                  protowire.Number = 4
	gSwapClickButtons            protowire.Number = 5
	gCurrentAILevel              protowire.Number = 6
	gNewbieModeEnable            protowire.Number = 7
	gTouchActivationAfterLiftOff protowire.Number = 8
	gFnLock                      protowire.Number = 9
	gDominantHand                protowire.Number = 12
	gBatterySavingModeEnable     protowire.Number = 14
	gKeySuppressorEnable         protowire.Number = 15
	gHoldDelayOnBorderEnable     protowire.Number = 16
	gSwapFnCtrl                  protowire.Number = 17
	gIdleBacklightTime           protowire.Number = 20
	gBacklightTime               protowire.Number = 21
	gAutoUsbSwitchEnable         protowire.Number = 22
	gTypeOnlyLongSwitchEnable    protowire.Number = 24
	gEcoBrightness               protowire.Number = 25
	gTap1fMouseButton            protowire.Number = 26
	gBrightness                  protowire.Number = 27
)

// AppSettings field numbers.
const (
	appGlobal        protowire.Number = 1
	appGlobalProfile protowire.Number = 2
	appCounter       protowire.Number = 3
)

// Response status values for GetSettingsResponse / SetSettingsResponse.
const (
	SettingsStatusOK              = 0
	SettingsStatusError           = 1
	SettingsStatusValidationError = 2
)

// Global mirrors GlobalSettings. Pointer fields distinguish "not set" from
// "set to zero", which is what makes a patch safe: absent fields keep the
// value the keyboard reported.
type Global struct {
	Tap1fEnable                 *bool   `json:"tap1fEnable,omitempty"`
	Tap2fEnable                 *bool   `json:"tap2fEnable,omitempty"`
	HoldEnable                  *bool   `json:"holdEnable,omitempty"`
	SwapClickButtons            *bool   `json:"swapClickButtons,omitempty"`
	CurrentAILevel              *uint32 `json:"currentAILevel,omitempty"`
	NewbieModeEnable            *bool   `json:"newbieModeEnable,omitempty"`
	TouchActivationAfterLiftOff *bool   `json:"touchActivationAfterLiftOff,omitempty"`
	FnLock                      *bool   `json:"fnLock,omitempty"`
	DominantHand                *string `json:"dominantHand,omitempty"` // LEFT, RIGHT, UNSELECTED
	BatterySavingModeEnable     *bool   `json:"batterySavingModeEnable,omitempty"`
	KeySuppressorEnable         *bool   `json:"keySuppressorEnable,omitempty"`
	HoldDelayOnBorderEnable     *bool   `json:"holdDelayOnBorderEnable,omitempty"`
	SwapFnCtrl                  *bool   `json:"swapFnCtrl,omitempty"`
	IdleBacklightTime           *uint32 `json:"idleBacklightTime,omitempty"` // seconds, 0 disables
	BacklightTime               *uint32 `json:"backlightTime,omitempty"`     // seconds, 0 = infinite
	AutoUsbSwitchEnable         *bool   `json:"autoUsbSwitchEnable,omitempty"`
	TypeOnlyLongSwitchEnable    *bool   `json:"typeOnlyLongSwitchEnable,omitempty"`
	EcoBrightness               *bool   `json:"ecoBrightness,omitempty"`
	Tap1fMouseButton            *string `json:"tap1fMouseButton,omitempty"` // LEFT_BUTTON, RIGHT_BUTTON
	Brightness                  *uint32 `json:"brightness,omitempty"`
}

var dominantHandNames = map[uint64]string{0: "LEFT", 1: "RIGHT", 2: "UNSELECTED"}
var dominantHandValues = map[string]uint64{"LEFT": 0, "RIGHT": 1, "UNSELECTED": 2}

var tap1fMouseButtonNames = map[uint64]string{0: "LEFT_BUTTON", 1: "RIGHT_BUTTON"}
var tap1fMouseButtonValues = map[string]uint64{"LEFT_BUTTON": 0, "RIGHT_BUTTON": 1}

// Merge copies every field set in patch over g. Fields left nil in patch are
// untouched, so a caller can send only what the user changed.
func (g *Global) Merge(patch *Global) {
	if g == nil || patch == nil {
		return
	}
	setBool(&g.Tap1fEnable, patch.Tap1fEnable)
	setBool(&g.Tap2fEnable, patch.Tap2fEnable)
	setBool(&g.HoldEnable, patch.HoldEnable)
	setBool(&g.SwapClickButtons, patch.SwapClickButtons)
	setUint32(&g.CurrentAILevel, patch.CurrentAILevel)
	setBool(&g.NewbieModeEnable, patch.NewbieModeEnable)
	setBool(&g.TouchActivationAfterLiftOff, patch.TouchActivationAfterLiftOff)
	setBool(&g.FnLock, patch.FnLock)
	setString(&g.DominantHand, patch.DominantHand)
	setBool(&g.BatterySavingModeEnable, patch.BatterySavingModeEnable)
	setBool(&g.KeySuppressorEnable, patch.KeySuppressorEnable)
	setBool(&g.HoldDelayOnBorderEnable, patch.HoldDelayOnBorderEnable)
	setBool(&g.SwapFnCtrl, patch.SwapFnCtrl)
	setUint32(&g.IdleBacklightTime, patch.IdleBacklightTime)
	setUint32(&g.BacklightTime, patch.BacklightTime)
	setBool(&g.AutoUsbSwitchEnable, patch.AutoUsbSwitchEnable)
	setBool(&g.TypeOnlyLongSwitchEnable, patch.TypeOnlyLongSwitchEnable)
	setBool(&g.EcoBrightness, patch.EcoBrightness)
	setString(&g.Tap1fMouseButton, patch.Tap1fMouseButton)
	setUint32(&g.Brightness, patch.Brightness)
}

func setBool(dst **bool, src *bool) {
	if src != nil {
		v := *src
		*dst = &v
	}
}

func setUint32(dst **uint32, src *uint32) {
	if src != nil {
		v := *src
		*dst = &v
	}
}

func setString(dst **string, src *string) {
	if src != nil {
		v := *src
		*dst = &v
	}
}

// Encode serialises GlobalSettings. Enum strings are validated here rather
// than silently dropped.
func (g *Global) Encode() ([]byte, error) {
	var b []byte
	if g == nil {
		return b, nil
	}
	b = appendBool(b, gTap1fEnable, g.Tap1fEnable)
	b = appendBool(b, gTap2fEnable, g.Tap2fEnable)
	b = appendBool(b, gHoldEnable, g.HoldEnable)
	b = appendBool(b, gSwapClickButtons, g.SwapClickButtons)
	b = appendUint32(b, gCurrentAILevel, g.CurrentAILevel)
	b = appendBool(b, gNewbieModeEnable, g.NewbieModeEnable)
	b = appendBool(b, gTouchActivationAfterLiftOff, g.TouchActivationAfterLiftOff)
	b = appendBool(b, gFnLock, g.FnLock)
	enum, err := enumValue(gDominantHand, g.DominantHand, dominantHandValues)
	if err != nil {
		return nil, err
	}
	b = appendUint32(b, gDominantHand, enum)
	b = appendBool(b, gBatterySavingModeEnable, g.BatterySavingModeEnable)
	b = appendBool(b, gKeySuppressorEnable, g.KeySuppressorEnable)
	b = appendBool(b, gHoldDelayOnBorderEnable, g.HoldDelayOnBorderEnable)
	b = appendBool(b, gSwapFnCtrl, g.SwapFnCtrl)
	b = appendUint32(b, gIdleBacklightTime, g.IdleBacklightTime)
	b = appendUint32(b, gBacklightTime, g.BacklightTime)
	b = appendBool(b, gAutoUsbSwitchEnable, g.AutoUsbSwitchEnable)
	b = appendBool(b, gTypeOnlyLongSwitchEnable, g.TypeOnlyLongSwitchEnable)
	b = appendBool(b, gEcoBrightness, g.EcoBrightness)
	enum, err = enumValue(gTap1fMouseButton, g.Tap1fMouseButton, tap1fMouseButtonValues)
	if err != nil {
		return nil, err
	}
	b = appendUint32(b, gTap1fMouseButton, enum)
	b = appendUint32(b, gBrightness, g.Brightness)
	return b, nil
}

// DecodeGlobal parses GlobalSettings. Presence, not value, decides whether a
// pointer field is set.
func DecodeGlobal(pb []byte) (*Global, error) {
	g := &Global{}
	for len(pb) > 0 {
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		pb = pb[n:]
		if typ != protowire.VarintType {
			m := protowire.ConsumeFieldValue(num, typ, pb)
			if m < 0 {
				return nil, protowire.ParseError(m)
			}
			pb = pb[m:]
			continue
		}
		v, m := protowire.ConsumeVarint(pb)
		if m < 0 {
			return nil, protowire.ParseError(m)
		}
		pb = pb[m:]
		switch num {
		case gTap1fEnable:
			g.Tap1fEnable = boolPtr(v)
		case gTap2fEnable:
			g.Tap2fEnable = boolPtr(v)
		case gHoldEnable:
			g.HoldEnable = boolPtr(v)
		case gSwapClickButtons:
			g.SwapClickButtons = boolPtr(v)
		case gCurrentAILevel:
			g.CurrentAILevel = uint32Ptr(v)
		case gNewbieModeEnable:
			g.NewbieModeEnable = boolPtr(v)
		case gTouchActivationAfterLiftOff:
			g.TouchActivationAfterLiftOff = boolPtr(v)
		case gFnLock:
			g.FnLock = boolPtr(v)
		case gDominantHand:
			g.DominantHand = enumName(v, dominantHandNames)
		case gBatterySavingModeEnable:
			g.BatterySavingModeEnable = boolPtr(v)
		case gKeySuppressorEnable:
			g.KeySuppressorEnable = boolPtr(v)
		case gHoldDelayOnBorderEnable:
			g.HoldDelayOnBorderEnable = boolPtr(v)
		case gSwapFnCtrl:
			g.SwapFnCtrl = boolPtr(v)
		case gIdleBacklightTime:
			g.IdleBacklightTime = uint32Ptr(v)
		case gBacklightTime:
			g.BacklightTime = uint32Ptr(v)
		case gAutoUsbSwitchEnable:
			g.AutoUsbSwitchEnable = boolPtr(v)
		case gTypeOnlyLongSwitchEnable:
			g.TypeOnlyLongSwitchEnable = boolPtr(v)
		case gEcoBrightness:
			g.EcoBrightness = boolPtr(v)
		case gTap1fMouseButton:
			g.Tap1fMouseButton = enumName(v, tap1fMouseButtonNames)
		case gBrightness:
			g.Brightness = uint32Ptr(v)
		}
	}
	return g, nil
}

// Settings is a decoded AppSettings. Global is typed; ProfileRaw keeps the
// per-profile settings (gestures, touch zones, backlight, F-row) byte-exact
// so writing global settings back cannot corrupt them.
type Settings struct {
	Global     *Global `json:"global,omitempty"`
	Counter    *uint32 `json:"counter,omitempty"`
	ProfileRaw []byte  `json:"-"`
	extra      []byte
}

// DecodeSettings parses an AppSettings message.
func DecodeSettings(pb []byte) (*Settings, error) {
	s := &Settings{}
	for len(pb) > 0 {
		fieldStart := pb
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		pb = pb[n:]
		m := protowire.ConsumeFieldValue(num, typ, pb)
		if m < 0 {
			return nil, protowire.ParseError(m)
		}
		body := pb[:m]
		pb = pb[m:]
		switch num {
		case appGlobal:
			raw, err := bytesValue(body)
			if err != nil {
				return nil, err
			}
			g, err := DecodeGlobal(raw)
			if err != nil {
				return nil, err
			}
			s.Global = g
		case appGlobalProfile:
			raw, err := bytesValue(body)
			if err != nil {
				return nil, err
			}
			s.ProfileRaw = append([]byte(nil), raw...)
		case appCounter:
			v, err := varintValue(body)
			if err != nil {
				return nil, err
			}
			s.Counter = uint32Ptr(v)
		default:
			// Keep unknown fields byte-exact (tag included) so a read-modify-write
			// cycle cannot drop settings this decoder does not model yet.
			s.extra = append(s.extra, fieldStart[:n+m]...)
		}
	}
	return s, nil
}

// Encode re-serialises the AppSettings message, preserving ProfileRaw and any
// field the decoder did not type.
func (s *Settings) Encode() ([]byte, error) {
	var b []byte
	if s == nil {
		return b, nil
	}
	global, err := s.Global.Encode()
	if err != nil {
		return nil, err
	}
	if global != nil {
		b = protowire.AppendTag(b, appGlobal, protowire.BytesType)
		b = protowire.AppendBytes(b, global)
	}
	if len(s.ProfileRaw) > 0 {
		b = protowire.AppendTag(b, appGlobalProfile, protowire.BytesType)
		b = protowire.AppendBytes(b, s.ProfileRaw)
	}
	if s.Counter != nil {
		b = protowire.AppendTag(b, appCounter, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(*s.Counter))
	}
	b = append(b, s.extra...)
	return b, nil
}

// EncodeGetSettings builds Request{type: GET_SETTINGS}.
func EncodeGetSettings() []byte {
	return encodeEnvelope(ReqGetSettings, nil)
}

// EncodeSetSettings builds
// Request{type: SET_SETTINGS, setSettings:{settings: AppSettings}}.
func EncodeSetSettings(appSettings []byte) []byte {
	var body []byte
	body = protowire.AppendTag(body, 1, protowire.BytesType) // SetSettingsRequest.settings
	body = protowire.AppendBytes(body, appSettings)
	return encodeEnvelope(ReqSetSettings, body)
}

// ParseSettingsResponse decodes a reply to GET_SETTINGS or SET_SETTINGS and
// returns the settings plus the device's status code (0 = OK).
func ParseSettingsResponse(r Response) (*Settings, uint32, error) {
	if r.Field != reqField[ReqGetSettings] && r.Field != reqField[ReqSetSettings] {
		return nil, 0, fmt.Errorf("proto: response field %d is not a settings reply", r.Field)
	}
	var status uint64
	var raw []byte
	pb := r.Body
	for len(pb) > 0 {
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return nil, 0, protowire.ParseError(n)
		}
		pb = pb[n:]
		m := protowire.ConsumeFieldValue(num, typ, pb)
		if m < 0 {
			return nil, 0, protowire.ParseError(m)
		}
		body := pb[:m]
		pb = pb[m:]
		switch num {
		case 1:
			v, err := varintValue(body)
			if err != nil {
				return nil, 0, err
			}
			status = v
		case 2:
			b, err := bytesValue(body)
			if err != nil {
				return nil, 0, err
			}
			raw = b
		}
	}
	if raw == nil {
		// SET_SETTINGS only echoes settings when validation fails, so a bare
		// status is a normal success reply.
		return nil, uint32(status), nil
	}
	s, err := DecodeSettings(raw)
	if err != nil {
		return nil, uint32(status), err
	}
	return s, uint32(status), nil
}

func appendBool(b []byte, num protowire.Number, v *bool) []byte {
	if v == nil {
		return b
	}
	var x uint64
	if *v {
		x = 1
	}
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, x)
}

func appendUint32(b []byte, num protowire.Number, v *uint32) []byte {
	if v == nil {
		return b
	}
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, uint64(*v))
}

func enumValue(num protowire.Number, name *string, values map[string]uint64) (*uint32, error) {
	if name == nil {
		return nil, nil
	}
	v, ok := values[*name]
	if !ok {
		return nil, fmt.Errorf("proto: unknown enum %q for field %d", *name, num)
	}
	return uint32Ptr(v), nil
}

func enumName(v uint64, names map[uint64]string) *string {
	name, ok := names[v]
	if !ok {
		name = "UNRECOGNIZED"
	}
	return &name
}

func boolPtr(v uint64) *bool {
	b := v != 0
	return &b
}

func uint32Ptr(v uint64) *uint32 {
	n := uint32(v)
	return &n
}

// varintValue extracts the value from a field body captured by
// ConsumeFieldValue; for varints the body is the minimal varint itself.
func varintValue(body []byte) (uint64, error) {
	v, n := protowire.ConsumeVarint(body)
	if n < 0 {
		return 0, protowire.ParseError(n)
	}
	return v, nil
}

func bytesValue(body []byte) ([]byte, error) {
	v, n := protowire.ConsumeBytes(body)
	if n < 0 {
		return nil, protowire.ParseError(n)
	}
	return v, nil
}
