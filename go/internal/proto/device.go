package proto

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// OSMode is the keyboard's host OS behavior mode.
type OSMode uint32

const (
	OSWindows OSMode = iota
	OSMac
	OSLinux
	OSIPad
	OSAppleTV
	OSUnknown
	OSAppleVision
	OSIOS
)

var osModeNames = map[OSMode]string{
	OSWindows: "WINDOWS", OSMac: "MAC", OSLinux: "LINUX", OSIPad: "IPAD",
	OSAppleTV: "APPLE_TV", OSUnknown: "UNKNOWN_OS", OSAppleVision: "APPLE_VISION", OSIOS: "IOS",
}

func (m OSMode) String() string {
	if name, ok := osModeNames[m]; ok {
		return name
	}
	return "UNRECOGNIZED"
}

func (m OSMode) MarshalJSON() ([]byte, error) { return json.Marshal(m.String()) }

func ParseOSMode(name string) (OSMode, error) {
	if name == "" {
		return OSLinux, nil
	}
	name = strings.ToUpper(name)
	for mode, candidate := range osModeNames {
		if name == candidate {
			return mode, nil
		}
	}
	return 0, fmt.Errorf("proto: invalid OS mode %q", name)
}

func EncodeSetOSMode(mode OSMode) ([]byte, error) {
	if _, ok := osModeNames[mode]; !ok {
		return nil, fmt.Errorf("proto: invalid OS mode %d", mode)
	}
	var body []byte
	if mode != OSWindows {
		body = protowire.AppendTag(body, 1, protowire.VarintType)
		body = protowire.AppendVarint(body, uint64(mode))
	}
	return encodeEnvelope(ReqSetOSMode, body), nil
}

// DeviceInfo mirrors app-protocol.ts. Enum values are exposed by name while
// unknown enum numbers remain UNRECOGNIZED rather than failing the read.
type DeviceInfo struct {
	RawBytes
	Name            *string     `json:"name,omitempty"`
	FWVersion       *uint32     `json:"fwVersion,omitempty"`
	BarCode         *string     `json:"barCode,omitempty"`
	SerialNumber    *uint32     `json:"serialNumber,omitempty"`
	ProtocolVersion *uint32     `json:"protocolVersion,omitempty"`
	Insider         *bool       `json:"insider,omitempty"`
	DeviceType      *string     `json:"deviceType,omitempty"`
	KBMInfo         *KBMInfo    `json:"kbmInfo,omitempty"`
	PCBAVersion     *uint32     `json:"pcbaVersion,omitempty"`
	ChipUID         []byte      `json:"chipUID,omitempty"`
	Color           *string     `json:"color,omitempty"`
	DeviceMode      *DeviceMode `json:"deviceMode,omitempty"`
	CLVXVersion     *string     `json:"CLVXVersion,omitempty"`
	Common          *uint32     `json:"common,omitempty"`
}

type DeviceMode struct {
	RawBytes
	OSMode    *OSMode `json:"osMode,omitempty"`
	InputMode *string `json:"inputMode,omitempty"`
}

type KBMInfo struct {
	RawBytes
	Keyboard    *KBMKeyboardInfo `json:"keyboard,omitempty"`
	TouchSensor *KBMSensorInfo   `json:"touchSensor,omitempty"`
	TouchIC     *KBMTouchICInfo  `json:"touchIc,omitempty"`
	Backlight   *KBMSensorInfo   `json:"backlight,omitempty"`
	Marker      *uint32          `json:"marker,omitempty"`
	Type        *string          `json:"type,omitempty"`
}

type KBMKeyboardInfo struct {
	RawBytes
	TypeLegacy *string `json:"typeLegacy,omitempty"`
	Color      *string `json:"color,omitempty"`
	Standard   *string `json:"standard,omitempty"`
	Language   *string `json:"language,omitempty"`
	Layout     *uint32 `json:"layout,omitempty"`
	OS         *string `json:"os,omitempty"`
}

type KBMSensorInfo struct {
	RawBytes
	Type    *uint32 `json:"type,omitempty"`
	Version *uint32 `json:"version,omitempty"`
}

type KBMTouchICInfo struct {
	RawBytes
	Chip      *uint32 `json:"chip,omitempty"`
	FWVersion *uint32 `json:"fwVersion,omitempty"`
	UID       []byte  `json:"UID,omitempty"`
}

var deviceTypeNames = map[uint64]string{0: "UNKNOWN", 1: "CLVX", 2: "Laptop", 3: "Detach", 4: "Milka"}
var deviceColorNames = map[uint64]string{0: "UNDEFINED", 1: "BLACK", 2: "WHITE"}
var clvxVersionNames = map[uint64]string{0: "Undefined", 1: "One", 2: "S", 3: "M"}
var inputModeNames = map[uint64]string{0: "MOUSE", 1: "TOUCHPAD"}
var kbmTypeNames = map[uint64]string{0: "TYPE_UNDEFINED", 1: "TYPE_420", 2: "TYPE_293", 3: "TYPE_275", 4: "TYPE_355", 5: "TYPE_293_SMOS", 6: "TYPE_293_V2"}
var kbmStandardNames = map[uint64]string{0: "STANDARD_UNDEFINED", 1: "ANSI", 2: "ISO", 3: "JIS"}
var kbmLanguageNames = map[uint64]string{0: "LANGUAGE_UNDEFINED", 1: "US", 2: "FR", 3: "DE", 4: "UK", 5: "JP"}
var kbmOSNames = map[uint64]string{0: "OS_UNDEFINED", 1: "WINDOWS", 2: "MAC", 3: "LINUX"}

