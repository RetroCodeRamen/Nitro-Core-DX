// manual_frames writes the pictures used by the manuals. It compiles the
// teaching programs and the overworld, runs them on the emulator core, and
// saves the framebuffer. This is not a player-facing tool.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/emulator"
)

func main() {
	out := "docs/manual/images"
	if err := os.MkdirAll(out, 0755); err != nil {
		fatal(err)
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	hello := compile(filepath.Join(root, "docs/manual_examples/hello.corelx"))
	run(hello, 90)
	save(hello, filepath.Join(out, "boot.png"))
	run(hello, 130)
	save(hello, filepath.Join(out, "hello.png"))

	counter := compile(filepath.Join(root, "docs/manual_examples/counter.corelx"))
	run(counter, 210)
	counter.emu.SetInputButtons(0)
	run(counter, 2)
	counter.emu.SetInputButtons(0x0010) // A
	run(counter, 3)
	counter.emu.SetInputButtons(0)
	run(counter, 5)
	save(counter, filepath.Join(out, "counter.png"))

	sprite := compile(filepath.Join(root, "docs/manual_examples/sprite.corelx"))
	run(sprite, 220)
	save(sprite, filepath.Join(out, "sprite.png"))

	floor := compile(filepath.Join(root, "docs/manual_examples/floor.corelx"))
	run(floor, 230)
	save(floor, filepath.Join(out, "floor.png"))

	over := compile(filepath.Join(root, "Games/NitroPackInDemo/corelx/overworld.corelx"))
	waitGlobal(over, "last_frame")
	run(over, 8)
	save(over, filepath.Join(out, "overworld.png"))
}

type loaded struct {
	emu    *emulator.Emulator
	result *corelx.CompileResult
}

func compile(path string) loaded {
	cart := filepath.Join(os.TempDir(), filepath.Base(path)+".cart")
	result, err := corelx.CompileProject(path, &corelx.CompileOptions{OutputPath: cart})
	if err != nil {
		fatal(fmt.Errorf("%s: %w", path, err))
	}
	data, err := os.ReadFile(cart)
	if err != nil {
		fatal(err)
	}
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(data); err != nil {
		fatal(err)
	}
	emu.SetFrameLimit(false)
	emu.Start()
	return loaded{emu: emu, result: result}
}

func run(l loaded, n int) {
	for i := 0; i < n; i++ {
		if err := l.emu.RunFrame(); err != nil {
			fatal(err)
		}
	}
}

func waitGlobal(l loaded, name string) {
	addr := uint16(0)
	found := false
	for _, e := range l.result.MemoryMap {
		if e.Name == name {
			addr = e.Address
			found = true
			break
		}
	}
	if !found {
		fatal(fmt.Errorf("global %s not in memory map", name))
	}
	for i := 0; i < 4000; i++ {
		if read16(l.emu, addr) != 0 {
			return
		}
		if err := l.emu.RunFrame(); err != nil {
			fatal(err)
		}
	}
	fatal(fmt.Errorf("timed out waiting for %s", name))
}

func read16(emu *emulator.Emulator, addr uint16) uint16 {
	lo := emu.CPU.Mem.Read8(0, addr)
	hi := emu.CPU.Mem.Read8(0, addr+1)
	return uint16(lo) | uint16(hi)<<8
}

func save(l loaded, path string) {
	fb := l.emu.GetOutputBuffer()
	const scale = 3
	img := image.NewRGBA(image.Rect(0, 0, 320*scale, 200*scale))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			c := fb[y*320+x]
			px := color.RGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 255}
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					img.Set(x*scale+sx, y*scale+sy, px)
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
