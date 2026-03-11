package ymstream

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	Magic = "NCDXMUS1"

	opEnd        = 0x00
	opWait8      = 0x10
	opWait16     = 0x11
	opWrite0     = 0x20
	opWrite1     = 0x21
	opBurst0     = 0x30
	opBurst1     = 0x31
)

type Write struct {
	Port uint8
	Addr uint8
	Data uint8
}

type Song struct {
	Frames       [][]Write
	FrameSamples uint32
	TotalSamples uint64
	WriteCount   int
}

type Stream struct {
	FrameSamples uint32
	Data         []byte
}

func ReadVGM(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if strings.EqualFold(filepath.Ext(path), ".vgz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		return io.ReadAll(gz)
	}
	return io.ReadAll(f)
}

func ParseVGM(data []byte) (*Song, error) {
	if len(data) < 0x100 || string(data[0:4]) != "Vgm " {
		return nil, errors.New("input is not a valid VGM stream")
	}

	version := binary.LittleEndian.Uint32(data[0x08:0x0C])
	rate := binary.LittleEndian.Uint32(data[0x24:0x28])
	dataOff := uint32(0x40)
	if version >= 0x150 {
		rel := binary.LittleEndian.Uint32(data[0x34:0x38])
		if rel != 0 {
			dataOff = 0x34 + rel
		}
	}

	frameSamples := uint32(735)
	if rate == 50 {
		frameSamples = 882
	}

	frames := make([][]Write, 0, 5000)
	current := make([]Write, 0, 64)
	var sampleAcc uint64
	var totalSamples uint64
	var writeCount int

	flushFrame := func() {
		cloned := make([]Write, len(current))
		copy(cloned, current)
		frames = append(frames, cloned)
		current = current[:0]
	}

	addWait := func(n uint64) {
		sampleAcc += n
		totalSamples += n
		for sampleAcc >= uint64(frameSamples) {
			flushFrame()
			sampleAcc -= uint64(frameSamples)
		}
	}

	for p := int(dataOff); p < len(data); {
		cmd := data[p]
		p++

		switch {
		case cmd == 0x66:
			if len(current) > 0 {
				flushFrame()
			}
			return &Song{
				Frames:       frames,
				FrameSamples: frameSamples,
				TotalSamples: totalSamples,
				WriteCount:   writeCount,
			}, nil

		case cmd == 0x56 || cmd == 0x57:
			if p+2 > len(data) {
				return nil, fmt.Errorf("truncated YM2608 write at 0x%X", p-1)
			}
			current = append(current, Write{Port: cmd - 0x56, Addr: data[p], Data: data[p+1]})
			writeCount++
			p += 2

		case cmd == 0x61:
			if p+2 > len(data) {
				return nil, fmt.Errorf("truncated wait(0x61) at 0x%X", p-1)
			}
			n := binary.LittleEndian.Uint16(data[p : p+2])
			addWait(uint64(n))
			p += 2

		case cmd == 0x62:
			addWait(735)

		case cmd == 0x63:
			addWait(882)

		case cmd >= 0x70 && cmd <= 0x7F:
			addWait(uint64((cmd & 0x0F) + 1))

		case cmd == 0x67:
			if p+7 > len(data) {
				return nil, fmt.Errorf("truncated data block at 0x%X", p-1)
			}
			if data[p] != 0x66 {
				return nil, fmt.Errorf("invalid data block marker at 0x%X", p)
			}
			p += 2
			sz := binary.LittleEndian.Uint32(data[p : p+4])
			p += 4
			if p+int(sz) > len(data) {
				return nil, fmt.Errorf("truncated data block payload at 0x%X", p)
			}
			p += int(sz)

		default:
			return nil, fmt.Errorf("unsupported VGM command 0x%02X at 0x%X", cmd, p-1)
		}
	}

	return nil, errors.New("VGM stream ended without 0x66 end marker")
}

func EncodeSong(song *Song) ([]byte, error) {
	if song == nil {
		return nil, errors.New("song is nil")
	}
	if song.FrameSamples == 0 {
		return nil, errors.New("frame sample rate must be non-zero")
	}

	out := make([]byte, 0, 16+song.WriteCount*3)
	out = append(out, []byte(Magic)...)
	frameBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(frameBuf, song.FrameSamples)
	out = append(out, frameBuf...)
	out = append(out, 0, 0, 0, 0)

	pendingWait := 0
	for _, frame := range song.Frames {
		if len(frame) > 0 && pendingWait > 0 {
			out = appendWait(out, pendingWait)
			pendingWait = 0
		}
		out = appendFrameWrites(out, frame)
		pendingWait++
	}
	if pendingWait > 0 {
		out = appendWait(out, pendingWait)
	}
	out = append(out, opEnd)
	return out, nil
}

