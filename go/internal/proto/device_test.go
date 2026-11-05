package proto

import (
	"bytes"
	"testing"
)

func TestDeviceInfoCapturedShapeAndUnknownFields(t *testing.T) {
	mode := testMessage(testVarintField(1, uint64(OSLinux)), testVarintField(2, 1))
	keyboard := testMessage(testVarintField(1, 2), testVarintField(2, 1), testVarintField(3, 2), testVarintField(4, 1), testVarintField(5, 42), testVarintField(6, 3))
	kbm := testMessage(testBytesField(1, keyboard), testBytesField(2, testMessage(testVarintField(1, 3), testVarintField(2, 232))), testBytesField(3, testMessage(testVarintField(1, 7), testVarintField(2, 9), testBytesField(3, []byte{1, 2, 3}))), testBytesField(4, testMessage(testVarintField(1, 3), testVarintField(2, 24))), testVarintField(5, 0), testVarintField(6, 2))
	raw := testMessage(
		testBytesField(1, []byte("CLVX 1")), testVarintField(3, 1234), testBytesField(9, []byte("ABC123")), testVarintField(10, 9876),
		testVarintField(11, 2816), testVarintField(12, 1), testVarintField(13, 1), testBytesField(14, kbm), testVarintField(15, 8),
		testBytesField(16, []byte{0xde, 0xad, 0xbe, 0xef}), testVarintField(17, 1), testBytesField(18, mode), testVarintField(19, 2), testVarintField(20, 7),
		testVarintField(99, 123),
	)
	info, err := DecodeDeviceInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if *info.Name != "CLVX 1" || *info.FWVersion != 1234 || *info.BarCode != "ABC123" || *info.SerialNumber != 9876 || *info.ProtocolVersion != 2816 || !*info.Insider || *info.DeviceType != "CLVX" || *info.CLVXVersion != "S" {
		t.Fatalf("device info mismatch: %+v", info)
	}
	if *info.DeviceMode.OSMode != OSLinux || info.DeviceMode.OSMode.String() != "LINUX" || *info.DeviceMode.InputMode != "TOUCHPAD" || !bytes.Equal(info.ChipUID, []byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("device mode/identity mismatch: %+v", info)
	}
	if info.KBMInfo.Keyboard == nil || *info.KBMInfo.Keyboard.OS != "LINUX" || *info.KBMInfo.Type != "TYPE_293" || !bytes.Equal(info.Raw, raw) {
		t.Fatalf("KBM info/raw mismatch: %+v", info.KBMInfo)
	}
	parsed, err := ParseDeviceInfoResponse(Response{Type: 2, Field: 5, Body: testBytesField(1, raw)})
	if err != nil || *parsed.Name != *info.Name {
		t.Fatalf("ParseDeviceInfoResponse: %+v, %v", parsed, err)
	}
}

func TestOSModeAndStatusResponses(t *testing.T) {
	if mode, err := ParseOSMode(""); err != nil || mode != OSLinux {
		t.Fatalf("default mode = %v, %v", mode, err)
	}
	if _, err := ParseOSMode("PLAN9"); err == nil {
		t.Fatal("invalid OS mode accepted")
	}
	pb, err := EncodeSetOSMode(OSLinux)
	if err != nil {
		t.Fatal(err)
	}
	req, err := DecodeRequestForTest(pb)
	if err != nil {
		t.Fatal(err)
	}
	if req.reqType != ReqSetOSMode || req.field != 10 || !bytes.Equal(req.body, testVarintField(1, 2)) {
		t.Fatalf("SetOSMode envelope: %+v", req)
	}
	status, err := ParseStatusResponse(Response{Type: 9, Field: 11, Body: testVarintField(1, 0)}, 11)
	if err != nil || status != 0 {
		t.Fatalf("SetOSMode status = %d, %v", status, err)
	}
	status, err = ParseStatusResponse(Response{Type: 12, Field: 14, Body: testVarintField(1, 1)}, 14)
	if err != nil || status != 1 {
		t.Fatalf("restart status = %d, %v", status, err)
	}
}

func TestStatisticsAndDiagnosticsFixture(t *testing.T) {
	gestures := testMessage(testVarintField(1, 10), testVarintField(2, 11), testVarintField(3, 12), testVarintField(4, 13), testVarintField(5, 14), testVarintField(6, 15), testVarintField(7, 16), testVarintField(8, 17), testVarintField(9, 18), testVarintField(10, 19), testVarintField(11, 20))
	channel := testMessage(testVarintField(1, 100), testBytesField(2, testMessage(testVarintField(1, 2), testVarintField(2, 90))))
	raw := testMessage(
		testBytesField(1, testVarintField(1, 12345)), testBytesField(2, gestures), testBytesField(3, testMessage(testVarintField(1, 4), testVarintField(2, 5))),
		testBytesField(4, testMessage(testBytesField(1, channel), testBytesField(2, channel))), testBytesField(5, testMessage(testVarintField(1, 200), testVarintField(2, 300))), testVarintField(99, 1),
	)
	stats, err := ParseStatisticsResponse(Response{Type: 14, Field: 16, Body: raw})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Keyboard.KeyPressedCounter != 12345 || stats.Gestures.HoldCount != 20 || stats.Sliders.RightSliderCount != 5 || stats.Channels.BT1.TotalTimeS != 100 || stats.Channels.BT1.OSTimes[0].OS != OSLinux || stats.TypeTouchOnly.TouchOnlyTimeS != 300 || !bytes.Equal(stats.Raw, raw) {
		t.Fatalf("statistics mismatch: %+v", stats)
	}
	diagRaw := testMessage(testVarintField(1, 1), testVarintField(2, 0), testVarintField(4, 1), testVarintField(3, 99))
	diag, err := ParseMiscDiagnosticsResponse(Response{Type: 13, Field: 15, Body: diagRaw})
	if err != nil || !*diag.TypeOnlyEnabled || *diag.BatteryMaxBrightnessEn || !*diag.IsWinSettings || !bytes.Equal(diag.Raw, diagRaw) {
		t.Fatalf("diagnostics mismatch: %+v, %v", diag, err)
	}
}
