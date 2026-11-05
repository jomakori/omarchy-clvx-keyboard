package proto

import "fmt"

// Heartbeat is the decoded HeartBeatResponse (app-protocol.ts). Field numbers
// come from the ts-proto encoder tags: status=1, appSettingsCounter=3,
// aiStateCounter=4, ChargingArmor=5, power=6, backlightMode=7, uptime=8,
// continuousTime=9, lastActivityTime=10; power: battery=1 (level=1,
// charging=2), mode=2.
type Heartbeat struct {
	Status           uint64
	AppSettingsCount uint64
	AIStateCount     uint64
	ChargingArmor    uint64
	BacklightMode    uint64
	UptimeS          uint64
	ContinuousTimeS  uint64
	LastActivityS    uint64
	BatteryLevel     uint64
	Charging         bool
	PowerMode        uint64
}

// DecodeHeartbeat parses a heartbeat response body.
func DecodeHeartbeat(body []byte) (Heartbeat, error) {
	var h Heartbeat
	fs, err := Fields(body)
	if err != nil {
		return h, err
	}
	for _, f := range fs {
		v, isVar := f.Value.(uint64)
		b, isBytes := f.Value.([]byte)
		switch {
		case f.Num == 1 && isVar:
			h.Status = v
		case f.Num == 3 && isVar:
			h.AppSettingsCount = v
		case f.Num == 4 && isVar:
			h.AIStateCount = v
		case f.Num == 5 && isVar:
			h.ChargingArmor = v
		case f.Num == 6 && isBytes:
			if err := decodePower(b, &h); err != nil {
				return h, err
			}
		case f.Num == 7 && isVar:
			h.BacklightMode = v
		case f.Num == 8 && isVar:
			h.UptimeS = v
		case f.Num == 9 && isVar:
			h.ContinuousTimeS = v
		case f.Num == 10 && isVar:
			h.LastActivityS = v
		}
	}
	return h, nil
}

func decodePower(pb []byte, h *Heartbeat) error {
	fs, err := Fields(pb)
	if err != nil {
		return fmt.Errorf("power: %w", err)
	}
	for _, f := range fs {
		switch v := f.Value.(type) {
		case uint64:
			if f.Num == 2 {
				h.PowerMode = v
			}
		case []byte:
			if f.Num != 1 {
				continue
			}
			bat, err := Fields(v)
			if err != nil {
				return fmt.Errorf("battery: %w", err)
			}
			for _, bf := range bat {
				if bv, ok := bf.Value.(uint64); ok {
					switch bf.Num {
					case 1:
						h.BatteryLevel = bv
					case 2:
						h.Charging = bv != 0
					}
				}
			}
		}
	}
	return nil
}
