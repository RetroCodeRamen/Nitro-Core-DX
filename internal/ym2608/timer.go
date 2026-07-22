package ym2608

type timer struct {
	running    bool
	flagEnable bool
	period     uint64
	counter    uint64
}

func (t *timer) start(period uint64) {
	if period == 0 {
		period = 1
	}
	t.running = true
	t.period = period
	t.counter = period
}

func (t *timer) stop() {
	t.running = false
	t.counter = 0
}

func (t *timer) step(clocks uint64) bool {
	if !t.running || t.counter == 0 || clocks == 0 {
		return false
	}

	expired := false
	remaining := clocks
	for remaining >= t.counter {
		remaining -= t.counter
		expired = true
		t.counter = t.period
		if t.counter == 0 {
			t.counter = 1
		}
	}
	t.counter -= remaining
	return expired
}
