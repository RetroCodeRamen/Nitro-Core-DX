package corelx

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveModulesDir returns the directory searched for `--! modules:`
// requests. An explicit CompileOptions.ModulesPath wins; otherwise it's a
// "modules" directory next to the main source file — modules are plain
// .corelx files installed alongside a project, not baked into the compiler.
func resolveModulesDir(sourcePath string, cfg CompileOptions) string {
	if cfg.ModulesPath != "" {
		return cfg.ModulesPath
	}
	return filepath.Join(filepath.Dir(sourcePath), "modules")
}

// loadModules resolves every module named in a `--! modules:` directive,
// parses each one (a module is a plain .corelx file, parsed with the same
// lexer/parser as any program), and merges its declarations into program:
// functions are namespaced by module name (charter: `walker.update`, the
// same dotted-call convention as builtins), so they resolve exactly like a
// builtin call once merged into program.Functions. Types/consts/globals are
// merged unprefixed — a module's own function bodies reference them by their
// plain names, and any name collision with the main program (or another
// module) is caught by the existing duplicate-declaration diagnostics in
// semantic analysis, which runs immediately after this.
func loadModules(program *Program, sourcePath string, cfg CompileOptions) []Diagnostic {
	if len(program.Modules) == 0 {
		return nil
	}
	var diags []Diagnostic
	modulesDir := resolveModulesDir(sourcePath, cfg)

	for _, name := range program.Modules {
		modPath := filepath.Join(modulesDir, name+".corelx")
		source, err := os.ReadFile(modPath)
		if err != nil {
			diags = append(diags, Diagnostic{
				Category: CategoryAssetReferenceError,
				Code:     "E_MODULE_NOT_INSTALLED",
				Message:  fmt.Sprintf("module `%s` not installed (looked for %s)", name, modPath),
				File:     sourcePath,
				Severity: SeverityError,
				Stage:    StageParser,
			})
			continue
		}

		lexer := NewLexer(string(source))
		tokens, lexErr := lexer.Tokenize()
		if lexErr != nil {
			diags = append(diags, Diagnostic{
				Category: CategorySyntaxError,
				Code:     "E_MODULE_LEX",
				Message:  fmt.Sprintf("module `%s`: %s", name, lexErr.Error()),
				File:     modPath,
				Severity: SeverityError,
				Stage:    StageLexer,
			})
			continue
		}

		modProgram, parseErr := NewParser(tokens).Parse()
		if parseErr != nil {
			diags = append(diags, Diagnostic{
				Category: CategorySyntaxError,
				Code:     "E_MODULE_PARSE",
				Message:  fmt.Sprintf("module `%s`: %s", name, parseErr.Error()),
				File:     modPath,
				Severity: SeverityError,
				Stage:    StageParser,
			})
			continue
		}

		for _, fn := range modProgram.Functions {
			namespaced := *fn
			namespaced.Name = name + "." + fn.Name
			program.Functions = append(program.Functions, &namespaced)
		}
		program.Types = append(program.Types, modProgram.Types...)
		program.Consts = append(program.Consts, modProgram.Consts...)
		program.Globals = append(program.Globals, modProgram.Globals...)
	}

	return diags
}
