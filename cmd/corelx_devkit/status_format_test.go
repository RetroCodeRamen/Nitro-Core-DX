package main

import (
	"testing"

	"nitro-core-dx/internal/devkit"
)

func TestFormatFrameClock(t *testing.T) {
	tests := []struct {
		name      string
		frame     uint64
		wantClock string
	}{
		{name: "zero", frame: 0, wantClock: "00:00.00"},
		{name: "oneSecond", frame: 60, wantClock: "00:01.00"},
		{name: "tenSeconds", frame: 600, wantClock: "00:10.00"},
		{name: "oneMinute", frame: 3600, wantClock: "01:00.00"},
		{name: "fractional", frame: 123, wantClock: "00:02.05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatFrameClock(tt.frame); got != tt.wantClock {
				t.Fatalf("formatFrameClock(%d) = %q, want %q", tt.frame, got, tt.wantClock)
			}
		})
	}
}

func TestFormatFrameMark(t *testing.T) {
	tests := []struct {
		name string
		snap devkit.EmulatorSnapshot
		want string
	}{
		{
			name: "running",
			snap: devkit.EmulatorSnapshot{
				Loaded:            true,
				Running:           true,
				Paused:            false,
				FPS:               59.9,
				CPUCyclesPerFrame: 127820,
				FrameCount:        420,
			},
			want: "Frame mark: frame=420 time=00:07.00 fps=59.9 cpu=127820 state=running",
		},
		{
			name: "paused",
			snap: devkit.EmulatorSnapshot{
				Loaded:            true,
				Running:           true,
				Paused:            true,
				FPS:               60.0,
				CPUCyclesPerFrame: 100,
				FrameCount:        60,
			},
			want: "Frame mark: frame=60 time=00:01.00 fps=60.0 cpu=100 state=paused",
		},
		{
			name: "stopped",
			snap: devkit.EmulatorSnapshot{
				Loaded:            true,
				Running:           false,
				Paused:            false,
				FPS:               0,
				CPUCyclesPerFrame: 0,
				FrameCount:        0,
			},
			want: "Frame mark: frame=0 time=00:00.00 fps=0.0 cpu=0 state=stopped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatFrameMark(tt.snap); got != tt.want {
				t.Fatalf("formatFrameMark(%+v) = %q, want %q", tt.snap, got, tt.want)
			}
		})
	}
}
