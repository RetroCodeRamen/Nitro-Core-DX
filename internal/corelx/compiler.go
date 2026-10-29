package corelx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"nitro-core-dx/internal/rom"
)

type CompileOptions struct {
	OutputPath            string
	ManifestOutputPath    string
	DiagnosticsOutputPath string
	BundleOutputPath      string
	AssetManifestPath     string
	// ModulesPath is the directory searched for `--! modules: name, ...`
	// requests (each resolved to <ModulesPath>/<name>.corelx). Empty means
	// the default: a "modules" directory next to the main source file.
	ModulesPath         string
	EntryBank           uint8
	EntryOffset         uint16
	MaxROMBytes         uint32
	SectionBudgets      map[string]uint32
	EmitROMBytes        bool
	EmitManifestJSON    bool
	EmitDiagnosticsJSON bool
	EmitBundleJSON      bool
	// NoROMBytes, NoManifestJSON, NoDiagnosticsJSON, and NoBundleJSON turn
	// those outputs off. A zero Emit* field does not: a partial options
	// struct such as &CompileOptions{OutputPath: path} keeps the defaults.
	NoROMBytes        bool
	NoManifestJSON    bool
	NoDiagnosticsJSON bool
	NoBundleJSON      bool
	// ForceBootSplash disables the under-`go test` __Boot() auto-bypass (see
	// injectTestBootBypass) so a test can observe the real default boot
	// sequence. Has no effect outside `go test` -- production compiles
	// always see the real behavior regardless of this field.
	ForceBootSplash bool
}

type CompileResult struct {
	Program          *Program
	NormalizedAssets []AssetIR
	AssetSourceFiles []string
	ROMBytes         []byte
	Manifest         *BuildManifest
	ManifestJSON     []byte
	DiagnosticsJSON  []byte
	BundleJSON       []byte
	MemoryMap        []MemoryMapEntry
	MemoryMapText    []byte
	Diagnostics      []Diagnostic
}

func defaultCompileOptions() CompileOptions {
	return CompileOptions{
		EntryBank:           1,
		EntryOffset:         0x8000,
		EmitROMBytes:        true,
		EmitManifestJSON:    true,
		EmitDiagnosticsJSON: true,
		EmitBundleJSON:      true,
	}
}

// CompileProject is the production compiler entrypoint scaffold.
// Current implementation compiles a single CoreLX source file and returns structured diagnostics.
func CompileProject(sourcePath string, opts *CompileOptions) (*CompileResult, error) {
	// Resolve a .ncdx container or project directory to its main source file.
	mainPath, cleanup, openErr := openProject(sourcePath)
	if openErr != nil {
		diag := Diagnostic{
			Category: CategoryIOError,
			Code:     "E_IO_OPEN_PROJECT",
			Message:  openErr.Error(),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StageIO,
		}
		return &CompileResult{Diagnostics: []Diagnostic{diag}}, &DiagnosticsError{Diagnostics: []Diagnostic{diag}}
	}
	defer cleanup()
	sourcePath = mainPath

	source, err := os.ReadFile(sourcePath)
	if err != nil {
		diag := Diagnostic{
			Category: CategoryIOError,
			Code:     "E_IO_READ_SOURCE",
			Message:  err.Error(),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StageIO,
		}
		if diag.EndLine == 0 {
			diag.EndLine = diag.Line
		}
		if diag.EndColumn == 0 {
			diag.EndColumn = diag.Column
		}
		return &CompileResult{Diagnostics: []Diagnostic{diag}}, &DiagnosticsError{Diagnostics: []Diagnostic{diag}}
	}
	return CompileSource(string(source), sourcePath, opts)
}

