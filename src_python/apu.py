"""
apu.py - APU (Audio Processing Unit) - 4-channel synthesizer
Python version - maintains BASIC-like simplicity
"""

import config
import math


def apu_reset():
    """Initialize APU to default state"""
    emu = config.emulator
    
    # Initialize all channels
    for ch in range(config.APU_NUM_CHANNELS):
        channel = config.apu_channels[ch]
        channel.waveform_type = config.WAVEFORM_SINE
        channel.frequency = 0
        channel.volume = 0
        channel.enabled = False
        channel.phase = 0.0
        channel.phase_increment = 0.0
        channel.noise_lfsr = 0x7FFF  # Initialize LFSR to non-zero
        channel.attack_rate = 0
        channel.decay_rate = 0
        channel.sustain_level = 255
        channel.release_rate = 0
    
    # Channel 3 defaults to noise
    config.apu_channels[3].waveform_type = config.WAVEFORM_NOISE
    
    # Clear sample buffer
    config.apu_sample_buffer[:] = [0] * len(config.apu_sample_buffer)
    
    emu.apu.buffer_write_pos = 0
    emu.apu.buffer_read_pos = 0
    emu.apu.master_volume = 255


def apu_read_reg(reg_addr):
    """Read APU register"""
    # TODO: Implement register reads
    # Most APU registers are write-only, but some status could be readable
    if config.APU_REG_CH0_CONTROL <= reg_addr <= config.APU_REG_CH3_CONTROL:
        # Return channel enable status
        ch = (reg_addr - config.APU_REG_CH0_CONTROL) // 4
        if ch < config.APU_NUM_CHANNELS:
            return 1 if config.apu_channels[ch].enabled else 0
        else:
            return 0
    else:
        return 0


def apu_write_reg(reg_addr, value):
    """Write APU register"""
    # Determine which channel (0-3)
    if config.APU_REG_CH0_FREQ_LOW <= reg_addr <= config.APU_REG_CH3_CONTROL:
        ch = (reg_addr - config.APU_REG_CH0_FREQ_LOW) // 4
        if ch >= config.APU_NUM_CHANNELS:
            return
        
        channel = config.apu_channels[ch]
        reg_field = (reg_addr - config.APU_REG_CH0_FREQ_LOW) % 4
        
        if reg_field == 0:
            # Frequency low byte
            freq = channel.frequency
            freq = (freq & 0xFF00) | value
            channel.frequency = freq
            apu_update_channel_frequency(ch)
        
        elif reg_field == 1:
            # Frequency high byte
            freq = channel.frequency
            freq = (freq & 0xFF) | (value << 8)
            channel.frequency = freq
            apu_update_channel_frequency(ch)
        
        elif reg_field == 2:
            # Volume
            channel.volume = value
        
        elif reg_field == 3:
            # Control
            channel.enabled = (value & 0x01) != 0
            if ch == 3:
                # Channel 3: bit 1 selects noise vs square
                if (value & 0x02) != 0:
                    channel.waveform_type = config.WAVEFORM_NOISE
                else:
                    channel.waveform_type = config.WAVEFORM_SQUARE
            else:
                # Other channels: waveform type from bits 1-2
                channel.waveform_type = (value >> 1) & 0x03
    
    elif reg_addr == config.APU_REG_MASTER_VOLUME:
        emu = config.emulator
        emu.apu.master_volume = value


def apu_update_channel_frequency(ch):
    """Update channel frequency and phase increment (internal helper)"""
    channel = config.apu_channels[ch]
    
    # Convert frequency value to Hz
    # TODO: Define frequency encoding (e.g., fixed-point or direct Hz value)
    # For now, assume frequency register is in Hz (will need scaling)
    freq_hz = channel.frequency
    
    # Calculate phase increment per sample
    channel.phase_increment = (freq_hz * 2.0 * math.pi) / config.APU_SAMPLE_RATE


def apu_generate_sample(ch):
    """Generate sample from oscillator (internal helper)"""
    channel = config.apu_channels[ch]
    
    if not channel.enabled:
        return 0.0
    
    volume = channel.volume / 255.0
    phase = channel.phase
    
    if channel.waveform_type == config.WAVEFORM_SINE:
        sample = math.sin(phase) * volume
    
    elif channel.waveform_type == config.WAVEFORM_SQUARE:
        if phase < math.pi:
            sample = volume
        else:
            sample = -volume
    
    elif channel.waveform_type == config.WAVEFORM_SAW:
        # Sawtooth: linear ramp from -1 to 1
        sample = ((phase / (2.0 * math.pi)) * 2.0 - 1.0) * volume
    
    elif channel.waveform_type == config.WAVEFORM_NOISE:
        # LFSR-based noise
        sample = apu_generate_noise(ch) * volume
    
    else:
        sample = 0.0
    
    # Update phase (wrap at 2*PI)
    channel.phase = phase + channel.phase_increment
    if channel.phase >= 2.0 * math.pi:
        channel.phase = channel.phase - 2.0 * math.pi
    
    return sample


def apu_generate_noise(ch):
    """Generate noise sample using LFSR (internal helper)"""
    channel = config.apu_channels[ch]
    lfsr = channel.noise_lfsr
    
    # 15-bit LFSR: tap bits 0 and 1, shift right
    bit = (lfsr ^ (lfsr >> 1)) & 1
    lfsr = (lfsr >> 1) | (bit << 14)
    
    # If LFSR becomes 0, reset it
    if lfsr == 0:
        lfsr = 0x7FFF
    
    channel.noise_lfsr = lfsr
    
    # Convert to signed sample (-1 to 1)
    if (lfsr & 1) != 0:
        return 1.0
    else:
        return -1.0


def apu_generate_audio_buffer(num_samples):
    """Mix all channels and fill audio buffer"""
    emu = config.emulator
    master_vol = emu.apu.master_volume / 255.0
    
    # Generate samples
    samples = []
    for i in range(num_samples):
        mixed = 0.0
        
        # Mix all channels
        for ch in range(config.APU_NUM_CHANNELS):
            sample = apu_generate_sample(ch)
            mixed = mixed + sample
        
        # Apply master volume and clamp
        mixed = mixed * master_vol
        if mixed > 1.0:
            mixed = 1.0
        if mixed < -1.0:
            mixed = -1.0
        
        # Convert to 16-bit integer (signed)
        sample_value = int(mixed * 32767)
        samples.append(sample_value)
    
    # Store in buffer for debugging
    for sample_value in samples:
        if emu.apu.buffer_write_pos < config.APU_BUFFER_SIZE:
            config.apu_sample_buffer[emu.apu.buffer_write_pos] = sample_value
            emu.apu.buffer_write_pos = (emu.apu.buffer_write_pos + 1) % config.APU_BUFFER_SIZE
    
    # Return samples as bytes for pygame
    import struct
    sample_bytes = b''.join(struct.pack('<h', s) for s in samples)
    return sample_bytes


# Global audio stream (set by main.py)
_audio_stream = None

def apu_set_audio_stream(stream):
    """Set the pygame audio stream for output"""
    global _audio_stream
    _audio_stream = stream

def apu_update():
    """Update APU (called once per frame)"""
    # Generate audio for one frame's worth of samples
    # At 60 FPS and 44100 Hz, that's 44100/60 = 735 samples per frame
    sample_bytes = apu_generate_audio_buffer(735)
    
    # Output to pygame audio stream if available
    if _audio_stream is not None:
        try:
            _audio_stream.queue(sample_bytes)
        except:
            pass  # Ignore errors (buffer full, etc.)

