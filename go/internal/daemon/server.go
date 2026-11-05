package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	cproto "clvx/internal/proto"
)

// SocketPath returns $XDG_RUNTIME_DIR/clvx/clvx.sock (or a /tmp fallback).
func SocketPath() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "clvx", "clvx.sock")
	}
	return filepath.Join(os.TempDir(), "clvx-"+os.Getenv("USER")+".sock")
}

// Request/Response are newline-delimited JSON on the Unix socket.
type Request struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	ID     int    `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Serve accepts clients until ctx is cancelled.
func Serve(ctx context.Context, d *Daemon, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return err
	}
	go func() { <-ctx.Done(); ln.Close(); os.Remove(path) }()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go handle(ctx, d, conn)
	}
}

func handle(ctx context.Context, d *Daemon, conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	enc := json.NewEncoder(conn)
	for sc.Scan() {
		var req Request
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			_ = enc.Encode(Response{Error: "bad json"})
			continue
		}
		switch req.Method {
		case "status":
			_ = enc.Encode(Response{ID: req.ID, Result: d.Snapshot()})
		case "get_settings":
			settings, err := d.GetSettings()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: settings})
		case "set_global":
			var p struct {
				Global *cproto.Global `json:"global"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			if p.Global == nil {
				_ = enc.Encode(Response{ID: req.ID, Error: "set_global needs a global object"})
				continue
			}
			applied, err := d.SetGlobal(p.Global)
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: applied})
		case "get_profile_settings":
			var p struct {
				ProfileID uint32 `json:"profileId"`
			}
			if len(req.Params) > 0 {
				if err := json.Unmarshal(req.Params, &p); err != nil {
					_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
					continue
				}
			}
			if p.ProfileID != 0 {
				_ = enc.Encode(Response{ID: req.ID, Error: "only profile id 0 is supported"})
				continue
			}
			profile, err := d.GetProfileSettings()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: profile})
		case "set_profile_settings":
			var p struct {
				Settings *cproto.ProfileSettings `json:"settings"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			if p.Settings == nil {
				_ = enc.Encode(Response{ID: req.ID, Error: "set_profile_settings needs settings.raw from get_profile_settings"})
				continue
			}
			profile, err := d.SetProfileSettings(p.Settings)
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: profile})
		case "device_info":
			info, err := d.GetDeviceInfo()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: info})
		case "default_settings":
			settings, err := d.GetDefaultSettings()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: settings})
		case "set_os_mode":
			var p struct {
				Mode string `json:"mode"`
			}
			if len(req.Params) > 0 {
				if err := json.Unmarshal(req.Params, &p); err != nil {
					_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
					continue
				}
			}
			mode, err := cproto.ParseOSMode(p.Mode)
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			status, err := d.SetOSMode(mode)
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: status})
		case "restart":
			status, err := d.PerformRestart()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: status})
		case "factory_reset":
			var p struct {
				Confirm bool `json:"confirm"`
			}
			if len(req.Params) == 0 {
				req.Params = json.RawMessage(`{}`)
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			if !p.Confirm {
				_ = enc.Encode(Response{ID: req.ID, Error: "factory_reset requires {\"confirm\":true}"})
				continue
			}
			status, err := d.PerformFullReset()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: status})
		case "statistics":
			statistics, err := d.GetStatistics()
			if err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			_ = enc.Encode(Response{ID: req.ID, Result: statistics})
		case "diagnostics":
			_ = enc.Encode(Response{ID: req.ID, Result: d.GetDiagnostics()})
		case "subscribe":
			ch, cancel := d.Subscribe()
			_ = enc.Encode(Response{ID: req.ID, Result: d.Snapshot()})
			for {
				select {
				case <-ctx.Done():
					cancel()
					return
				case s := <-ch:
					if err := enc.Encode(Response{ID: req.ID, Result: s}); err != nil {
						cancel()
						return
					}
				}
			}
		default:
			_ = enc.Encode(Response{ID: req.ID, Error: "unknown method " + req.Method})
		}
	}
}
