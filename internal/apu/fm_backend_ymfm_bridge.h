#pragma once

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ncdx_ymfm_opna ncdx_ymfm_opna;

ncdx_ymfm_opna *ncdx_ymfm_opna_create(uint32_t master_clock_hz);
void ncdx_ymfm_opna_destroy(ncdx_ymfm_opna *chip);
void ncdx_ymfm_opna_reset(ncdx_ymfm_opna *chip);

uint32_t ncdx_ymfm_opna_sample_rate(ncdx_ymfm_opna *chip);

void ncdx_ymfm_opna_write_port(ncdx_ymfm_opna *chip, uint16_t offset, uint8_t data);
uint8_t ncdx_ymfm_opna_read_port(ncdx_ymfm_opna *chip, uint16_t offset);

void ncdx_ymfm_opna_step_clocks(ncdx_ymfm_opna *chip, uint64_t clocks);
void ncdx_ymfm_opna_generate_sample(ncdx_ymfm_opna *chip, int32_t *left, int32_t *right);
int ncdx_ymfm_opna_irq_pending(ncdx_ymfm_opna *chip);

#ifdef __cplusplus
}
#endif
