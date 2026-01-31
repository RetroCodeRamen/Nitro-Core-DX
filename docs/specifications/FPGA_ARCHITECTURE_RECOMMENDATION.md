# FPGA Architecture Recommendation

**Version 1.0**  
**Last Updated: January 30, 2026**  
**Purpose: FPGA implementation strategy and chip selection for Nitro-Core-DX**

---

## Executive Summary

**Recommendation: Single FPGA Solution**

For cost-effective production, use **one medium-sized FPGA** (similar to MiSTer's approach but smaller). This provides the best balance of cost, complexity, and performance.

**Recommended FPGA**: Xilinx Spartan-7 (XC7S50) or Intel Cyclone IV (EP4CE22)
**Estimated Cost**: $15-30 per FPGA chip (volume pricing)
**Total System Cost**: ~$50-80 per console (FPGA + supporting components)

---

## System Resource Requirements Analysis

### Component Complexity Breakdown

| Component | Estimated LUTs | Estimated BRAM | Notes |
|-----------|----------------|----------------|-------|
| **CPU Core** | 5,000-10,000 | 2-4 blocks | Custom 10 MHz CPU, simple instruction set |
| **PPU (Graphics)** | 15,000-30,000 | 20-40 blocks | Most complex: 4 layers, 128 sprites, matrix mode |
| **APU (Audio)** | 2,000-5,000 | 4-8 blocks | 4 channels, waveform generation |
| **Memory Controllers** | 3,000-5,000 | 10-20 blocks | WRAM, VRAM, CGRAM, OAM, cartridge |
| **I/O Interfaces** | 1,000-2,000 | 0-2 blocks | Controllers, expansion port |
| **System Logic** | 2,000-3,000 | 2-4 blocks | Clocking, reset, synchronization |
| **TOTAL** | **28,000-55,000 LUTs** | **40-78 BRAM blocks** | Need 30-50% headroom |

### Memory Requirements

| Memory Type | Size | Implementation |
|-------------|------|----------------|
| WRAM | 32KB | BRAM (8 blocks @ 4KB each) |
| Extended WRAM | 128KB | BRAM (32 blocks) or external SRAM |
| VRAM | 64KB | BRAM (16 blocks) |
| CGRAM | 512 bytes | BRAM (1 block) |
| OAM | 768 bytes | BRAM (1 block) |
| **Total BRAM** | **~58 blocks** | Can use external SRAM for WRAM |

**Note**: External SRAM can reduce BRAM usage significantly (save ~40 blocks by using external SRAM for WRAM).

---

## FPGA Architecture Options

### Option 1: Single FPGA (RECOMMENDED) ⭐

**Architecture:**
```
┌─────────────────────────────────────┐
│         Single FPGA                 │
│  ┌──────┐  ┌──────┐  ┌──────┐      │
│  │ CPU  │  │ PPU  │  │ APU  │      │
│  └──┬───┘  └──┬───┘  └──┬───┘      │
│     │         │         │           │
│  ┌──▼─────────▼─────────▼───┐      │
│  │   Memory Bus & I/O        │      │
│  └───────────────────────────┘      │
└─────────────────────────────────────┘
```

**Pros:**
- ✅ **Lowest cost** (~$15-30 per FPGA)
- ✅ **Simplest PCB** (single chip, easier routing)
- ✅ **No inter-FPGA communication** (no latency, no sync issues)
- ✅ **Lower power consumption** (one chip vs multiple)
- ✅ **Easier debugging** (all logic in one place)
- ✅ **Lower manufacturing cost** (simpler assembly)
- ✅ **Better performance** (no communication overhead)

**Cons:**
- ⚠️ Less modular (can't upgrade CPU/PPU separately)
- ⚠️ All eggs in one basket (if FPGA fails, whole system fails)

**Recommended FPGAs:**
1. **Xilinx Spartan-7 XC7S50** (~$20-25)
   - 52,000 LUTs, 180 BRAM blocks
   - Good headroom for future features
   - Modern, well-supported

2. **Intel Cyclone IV EP4CE22** (~$15-20)
   - 22,000 LUTs, 66 BRAM blocks
   - Might be tight, but cheaper
   - Well-documented, many examples

3. **Lattice ECP5 LFE5U-25F** (~$18-22)
   - 24,000 LUTs, 84 BRAM blocks
   - Open-source toolchain (Project Trellis)
   - Good for hobbyist projects

**Cost Breakdown (Single FPGA):**
- FPGA chip: $15-30
- PCB (4-layer): $5-10
- Supporting components: $10-15
- Connectors, power, etc.: $10-15
- **Total per console: ~$40-70**

---

### Option 2: Two FPGAs

**Architecture:**
```
┌──────────────┐      ┌──────────────┐
│  FPGA 1      │      │  FPGA 2      │
│  ┌────────┐  │      │  ┌────────┐  │
│  │  CPU   │  │◄────►│  │  PPU   │  │
│  │ Memory │  │      │  │  APU   │  │
│  │  Bus   │  │      │  │        │  │
│  └────────┘  │      │  └────────┘  │
└──────────────┘      └──────────────┘
```

**Pros:**
- ✅ Some modularity (can upgrade CPU separately)
- ✅ Can use smaller, cheaper FPGAs

**Cons:**
- ❌ **Higher cost** (2× FPGA chips = $30-60)
- ❌ **Complex inter-FPGA communication** (needs high-speed bus)
- ❌ **More complex PCB** (routing between chips, signal integrity)
- ❌ **Synchronization challenges** (clock domains, data sync)
- ❌ **Higher power consumption** (2× chips)
- ❌ **More debugging complexity** (two chips to debug)
- ❌ **Higher manufacturing cost** (more components, more assembly)

**FPGA Options:**
- FPGA 1 (CPU): Xilinx Spartan-7 XC7S15 (~$8-12)
- FPGA 2 (PPU/APU): Xilinx Spartan-7 XC7S25 (~$12-18)
- **Total: ~$20-30** (but more complex PCB)

**Cost Breakdown (Two FPGAs):**
- FPGA chips: $20-30
- PCB (6-layer, complex routing): $10-20
- Supporting components: $15-20
- Connectors, power, etc.: $15-20
- **Total per console: ~$60-90**

---

### Option 3: Three FPGAs

**Architecture:**
```
┌──────────┐    ┌──────────┐    ┌──────────┐
│ FPGA 1   │    │ FPGA 2   │    │ FPGA 3   │
│  CPU     │◄──►│  PPU     │◄──►│  APU     │
│ Memory   │    │          │    │          │
└──────────┘    └──────────┘    └──────────┘
```

**Pros:**
- ✅ Maximum modularity
- ✅ Can use very small FPGAs

**Cons:**
- ❌ **Highest cost** (3× FPGA chips = $24-45)
- ❌ **Very complex inter-FPGA communication** (3-way bus)
- ❌ **Most complex PCB** (6-8 layer, very difficult routing)
- ❌ **Severe synchronization challenges** (3 clock domains)
- ❌ **Highest power consumption** (3× chips)
- ❌ **Most debugging complexity** (three chips)
- ❌ **Highest manufacturing cost** (most components, most assembly)
- ❌ **Not cost-effective** for this system

**Cost Breakdown (Three FPGAs):**
- FPGA chips: $24-45
- PCB (8-layer, very complex): $20-30
- Supporting components: $20-25
- Connectors, power, etc.: $20-25
- **Total per console: ~$84-125**

---

## Detailed Cost Analysis

### Production Volume Considerations

| Volume | Single FPGA | Two FPGAs | Three FPGAs |
|--------|-------------|-----------|------------|
| **1-10 units** | $50-80 | $70-110 | $100-150 |
| **100 units** | $40-60 | $60-85 | $90-120 |
| **1,000 units** | $30-50 | $50-70 | $80-100 |
| **10,000 units** | $25-40 | $45-60 | $70-85 |

**Note**: Single FPGA scales better with volume due to simpler manufacturing.

### PCB Complexity Impact

| Option | PCB Layers | Routing Complexity | Manufacturing Cost |
|--------|------------|-------------------|-------------------|
| Single FPGA | 4 layers | Low | $5-10 |
| Two FPGAs | 6 layers | Medium-High | $10-20 |
| Three FPGAs | 8 layers | Very High | $20-30 |

**Key Point**: Multi-FPGA designs require more PCB layers and complex routing, significantly increasing cost.

---

## Performance Considerations

### Inter-FPGA Communication Overhead

**Single FPGA:**
- All communication is internal (no overhead)
- Clock domain crossing is handled by FPGA tools
- No latency between components

**Two/Three FPGAs:**
- Need high-speed bus between chips (LVDS or parallel bus)
- Additional latency (10-50ns per transfer)
- Clock domain synchronization required
- Signal integrity challenges (crosstalk, timing)

**Impact on System:**
- CPU-PPU communication: Critical for rendering
- CPU-APU communication: Less critical (audio buffering)
- Multi-FPGA adds latency that could affect frame timing

---

## Recommended FPGA Selection

### Primary Recommendation: Xilinx Spartan-7 XC7S50

**Specifications:**
- **LUTs**: 52,000 (plenty of headroom)
- **BRAM**: 180 blocks (more than enough)
- **DSP Slices**: 120 (useful for matrix math)
- **I/O Pins**: 200+ (enough for all interfaces)
- **Cost**: ~$20-25 (volume pricing)
- **Toolchain**: Vivado (free WebPack edition)

**Why This FPGA:**
- ✅ Enough resources for entire system + 30-50% headroom
- ✅ Good price point
- ✅ Modern architecture (7-series)
- ✅ Well-documented
- ✅ Good toolchain support
- ✅ Available in multiple package sizes

### Alternative: Intel Cyclone IV EP4CE22

**Specifications:**
- **LEs**: 22,000 (equivalent to ~22,000 LUTs)
- **M9K Blocks**: 66 (enough for memory)
- **I/O Pins**: 153 (enough for interfaces)
- **Cost**: ~$15-20 (volume pricing)
- **Toolchain**: Quartus Prime Lite (free)

**Why This FPGA:**
- ✅ Lower cost
- ✅ Still enough resources (tight but workable)
- ✅ Well-documented (many retro console projects use it)
- ⚠️ Might be tight on resources (need optimization)

### Budget Option: Lattice ECP5 LFE5U-25F

**Specifications:**
- **LUTs**: 24,000 (might be tight)
- **EBR**: 84 blocks (enough)
- **I/O Pins**: 335 (plenty)
- **Cost**: ~$18-22 (volume pricing)
- **Toolchain**: Open-source (Project Trellis)

**Why This FPGA:**
- ✅ Open-source toolchain (no vendor lock-in)
- ✅ Good for hobbyist projects
- ⚠️ Might need optimization to fit
- ⚠️ Less community support than Xilinx/Intel

---

## Implementation Strategy

### Phase 1: Prototype (Single FPGA)

1. **Start with Xilinx Spartan-7 XC7S50**
   - Use development board (Arty S7-50, ~$100)
   - Prototype entire system on one FPGA
   - Verify all components work together

2. **Optimize as needed**
   - Profile resource usage
   - Optimize PPU (biggest consumer)
   - Consider external SRAM for WRAM if needed

### Phase 2: Production Design

1. **Custom PCB with single FPGA**
   - Use same FPGA or smaller if optimization allows
   - Design 4-layer PCB
   - Include all connectors (cartridge, controllers, expansion)

2. **Cost optimization**
   - Volume pricing for FPGA
   - Optimize component selection
   - Consider alternative FPGAs if cost is critical

---

## Comparison with MiSTer

**MiSTer uses**: Intel Cyclone V (DE-10 Nano)
- **LUTs**: ~110,000
- **Cost**: ~$200 (development board)
- **Why**: MiSTer emulates multiple systems, needs more resources

**Nitro-Core-DX needs**: ~28,000-55,000 LUTs
- **Recommendation**: Smaller FPGA (Spartan-7 XC7S50 or Cyclone IV)
- **Cost**: ~$15-30 (chip only)
- **Why**: Single system, simpler requirements

**Key Difference**: MiSTer is a multi-system emulator, Nitro-Core-DX is a single-system console. We don't need MiSTer's resources.

---

## Power Consumption Analysis

### Single FPGA

| Component | Power | Notes |
|-----------|-------|-------|
| FPGA (active) | 0.5-1.5W | Depends on utilization |
| External SRAM | 0.1-0.3W | If used for WRAM |
| Supporting components | 0.2-0.5W | Regulators, etc. |
| **Total** | **0.8-2.3W** | Very efficient |

### Two FPGAs

| Component | Power | Notes |
|-----------|-------|-------|
| FPGA 1 (CPU) | 0.3-0.8W | |
| FPGA 2 (PPU/APU) | 0.5-1.2W | |
| Inter-FPGA bus | 0.1-0.2W | Communication overhead |
| External SRAM | 0.1-0.3W | |
| Supporting components | 0.3-0.6W | |
| **Total** | **1.3-3.1W** | Higher power |

**Conclusion**: Single FPGA is more power-efficient.

---

## Final Recommendation

### ✅ Use Single FPGA: Xilinx Spartan-7 XC7S50

**Rationale:**
1. **Cost-effective**: $20-25 per chip (vs $30-60 for two, $45-75 for three)
2. **Simpler design**: Single chip, easier PCB, lower manufacturing cost
3. **Better performance**: No inter-FPGA communication overhead
4. **Sufficient resources**: 52,000 LUTs is plenty (need ~28,000-55,000)
5. **Lower power**: Single chip is more efficient
6. **Easier debugging**: All logic in one place
7. **Production-friendly**: Simpler assembly, higher yield

**Estimated Production Cost:**
- **Prototype**: ~$100-150 (development board)
- **Small batch (10-100)**: ~$50-80 per console
- **Medium batch (1,000)**: ~$40-60 per console
- **Large batch (10,000+)**: ~$30-50 per console

**This is cost-effective for a retro console project!**

---

## Future Considerations

### If Resources Become Tight

1. **Use external SRAM for WRAM**
   - Reduces BRAM usage by ~40 blocks
   - Adds one SRAM chip (~$2-3)
   - Still cheaper than second FPGA

2. **Optimize PPU**
   - Use DSP slices for matrix math
   - Pipeline rendering operations
   - Optimize sprite rendering

3. **Consider larger FPGA**
   - Xilinx Spartan-7 XC7S75 (~$25-30)
   - More headroom for future features

### If Cost Becomes Critical

1. **Use Intel Cyclone IV EP4CE22**
   - Lower cost (~$15-20)
   - Need careful optimization
   - Still single FPGA (simpler than multi-FPGA)

2. **Use external SRAM**
   - Reduces FPGA BRAM requirements
   - Allows smaller/cheaper FPGA

---

## Conclusion

**Single FPGA is the clear winner** for Nitro-Core-DX:

- ✅ **Lowest cost** (~$30-50 per console in production)
- ✅ **Simplest design** (easier to manufacture and debug)
- ✅ **Best performance** (no communication overhead)
- ✅ **Sufficient resources** (Spartan-7 XC7S50 has plenty of headroom)
- ✅ **Production-friendly** (simpler PCB, higher yield)

**Multi-FPGA designs add cost and complexity without significant benefits** for this system.

---

## References

- **Hardware Specification**: `HARDWARE_SPECIFICATION.md`
- **Cartridge Pin Spec**: `CARTRIDGE_PIN_SPECIFICATION.md`
- **Controller Pin Spec**: `CONTROLLER_PIN_SPECIFICATION.md`
- **MiSTer Project**: https://github.com/MiSTer-devel/Main_MiSTer/wiki

---

**End of Recommendation**