// CompileSource compiles CoreLX source text with optional sourcePath metadata.
func CompileSource(source, sourcePath string, opts *CompileOptions) (result *CompileResult, err error) {
	cfg := defaultCompileOptions()
	if opts != nil {
		mergeCompileOptions(&cfg, *opts)
	}

	currentStage := StageIO
	defer func() {
		if r := recover(); r != nil {
			if result == nil {
				result = &CompileResult{}
			}
			diag := internalCompilerDiagnostic(currentStage, sourcePath, r)
			result.Diagnostics = append(result.Diagnostics, diag)
			err = &DiagnosticsError{Diagnostics: result.Diagnostics}
		}

		if result == nil {
			return
		}
		normalizeDiagnosticRanges(result.Diagnostics)
		if cfg.EmitDiagnosticsJSON || cfg.DiagnosticsOutputPath != "" {
			if b, mErr := json.MarshalIndent(result.Diagnostics, "", "  "); mErr == nil {
				result.DiagnosticsJSON = b
				if cfg.DiagnosticsOutputPath != "" {
					if wErr := os.WriteFile(cfg.DiagnosticsOutputPath, b, 0644); wErr != nil {
						result.Diagnostics = append(result.Diagnostics, Diagnostic{
							Category: CategoryIOError,
							Code:     "E_IO_WRITE_DIAGNOSTICS",
							Message:  wErr.Error(),
							File:     cfg.DiagnosticsOutputPath,
							Severity: SeverityError,
							Stage:    StageIO,
						})
						// refresh DiagnosticsJSON with the appended diagnostic if possible
						if b2, mErr2 := json.MarshalIndent(result.Diagnostics, "", "  "); mErr2 == nil {
							result.DiagnosticsJSON = b2
						}
						if err == nil {
							err = &DiagnosticsError{Diagnostics: result.Diagnostics}
						}
					}
				}
			}
		}
		if cfg.EmitBundleJSON || cfg.BundleOutputPath != "" {
			bundle := BuildCompileBundle(result)
			if b, mErr := json.MarshalIndent(bundle, "", "  "); mErr == nil {
				result.BundleJSON = b
				if cfg.BundleOutputPath != "" {
					if wErr := os.WriteFile(cfg.BundleOutputPath, b, 0644); wErr != nil {
						result.Diagnostics = append(result.Diagnostics, Diagnostic{
							Category: CategoryIOError,
							Code:     "E_IO_WRITE_BUNDLE",
							Message:  wErr.Error(),
							File:     cfg.BundleOutputPath,
							Severity: SeverityError,
							Stage:    StageIO,
						})
						// Refresh derived JSON outputs after appending the write error.
						if b2, err2 := json.MarshalIndent(result.Diagnostics, "", "  "); err2 == nil {
							result.DiagnosticsJSON = b2
						}
						if b3, err3 := json.MarshalIndent(BuildCompileBundle(result), "", "  "); err3 == nil {
							result.BundleJSON = b3
						}
						if err == nil {
							err = &DiagnosticsError{Diagnostics: result.Diagnostics}
						}
					}
				}
			}
		}
	}()

	result = &CompileResult{
		Diagnostics: make([]Diagnostic, 0),
	}

	lexer := NewLexer(source)
	currentStage = StageLexer
	tokens, err := lexer.Tokenize()
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Category: CategoryLexError,
			Code:     "E_LEX_TOKENIZE",
			Message:  err.Error(),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StageLexer,
		})
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	for _, tok := range tokens {
		if tok.Type == TOKEN_ERROR {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Category: CategoryLexError,
				Code:     "E_LEX_TOKEN",
				Message:  tok.Literal,
				File:     sourcePath,
				Line:     tok.Line,
				Column:   tok.Column,
				Severity: SeverityError,
				Stage:    StageLexer,
			})
		}
	}
	if HasErrors(result.Diagnostics) {
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	parser := NewParser(tokens)
	currentStage = StageParser
	program, err := parser.Parse()
	if err != nil {
		var pe *ParseErrors
		if errors.As(err, &pe) {
			for _, e := range pe.Errs {
				result.Diagnostics = append(result.Diagnostics, parseDiagnostic(e, sourcePath))
			}
		} else {
			result.Diagnostics = append(result.Diagnostics, parseDiagnostic(err, sourcePath))
		}
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}
	result.Program = program
	if bootErr := injectBootEntry(program, cfg); bootErr != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Category: CategoryBackendCodegenError,
			Code:     "E_BOOT_SEQUENCE",
			Message:  bootErr.Error(),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StageParser,
		})
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	currentStage = StageParser
	moduleDiags := loadModules(program, sourcePath, cfg)
	result.Diagnostics = append(result.Diagnostics, moduleDiags...)
	if HasErrors(result.Diagnostics) {
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	currentStage = StageAsset
	externalAssets, externalSources, externalDiags := loadProjectAssets(sourcePath, cfg)
	result.Diagnostics = append(result.Diagnostics, externalDiags...)
	if HasErrors(result.Diagnostics) {
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}
	if len(externalAssets) > 0 {
		program.Assets = append(program.Assets, externalAssets...)
		result.AssetSourceFiles = append(result.AssetSourceFiles, externalSources...)
	}

	currentStage = StageSemantic
	semDiags := AnalyzeWithDiagnostics(program)
	stampDiagnosticsFile(semDiags, sourcePath)
	result.Diagnostics = append(result.Diagnostics, semDiags...)
	if HasErrors(result.Diagnostics) {
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	currentStage = StageAsset
	assets, assetDiags := NormalizeAssets(program, sourcePath)
	result.NormalizedAssets = assets
	result.Diagnostics = append(result.Diagnostics, assetDiags...)
	if HasErrors(result.Diagnostics) {
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	// Load external image (.cxasset) bitmaps and music (.ncdxmusic) YM2608
	// streams, laid out in the ROM data region starting at a provisional
	// bank (2 -- the single-bank default). This only affects baked-in DMA
	// bank/offset immediates, not instruction count, so it's a safe guess
	// for pass 1; if pass 1's code doesn't fit in one bank, the multi-bank
	// path below reloads both at their real starting bank once the code
	// bank count is known.
	const singleBankDataStart = 2
	imageAssets, imageRegion, imgErr := loadImageAssets(program, sourcePath, singleBankDataStart)
	if imgErr != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Category: CategoryAssetParseError,
			Code:     "E_IMAGE_ASSET",
			Message:  imgErr.Error(),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StageAsset,
		})
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}
	musicAssets, musicRegion, musErr := loadMusicAssets(program, sourcePath, singleBankDataStart, len(imageRegion))
	if musErr != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Category: CategoryAssetParseError,
			Code:     "E_MUSIC_ASSET",
			Message:  musErr.Error(),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StageAsset,
		})
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	// Pass 1: compact, single-bank compile -- today's exact behavior byte
	// for byte when it fits. Real codegen errors (unrelated to ROM size)
	// are surfaced here directly, same as always; only a ROM-size overflow
	// (code alone, or code + data together) triggers the multi-bank
	// fallback below.
	pass1Builder := rom.NewROMBuilder()
	generator := NewCodeGenerator(program, pass1Builder)
	generator.SetNormalizedAssets(assets)
	generator.SetImageAssets(imageAssets)
	generator.SetMusicAssets(musicAssets)
	currentStage = StageCodegen
	genErr := generator.Generate()
	needsMultiBank := errors.Is(genErr, errCodeOverflowsBank)
	if genErr != nil && !needsMultiBank {
		result.Diagnostics = append(result.Diagnostics, diagnosticFromError(genErr, sourcePath, "E_CODEGEN_GENERATE", CategoryBackendCodegenError, StageCodegen))
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}

	// Image and music bytes share one contiguous ROM data region (images first,
	// then music — matching the bank/offset cursor used during placement).
	dataRegion := append(append([]byte{}, imageRegion...), musicRegion...)

	if !needsMultiBank {
		needsMultiBank = pass1Builder.GetCodeLength()*2 > int(rom.ROMBankSizeBytes)
	}
	var (
		romBytes  []byte
		codeBytes uint32
	)
	if !needsMultiBank {
		if len(dataRegion) > 0 {
			pass1Builder.SetDataRegion(singleBankDataStart, dataRegion)
		}
		currentStage = StagePack
		rb, buildErr := pass1Builder.BuildROMBytes(cfg.EntryBank, cfg.EntryOffset)
		if buildErr == nil {
			romBytes = rb
			codeBytes = uint32(pass1Builder.GetCodeLength() * 2)
		} else {
			// Code alone fit, but code+data together didn't -- same
			// remedy as a pure code overflow: fall back to multi-bank.
			needsMultiBank = true
		}
	}

	if needsMultiBank {
		mgen, mRomBytes, mCodeBytes, mErr := compileMultiBank(program, sourcePath, assets, cfg)
		if mErr != nil {
			result.Diagnostics = append(result.Diagnostics, diagnosticFromError(mErr, sourcePath, "E_PACK_MULTIBANK", CategoryLayoutError, StagePack))
			return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
		}
		generator = mgen
		romBytes = mRomBytes
		codeBytes = mCodeBytes
	}

	if cfg.EmitROMBytes {
		result.ROMBytes = romBytes
	}
	result.MemoryMap = generator.MemoryMap()
	result.MemoryMapText = formatMemoryMap(result.MemoryMap)
	if cfg.OutputPath != "" && len(result.MemoryMap) > 1 {
		// Listing emitted alongside the ROM for debugger/symbol use
		// (charter memory model: tooling-visible allocation).
		mapPath := cfg.OutputPath + ".memmap"
		if wErr := os.WriteFile(mapPath, result.MemoryMapText, 0644); wErr != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Category: CategoryIOError,
				Code:     "E_IO_WRITE_MEMMAP",
				Message:  wErr.Error(),
				File:     mapPath,
				Severity: SeverityWarning,
				Stage:    StageIO,
			})
		}
	}
	result.Manifest = buildManifestFromCompileState(sourcePath, cfg.EntryBank, cfg.EntryOffset, codeBytes, uint32(len(romBytes)), program, assets)
	if result.Manifest != nil && len(result.AssetSourceFiles) > 0 {
		result.Manifest.SourceFiles = uniqueStrings(append(result.Manifest.SourceFiles, result.AssetSourceFiles...))
	}
	currentStage = StagePack
	packDiags := validatePackBudgets(result.Manifest, cfg, sourcePath)
	result.Diagnostics = append(result.Diagnostics, packDiags...)
	if HasErrors(result.Diagnostics) {
		return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
	}
	if result.Manifest != nil && (cfg.EmitManifestJSON || cfg.ManifestOutputPath != "") {
		manifestJSON, mErr := json.MarshalIndent(result.Manifest, "", "  ")
		if mErr != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Category: CategoryInternalCompilerError,
				Code:     "E_MANIFEST_MARSHAL",
				Message:  mErr.Error(),
				File:     sourcePath,
				Severity: SeverityError,
				Stage:    StagePack,
			})
			return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
		}
		result.ManifestJSON = manifestJSON
		if cfg.ManifestOutputPath != "" {
			currentStage = StageIO
			if wErr := os.WriteFile(cfg.ManifestOutputPath, manifestJSON, 0644); wErr != nil {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Category: CategoryIOError,
					Code:     "E_IO_WRITE_MANIFEST",
					Message:  wErr.Error(),
					File:     cfg.ManifestOutputPath,
					Severity: SeverityError,
					Stage:    StageIO,
				})
				return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
			}
		}
	}

	if cfg.OutputPath != "" {
		currentStage = StageIO
		if err := os.WriteFile(cfg.OutputPath, romBytes, 0644); err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Category: CategoryIOError,
				Code:     "E_IO_WRITE_ROM",
				Message:  err.Error(),
				File:     cfg.OutputPath,
				Severity: SeverityError,
				Stage:    StageIO,
			})
			return result, &DiagnosticsError{Diagnostics: result.Diagnostics}
		}
	}

	return result, nil
}