func DecodeDeviceInfo(pb []byte) (*DeviceInfo, error) {
	d := &DeviceInfo{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ == protowire.BytesType {
			b, ok := fieldBytes(body)
			if !ok {
				return nil
			}
			switch num {
			case 1:
				v := string(b)
				d.Name = &v
			case 9:
				v := string(b)
				d.BarCode = &v
			case 14:
				info, err := decodeKBMInfo(b)
				d.KBMInfo = info
				return err
			case 16:
				d.ChipUID = append([]byte(nil), b...)
			case 18:
				mode, err := decodeDeviceMode(b)
				d.DeviceMode = mode
				return err
			}
			return nil
		}
		if typ != protowire.VarintType {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		switch num {
		case 3:
			d.FWVersion = uint32Ptr(v)
		case 10:
			d.SerialNumber = uint32Ptr(v)
		case 11:
			d.ProtocolVersion = uint32Ptr(v)
		case 12:
			d.Insider = boolPtr(v)
		case 13:
			d.DeviceType = enumName(v, deviceTypeNames)
		case 15:
			d.PCBAVersion = uint32Ptr(v)
		case 17:
			d.Color = enumName(v, deviceColorNames)
		case 19:
			d.CLVXVersion = enumName(v, clvxVersionNames)
		case 20:
			d.Common = uint32Ptr(v)
		}
		return nil
	})
	return d, err
}

func ParseDeviceInfoResponse(r Response) (*DeviceInfo, error) {
	if r.Field != 5 {
		return nil, fmt.Errorf("proto: response field %d is not device info", r.Field)
	}
	var info *DeviceInfo
	err := walkFields(r.Body, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) (err error) { info, err = DecodeDeviceInfo(b); return })
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("proto: device info response has no deviceInfo")
	}
	return info, nil
}

func decodeDeviceMode(pb []byte) (*DeviceMode, error) {
	d := &DeviceMode{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		if num == 1 {
			m := OSMode(v)
			d.OSMode = &m
		} else if num == 2 {
			d.InputMode = enumName(v, inputModeNames)
		}
		return nil
	})
	return d, err
}

func decodeKBMInfo(pb []byte) (*KBMInfo, error) {
	k := &KBMInfo{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ == protowire.BytesType {
			switch num {
			case 1:
				return decodeChild(body, func(b []byte) (err error) { k.Keyboard, err = decodeKBMKeyboard(b); return })
			case 2, 4:
				return decodeChild(body, func(b []byte) error {
					v, err := decodeKBMSensor(b)
					if num == 2 {
						k.TouchSensor = v
					} else {
						k.Backlight = v
					}
					return err
				})
			case 3:
				return decodeChild(body, func(b []byte) (err error) { k.TouchIC, err = decodeKBMTouchIC(b); return })
			}
		}
		if typ == protowire.VarintType {
			v, ok := fieldVarint(body)
			if !ok {
				return nil
			}
			if num == 5 {
				k.Marker = uint32Ptr(v)
			} else if num == 6 {
				k.Type = enumName(v, kbmTypeNames)
			}
		}
		return nil
	})
	return k, err
}

func decodeKBMKeyboard(pb []byte) (*KBMKeyboardInfo, error) {
	k := &KBMKeyboardInfo{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		switch num {
		case 1:
			k.TypeLegacy = enumName(v, kbmTypeNames)
		case 2:
			k.Color = enumName(v, deviceColorNames)
		case 3:
			k.Standard = enumName(v, kbmStandardNames)
		case 4:
			k.Language = enumName(v, kbmLanguageNames)
		case 5:
			k.Layout = uint32Ptr(v)
		case 6:
			k.OS = enumName(v, kbmOSNames)
		}
		return nil
	})
	return k, err
}

func decodeKBMSensor(pb []byte) (*KBMSensorInfo, error) {
	s := &KBMSensorInfo{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		if num == 1 {
			s.Type = uint32Ptr(v)
		} else if num == 2 {
			s.Version = uint32Ptr(v)
		}
		return nil
	})
	return s, err
}

func decodeKBMTouchIC(pb []byte) (*KBMTouchICInfo, error) {
	t := &KBMTouchICInfo{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 3 && typ == protowire.BytesType {
			b, ok := fieldBytes(body)
			if ok {
				t.UID = append([]byte(nil), b...)
			}
			return nil
		}
		if typ != protowire.VarintType {
			return nil
		}
		v, ok := fieldVarint(body)
		if !ok {
			return nil
		}
		if num == 1 {
			t.Chip = uint32Ptr(v)
		} else if num == 2 {
			t.FWVersion = uint32Ptr(v)
		}
		return nil
	})
	return t, err
}

// ParseStatusResponse handles SetOSMode (field 11), full reset (field 12),
// and restart (field 14). All three use status:1, OK=0/ERROR=1.
func ParseStatusResponse(r Response, field protowire.Number) (uint32, error) {
	if r.Field != field {
		return 0, fmt.Errorf("proto: response field %d, want %d", r.Field, field)
	}
	var status uint64
	err := walkFields(r.Body, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.VarintType {
			v, err := varintValue(body)
			if err != nil {
				return err
			}
			status = v
		}
		return nil
	})
	return uint32(status), err
}