func DecodeStream(data []byte) (*Song, error) {
	if len(data) < 16 {
		return nil, errors.New("stream too short")
	}
	if string(data[:8]) != Magic {
		return nil, errors.New("invalid music stream magic")
	}
	frameSamples := binary.LittleEndian.Uint32(data[8:12])
	if frameSamples == 0 {
		return nil, errors.New("invalid frame sample rate")
	}

	frames := make([][]Write, 0, 4096)
	current := make([]Write, 0, 64)
	var totalSamples uint64
	var writeCount int

	flushWait := func(n int) {
		cloned := make([]Write, len(current))
		copy(cloned, current)
		frames = append(frames, cloned)
		for i := 1; i < n; i++ {
			frames = append(frames, nil)
		}
		current = current[:0]
		totalSamples += uint64(n) * uint64(frameSamples)
	}

	for p := 16; p < len(data); {
		op := data[p]
		p++
		switch op {
		case opEnd:
			if len(current) > 0 {
				cloned := make([]Write, len(current))
				copy(cloned, current)
				frames = append(frames, cloned)
				totalSamples += uint64(frameSamples)
			}
			return &Song{
				Frames:       frames,
				FrameSamples: frameSamples,
				TotalSamples: totalSamples,
				WriteCount:   writeCount,
			}, nil
		case opWait8:
			if p >= len(data) {
				return nil, errors.New("truncated wait8")
			}
			n := int(data[p])
			p++
			if n == 0 {
				return nil, errors.New("wait8 of zero is invalid")
			}
			flushWait(n)
		case opWait16:
			if p+2 > len(data) {
				return nil, errors.New("truncated wait16")
			}
			n := int(binary.LittleEndian.Uint16(data[p : p+2]))
			p += 2
			if n == 0 {
				return nil, errors.New("wait16 of zero is invalid")
			}
			flushWait(n)
		case opWrite0, opWrite1:
			if p+2 > len(data) {
				return nil, errors.New("truncated write")
			}
			current = append(current, Write{
				Port: op - opWrite0,
				Addr: data[p],
				Data: data[p+1],
			})
			writeCount++
			p += 2
		case opBurst0, opBurst1:
			if p+2 > len(data) {
				return nil, errors.New("truncated burst header")
			}
			addr := data[p]
			count := int(data[p+1])
			p += 2
			if count <= 0 || p+count > len(data) {
				return nil, errors.New("invalid burst payload")
			}
			for i := 0; i < count; i++ {
				current = append(current, Write{
					Port: op - opBurst0,
					Addr: addr + uint8(i),
					Data: data[p+i],
				})
				writeCount++
			}
			p += count
		default:
			return nil, fmt.Errorf("unknown stream opcode 0x%02X", op)
		}
	}

	return nil, errors.New("stream ended without end opcode")
}

func appendFrameWrites(out []byte, writes []Write) []byte {
	for i := 0; i < len(writes); {
		if count := burstLength(writes, i); count >= 3 {
			if writes[i].Port == 0 {
				out = append(out, opBurst0)
			} else {
				out = append(out, opBurst1)
			}
			out = append(out, writes[i].Addr, uint8(count))
			for j := 0; j < count; j++ {
				out = append(out, writes[i+j].Data)
			}
			i += count
			continue
		}
		if writes[i].Port == 0 {
			out = append(out, opWrite0, writes[i].Addr, writes[i].Data)
		} else {
			out = append(out, opWrite1, writes[i].Addr, writes[i].Data)
		}
		i++
	}
	return out
}

func burstLength(writes []Write, start int) int {
	if start >= len(writes) {
		return 0
	}
	port := writes[start].Port
	addr := writes[start].Addr
	n := 1
	for start+n < len(writes) && n < 255 {
		next := writes[start+n]
		if next.Port != port || next.Addr != addr+uint8(n) {
			break
		}
		n++
	}
	return n
}

func appendWait(out []byte, n int) []byte {
	for n > 0 {
		if n <= 0xFF {
			out = append(out, opWait8, uint8(n))
			return out
		}
		chunk := n
		if chunk > 0xFFFF {
			chunk = 0xFFFF
		}
		out = append(out, opWait16, uint8(chunk&0xFF), uint8((chunk>>8)&0xFF))
		n -= chunk
	}
	return out
}
