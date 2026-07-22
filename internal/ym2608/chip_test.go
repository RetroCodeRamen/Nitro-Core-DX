package ym2608

import "testing"

func writeReg(c *Chip, port int, addr, data uint8) {
	if port == 0 {
		c.WritePort(Port0Addr, addr)
		c.WritePort(Port0Data, data)
		return
	}
	c.WritePort(Port1Addr, addr)
	c.WritePort(Port1Data, data)
}

func TestDualPortRegisterShadowing(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x28, 0xf0)
	writeReg(c, 1, 0x10, 0x34)

	if got := c.Address(0); got != 0x28 {
		t.Fatalf("port0 address: got 0x%02X, want 0x28", got)
	}
	if got := c.Address(1); got != 0x10 {
		t.Fatalf("port1 address: got 0x%02X, want 0x10", got)
	}
	if got := c.ReadPort(Port0Data); got != 0xf0 {
		t.Fatalf("port0 data readback: got 0x%02X, want 0xF0", got)
	}
	if got := c.ReadPort(Port1Data); got != 0x34 {
		t.Fatalf("port1 data readback: got 0x%02X, want 0x34", got)
	}
}

func TestBusyFlagClearsAfterStep(t *testing.T) {
	c := New(Config{})

	c.WritePort(Port0Addr, 0x22)
	if got := c.Status(); got&StatusBusy == 0 {
		t.Fatalf("busy flag was not set after host write, status=0x%02X", got)
	}

	c.Step(31)
	if got := c.Status(); got&StatusBusy == 0 {
		t.Fatalf("busy flag cleared too early, status=0x%02X", got)
	}

	c.Step(1)
	if got := c.Status(); got&StatusBusy != 0 {
		t.Fatalf("busy flag did not clear after 32 clocks, status=0x%02X", got)
	}
}

func TestTimerAOverflowSetsStatus(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, regTimerAHigh, 0xff)
	writeReg(c, 0, regTimerALow, 0x03)
	writeReg(c, 0, regTimerCtrl, 0x05) // load A + enable A flag

	c.Step(71)
	if got := c.Status(); got&StatusTimerA != 0 {
		t.Fatalf("timer A fired early, status=0x%02X", got)
	}

	c.Step(1)
	if got := c.Status(); got&StatusTimerA != StatusTimerA {
		t.Fatalf("timer A overflow did not set flag, status=0x%02X", got)
	}
	if !c.IRQPending() {
		t.Fatalf("IRQPending false after enabled timer A status")
	}
}

func TestTimerAOverflowWithoutFlagEnableDoesNotSetStatus(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, regTimerAHigh, 0xff)
	writeReg(c, 0, regTimerALow, 0x03)
	writeReg(c, 0, regTimerCtrl, 0x01) // start A only, no flag enable
	c.Step(72)

	if got := c.Status(); got&StatusTimerA != 0 {
		t.Fatalf("timer A flag set without flag enable, status=0x%02X", got)
	}
}

func TestTimerACSMKeyOnOff(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x52, 0x1f)
	writeReg(c, 0, 0x82, 0x03)
	writeReg(c, 0, regTimerAHigh, 0xff)
	writeReg(c, 0, regTimerALow, 0x03)
	writeReg(c, 0, regTimerCtrl, 0x81) // start A + CSM, no flag enable
	c.Step(72)

	for i, op := range c.FM().Channels[2].Operators {
		if op.KeyOn {
			t.Fatalf("CSM operator %d key latch remained on after TimerA key-on/off", i)
		}
		if op.Phase != egRelease && op.Phase != egOff {
			t.Fatalf("CSM operator %d phase: got %v, want release/off", i, op.Phase)
		}
	}
	if got := c.Status(); got&StatusTimerA != 0 {
		t.Fatalf("CSM-only TimerA unexpectedly set status, status=0x%02X", got)
	}
}

func TestTimerClearAndFlagDisable(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, regTimerAHigh, 0xff)
	writeReg(c, 0, regTimerALow, 0x03)
	writeReg(c, 0, regTimerCtrl, 0x05)
	c.Step(72)

	writeReg(c, 0, regTimerCtrl, 0x15) // keep A running, enable A flag, clear A
	if got := c.Status(); got&StatusTimerA != 0 {
		t.Fatalf("timer A clear did not clear flag, status=0x%02X", got)
	}

	c.Step(72)
	if got := c.Status(); got&StatusTimerA != StatusTimerA {
		t.Fatalf("timer A did not continue after clear, status=0x%02X", got)
	}

	writeReg(c, 0, regTimerCtrl, 0x01)
	writeReg(c, 0, regTimerCtrl, 0x11)
	c.Step(72)
	if got := c.Status(); got&StatusTimerA != 0 {
		t.Fatalf("disabled timer A flag should stay clear, status=0x%02X", got)
	}
}

