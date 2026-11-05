// Package frame implements the CLVX app-protocol wire framing for both transports.
//
// USB (HID, vendor interface usagePage 0xFF00):
//
//	payload = ascii(base64(protobuf)) || 0x0A
//	split into 63-byte chunks, each sent as a 64-byte output report [0x23, chunk...]
//	input reports are [0x24, 63 bytes]; strip the ID, accumulate until 0x0A.
//
// BLE (GATT vendor characteristic, see vendor index.js sendCrc path):
//
//	payload = 0x23 || base64(protobuf || LE32(crc32(protobuf))) || 0x0A
//	split into 56-byte writes; notifications accumulate until 0x0A, first byte 0x23.
package frame

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	Terminator   = 0x0A
	USBReportOut = 0x23
	USBReportIn  = 0x24
	USBChunk     = 63
	USBReportLen = 64
	BLEPrefix    = 0x23
	BLEChunk     = 56
)

var ErrNoTerminator = errors.New("frame: no terminator yet")

// USBReports returns the 64-byte output reports for one protobuf message.
func USBReports(pb []byte) [][]byte {
	payload := append([]byte(base64.StdEncoding.EncodeToString(pb)), Terminator)
	var reports [][]byte
	for i := 0; i < len(payload); i += USBChunk {
		end := min(i+USBChunk, len(payload))
		r := make([]byte, USBReportLen)
		r[0] = USBReportOut
		copy(r[1:], payload[i:end])
		reports = append(reports, r)
	}
	return reports
}

// USBDecodeInput consumes one 64-byte input report into acc and returns the
// decoded protobuf once the terminator has arrived.
func USBDecodeInput(acc []byte, report []byte) ([]byte, []byte, error) {
	if len(report) < 2 || report[0] != USBReportIn {
		return acc, nil, errors.New("frame: unexpected input report id")
	}
	// The device zero-pads each report; base64 never contains NUL, so padding
	// is dropped before accumulating (base64 may span report boundaries).
	acc = append(acc, bytes.ReplaceAll(report[1:], []byte{0}, nil)...)
	i := bytes.IndexByte(acc, Terminator)
	if i < 0 {
		return acc, nil, ErrNoTerminator
	}
	pb, err := base64.StdEncoding.DecodeString(string(acc[:i]))
	return nil, pb, err
}

// BLEWrites returns the chunks to write for one protobuf message.
func BLEWrites(pb []byte) [][]byte {
	crc := make([]byte, 4)
	binary.LittleEndian.PutUint32(crc, crc32.ChecksumIEEE(pb))
	body := base64.StdEncoding.EncodeToString(append(append([]byte{}, pb...), crc...))
	payload := append([]byte{BLEPrefix}, []byte(body)...)
	payload = append(payload, Terminator)
	var out [][]byte
	for i := 0; i < len(payload); i += BLEChunk {
		out = append(out, payload[i:min(i+BLEChunk, len(payload))])
	}
	return out
}

// BLEDecode validates a complete notification stream (up to and including the
// terminator) and returns the protobuf with its CRC removed.
func BLEDecode(stream []byte) ([]byte, error) {
	i := bytes.IndexByte(stream, Terminator)
	if i < 0 {
		return nil, ErrNoTerminator
	}
	msg := stream[:i]
	if len(msg) == 0 || msg[0] != BLEPrefix {
		return nil, errors.New("frame: bad BLE prefix")
	}
	body, err := base64.StdEncoding.DecodeString(string(msg[1:]))
	if err != nil {
		return nil, err
	}
	if len(body) < 4 {
		return nil, errors.New("frame: BLE body too short")
	}
	pb, crc := body[:len(body)-4], binary.LittleEndian.Uint32(body[len(body)-4:])
	if crc32.ChecksumIEEE(pb) != crc {
		return nil, errors.New("frame: BLE CRC mismatch")
	}
	return pb, nil
}