// compileMultiBank compiles program via the 3-pass multi-bank strategy,
// used only when a single-bank (pass 1) compile doesn't fit. Pass 2
// measures every function/helper's size in wide-call form (a flat
// single-bank ROMBuilder, but every call site sized as if it were a far
// call) and greedily packs them into ROM banks in emission order -- the
// entry function is always first, so it always lands in bank 1, matching
// the hardware boot vector requirement. With the resulting code-bank count
// known, image/music assets are reloaded at their real starting bank
// (immediately above the code banks), and pass 3 does the final emission
// against a BankedROMBuilder using pass 2's schedule.
func compileMultiBank(program *Program, sourcePath string, assets []AssetIR, cfg CompileOptions) (*CodeGenerator, []byte, uint32, error) {
	// Pass 2: measurement. The provisional data-start-bank guess doesn't
	// matter here -- only instruction counts feed the bank schedule, and
	// asset bank/offset values are baked into immediates, not sizes.
	const provisionalDataStart = 2
	measureImageAssets, measureImageRegion, imgErr := loadImageAssets(program, sourcePath, provisionalDataStart)
	if imgErr != nil {
		return nil, nil, 0, imgErr
	}
	measureMusicAssets, _, musErr := loadMusicAssets(program, sourcePath, provisionalDataStart, len(measureImageRegion))
	if musErr != nil {
		return nil, nil, 0, musErr
	}

	measureBuilder := rom.NewROMBuilder()
	measureGen := NewCodeGenerator(program, measureBuilder)
	measureGen.SetNormalizedAssets(assets)
	measureGen.SetImageAssets(measureImageAssets)
	measureGen.SetMusicAssets(measureMusicAssets)
	measureGen.EnableWideCallMode()
	if err := measureGen.Generate(); err != nil {
		return nil, nil, 0, fmt.Errorf("bank measurement pass: %w", err)
	}

	schedule, bankEnds, codeBankCount, packErr := packFunctionBanks(measureGen)
	if packErr != nil {
		return nil, nil, 0, packErr
	}

	// Finalize: image/music data starts immediately above the code banks
	// pass 2 determined are needed.
	dataStartBank := uint8(1 + codeBankCount)
	finalImageAssets, finalImageRegion, imgErr := loadImageAssets(program, sourcePath, dataStartBank)
	if imgErr != nil {
		return nil, nil, 0, imgErr
	}
	finalMusicAssets, finalMusicRegion, musErr := loadMusicAssets(program, sourcePath, dataStartBank, len(finalImageRegion))
	if musErr != nil {
		return nil, nil, 0, musErr
	}
	dataRegion := append(append([]byte{}, finalImageRegion...), finalMusicRegion...)

	// Pass 3: final emission -- BankedROMBuilder via a bankCursor adapter,
	// wide-call mode, real bank schedule.
	banked := rom.NewBankedROMBuilder()
	finalGen := NewCodeGenerator(program, nil)
	finalGen.builder = &bankCursor{b: banked, cg: finalGen}
	finalGen.SetNormalizedAssets(assets)
	finalGen.SetImageAssets(finalImageAssets)
	finalGen.SetMusicAssets(finalMusicAssets)
	finalGen.EnableWideCallMode()
	finalGen.SetBankedBuilder(banked, schedule)
	finalGen.SetFunctionBankEnds(bankEnds)
	if err := finalGen.Generate(); err != nil {
		return nil, nil, 0, fmt.Errorf("final multi-bank emission: %w", err)
	}

	if len(dataRegion) > 0 {
		banked.SetDataRegion(dataStartBank, dataRegion)
	}
	romBytes, err := banked.BuildROMBytes(cfg.EntryBank, cfg.EntryOffset)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("final multi-bank emission: %w", err)
	}

	var codeWords int
	for bank := uint8(1); bank <= uint8(codeBankCount); bank++ {
		codeWords += banked.GetCodeLength(bank)
	}
	return finalGen, romBytes, uint32(codeWords * 2), nil
}

