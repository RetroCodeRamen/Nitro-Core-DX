#!/usr/bin/env python3
"""
Profile CPU execution to identify bottlenecks
"""

import sys
import os
import cProfile
import pstats
import io

# Add src_python to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src_python'))

import config
import cpu
import memory
import rom

def profile_cpu_execution():
    """Profile CPU execution to find bottlenecks"""
    print("Profiling CPU execution...")
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
    
    # Disable logging for profiling
    import ui
    ui.logger.enabled = False
    
    # Profile CPU execution
    profiler = cProfile.Profile()
    profiler.enable()
    
    # Run CPU for 10,000 cycles (small sample for profiling)
    target_cycles = config.emulator.cpu.cycles + 10000
    cpu.cpu_run_cycles(target_cycles)
    
    profiler.disable()
    
    # Analyze results
    s = io.StringIO()
    ps = pstats.Stats(profiler, stream=s)
    ps.sort_stats('cumulative')
    ps.print_stats(30)  # Top 30 functions
    
    print(s.getvalue())
    
    # Also print by time spent
    print("\n" + "=" * 60)
    print("Top functions by time spent:")
    print("=" * 60)
    s2 = io.StringIO()
    ps2 = pstats.Stats(profiler, stream=s2)
    ps2.sort_stats('time')
    ps2.print_stats(30)
    print(s2.getvalue())
    
    # Save to file
    profile_file = os.path.join(os.path.dirname(__file__), '..', 'logs', 'cpu_profile.stats')
    os.makedirs(os.path.dirname(profile_file), exist_ok=True)
    profiler.dump_stats(profile_file)
    print(f"\nFull profile saved to: {profile_file}")
    print("View with: python3 -m pstats {profile_file}")

if __name__ == "__main__":
    profile_cpu_execution()