func TestTimerBManualPeriod(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, regTimerB, 0xff)
	writeReg(c, 0, regTimerCtrl, 0x0a) // load B + enable B flag

	c.Step(1151)
	if got := c.Status(); got&StatusTimerB != 0 {
		t.Fatalf("timer B fired early, status=0x%02X", got)
	}
	c.Step(1)
	if got := c.Status(); got&StatusTimerB != StatusTimerB {
		t.Fatalf("timer B overflow did not set flag, status=0x%02X", got)
	}
}

func TestResetClearsState(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x28, 0xf0)
	writeReg(c, 1, 0x10, 0x34)
	writeReg(c, 0, regTimerAHigh, 0xff)
	writeReg(c, 0, regTimerALow, 0x03)
	writeReg(c, 0, regTimerCtrl, 0x05)
	c.Step(72)

	c.WritePort(PortControl, 0x80)

	if got := c.Status(); got != 0 {
		t.Fatalf("status after reset: got 0x%02X, want 0", got)
	}
	if got := c.Register(0, 0x28); got != 0 {
		t.Fatalf("port0 register after reset: got 0x%02X, want 0", got)
	}
	if got := c.Register(1, 0x10); got != 0 {
		t.Fatalf("port1 register after reset: got 0x%02X, want 0", got)
	}
	if c.IRQPending() {
		t.Fatalf("timer status should be cleared by reset")
	}
}

func TestGenerateSampleFixedStartsSilent(t *testing.T) {
	c := New(Config{})

	left, right := c.GenerateSampleFixed()
	if left != 0 || right != 0 {
		t.Fatalf("initial synthesis should be silent, got L=%d R=%d", left, right)
	}
}

func TestPSGToneAGeneratesAudio(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x00, 0x20)
	writeReg(c, 0, 0x01, 0x00)
	writeReg(c, 0, 0x07, 0x3e) // tone A enabled, other tone/noise disabled
	writeReg(c, 0, 0x08, 0x0f)

	nonZero := false
	for i := 0; i < 128; i++ {
		left, right := c.GenerateSampleFixed()
		if left != right {
			t.Fatalf("PSG mono fold-down mismatch: L=%d R=%d", left, right)
		}
		if left != 0 {
			nonZero = true
		}
	}
	if !nonZero {
		t.Fatalf("PSG tone A generated only silence")
	}
}

func TestPSGChannelMaskCanSilenceTone(t *testing.T) {
	psg := NewPSG(DefaultMasterClockHz/8, DefaultSampleRateHz)

	psg.SetReg(0x00, 0x20)
	psg.SetReg(0x01, 0x00)
	psg.SetReg(0x07, 0x3e)
	psg.SetReg(0x08, 0x0f)
	psg.SetChannelMask(0)

	buf := make([]int32, 64)
	psg.Mix(buf)
	for i, sample := range buf {
		if sample != 0 {
			t.Fatalf("masked PSG sample %d: got %d, want 0", i, sample)
		}
	}
}

func TestPSGEnvelopeGeneratesChangingOutput(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x00, 0x10)
	writeReg(c, 0, 0x01, 0x00)
	writeReg(c, 0, 0x07, 0x3e)
	writeReg(c, 0, 0x08, 0x10) // channel A uses envelope
	writeReg(c, 0, 0x0b, 0x01)
	writeReg(c, 0, 0x0c, 0x00)
	writeReg(c, 0, 0x0d, 0x0a)

	seen := map[int16]bool{}
	for i := 0; i < 512; i++ {
		left, _ := c.GenerateSampleFixed()
		seen[left] = true
	}
	if len(seen) < 3 {
		t.Fatalf("PSG envelope output did not vary enough, unique samples=%d", len(seen))
	}
}

func TestPSGRegisterReadbackAfterReset(t *testing.T) {
	c := New(Config{})

	c.WritePort(Port0Addr, 0x07)
	if got := c.ReadPort(Port0Data); got != 0xff {
		t.Fatalf("PSG mixer reset register: got 0x%02X, want 0xFF", got)
	}
	c.WritePort(Port0Addr, 0x0e)
	if got := c.ReadPort(Port0Data); got != 0xff {
		t.Fatalf("PSG IO port A reset register: got 0x%02X, want 0xFF", got)
	}
}