// packFunctionBanks computes a function/helper -> ROM bank assignment from
// pass 2's flat measurement build: each entry's size is the word-index
// delta to the next entry in emission order, so the last entry's size
// naturally includes the trailing __irqstub bytes patchIRQVector appends
// after every named function/helper. Functions are packed greedily in
// emission order into ROMBankSizeWords-sized banks, starting a new bank
// whenever the next function wouldn't fit whole -- the entry function is
// always first in emission order, so it always lands in bank 1.
//
// A function whose measured body is itself larger than one bank is given a
// contiguous run of banks. Pass 3 continues that function across the run
// (see CodeGenerator.maybeSpill). The returned end map is the inclusive
// last bank reserved for each function; codeBankCount is the highest bank
// the schedule uses.
func packFunctionBanks(measureGen *CodeGenerator) (map[string]uint8, map[string]uint8, int, error) {
	order := measureGen.emitOrder
	schedule := make(map[string]uint8, len(order))
	ends := make(map[string]uint8, len(order))
	if len(order) == 0 {
		return schedule, ends, 1, nil
	}
	totalWords := measureGen.builder.GetCodeLength()
	sizes := make([]int, len(order))
	for i, name := range order {
		start := measureGen.functionAddrs[name].index
		end := totalWords
		if i+1 < len(order) {
			end = measureGen.functionAddrs[order[i+1]].index
		}
		sizes[i] = end - start
	}

	bank := uint8(1)
	used := 0
	maxBank := uint8(1)
	for i, name := range order {
		if sizes[i] > rom.ROMBankSizeWords {
			if used > 0 {
				if bank == rom.ROMMaxProgramBank {
					return nil, nil, 0, fmt.Errorf("function %s does not fit in the remaining ROM banks", name)
				}
				bank++
				used = 0
			}
			span := banksForOversizedFunction(sizes[i])
			endBank := int(bank) + span - 1
			if endBank > int(rom.ROMMaxProgramBank) {
				return nil, nil, 0, fmt.Errorf("function %s needs %d ROM banks starting at bank %d, past bank %d", name, span, bank, rom.ROMMaxProgramBank)
			}
			schedule[name] = bank
			ends[name] = uint8(endBank)
			if ends[name] > maxBank {
				maxBank = ends[name]
			}
			if i+1 < len(order) && endBank >= int(rom.ROMMaxProgramBank) {
				return nil, nil, 0, fmt.Errorf("function %s uses the last ROM bank; %s has nowhere to go", name, order[i+1])
			}
			if endBank < int(rom.ROMMaxProgramBank) {
				bank = uint8(endBank + 1)
			}
			used = 0
			continue
		}
		if used > 0 && used+sizes[i] > rom.ROMBankSizeWords {
			if bank == rom.ROMMaxProgramBank {
				return nil, nil, 0, fmt.Errorf("function %s does not fit in the remaining ROM banks", name)
			}
			bank++
			used = 0
		}
		schedule[name] = bank
		ends[name] = bank
		used += sizes[i]
		if bank > maxBank {
			maxBank = bank
		}
	}
	return schedule, ends, int(maxBank), nil
}

