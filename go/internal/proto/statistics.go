package proto

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Statistics contains the device's cumulative usage counters.
type Statistics struct {
	RawBytes
	Keyboard      *KeyboardStatistics  `json:"keyboard,omitempty"`
	Gestures      *GestureStatistics   `json:"gestures,omitempty"`
	Sliders       *SliderStatistics    `json:"sliders,omitempty"`
	Channels      *ChannelStatistics   `json:"channels,omitempty"`
	TypeTouchOnly *TypeTouchStatistics `json:"typeTouchOnly,omitempty"`
}

// KeyboardStatistics contains cumulative keyboard usage counters.
type KeyboardStatistics struct {
	RawBytes
	KeyPressedCounter uint32 `json:"keyPressedCounter"`
}

// GestureStatistics contains cumulative gesture usage counters.
type GestureStatistics struct {
	RawBytes
	Tap1fCount   uint32 `json:"tap1fCount"`
	Tap2fCount   uint32 `json:"tap2fCount"`
	Tap3fCount   uint32 `json:"tap3fCount"`
	Tap4fCount   uint32 `json:"tap4fCount"`
	PinchCount   uint32 `json:"pinchCount"`
	Pinch4fCount uint32 `json:"pinch4fCount"`
	Slide1fCount uint32 `json:"slide1fCount"`
	Slide2fCount uint32 `json:"slide2fCount"`
	Slide3fCount uint32 `json:"slide3fCount"`
	Slide4fCount uint32 `json:"slide4fCount"`
	HoldCount    uint32 `json:"holdCount"`
}

// SliderStatistics contains cumulative left and right slider counters.
type SliderStatistics struct {
	RawBytes
	LeftSliderCount  uint32 `json:"lsliderCount"`
	RightSliderCount uint32 `json:"rsliderCount"`
}

// ChannelStatistics contains usage counters for each wireless or USB channel.
type ChannelStatistics struct {
	RawBytes
	USB *ChannelStatistic `json:"usb,omitempty"`
	BT1 *ChannelStatistic `json:"bt1,omitempty"`
	BT2 *ChannelStatistic `json:"bt2,omitempty"`
	BT3 *ChannelStatistic `json:"bt3,omitempty"`
}

// ChannelStatistic contains the time a channel spent active by host OS.
type ChannelStatistic struct {
	RawBytes
	TotalTimeS uint32   `json:"totalTimeS"`
	OSTimes    []OSTime `json:"osTimes,omitempty"`
}

// OSTime records the time spent using one host OS mode.
type OSTime struct {
	RawBytes
	OS    OSMode `json:"os"`
	TimeS uint32 `json:"timeS"`
}

// TypeTouchStatistics contains cumulative typing-only and touching-only times.
type TypeTouchStatistics struct {
	RawBytes
	TypeOnlyTimeS  uint32 `json:"typeOnlyTimeS"`
	TouchOnlyTimeS uint32 `json:"touchOnlyTimeS"`
}

// ParseStatisticsResponse validates and decodes a statistics response.
func ParseStatisticsResponse(r Response) (*Statistics, error) {
	if r.Field != 16 {
		return nil, fmt.Errorf("proto: response field %d is not statistics", r.Field)
	}
	return DecodeStatistics(r.Body)
}

// DecodeStatistics parses cumulative usage counters while preserving raw bytes.
func DecodeStatistics(pb []byte) (*Statistics, error) {
	s := &Statistics{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType {
			return nil
		}
		switch num {
		case 1:
			return decodeChild(body, func(b []byte) error {
				var err error
				s.Keyboard, err = decodeKeyboardStatistics(b)
				return err
			})
		case 2:
			return decodeChild(body, func(b []byte) error {
				var err error
				s.Gestures, err = decodeGestureStatistics(b)
				return err
			})
		case 3:
			return decodeChild(body, func(b []byte) error {
				var err error
				s.Sliders, err = decodeSliderStatistics(b)
				return err
			})
		case 4:
			return decodeChild(body, func(b []byte) error {
				var err error
				s.Channels, err = decodeChannelStatistics(b)
				return err
			})
		case 5:
			return decodeChild(body, func(b []byte) error {
				var err error
				s.TypeTouchOnly, err = decodeTypeTouchStatistics(b)
				return err
			})
		}
		return nil
	})
	return s, err
}

func counterValue(body []byte) uint32 {
	v, _ := fieldVarint(body)
	return uint32(v)
}

