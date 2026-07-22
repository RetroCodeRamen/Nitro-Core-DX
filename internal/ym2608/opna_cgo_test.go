//go:build cgo

package ym2608

import "testing"

func TestOPNAPSGPathMatchesPMDWinCThroughFullChip(t *testing.T) {
	writes := []opnaWrite{
		{0x00, 0x20},
		{0x01, 0x00},
		{0x07, 0x3e},
		{0x08, 0x0f},
	}

	goBuf := renderGoOPNA(writes, 256)
	cBuf := mustRenderCOPNA(t, writes, 256)
	assertInt16Equal(t, goBuf, cBuf)
}

func TestOPNAPSGPrescalerMatchesPMDWinC(t *testing.T) {
	for _, prescalerReg := range []uint16{0x2d, 0x2e, 0x2f} {
		t.Run(itoa(int(prescalerReg)), func(t *testing.T) {
			writes := []opnaWrite{
				{prescalerReg, 0x00},
				{0x00, 0x20},
				{0x01, 0x00},
				{0x07, 0x3e},
				{0x08, 0x0f},
			}
			assertOPNAProgramMatches(t, writes, 256)
		})
	}
}

func TestOPNARhythmBassDrumMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0x11, 0x00},
		{0x18, 0xc0},
		{0x10, 0x01},
	}
	assertOPNAProgramMatches(t, writes, 512)
}

func TestOPNARhythmAllVoicesMatchPMDWinC(t *testing.T) {
	cases := []struct {
		name string
		reg  uint16
		key  uint8
	}{
		{"bass", 0x18, 0x01},
		{"snare", 0x19, 0x02},
		{"top", 0x00, 0x04},
		{"hihat", 0x1b, 0x08},
		{"tom", 0x1c, 0x10},
		{"rim", 0x1d, 0x20},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writes := []opnaWrite{{0x11, 0x00}}
			if tc.reg != 0 {
				writes = append(writes, opnaWrite{tc.reg, 0xc0})
			}
			writes = append(writes, opnaWrite{0x10, tc.key})
			assertOPNAProgramMatches(t, writes, 512)
		})
	}
}

func TestOPNARhythmDumpDuringPlaybackMatchesPMDWinC(t *testing.T) {
	events := []opnaEvent{
		{writes: []opnaWrite{{0x11, 0x00}, {0x18, 0xc0}, {0x10, 0x01}}},
		{at: 64, writes: []opnaWrite{{0x10, 0x81}}},
	}
	assertOPNAEventsMatch(t, events, 256)
}

func TestOPNARhythmSetVolumeMatchesPMDWinC(t *testing.T) {
	for _, db := range []int{-24, -6, 0, 12, 20, 32} {
		t.Run(itoa(db), func(t *testing.T) {
			events := []opnaEvent{
				{
					rhythmVolumes: []opnaRhythmVolume{{index: 0, db: db}},
					writes:        []opnaWrite{{0x11, 0x00}, {0x18, 0xc0}, {0x10, 0x01}},
				},
			}
			assertOPNAEventsMatch(t, events, 256)
		})
	}
}

func TestOPNASetChannelMaskMatchesPMDWinC(t *testing.T) {
	cases := []struct {
		name string
		mask uint32
	}{
		{"all-muted", 0x000},
		{"fm0-only", 0x001},
		{"psg-a-only", 0x040},
		{"fm0-and-psg-a", 0x041},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := []opnaEvent{
				{
					channelMask:    tc.mask,
					setChannelMask: true,
					writes: []opnaWrite{
						{0x00, 0x20},
						{0x01, 0x00},
						{0x07, 0x3e},
						{0x08, 0x0f},
						{0xb0, 0x07},
						{0xa4, 0x24},
						{0xa0, 0x68},
						{0x40, 0x00},
						{0x50, 0x1f},
						{0x80, 0x00},
						{0x28, 0x10},
					},
				},
			}
			assertOPNAEventsMatch(t, events, 256)
		})
	}
}

func TestOPNAInit48000MatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0x00, 0x20},
		{0x01, 0x00},
		{0x07, 0x3e},
		{0x08, 0x0f},
		{0x11, 0x00},
		{0x18, 0xc0},
		{0x10, 0x01},
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x40, 0x00},
		{0x50, 0x1f},
		{0x80, 0x00},
		{0x28, 0x10},
	}
	goBuf := renderGoOPNAWithConfig(writes, 512, Config{SampleRateHz: 48000})
	cBuf, err := renderCOPNAReferenceWithRate(writes, 512, 48000)
	if err != nil {
		t.Fatal(err)
	}
	assertInt16Equal(t, goBuf, cBuf)
}

func TestOPNAInitInterpolatedMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x40, 0x00},
		{0x50, 0x1f},
		{0x80, 0x00},
		{0x28, 0x10},
	}
	goBuf := renderGoOPNAWithConfig(writes, 512, Config{Interpolation: true})
	cBuf, err := renderCOPNAReferenceWithRateAndInterpolation(writes, 512, DefaultSampleRateHz, true)
	if err != nil {
		t.Fatal(err)
	}
	assertInt16Equal(t, goBuf, cBuf)
}

func TestOPNASetRateDuringPlaybackMatchesPMDWinC(t *testing.T) {
	events := []opnaEvent{
		{writes: simpleFMCarrierProgram()},
		{at: 64, sampleRate: 48000, setSampleRate: true},
	}
	assertOPNAEventsMatch(t, events, 256)
}

func TestOPNASetRateInterpolatedDuringPlaybackMatchesPMDWinC(t *testing.T) {
	events := []opnaEvent{
		{writes: simpleFMCarrierProgram()},
		{at: 64, sampleRate: 48000, interpolation: true, setSampleRate: true},
	}
	assertOPNAEventsMatch(t, events, 256)
}

func TestOPNAFMCarrierMatchesPMDWinC(t *testing.T) {
	assertOPNAProgramMatches(t, simpleFMCarrierProgram(), 256)
}

func TestOPNAFMOperator1CarrierMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x48, 0x00},
		{0x58, 0x1f},
		{0x88, 0x00},
		{0x28, 0x20},
	}
	assertOPNAProgramMatches(t, writes, 256)
}

func TestOPNAFMHighTotalLevelPMDWinUBDiagnostic(t *testing.T) {
	writes := []opnaWrite{
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x40, 0x7f},
		{0x50, 0x1f},
		{0x80, 0x00},
		{0x28, 0x10},
	}
	goBuf := renderGoOPNA(writes, 256)
	cBuf := mustRenderCOPNA(t, writes, 256)
	t.Logf("PMDWin high-TL UB diagnostic: %s", firstInt16Mismatch(goBuf, cBuf))
}

func TestOPNAFMMultiCarrierMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x40, 0x00},
		{0x48, 0x08},
		{0x50, 0x1f},
		{0x58, 0x1f},
		{0x80, 0x00},
		{0x88, 0x00},
		{0x28, 0x30},
	}
	assertOPNAProgramMatches(t, writes, 256)
}

func TestOPNAFMAlgorithmsMatchPMDWinC(t *testing.T) {
	for alg := uint8(0); alg < 8; alg++ {
		t.Run(itoa(int(alg)), func(t *testing.T) {
			writes := fmAlgorithmProgram(alg)
			assertOPNAProgramMatches(t, writes, 512)
		})
	}
}

func TestOPNAFMLFOMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0x22, 0x0f},
		{0xb4, 0x10},
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x40, 0x00},
		{0x50, 0x1f},
		{0x60, 0x80},
		{0x80, 0x00},
		{0x28, 0x10},
	}
	assertOPNAProgramMatches(t, writes, 1024)
}

func TestOPNAFMSSGEGLatchMatchesPMDWinC(t *testing.T) {
	base := simpleFMCarrierProgram()
	withSSGEG := append([]opnaWrite{{0x90, 0x0f}}, base...)

	goBase := renderGoOPNA(base, 256)
	goSSGEG := renderGoOPNA(withSSGEG, 256)
	cSSGEG := mustRenderCOPNA(t, withSSGEG, 256)
	assertInt16Equal(t, goSSGEG, cSSGEG)
	assertInt16Equal(t, goSSGEG, goBase)
}

func TestOPNAFMChannel3SpecialFNumMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0xb2, 0x07},
		{0xa6, 0x24},
		{0xa2, 0x68},
		{0xac, 0x25},
		{0xa8, 0x00},
		{0xad, 0x26},
		{0xa9, 0x00},
		{0xae, 0x27},
		{0xaa, 0x00},
		{0x42, 0x00},
		{0x52, 0x1f},
		{0x82, 0x00},
		{0x27, 0x40},
		{0x28, 0x12},
	}
	assertOPNAProgramMatches(t, writes, 512)
}