// banksForOversizedFunction is how many ROM banks pass 3 should reserve for
// a measured body that does not fit in one bank. The payload matches
// maybeSpill's slack so the reserved run covers the continuation jumps, and
// one spare bank absorbs a spill that lands on the final RET.
func banksForOversizedFunction(sizeWords int) int {
	payload := rom.ROMBankSizeWords - bankSpillSlackWords - farJumpWords
	if payload < 1 {
		payload = 1
	}
	n := (sizeWords + payload - 1) / payload
	if n < 2 {
		n = 2
	}
	return n + 1
}

func mergeCompileOptions(dst *CompileOptions, src CompileOptions) {
	if src.OutputPath != "" {
		dst.OutputPath = src.OutputPath
	}
	if src.ManifestOutputPath != "" {
		dst.ManifestOutputPath = src.ManifestOutputPath
	}
	if src.DiagnosticsOutputPath != "" {
		dst.DiagnosticsOutputPath = src.DiagnosticsOutputPath
	}
	if src.BundleOutputPath != "" {
		dst.BundleOutputPath = src.BundleOutputPath
	}
	if src.AssetManifestPath != "" {
		dst.AssetManifestPath = src.AssetManifestPath
	}
	if src.ModulesPath != "" {
		dst.ModulesPath = src.ModulesPath
	}
	if src.EntryBank != 0 {
		dst.EntryBank = src.EntryBank
	}
	if src.EntryOffset != 0 {
		dst.EntryOffset = src.EntryOffset
	}
	if src.MaxROMBytes != 0 {
		dst.MaxROMBytes = src.MaxROMBytes
	}
	if src.SectionBudgets != nil {
		dst.SectionBudgets = src.SectionBudgets
	}
	// A zero Emit* bool keeps the default. Callers that want an output off
	// set the matching No* field. ForceBootSplash defaults off, so setting
	// it true is the only override it needs.
	if src.EmitROMBytes {
		dst.EmitROMBytes = true
	}
	if src.EmitManifestJSON {
		dst.EmitManifestJSON = true
	}
	if src.EmitDiagnosticsJSON {
		dst.EmitDiagnosticsJSON = true
	}
	if src.EmitBundleJSON {
		dst.EmitBundleJSON = true
	}
	if src.NoROMBytes {
		dst.EmitROMBytes = false
	}
	if src.NoManifestJSON {
		dst.EmitManifestJSON = false
	}
	if src.NoDiagnosticsJSON {
		dst.EmitDiagnosticsJSON = false
	}
	if src.NoBundleJSON {
		dst.EmitBundleJSON = false
	}
	if src.ForceBootSplash {
		dst.ForceBootSplash = true
	}
}

