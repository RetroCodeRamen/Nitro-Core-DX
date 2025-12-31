#!/usr/bin/env python3
"""
Performance test script for Nitro-Core-DX emulator
Tests CPU and PPU performance with optimizations
"""

import sys
import os
import time

# Add src_python to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src_python'))

import config
import cpu
import ppu
import rom
import memory

def test_cpu_performance():
    """Test CPU execution speed"""
    print("=" * 60)
    print("CPU Performance Test")
    print("=" * 60)
    
    # Reset emulator
    config.emulator = config.EmulatorState()
    cpu.cpu_reset()
    memory.memory_reset()
    
    # Load test ROM
    rom_file = os.path.join(os.path.dirname(__file__), '..', 'roms', 'graphics.rom')
    if not os.path.exists(rom_file):
        print(f"ERROR: ROM file not found: {rom_file}")
        return
    
    rom.rom_load_file(rom_file)
    cpu.cpu_reset()
    
    # Test: Execute 166,667 cycles (one frame worth)
    cycles_per_frame = config.CPU_CYCLES_PER_FRAME
    target_cycles = cycles_per_frame
    
    print(f"Target: {cycles_per_frame:,} cycles (one frame at 10 MHz)")
    print("Running CPU for one frame...")
    
    start_time = time.time()
    cpu.cpu_run_cycles(target_cycles)
    end_time = time.time()
    
    elapsed = end_time - start_time
    actual_cycles = config.emulator.cpu.cycles
    cycles_per_second = actual_cycles / elapsed if elapsed > 0 else 0
    fps_estimate = cycles_per_second / cycles_per_frame if cycles_per_second > 0 else 0
    
    print(f"\nResults:")
    print(f"  Elapsed time: {elapsed*1000:.2f} ms")
    print(f"  Cycles executed: {actual_cycles:,}")
    print(f"  Cycles/second: {cycles_per_second:,.0f}")
    print(f"  Estimated FPS (CPU only): {fps_estimate:.2f}")
    print(f"  Target FPS: 60.00")
    print(f"  CPU performance: {fps_estimate/60*100:.1f}% of target")


def test_ppu_performance():
    """Test PPU rendering speed"""
    print("\n" + "=" * 60)
    print("PPU Performance Test")
    print("=" * 60)
    
    # Reset emulator
    config.emulator = config.EmulatorState()
    ppu.ppu_reset()
    
    # Setup a simple scene (enable BG0 with a tile)
    emu = config.emulator
    emu.ppu.bg0.enabled = True
    emu.ppu.bg0.tile_size = 8
    emu.ppu.bg0.scroll_x = 0
    emu.ppu.bg0.scroll_y = 0
    emu.ppu.bg0.tile_data_base = 0x0000
    emu.ppu.bg0.tile_map_base = 0x1000
    
    # Create a simple tile in VRAM (white tile)
    # Tile data: 8x8 tile, all pixels = 1 (white)
    tile_data = [0x11] * 32  # 4bpp: 2 pixels per byte, 8 rows * 4 bytes = 32 bytes
    for i in range(32):
        if i < len(config.ppu_vram):
            config.ppu_vram[i] = tile_data[i]
    
    # Set tilemap entry (tile 1 at position 0,0)
    tilemap_addr = 0x1000
    if tilemap_addr + 1 < len(config.ppu_vram):
        config.ppu_vram[tilemap_addr] = 1  # Tile index
        config.ppu_vram[tilemap_addr + 1] = 0x10  # Palette 1
    
    # Set palette color 1 to white
    ppu.ppu_set_palette_color(1, 31, 31, 31)
    
    print("Rendering 100 frames...")
    
    start_time = time.time()
    for i in range(100):
        ppu.ppu_render_frame()
    end_time = time.time()
    
    elapsed = end_time - start_time
    fps = 100 / elapsed if elapsed > 0 else 0
    ms_per_frame = (elapsed / 100) * 1000 if elapsed > 0 else 0
    
    print(f"\nResults:")
    print(f"  Frames rendered: 100")
    print(f"  Elapsed time: {elapsed*1000:.2f} ms")
    print(f"  Average time per frame: {ms_per_frame:.2f} ms")
    print(f"  Estimated FPS (PPU only): {fps:.2f}")
    print(f"  Target FPS: 60.00")
    print(f"  PPU performance: {fps/60*100:.1f}% of target")


def test_combined_performance():
    """Test combined CPU + PPU performance"""
    print("\n" + "=" * 60)
    print("Combined CPU + PPU Performance Test")
    print("=" * 60)
    
    # Reset emulator
    config.emulator = config.EmulatorState()
    cpu.cpu_reset()
    ppu.ppu_reset()
    memory.memory_reset()
    
    # Load test ROM
    rom_file = os.path.join(os.path.dirname(__file__), '..', 'roms', 'graphics.rom')
    if not os.path.exists(rom_file):
        print(f"ERROR: ROM file not found: {rom_file}")
        return
    
    rom.rom_load_file(rom_file)
    cpu.cpu_reset()
    
    # Setup PPU
    emu = config.emulator
    emu.ppu.bg0.enabled = True
    
    print("Running 60 frames (1 second at target FPS)...")
    
    start_time = time.time()
    for frame in range(60):
        # Run CPU for one frame
        target_cycles = emu.cpu.cycles + config.CPU_CYCLES_PER_FRAME
        cpu.cpu_run_cycles(target_cycles)
        
        # Render frame
        ppu.ppu_render_frame()
        
        # Update frame counter
        emu.frame_count += 1
    end_time = time.time()
    
    elapsed = end_time - start_time
    fps = 60 / elapsed if elapsed > 0 else 0
    ms_per_frame = (elapsed / 60) * 1000 if elapsed > 0 else 0
    
    print(f"\nResults:")
    print(f"  Frames: 60")
    print(f"  Elapsed time: {elapsed*1000:.2f} ms")
    print(f"  Average time per frame: {ms_per_frame:.2f} ms")
    print(f"  Actual FPS: {fps:.2f}")
    print(f"  Target FPS: 60.00")
    print(f"  Overall performance: {fps/60*100:.1f}% of target")
    
    if fps >= 60:
        print("  ✅ Performance target met!")
    elif fps >= 30:
        print("  ⚠️  Performance acceptable but below target")
    else:
        print("  ❌ Performance below acceptable threshold")


if __name__ == "__main__":
    print("Nitro-Core-DX Performance Test")
    print("=" * 60)
    print()
    
    # Disable logging for performance test
    import ui
    ui.logger.enabled = False
    
    try:
        test_cpu_performance()
        test_ppu_performance()
        test_combined_performance()
        
        print("\n" + "=" * 60)
        print("Performance Test Complete")
        print("=" * 60)
    except Exception as e:
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()