func decodeKeyboardStatistics(pb []byte) (*KeyboardStatistics, error) {
	k := &KeyboardStatistics{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.VarintType {
			k.KeyPressedCounter = counterValue(body)
		}
		return nil
	})
	return k, err
}

func decodeGestureStatistics(pb []byte) (*GestureStatistics, error) {
	g := &GestureStatistics{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		v := counterValue(body)
		switch num {
		case 1:
			g.Tap1fCount = v
		case 2:
			g.Tap2fCount = v
		case 3:
			g.Tap3fCount = v
		case 4:
			g.Tap4fCount = v
		case 5:
			g.PinchCount = v
		case 6:
			g.Pinch4fCount = v
		case 7:
			g.Slide1fCount = v
		case 8:
			g.Slide2fCount = v
		case 9:
			g.Slide3fCount = v
		case 10:
			g.Slide4fCount = v
		case 11:
			g.HoldCount = v
		}
		return nil
	})
	return g, err
}

func decodeSliderStatistics(pb []byte) (*SliderStatistics, error) {
	s := &SliderStatistics{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		if num == 1 {
			s.LeftSliderCount = counterValue(body)
		} else if num == 2 {
			s.RightSliderCount = counterValue(body)
		}
		return nil
	})
	return s, err
}

func decodeChannelStatistics(pb []byte) (*ChannelStatistics, error) {
	c := &ChannelStatistics{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.BytesType || num < 1 || num > 4 {
			return nil
		}
		return decodeChild(body, func(b []byte) error {
			v, err := decodeChannelStatistic(b)
			switch num {
			case 1:
				c.USB = v
			case 2:
				c.BT1 = v
			case 3:
				c.BT2 = v
			case 4:
				c.BT3 = v
			}
			return err
		})
	})
	return c, err
}

func decodeChannelStatistic(pb []byte) (*ChannelStatistic, error) {
	c := &ChannelStatistic{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if num == 1 && typ == protowire.VarintType {
			c.TotalTimeS = counterValue(body)
		}
		if num == 2 && typ == protowire.BytesType {
			return decodeChild(body, func(b []byte) error {
				v, err := decodeOSTime(b)
				c.OSTimes = append(c.OSTimes, *v)
				return err
			})
		}
		return nil
	})
	return c, err
}

func decodeOSTime(pb []byte) (*OSTime, error) {
	o := &OSTime{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		v, _ := fieldVarint(body)
		if num == 1 {
			o.OS = OSMode(v)
		} else if num == 2 {
			o.TimeS = uint32(v)
		}
		return nil
	})
	return o, err
}

func decodeTypeTouchStatistics(pb []byte) (*TypeTouchStatistics, error) {
	t := &TypeTouchStatistics{RawBytes: rawBytes(pb)}
	err := walkFields(pb, func(num protowire.Number, typ protowire.Type, body []byte) error {
		if typ != protowire.VarintType {
			return nil
		}
		if num == 1 {
			t.TypeOnlyTimeS = counterValue(body)
		} else if num == 2 {
			t.TouchOnlyTimeS = counterValue(body)
		}
		return nil
	})
	return t, err
}

// MiscDiagnostics is separate from the temporary multi-message diagnostics
// response; this is the small GET_DIAGNOSTICS miscellaneous-info reply.
type MiscDiagnostics struct {
	RawBytes
	TypeOnlyEnabled        *bool `json:"typeOnlyEnabled,omitempty"`
	BatteryMaxBrightnessEn *bool `json:"batteryMaxBrightnessEn,omitempty"`
	IsWinSettings          *bool `json:"isWinSettings,omitempty"`
}

// ParseMiscDiagnosticsResponse validates and decodes a diagnostics response.
func ParseMiscDiagnosticsResponse(r Response) (*MiscDiagnostics, error) {
	if r.Field != 15 {
		return nil, fmt.Errorf("proto: response field %d is not misc diagnostics", r.Field)
	}
	return DecodeMiscDiagnostics(r.Body)
}

// DecodeMiscDiagnostics parses miscellaneous diagnostics while preserving raw bytes.
func DecodeMiscDiagnostics(pb []byte) (*MiscDiagnostics, error) {
	d := &MiscDiagnostics{RawBytes: rawBytes(pb)}
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
			d.TypeOnlyEnabled = boolPtr(v)
		case 2:
			d.BatteryMaxBrightnessEn = boolPtr(v)
		case 4:
			d.IsWinSettings = boolPtr(v)
		}
		return nil
	})
	return d, err
}