func validatePackBudgets(manifest *BuildManifest, cfg CompileOptions, sourcePath string) []Diagnostic {
	if manifest == nil {
		return nil
	}
	diags := make([]Diagnostic, 0)
	if cfg.MaxROMBytes > 0 && manifest.PlannedROMSizeBytes > cfg.MaxROMBytes {
		diags = append(diags, Diagnostic{
			Category: CategoryOverflowError,
			Code:     "E_OVERFLOW_ROM",
			Message:  fmt.Sprintf("planned ROM size %d exceeds budget %d", manifest.PlannedROMSizeBytes, cfg.MaxROMBytes),
			File:     sourcePath,
			Severity: SeverityError,
			Stage:    StagePack,
			Notes: []string{
				"Reduce code/assets, or increase the ROM budget in CompileOptions.",
			},
		})
	}
	for _, s := range manifest.Sections {
		if cfg.SectionBudgets == nil {
			continue
		}
		budget, ok := cfg.SectionBudgets[s.Name]
		if !ok || budget == 0 {
			continue
		}
		if s.UsedBytes > budget {
			diags = append(diags, Diagnostic{
				Category: CategoryOverflowError,
				Code:     "E_OVERFLOW_SECTION",
				Message:  fmt.Sprintf("section %q uses %d bytes and exceeds budget %d", s.Name, s.UsedBytes, budget),
				File:     sourcePath,
				Severity: SeverityError,
				Stage:    StagePack,
				Notes: []string{
					"Adjust asset sizes or raise the section budget.",
				},
			})
		}
	}
	return diags
}

