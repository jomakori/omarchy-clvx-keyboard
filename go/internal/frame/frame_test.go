package frame

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestUSBRoundTrip(t *testing.T) {
	pb := []byte{0x08, 0x03, 0x32, 0x00}
	reports := USBReports(pb)
	if len(reports) != 1 || len(reports[0]) != USBReportLen || reports[0][0] != USBReportOut {
		t.Fatalf("unexpected report layout: %d reports", len(reports))
	}
	// Feed the same bytes back as input reports (ID 0x24) and decode.
	var acc []byte
	var got []byte
	for _, r := range reports {
		in := append([]byte{USBReportIn}, r[1:]...)
		var err error
		acc, got, err = USBDecodeInput(acc, in)
		if err != nil && err != ErrNoTerminator {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(got, pb) {
		t.Fatalf("round trip mismatch: %x", got)
	}
}

func TestBLEVectors(t *testing.T) {
	// Vector observed live from the keyboard: GET_SETTINGS request 1200.
	writes := BLEWrites([]byte{0x12, 0x00})
	var stream []byte
	for _, w := range writes {
		stream = append(stream, w...)
	}
	if stream[0] != BLEPrefix || stream[len(stream)-1] != Terminator {
		t.Fatalf("bad BLE framing: %q", stream)
	}
	body, err := base64.StdEncoding.DecodeString(string(stream[1 : len(stream)-1]))
	if err != nil || !bytes.Equal(body[:2], []byte{0x12, 0x00}) || len(body) != 6 {
		t.Fatalf("payload/CRC layout wrong: %x %v", body, err)
	}
	// Device-side replies use the same framing; decode a synthetic reply.
	pb, err := BLEDecode(stream)
	if err != nil || !bytes.Equal(pb, []byte{0x12, 0x00}) {
		t.Fatalf("BLEDecode: %x %v", pb, err)
	}
}

func TestUSBPaddedInput(t *testing.T) {
	// Replies arrive zero-padded per report and base64 spans reports.
	pb := []byte("hello-protobuf-payload-xyz") // any bytes
	b64 := []byte(base64.StdEncoding.EncodeToString(pb) + "\n")
	half := len(b64) / 2
	r1 := append([]byte{USBReportIn}, b64[:half]...)
	r1 = append(r1, make([]byte, USBReportLen-len(r1))...)
	r2 := append([]byte{USBReportIn}, b64[half:]...)
	r2 = append(r2, make([]byte, USBReportLen-len(r2))...)
	var acc, got []byte
	var err error
	acc, got, err = USBDecodeInput(acc, r1)
	if err != ErrNoTerminator {
		t.Fatalf("first report: %v", err)
	}
	_, got, err = USBDecodeInput(acc, r2)
	if err != nil || !bytes.Equal(got, pb) {
		t.Fatalf("padded reassembly failed: %q %v", got, err)
	}
}
