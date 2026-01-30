package main

import (
	"fmt"
	"os"
	"path/filepath"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/rom"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.corelx> <output.rom>\n", os.Args[0])
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	// Read source file
	source, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Lex
	lexer := corelx.NewLexer(string(source))
	tokens, err := lexer.Tokenize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lexer error: %v\n", err)
		os.Exit(1)
	}

	// Check for lexer errors
	for _, tok := range tokens {
		if tok.Type == corelx.TOKEN_ERROR {
			fmt.Fprintf(os.Stderr, "Lexer error: %s\n", tok.Literal)
			os.Exit(1)
		}
	}

	// Parse
	parser := corelx.NewParser(tokens)
	program, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// Semantic analysis
	if err := corelx.Analyze(program); err != nil {
		fmt.Fprintf(os.Stderr, "Semantic error: %v\n", err)
		os.Exit(1)
	}

	// Code generation
	builder := rom.NewROMBuilder()
	codegen := corelx.NewCodeGenerator(program, builder)

	// Find entry point function (__Boot takes priority over Start)
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)
	
	var entryFunction *corelx.FunctionDecl
	hasBoot := false
	
	// Check for __Boot() function first
	for _, fn := range program.Functions {
		if fn.Name == "__Boot" {
			entryFunction = fn
			hasBoot = true
			break
		}
	}
	
	// Fall back to Start() if no __Boot
	if entryFunction == nil {
		for _, fn := range program.Functions {
			if fn.Name == "Start" {
				entryFunction = fn
				break
			}
		}
	}
	
	if entryFunction == nil {
		fmt.Fprintf(os.Stderr, "Error: No entry point function found (__Boot or Start required)\n")
		os.Exit(1)
	}

	// Generate code (functions are generated in order, so entry point will be at 0x8000)
	if err := codegen.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Code generation error: %v\n", err)
		os.Exit(1)
	}

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "ROM build error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled %s -> %s\n", filepath.Base(inputPath), filepath.Base(outputPath))
	if hasBoot {
		fmt.Println("Note: Using __Boot() as entry point - boot animation will be skipped")
	} else {
		fmt.Println("Note: Using Start() as entry point")
	}
}