func TestCStyleOPNARegRead(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x07, 0x3e)
	if got := c.OPNAReg(0x07); got != 0x3e {
		t.Fatalf("OPNAReg PSG read: got 0x%02X, want 0x3E", got)
	}
	if got := c.OPNAReg(0xff); got != 1 {
		t.Fatalf("OPNAReg 0xFF: got 0x%02X, want 1", got)
	}
	if got := c.OPNAReg(0x28); got != 0 {
		t.Fatalf("OPNAReg non-PSG: got 0x%02X, want 0", got)
	}
}

func TestChipChannelMaskControlsPSGAndFM(t *testing.T) {
	c := New(Config{})

	c.SetChannelMask(0)
	writeReg(c, 0, 0x00, 0x20)
	writeReg(c, 0, 0x01, 0x00)
	writeReg(c, 0, 0x07, 0x3e)
	writeReg(c, 0, 0x08, 0x0f)
	writeReg(c, 0, 0xb0, 0x07)
	writeReg(c, 0, 0xa4, 0x24)
	writeReg(c, 0, 0xa0, 0x68)
	writeReg(c, 0, 0x40, 0x00)
	writeReg(c, 0, 0x50, 0x1f)
	writeReg(c, 0, 0x80, 0x00)
	writeReg(c, 0, 0x28, 0x10)

	for i := 0; i < 32; i++ {
		if left, right := c.GenerateSampleFixed(); left != 0 || right != 0 {
			t.Fatalf("masked chip generated audio: L=%d R=%d", left, right)
		}
	}
}

func TestRhythmKeyOnGeneratesAudio(t *testing.T) {
	c := New(Config{})
	c.SetRhythmSample(0, []int8{8, 16, -8, -16}, DefaultSampleRateHz)

	writeReg(c, 0, 0x11, 0x00)
	writeReg(c, 0, 0x18, 0xc0)
	writeReg(c, 0, 0x10, 0x01)

	nonZero := false
	for i := 0; i < 4; i++ {
		left, right := c.GenerateSampleFixed()
		if left != right {
			t.Fatalf("rhythm mono fold-down mismatch: L=%d R=%d", left, right)
		}
		if left != 0 {
			nonZero = true
		}
	}
	if !nonZero {
		t.Fatalf("rhythm key-on generated only silence")
	}
}

func TestRhythmDumpStopsVoice(t *testing.T) {
	c := New(Config{})
	c.SetRhythmSample(0, []int8{32, 32, 32, 32}, DefaultSampleRateHz)

	writeReg(c, 0, 0x18, 0xc0)
	writeReg(c, 0, 0x10, 0x01)
	if left, _ := c.GenerateSampleFixed(); left == 0 {
		t.Fatalf("expected rhythm voice before dump")
	}

	writeReg(c, 0, 0x10, 0x81)
	for i := 0; i < 4; i++ {
		if left, right := c.GenerateSampleFixed(); left != 0 || right != 0 {
			t.Fatalf("rhythm voice continued after dump: L=%d R=%d", left, right)
		}
	}
}

func TestFMKeyOnPort0AndPort1ChannelMapping(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x28, 0xf2) // ops 1-4 on, channel 2
	for i, op := range c.FM().Channels[2].Operators {
		if !op.KeyOn {
			t.Fatalf("channel 2 operator %d key-on false", i)
		}
	}

	writeReg(c, 0, 0x28, 0x14) // op 1 on, high-bank channel 3
	if !c.FM().Channels[3].Operators[0].KeyOn {
		t.Fatalf("channel 3 operator 0 key-on false")
	}
	if c.FM().Channels[3].Operators[1].KeyOn {
		t.Fatalf("channel 3 operator 1 key-on true")
	}
}

func TestFMFNumDecodeAcrossPorts(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0xa4, 0x2a)
	writeReg(c, 0, 0xa0, 0x34)
	if got := c.FM().FNum(0); got != 0x2a34 {
		t.Fatalf("channel 0 fnum: got 0x%04X, want 0x2A34", got)
	}
	wantDP := (uint32(0x2a34) & 2047) << ((uint32(0x2a34) >> 11) & 7)
	if got := c.FM().Channels[0].Operators[0].DP; got != wantDP {
		t.Fatalf("channel 0 operator DP: got %d, want %d", got, wantDP)
	}

	writeReg(c, 1, 0xa5, 0x1b)
	writeReg(c, 1, 0xa1, 0x55)
	if got := c.FM().FNum(4); got != 0x1b55 {
		t.Fatalf("channel 4 fnum: got 0x%04X, want 0x1B55", got)
	}
}

