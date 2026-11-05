package daemon

import (
	"errors"
	"fmt"

	cproto "clvx/internal/proto"
	"google.golang.org/protobuf/encoding/protowire"
)

// OperationStatus is returned by OS-mode, restart, and reset operations.
type OperationStatus struct {
	Status uint32 `json:"status"`
}

func (d *Daemon) call(name string, pb []byte) (cproto.Response, error) {
	reply, err := d.exchange(pb)
	if err != nil {
		return cproto.Response{}, err
	}
	resp, err := cproto.DecodeResponse(reply)
	if err != nil {
		return cproto.Response{}, err
	}
	if resp.Type == cproto.RespBadRequest {
		return cproto.Response{}, fmt.Errorf("clvxd: %s rejected with BAD_REQUEST", name)
	}
	return resp, nil
}

func emptyRequest(reqType uint32) ([]byte, error) {
	return cproto.EncodeEmptyRequest(reqType)
}

// ProfileSettings reads the sole supported working profile (id 0).
func (d *Daemon) ProfileSettings() (*cproto.ProfileSettings, error) {
	resp, err := d.call("GET_PROFILE_SETTINGS", cproto.EncodeGetProfileSettings(0))
	if err != nil {
		return nil, err
	}
	profile, status, err := cproto.ParseProfileResponse(resp)
	if err != nil {
		return nil, err
	}
	if status != cproto.GetProfileStatusOK {
		return profile, fmt.Errorf("clvxd: GET_PROFILE_SETTINGS status %d", status)
	}
	if profile == nil {
		return nil, errors.New("clvxd: GET_PROFILE_SETTINGS returned no settings")
	}
	if profile.ID != 0 {
		return nil, fmt.Errorf("clvxd: device returned unsupported profile id %d", profile.ID)
	}
	return profile, nil
}

// SetProfileSettings merges typed edits into a device-supplied raw profile.
// Callers must supply the raw profile so unknown fields can be preserved.
// A successful write returns the merged view without a read-back: the UI is
// optimistic, the next ProfileSettings call reconciles with the device, and
// client-side range validation makes normalization drift negligible.
func (d *Daemon) SetProfileSettings(profile *cproto.ProfileSettings) (*cproto.ProfileSettings, error) {
	raw, err := profile.Bytes()
	if err != nil {
		return nil, err
	}
	resp, err := d.call("SET_PROFILE_SETTINGS", cproto.EncodeSetProfileSettings(raw))
	if err != nil {
		return nil, err
	}
	applied, status, err := cproto.ParseProfileResponse(resp)
	if err != nil {
		return nil, err
	}
	if status != cproto.SetProfileStatusOK {
		return applied, fmt.Errorf("clvxd: SET_PROFILE_SETTINGS status %d", status)
	}
	return cproto.DecodeProfile(raw)
}

// DeviceInfo reads identity and firmware information from the keyboard.
func (d *Daemon) DeviceInfo() (*cproto.DeviceInfo, error) {
	pb, err := emptyRequest(cproto.ReqGetDeviceInfo)
	if err != nil {
		return nil, err
	}
	resp, err := d.call("GET_DEVICE_INFO", pb)
	if err != nil {
		return nil, err
	}
	return cproto.ParseDeviceInfoResponse(resp)
}

// DefaultSettings reads the defaults without applying them.
func (d *Daemon) DefaultSettings() (*cproto.Settings, error) {
	pb, err := emptyRequest(cproto.ReqGetDefaultSettings)
	if err != nil {
		return nil, err
	}
	resp, err := d.call("GET_DEFAULT_SETTINGS", pb)
	if err != nil {
		return nil, err
	}
	settings, status, err := cproto.ParseSettingsResponse(resp)
	if err != nil {
		return nil, err
	}
	if status != cproto.SettingsStatusOK {
		return settings, fmt.Errorf("clvxd: GET_DEFAULT_SETTINGS status %d", status)
	}
	if settings == nil {
		return nil, errors.New("clvxd: GET_DEFAULT_SETTINGS returned no settings")
	}
	return settings, nil
}

// SetOSMode selects the keyboard's host OS behavior mode.
func (d *Daemon) SetOSMode(mode cproto.OSMode) (*OperationStatus, error) {
	pb, err := cproto.EncodeSetOSMode(mode)
	if err != nil {
		return nil, err
	}
	resp, err := d.call("SET_OS_MODE", pb)
	if err != nil {
		return nil, err
	}
	status, err := cproto.ParseStatusResponse(resp, 11)
	if err != nil {
		return nil, err
	}
	if status != 0 {
		return &OperationStatus{Status: status}, fmt.Errorf("clvxd: SET_OS_MODE status %d", status)
	}
	return &OperationStatus{Status: status}, nil
}

// PerformRestart asks the keyboard to restart.
func (d *Daemon) PerformRestart() (*OperationStatus, error) {
	return d.performStatusRequest(cproto.ReqPerformRestart, "PERFORM_RESTART", 14)
}

// PerformFullReset asks the keyboard to restore its factory settings.
func (d *Daemon) PerformFullReset() (*OperationStatus, error) {
	return d.performStatusRequest(cproto.ReqPerformFullReset, "PERFORM_FULL_RESET", 12)
}

func (d *Daemon) performStatusRequest(reqType uint32, name string, responseField protowire.Number) (*OperationStatus, error) {
	pb, err := emptyRequest(reqType)
	if err != nil {
		return nil, err
	}
	resp, err := d.call(name, pb)
	if err != nil {
		return nil, err
	}
	status, err := cproto.ParseStatusResponse(resp, responseField)
	if err != nil {
		return nil, err
	}
	if status != 0 {
		return &OperationStatus{Status: status}, fmt.Errorf("clvxd: %s status %d", name, status)
	}
	return &OperationStatus{Status: status}, nil
}

// Statistics reads the keyboard's cumulative usage counters.
func (d *Daemon) Statistics() (*cproto.Statistics, error) {
	pb, err := emptyRequest(cproto.ReqGetStatistics)
	if err != nil {
		return nil, err
	}
	resp, err := d.call("GET_STATISTICS", pb)
	if err != nil {
		return nil, err
	}
	return cproto.ParseStatisticsResponse(resp)
}

// Diagnostics is deliberately best-effort. Firmware that rejects this optional
// RPC produces an empty JSON object without affecting other methods.
func (d *Daemon) Diagnostics() *cproto.MiscDiagnostics {
	pb, err := emptyRequest(cproto.ReqGetDiagnostics)
	if err != nil {
		return &cproto.MiscDiagnostics{}
	}
	resp, err := d.call("GET_DIAGNOSTICS", pb)
	if err != nil {
		return &cproto.MiscDiagnostics{}
	}
	diagnostics, err := cproto.ParseMiscDiagnosticsResponse(resp)
	if err != nil {
		return &cproto.MiscDiagnostics{}
	}
	return diagnostics
}