func TestOPNAFMPort1CarrierMatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0x29, 0x9f},
		{0x1b0, 0x07},
		{0x1a4, 0x24},
		{0x1a0, 0x68},
		{0x140, 0x00},
		{0x150, 0x1f},
		{0x180, 0x00},
		{0x28, 0x14},
	}
	assertOPNAProgramMatches(t, writes, 256)
}

func TestOPNAFMPort1CarrierDisabledByReg29MatchesPMDWinC(t *testing.T) {
	writes := []opnaWrite{
		{0x1b0, 0x07},
		{0x1a4, 0x24},
		{0x1a0, 0x68},
		{0x140, 0x00},
		{0x150, 0x1f},
		{0x180, 0x00},
		{0x28, 0x14},
	}

	goBuf := renderGoOPNA(writes, 256)
	cBuf := mustRenderCOPNA(t, writes, 256)
	assertInt16Equal(t, goBuf, cBuf)
	if !allZero(cBuf) {
		t.Fatalf("expected high-bank FM to be silent without reg29 bit 7")
	}
}

func TestOPNAFMKeyOffReleaseMatchesPMDWinC(t *testing.T) {
	writes := append(simpleFMCarrierProgram(), opnaWrite{0x28, 0x00})
	assertOPNAProgramMatches(t, writes, 256)
}

func TestOPNAFMKeyOffDuringPlaybackMatchesPMDWinC(t *testing.T) {
	events := []opnaEvent{
		{writes: simpleFMCarrierProgram()},
		{at: 64, writes: []opnaWrite{{0x28, 0x00}}},
	}
	assertOPNAEventsMatch(t, events, 256)
}

func TestOPNATimerCountMatchesPMDWinC(t *testing.T) {
	cases := []struct {
		name   string
		writes []opnaWrite
		steps  []int32
	}{
		{
			name:   "timer-a-min-flag-enabled",
			writes: []opnaWrite{{0x24, 0xff}, {0x25, 0x03}, {0x27, 0x05}},
			steps:  []int32{18, 1, 18, 1},
		},
		{
			name:   "timer-a-started-flag-disabled",
			writes: []opnaWrite{{0x24, 0xff}, {0x25, 0x03}, {0x27, 0x01}},
			steps:  []int32{19},
		},
		{
			name:   "timer-b-min-flag-enabled",
			writes: []opnaWrite{{0x26, 0xff}, {0x27, 0x0a}},
			steps:  []int32{287, 1, 287, 1},
		},
		{
			name:   "prescaler-two-timer-a",
			writes: []opnaWrite{{0x2f, 0x00}, {0x24, 0xff}, {0x25, 0x03}, {0x27, 0x05}},
			steps:  []int32{6, 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			goResults := runGoTimer(tc.writes, tc.steps)
			cResults := mustRunCOPNATimer(t, tc.writes, tc.steps)
			if len(goResults) != len(cResults) {
				t.Fatalf("timer result length: go=%d c=%d", len(goResults), len(cResults))
			}
			for i := range goResults {
				if goResults[i] != cResults[i] {
					t.Fatalf("timer step %d: go=%+v c=%+v", i, goResults[i], cResults[i])
				}
			}
		})
	}
}

func simpleFMCarrierProgram() []opnaWrite {
	return []opnaWrite{
		{0xb0, 0x07},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x40, 0x00},
		{0x50, 0x1f},
		{0x80, 0x00},
		{0x28, 0x10},
	}
}

func fmAlgorithmProgram(alg uint8) []opnaWrite {
	return []opnaWrite{
		{0xb0, alg | 0x08},
		{0xa4, 0x24},
		{0xa0, 0x68},
		{0x30, 0x01},
		{0x34, 0x01},
		{0x38, 0x01},
		{0x3c, 0x01},
		{0x40, 0x08},
		{0x44, 0x10},
		{0x48, 0x14},
		{0x4c, 0x18},
		{0x50, 0x1f},
		{0x54, 0x1f},
		{0x58, 0x1f},
		{0x5c, 0x1f},
		{0x60, 0x00},
		{0x64, 0x00},
		{0x68, 0x00},
		{0x6c, 0x00},
		{0x70, 0x00},
		{0x74, 0x00},
		{0x78, 0x00},
		{0x7c, 0x00},
		{0x80, 0x00},
		{0x84, 0x00},
		{0x88, 0x00},
		{0x8c, 0x00},
		{0x28, 0xf0},
	}
}