func TestFMAlgorithmFeedbackAndPMSAMS(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0xb0, 0x3f)
	ch := c.FM().Channels[0]
	if ch.Algorithm != 7 {
		t.Fatalf("algorithm: got %d, want 7", ch.Algorithm)
	}
	if ch.Feedback != 1 {
		t.Fatalf("feedback: got %d, want 1", ch.Feedback)
	}
	if ch.Index != [6]uint8{1, 1, 1, 0, 0, 0} {
		t.Fatalf("algorithm index: got %+v", ch.Index)
	}

	writeReg(c, 1, 0xb6, 0xa5)
	for i, op := range c.FM().Channels[5].Operators {
		if op.MS != 0xa5 {
			t.Fatalf("channel 5 operator %d MS: got 0x%02X, want 0xA5", i, op.MS)
		}
	}
}

func TestFMOperatorRegisterDecode(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x34, 0x71) // channel 0, slot 2 by C slottable
	op := c.FM().Channels[0].Operators[2]
	if op.Detune != 0xe0 || op.Multiple != 0x01 {
		t.Fatalf("DT/MULTI decode: got detune=0x%02X multiple=0x%X", op.Detune, op.Multiple)
	}

	writeReg(c, 1, 0x52, 0xdf) // channel 5, slot 0
	op = c.FM().Channels[5].Operators[0]
	if op.KS != 3 || op.AR != 62 {
		t.Fatalf("KS/AR decode: got KS=%d AR=%d", op.KS, op.AR)
	}

	writeReg(c, 0, 0x80, 0xf3)
	op = c.FM().Channels[0].Operators[0]
	if op.SL != 60 || op.RR != 14 {
		t.Fatalf("SL/RR decode: got SL=%d RR=%d", op.SL, op.RR)
	}
}

func TestFMSimpleCarrierGeneratesAudio(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0xb0, 0x07) // ALG7: all operators directly audible
	writeReg(c, 0, 0xa4, 0x24)
	writeReg(c, 0, 0xa0, 0x68)
	writeReg(c, 0, 0x40, 0x00) // operator 0 TL loud
	writeReg(c, 0, 0x50, 0x1f) // operator 0 fast attack
	writeReg(c, 0, 0x80, 0x00) // sustain immediately
	writeReg(c, 0, 0x28, 0x10) // key-on operator 0 channel 0

	nonZero := false
	for i := 0; i < 256; i++ {
		left, right := c.GenerateSampleFixed()
		if left != right {
			t.Fatalf("FM mono fold-down mismatch: L=%d R=%d", left, right)
		}
		if left != 0 {
			nonZero = true
		}
	}
	if !nonZero {
		t.Fatalf("FM simple carrier generated only silence")
	}
}

func TestFMPhaseProgressesAfterKeyOn(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0xb0, 0x07)
	writeReg(c, 0, 0xa4, 0x24)
	writeReg(c, 0, 0xa0, 0x68)
	writeReg(c, 0, 0x40, 0x00)
	writeReg(c, 0, 0x50, 0x1f)
	writeReg(c, 0, 0x80, 0x00)
	writeReg(c, 0, 0x28, 0x10)

	before := c.FM().Channels[0].Operators[0].PGCount
	for i := 0; i < 8; i++ {
		c.GenerateSampleFixed()
	}
	after := c.FM().Channels[0].Operators[0].PGCount
	if after == before {
		t.Fatalf("FM operator phase did not advance after key-on")
	}
}

func TestFMKeyOffMovesOperatorTowardRelease(t *testing.T) {
	c := New(Config{})

	writeReg(c, 0, 0x50, 0x1f)
	writeReg(c, 0, 0x80, 0x03)
	writeReg(c, 0, 0x28, 0x10)
	if !c.FM().Channels[0].Operators[0].KeyOn {
		t.Fatalf("operator did not key on")
	}

	writeReg(c, 0, 0x28, 0x00)
	op := c.FM().Channels[0].Operators[0]
	if op.KeyOn {
		t.Fatalf("operator key-on latch remained set after key-off")
	}
	if op.Phase != egRelease && op.Phase != egOff {
		t.Fatalf("operator phase after key-off: got %v, want release/off", op.Phase)
	}
}

func TestBackendShapeForFutureAPUSwap(t *testing.T) {
	b := NewBackend(Config{})

	b.Write8(Port0Addr, 0x22)
	b.Write8(Port0Data, 0x08)
	if got := b.Read8(Port0Data); got != 0x08 {
		t.Fatalf("backend data readback: got 0x%02X, want 0x08", got)
	}

	b.SetSampleRate(48000)
	if got := b.Chip().Config().SampleRateHz; got != 48000 {
		t.Fatalf("backend sample rate: got %d, want 48000", got)
	}

	b.SetEnabledMuted(true, false)
	if got := b.GenerateSampleFixed(); got != 0 {
		t.Fatalf("backend should stay silent until synthesis slices land, got %d", got)
	}
}
