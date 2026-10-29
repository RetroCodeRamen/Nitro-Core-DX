package ymstream

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

func buildVGM(rate uint32, commands []byte) []byte {
	n := 0x40 + len(commands)
	if n < 0x100 {
		n = 0x100
	}
	data := make([]byte, n)
	copy(data[0:4], "Vgm ")
	binary.LittleEndian.PutUint32(data[0x08:0x0C], 0x100)
	binary.LittleEndian.PutUint32(data[0x24:0x28], rate)
	copy(data[0x40:], commands)
	return data
}

func TestConvertVGMRoundTrip(t *testing.T) {
	raw := buildVGM(60, []byte{
		0x56, 0x10, 0xAA,
		0x62,
		0x66,
	})
	encoded, report, err := ConvertVGM(raw)
	if err != nil {
		t.Fatalf("ConvertVGM: %v", err)
	}
	if report.FrameCount != 1 || report.WriteCount != 1 {
		t.Fatalf("report frames/writes = %d/%d, want 1/1", report.FrameCount, report.WriteCount)
	}
	if report.FrameSamples != 735 {
		t.Fatalf("frame samples = %d, want 735", report.FrameSamples)
	}
	if report.ROMBytes != 9 {
		t.Fatalf("ROM footprint = %d, want 9", report.ROMBytes)
	}
	if report.CompactBytes != len(encoded) {
		t.Fatalf("compact size = %d, encoded len = %d", report.CompactBytes, len(encoded))
	}
	if report.TempoNote != "" {
		t.Fatalf("unexpected tempo note: %s", report.TempoNote)
	}
	if report.Duration != time.Second/60 {
		t.Fatalf("duration = %s, want one 60 Hz frame", report.Duration)
	}

	decoded, err := DecodeStream(encoded)
	if err != nil {
		t.Fatalf("DecodeStream: %v", err)
	}
	if len(decoded.Frames) != 1 || len(decoded.Frames[0]) != 1 {
		t.Fatalf("decoded frames = %#v", decoded.Frames)
	}
	got := decoded.Frames[0][0]
	if got.Port != 0 || got.Addr != 0x10 || got.Data != 0xAA {
		t.Fatalf("decoded write = %+v", got)
	}
}

func TestConvertVGMGzip(t *testing.T) {
	raw := buildVGM(60, []byte{0x56, 0x22, 0x33, 0x62, 0x66})
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	encoded, report, err := ConvertVGM(buf.Bytes())
	if err != nil {
		t.Fatalf("ConvertVGM gzip: %v", err)
	}
	if report.WriteCount != 1 || report.CompactBytes != len(encoded) {
		t.Fatalf("gzip report = %+v", report)
	}
}

func TestConvertVGMRejectsUnsupportedCommand(t *testing.T) {
	raw := buildVGM(60, []byte{0x50, 0x00, 0x66})
	encoded, _, err := ConvertVGM(raw)
	if err == nil {
		t.Fatal("expected unsupported command error")
	}
	if encoded != nil {
		t.Fatalf("expected no output bytes, got %d", len(encoded))
	}
	msg := err.Error()
	if !strings.Contains(msg, "unsupported VGM command 0x50") || !strings.Contains(msg, "0x40") {
		t.Fatalf("error = %q", msg)
	}
}

func TestConvertVGMCountsSkippedDataBlock(t *testing.T) {
	raw := buildVGM(60, []byte{
		0x67, 0x66, 0x00,
		0x01, 0x00, 0x00, 0x00,
		0xFF,
		0x56, 0x10, 0x01,
		0x62,
		0x66,
	})
	_, report, err := ConvertVGM(raw)
	if err != nil {
		t.Fatalf("ConvertVGM: %v", err)
	}
	if report.SkippedDataBlocks != 1 {
		t.Fatalf("skipped blocks = %d, want 1", report.SkippedDataBlocks)
	}
	if !strings.Contains(report.String(), "Skipped data blocks: 1") {
		t.Fatalf("report text missing skip count:\n%s", report.String())
	}
	if report.WriteCount != 1 {
		t.Fatalf("writes = %d, want 1", report.WriteCount)
	}
}

func TestConvertVGM50HzNote(t *testing.T) {
	raw := buildVGM(50, []byte{0x56, 0x10, 0xAA, 0x63, 0x66})
	_, report, err := ConvertVGM(raw)
	if err != nil {
		t.Fatalf("ConvertVGM: %v", err)
	}
	if report.FrameSamples != 882 {
		t.Fatalf("frame samples = %d, want 882", report.FrameSamples)
	}
	if !strings.Contains(report.TempoNote, "50 Hz") || !strings.Contains(report.TempoNote, "60 Hz") {
		t.Fatalf("tempo note = %q", report.TempoNote)
	}
}