// CompileFile is a convenience wrapper for the current single-file workflow.
// It now uses the production compiler entrypoint and structured diagnostics internally.
func CompileFile(sourcePath, outputPath string) error {
	_, err := CompileProject(sourcePath, &CompileOptions{OutputPath: outputPath})
	return err
}

func stampDiagnosticsFile(diags []Diagnostic, file string) {
	if file == "" {
		return
	}
	for i := range diags {
		if diags[i].File == "" {
			diags[i].File = file
		}
		for j := range diags[i].Related {
			if diags[i].Related[j].File == "" {
				diags[i].Related[j].File = file
			}
		}
	}
}

func diagnosticFromError(err error, file, code string, category DiagnosticCategory, stage DiagnosticStage) Diagnostic {
	d := Diagnostic{
		Category: category,
		Code:     code,
		Message:  err.Error(),
		File:     file,
		Severity: SeverityError,
		Stage:    stage,
	}
	var ce *codeError
	if errors.As(err, &ce) && ce.pos.Line > 0 {
		d.Line = ce.pos.Line
		d.Column = ce.pos.Column
		d.Message = ce.err.Error()
	}
	return d
}

var parseErrRe = regexp.MustCompile(`parse error at line ([0-9]+), column ([0-9]+): (.*)$`)

func parseDiagnostic(err error, file string) Diagnostic {
	msg := err.Error()
	diag := Diagnostic{
		Category: CategorySyntaxError,
		Code:     "E_PARSE",
		Message:  msg,
		File:     file,
		Severity: SeverityError,
		Stage:    StageParser,
	}
	m := parseErrRe.FindStringSubmatch(msg)
	if len(m) == 4 {
		line, _ := strconv.Atoi(m[1])
		col, _ := strconv.Atoi(m[2])
		diag.Line = line
		diag.Column = col
		diag.Message = strings.TrimSpace(m[3])
	}
	return diag
}

func internalCompilerDiagnostic(stage DiagnosticStage, file string, recovered any) Diagnostic {
	return Diagnostic{
		Category: CategoryInternalCompilerError,
		Code:     "E_INTERNAL_PANIC",
		Message:  fmt.Sprintf("internal compiler panic: %v", recovered),
		File:     file,
		Severity: SeverityError,
		Stage:    stage,
	}
}

func normalizeDiagnosticRanges(diags []Diagnostic) {
	for i := range diags {
		if diags[i].Line > 0 && diags[i].EndLine == 0 {
			diags[i].EndLine = diags[i].Line
		}
		if diags[i].Line > 0 && diags[i].Column > 0 && diags[i].EndColumn == 0 {
			diags[i].EndColumn = diags[i].Column
		}
	}
}

// formatMemoryMap renders the WRAM allocation listing emitted with each build.
func formatMemoryMap(entries []MemoryMapEntry) []byte {
	var b strings.Builder
	b.WriteString("# CoreLX WRAM memory map (name  address  size  kind)\n")
	b.WriteString("# user scratch region (never compiler-allocated): 0x7000-0x7FFF\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%-24s 0x%04X  %4d  %s\n", e.Name, e.Address, e.Size, e.Kind)
	}
	return []byte(b.String())
}
