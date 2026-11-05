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
	pb, err := cproto.EncodeEmptyRequest(reqType)
	if err != nil {
		return nil, err
	}
	return pb, nil
}

// GetProfileSettings reads the sole supported working profile (id 0).
func (d *Daemon) GetProfileSettings() (*cproto.ProfileSettings, error) {
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

// SetProfileSettings performs a byte-exact profile write. Typed profile
// mutation remains decode-only; callers must supply a profile obtained from
// the device so unknown fields cannot be discarded.
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
	return d.GetProfileSettings()
}

func (d *Daemon) GetDeviceInfo() (*cproto.DeviceInfo, error) {
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

// GetDefaultSettings is read-only: it never applies the returned settings.
func (d *Daemon) GetDefaultSettings() (*cproto.Settings, error) {
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

func (d *Daemon) PerformRestart() (*OperationStatus, error) {
	return d.performStatusRequest(cproto.ReqPerformRestart, "PERFORM_RESTART", 14)
}

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

func (d *Daemon) GetStatistics() (*cproto.Statistics, error) {
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

// GetDiagnostics is deliberately best-effort. Firmware that rejects this
// optional RPC produces an empty JSON object without affecting other methods.
func (d *Daemon) GetDiagnostics() *cproto.MiscDiagnostics {
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