func renderGoOPNA(writes []opnaWrite, samples int) []int16 {
	return renderGoOPNAEvents([]opnaEvent{{writes: writes}}, samples)
}

func renderGoOPNAWithConfig(writes []opnaWrite, samples int, cfg Config) []int16 {
	return renderGoOPNAEventsWithConfig([]opnaEvent{{writes: writes}}, samples, cfg)
}

func renderGoOPNAEvents(events []opnaEvent, samples int) []int16 {
	return renderGoOPNAEventsWithConfig(events, samples, Config{})
}

func renderGoOPNAEventsWithConfig(events []opnaEvent, samples int, cfg Config) []int16 {
	chip := New(cfg)
	out := make([]int16, samples)
	cursor := 0
	for _, event := range events {
		for cursor < event.at {
			left, _ := chip.GenerateSampleFixed()
			out[cursor] = left
			cursor++
		}
		applyGoOPNAEvent(chip, event)
	}
	for cursor < len(out) {
		left, _ := chip.GenerateSampleFixed()
		out[cursor] = left
		cursor++
	}
	return out
}

func applyGoOPNAEvent(chip *Chip, event opnaEvent) {
	if event.setChannelMask {
		chip.SetChannelMask(event.channelMask)
	}
	if event.setSampleRate {
		chip.SetRate(event.sampleRate, event.interpolation)
	}
	for _, volume := range event.rhythmVolumes {
		chip.SetRhythmVolume(volume.index, volume.db)
	}
	for _, w := range event.writes {
		port := int((w.addr >> 8) & 0x01)
		writeReg(chip, port, uint8(w.addr), w.data)
	}
}

func assertOPNAProgramMatches(t *testing.T, writes []opnaWrite, samples int) {
	t.Helper()
	goBuf := renderGoOPNA(writes, samples)
	cBuf := mustRenderCOPNA(t, writes, samples)
	if allZero(cBuf) {
		t.Fatalf("C OPNA reference produced only silence")
	}
	assertInt16Equal(t, goBuf, cBuf)
}

func assertOPNAEventsMatch(t *testing.T, events []opnaEvent, samples int) {
	t.Helper()
	goBuf := renderGoOPNAEvents(events, samples)
	cBuf := mustRenderCOPNAEvents(t, events, samples)
	assertInt16Equal(t, goBuf, cBuf)
}

func mustRenderCOPNA(t *testing.T, writes []opnaWrite, samples int) []int16 {
	t.Helper()
	out, err := renderCOPNAReference(writes, samples)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func mustRenderCOPNAEvents(t *testing.T, events []opnaEvent, samples int) []int16 {
	t.Helper()
	out, err := renderCOPNAReferenceEvents(events, samples)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func mustRunCOPNATimer(t *testing.T, writes []opnaWrite, steps []int32) []opnaTimerResult {
	t.Helper()
	out, err := runCOPNATimerReference(writes, steps)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func runGoTimer(writes []opnaWrite, steps []int32) []opnaTimerResult {
	chip := New(Config{})
	for _, w := range writes {
		port := int((w.addr >> 8) & 0x01)
		writeReg(chip, port, uint8(w.addr), w.data)
	}
	out := make([]opnaTimerResult, len(steps))
	for i, us := range steps {
		event := chip.TimerCount(us)
		out[i] = opnaTimerResult{
			event:  event,
			status: chip.Status() & (StatusTimerA | StatusTimerB),
		}
	}
	return out
}

func assertInt16Equal(t *testing.T, got, want []int16) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got=%d want=%d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("first mismatch at %d: got=%d want=%d", i, got[i], want[i])
		}
	}
}

func allZero(samples []int16) bool {
	for _, sample := range samples {
		if sample != 0 {
			return false
		}
	}
	return true
}

func firstInt16Mismatch(a, b []int16) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return "index " + itoa(i) + ": go=" + itoa(int(a[i])) + " c=" + itoa(int(b[i]))
		}
	}
	if len(a) != len(b) {
		return "length: go=" + itoa(len(a)) + " c=" + itoa(len(b))
	}
	return "none"
}
