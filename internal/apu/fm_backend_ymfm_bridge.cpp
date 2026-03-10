//go:build ymfm_cgo

#include "fm_backend_ymfm_bridge.h"

#include <array>
#include <cstdint>
#include <vector>

#include "ymfm_opn.h"

struct ncdx_ymfm_opna : public ymfm::ymfm_interface {
	explicit ncdx_ymfm_opna(uint32_t master_clock)
		: chip(*this), master_clock_hz(master_clock == 0 ? 8000000 : master_clock) {
		chip.set_fidelity(ymfm::OPN_FIDELITY_MIN);
		chip.reset();
		chip_sample_rate = chip.sample_rate(master_clock_hz);
		clocks_per_sample = chip_sample_rate ? (master_clock_hz / chip_sample_rate) : 1;
		if (clocks_per_sample == 0) {
			clocks_per_sample = 1;
		}
		timer_remaining[0] = -1;
		timer_remaining[1] = -1;
	}

	ymfm::ym2608 chip;
	uint8_t addr0 = 0;
	uint8_t addr1 = 0;
	bool irq_asserted = false;
	uint32_t master_clock_hz = 8000000;
	uint32_t chip_sample_rate = 0;
	uint32_t clocks_per_sample = 1;
	uint64_t chip_clock = 0;
	uint64_t busy_until_clock = 0;
	std::array<int64_t, 2> timer_remaining{};
	std::array<std::vector<uint8_t>, ymfm::ACCESS_CLASSES> ext;

	void ymfm_set_busy_end(uint32_t clocks) override { busy_until_clock = chip_clock + clocks; }
	bool ymfm_is_busy() override { return chip_clock < busy_until_clock; }
	void ymfm_update_irq(bool asserted) override { irq_asserted = asserted; }

	void ymfm_set_timer(uint32_t tnum, int32_t duration_in_clocks) override {
		if (tnum > 1) {
			return;
		}
		if (duration_in_clocks < 0) {
			timer_remaining[tnum] = -1;
			return;
		}
		timer_remaining[tnum] = duration_in_clocks;
	}

	uint8_t ymfm_external_read(ymfm::access_class type, uint32_t address) override {
		auto &vec = ext[static_cast<std::size_t>(type)];
		if (address < vec.size()) {
			return vec[address];
		}
		return 0;
	}

	void ymfm_external_write(ymfm::access_class type, uint32_t address, uint8_t data) override {
		auto &vec = ext[static_cast<std::size_t>(type)];
		if (address >= vec.size()) {
			vec.resize(address + 1);
		}
		vec[address] = data;
	}

	void step_timers(uint64_t clocks) {
		chip_clock += clocks;
		for (uint32_t tnum = 0; tnum < 2; tnum++) {
			if (timer_remaining[tnum] < 0) {
				continue;
			}
			timer_remaining[tnum] -= static_cast<int64_t>(clocks);
			int guard = 0;
			while (timer_remaining[tnum] <= 0 && timer_remaining[tnum] >= -0x7fffffff) {
				int64_t overshoot = -timer_remaining[tnum];
				m_engine->engine_timer_expired(tnum);
				guard++;
				if (guard > 32) {
					break;
				}
				if (timer_remaining[tnum] < 0) {
					break;
				}
				if (overshoot > 0) {
					timer_remaining[tnum] -= overshoot;
				}
			}
		}
	}
};

extern "C" {

ncdx_ymfm_opna *ncdx_ymfm_opna_create(uint32_t master_clock_hz) {
	return new ncdx_ymfm_opna(master_clock_hz);
}

void ncdx_ymfm_opna_destroy(ncdx_ymfm_opna *chip) {
	delete chip;
}

void ncdx_ymfm_opna_reset(ncdx_ymfm_opna *chip) {
	if (chip == nullptr) {
		return;
	}
	chip->chip.reset();
	chip->chip_clock = 0;
	chip->busy_until_clock = 0;
	chip->timer_remaining[0] = -1;
	chip->timer_remaining[1] = -1;
}

uint32_t ncdx_ymfm_opna_sample_rate(ncdx_ymfm_opna *chip) {
	if (chip == nullptr) {
		return 0;
	}
	return chip->chip_sample_rate;
}

void ncdx_ymfm_opna_write_port(ncdx_ymfm_opna *chip, uint16_t offset, uint8_t data) {
	if (chip == nullptr) {
		return;
	}
	switch (offset & 0x00ff) {
	case 0x00:
		chip->addr0 = data;
		chip->chip.write(0, data);
		break;
	case 0x01:
		chip->chip.write(1, data);
		break;
	case 0x04:
		chip->addr1 = data;
		chip->chip.write(2, data);
		break;
	case 0x05:
		chip->chip.write(3, data);
		break;
	default:
		break;
	}
}

uint8_t ncdx_ymfm_opna_read_port(ncdx_ymfm_opna *chip, uint16_t offset) {
	if (chip == nullptr) {
		return 0;
	}
	switch (offset & 0x00ff) {
	case 0x00:
		return chip->addr0;
	case 0x01:
		return chip->chip.read(1);
	case 0x02:
		return chip->chip.read_status();
	case 0x04:
		return chip->addr1;
	case 0x05:
		return chip->chip.read(3);
	default:
		return 0;
	}
}

void ncdx_ymfm_opna_step_clocks(ncdx_ymfm_opna *chip, uint64_t clocks) {
	if (chip == nullptr || clocks == 0) {
		return;
	}
	chip->step_timers(clocks);
}

void ncdx_ymfm_opna_generate_sample(ncdx_ymfm_opna *chip, int32_t *left, int32_t *right) {
	if (chip == nullptr) {
		if (left != nullptr) {
			*left = 0;
		}
		if (right != nullptr) {
			*right = 0;
		}
		return;
	}

	ymfm::ym2608::output_data out;
	chip->chip.generate(&out, 1);
	chip->step_timers(chip->clocks_per_sample);

	int32_t l = out.data[0] + out.data[2];
	int32_t r = out.data[1] + out.data[2];
	if (left != nullptr) {
		*left = l;
	}
	if (right != nullptr) {
		*right = r;
	}
}

int ncdx_ymfm_opna_irq_pending(ncdx_ymfm_opna *chip) {
	if (chip == nullptr) {
		return 0;
	}
	return chip->irq_asserted ? 1 : 0;
}

} // extern "C"
