# ROM Compatibility Fix

## Issue Found

The sprite blending implementation had a backward compatibility issue that could break existing ROMs.

### Problem

ROMs like `moving_sprite_colored.rom` use control byte `0x03`:
- Bit 0 = 1 (enabled)
- Bit 1 = 1 (16x16)
- Bits [3:2] = 0 (blend mode 0 = normal)
- Bits [7:4] = 0 (alpha = 0)

The original code checked:
```go
if blendMode == 0 && alpha == 15 {
    // Normal mode, fully opaque
    p.OutputBuffer[y*320+x] = spriteColor
} else {
    // Apply blend (would use alpha=0, making sprite transparent!)
    blendedColor := p.blendColor(spriteColor, backgroundColor, blendMode, alpha)
}
```

This would cause sprites with `alpha=0` to be rendered as transparent, even in normal mode.

### Fix

Changed the logic to:
```go
if blendMode == 0 {
    // Normal mode (opaque) - ignore alpha, just write sprite color
    // This maintains backward compatibility with ROMs that use control byte 0x03
    p.OutputBuffer[y*320+x] = spriteColor
} else {
    // Blending modes (alpha, additive, subtractive) - need background color
    backgroundColor := p.OutputBuffer[y*320+x]
    blendedColor := p.blendColor(spriteColor, backgroundColor, blendMode, alpha)
    p.OutputBuffer[y*320+x] = blendedColor
}
```

### Result

- ✅ Normal mode (blendMode=0) now ignores alpha value
- ✅ Maintains backward compatibility with existing ROMs
- ✅ Blending modes (1-3) still work correctly with alpha
- ✅ All tests pass

## Verification

The ROM `moving_sprite_colored.rom` should now work correctly with the updated sprite rendering system.
