package corelx

import (
	_ "embed"
	"fmt"
	"testing"
)

// Lives in a subdirectory (not directly in internal/corelx/) so
// loadImageAssets's orphan-.cxasset check -- a non-recursive scan of the
// compiled *project's* source directory -- never sees it and mistakes it
// for a stray project asset file.
//
//go:embed bootasset/embedded_boot_logo.cxasset
var embeddedBootLogoCxasset string

// bootLogoSentinelPath is a synthetic FilePath used on the AssetDecl this
// file injects for the embedded boot logo. loadImageAssets special-cases
// it: instead of os.ReadFile, it parses embeddedBootLogoCxasset directly.
const bootLogoSentinelPath = "\x00__embedded_boot_logo__"

// bootShowDefaultFuncName is the merged-in helper's name: the compiler-
// embedded logo slides down from off-screen and holds, then returns
// (caller decides what happens next -- the auto-injected default __Boot()
// falls straight into Start(); a user's own __Boot() calling this via
// boot.show_default() decides for itself, e.g. read input first).
//
// Written as plain CoreLX (not hand-built AST or raw codegen) so it goes
// through the exact same lex/parse/semantic/codegen path as any other
// program -- including the frame_counter() debounce every per-frame CoreLX
// loop needs (see corelx-wait-vblank-multi-iteration-pitfall). ~40 frames
// sliding (~0.7s) + 150 frames held (~2.5s) at 60fps.
const bootShowDefaultFuncName = "__boot_show_default"

const bootShowDefaultSource = `
var __boot_scroll: int = 200
var __boot_hold: int = 0
var __boot_last_frame: int = 0

function __boot_show_default()
    bg.enable(0)
    bg.bind_transform(0, 0)
    bg.set_priority(0, 0)
    matrix.enable(0)
    matrix.identity(0)
    matrix.set_flags(0, false, false, 1, false)
    matrix_plane.enable(0, 32)
    matrix_plane.load_bitmap(__BootLogo, 0)
    -- The logo plane is a 256x256 canvas on a 320x200 screen: scrollX=-32
    -- centers it horizontally (320-256)/2, and the slide settles at
    -- scrollY=30 rather than 0 so the logo centers vertically too (its own
    -- art sits in the upper portion of the 256-tall canvas, so stopping the
    -- slide at 0 left it sitting high with excess black space below it).
    bg.set_scroll(0, 0 - 32, 230)
    ppu.enable_display()

    __boot_scroll = 230
    __boot_hold = 0
    __boot_last_frame = frame_counter()
    while __boot_scroll > 30
        while frame_counter() == __boot_last_frame
            wait_vblank()
        __boot_last_frame = frame_counter()
        __boot_scroll = __boot_scroll - 5
        if __boot_scroll < 30
            __boot_scroll = 30
        bg.set_scroll(0, 0 - 32, __boot_scroll)

    while __boot_hold < 150
        while frame_counter() == __boot_last_frame
            wait_vblank()
        __boot_last_frame = frame_counter()
        __boot_hold = __boot_hold + 1

    bg.disable(0)
    matrix_plane.disable(0)
    matrix.disable(0)
`

// injectBootEntry always makes the embedded default logo + its show
// sequence available (as __BootLogo / __boot_show_default(), for
// boot.show_default() to call from a user's own __Boot()), then decides how
// the real entry point behaves relative to Start(), in one of three
// mutually exclusive ways:
//
//  1. The program already defines its own __Boot() -- leave entry-point
//     selection alone. That source is intentionally taking over the entry
//     point itself (not new plumbing: Generate()/semantic.go already treat
//     a user-defined __Boot() as the entire program instead of Start(),
//     exactly the same as test/roms/pellet_game.corelx already does).
//  2. Running under `go test` and the caller didn't ask to see the real
//     splash (cfg.ForceBootSplash): inject a trivial `__Boot(): Start()` so
//     the hundreds of existing tests that compile small CoreLX snippets and
//     assert state a few frames later don't need touching to skip a ~3.2s
//     hold they never asked to test. Production compiles (the real `corelx`
//     CLI) never run under `go test`, so real games are unaffected.
//  3. Otherwise (production, or a test with ForceBootSplash): inject
//     `__Boot(): __boot_show_default(); Start()`.
func injectBootEntry(program *Program, cfg CompileOptions) error {
	hasBoot := false
	hasStart := false
	for _, fn := range program.Functions {
		if fn.Name == "__Boot" {
			hasBoot = true
		}
		if fn.Name == "Start" {
			hasStart = true
		}
	}
	if !hasStart && !hasBoot {
		return nil
	}

	fastTestBypass := !hasBoot && testing.Testing() && !cfg.ForceBootSplash
	if fastTestBypass {
		// Skip the parse+merge below entirely -- no __Boot() exists yet (so
		// nothing in this program can reach boot.show_default()), and this
		// is the hot path for the hundreds of tests that never asked to see
		// the splash. Keeps `go test` compile time unaffected by this file.
		program.Functions = append(program.Functions, &FunctionDecl{
			Name: "__Boot",
			Body: []Stmt{
				&ExprStmt{Expr: &CallExpr{Func: &IdentExpr{Name: "Start"}}},
			},
		})
		return nil
	}

	lexer := NewLexer(bootShowDefaultSource)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("internal: default boot sequence lex: %w", err)
	}
	bootProgram, err := NewParser(tokens).Parse()
	if err != nil {
		return fmt.Errorf("internal: default boot sequence parse: %w", err)
	}
	program.Functions = append(program.Functions, bootProgram.Functions...)
	program.Globals = append(program.Globals, bootProgram.Globals...)
	program.Assets = append(program.Assets, &AssetDecl{
		Name:     "__BootLogo",
		Type:     "image",
		FilePath: bootLogoSentinelPath,
	})

	if hasBoot {
		// The program's own __Boot() takes over the entry point; it may or
		// may not call boot.show_default() -- either way, nothing more to
		// inject here.
		return nil
	}

	program.Functions = append(program.Functions, &FunctionDecl{
		Name: "__Boot",
		Body: []Stmt{
			&ExprStmt{Expr: &CallExpr{Func: &IdentExpr{Name: bootShowDefaultFuncName}}},
			&ExprStmt{Expr: &CallExpr{Func: &IdentExpr{Name: "Start"}}},
		},
	})
	return nil
}
