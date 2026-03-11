package emulator

import (
	"image"
	"image/color"
	"testing"

	ppucore "nitro-core-dx/internal/ppu"
)

func TestBuildBitmapMatrixPlaneAssetFromImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x < 4 {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{G: 255, A: 255})
			}
		}
	}

	asset, err := BuildBitmapMatrixPlaneAssetFromImage(img, 0, ppucore.TilemapSize32x32, 2)
	if err != nil {
		t.Fatalf("BuildBitmapMatrixPlaneAssetFromImage: %v", err)
	}
	if asset.Program.SourceMode != ppucore.MatrixPlaneSourceBitmap {
		t.Fatalf("source mode = %d, want bitmap", asset.Program.SourceMode)
	}
	if asset.Program.BitmapPalette != 2 {
		t.Fatalf("bitmap palette = %d, want 2", asset.Program.BitmapPalette)
	}
	if len(asset.Program.Bitmap) == 0 {
		t.Fatal("bitmap payload is empty")
	}
	if len(asset.Palette) != 16 {
		t.Fatalf("palette length = %d, want 16", len(asset.Palette))
	}
	hasNonZeroPackedByte := false
	for _, b := range asset.Program.Bitmap {
		if b != 0x00 {
			hasNonZeroPackedByte = true
			break
		}
	}
	if !hasNonZeroPackedByte {
		t.Fatal("bitmap payload is all zero bytes, want non-empty packed pixel data")
	}
	hasRedish := false
	hasGreenish := false
	for _, c := range asset.Palette {
		r, g, b := rgb555Components(c)
		if r > g && r > b {
			hasRedish = true
		}
		if g > r && g > b {
			hasGreenish = true
		}
	}
	if !hasRedish {
		t.Fatal("palette did not preserve a red-dominant entry")
	}
	if !hasGreenish {
		t.Fatal("palette did not preserve a green-dominant entry")
	}
}

func TestBuildBitmapMatrixPlaneAssetFromImagePreservesTransparencyAsIndexZero(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x < 4 {
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{A: 0})
			}
		}
	}

	asset, err := BuildBitmapMatrixPlaneAssetFromImage(img, 0, ppucore.TilemapSize32x32, 3)
	if err != nil {
		t.Fatalf("BuildBitmapMatrixPlaneAssetFromImage: %v", err)
	}
	if asset.Palette[0] != 0 {
		t.Fatalf("palette[0] = 0x%04X, want transparent zero entry", asset.Palette[0])
	}
	firstRowLastByte := asset.Program.Bitmap[(256/2)-1]
	if firstRowLastByte != 0x00 {
		t.Fatalf("first row trailing byte = 0x%02X, want transparent packed byte 0x00", firstRowLastByte)
	}
}

func rgb555Components(v uint16) (uint8, uint8, uint8) {
	r := uint8((uint32(v&0x1F) * 255) / 31)
	g := uint8((uint32((v>>5)&0x1F) * 255) / 31)
	b := uint8((uint32((v>>10)&0x1F) * 255) / 31)
	return r, g, b
}
