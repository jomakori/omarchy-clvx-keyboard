package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
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
		case "set_profile":
			var p struct {
				Profile uint32 `json:"profile"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				_ = enc.Encode(Response{ID: req.ID, Error: err.Error()})
				continue
			}
			d.SetProfile(p.Profile)
			_ = enc.Encode(Response{ID: req.ID, Result: d.Snapshot()})
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
