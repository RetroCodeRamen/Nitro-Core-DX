# CoreLX compiler contract

This is the contract for the compiler in `internal/corelx`. The programming
manual is the tutorial. This file is the short list of rules a
reimplementation has to match. When the two disagree, the compiler and its
tests win, and this file should be updated with them.

## Pipeline

A compile is lex, parse, semantic check, then code generation.

The lexer keeps one keyword map for the whole process. A newline inside
parentheses or brackets is not a statement break. A newline outside them
ends the statement. `--` comments and a carriage return before a newline do
not change indentation.

The parser reports every statement error it can, not just the first. A
missing `)` and a token that cannot start an expression name the token they
found. Parse errors stop the compile before semantic analysis.

## Types

Storage types are `int` (also spelled `u16` or `i16`), `u8`, `fixed` (8.8),
and `bool`. Integer literals `32768`–`65535` are accepted and compare as the
signed value of those bits, so `40000` is `-25536`. `70000` and a fixed
literal outside `-32768`..`32767` in 8.8 bits are compile errors. `010` is
decimal ten. `0x` is hex and `0b` is binary.

`int()` floors toward negative infinity. `/` and `%` round toward zero.
Those are different rules. Fixed divided by fixed is 8.8 toward zero, with
16-bit wrap. `%` on a fixed value is an error. A constant divide by zero is
a compile error. A divide by zero at run time yields `0xFFFF`. `>>` is an
arithmetic shift. Constant folding uses the same 16-bit rules as the
generated code, including `-32768 / -1` wrapping to `0x8000`.

`:=` takes the type of the expression: fixed stays fixed through a unary
minus, a multiply, `fixed()`, a fixed return, and a fixed array slot.
There is no pointer type and no address-of operator. `var x: *int` is a
parse error.

## Names and scope

Each `if`, `elseif`, `else`, `while`, and `for` body is its own block. A
name can be declared again in a sibling block. Declaring it twice in the
same block is an error. A `for` variable does not exist after the loop.
Globals and outer locals can be shadowed and come back when the block ends.

The twelve controller names `UP DOWN LEFT RIGHT A B X Y L R START Z` are
reserved. Redeclaring one says that it is a controller button name.

A function may call another function. A function may not call itself, and
two functions may not call each other in a cycle. The diagnostic is
`E_RECURSION`.

## Data

`type Name = struct` and `struct Name:` both declare a struct. A struct
global, a struct array (`var enemies: Player[8]`), and a nested field
(`body.pos.x`) are stored inline and start at zero. A local array
(`xs: int[4]`) is the same: no value initializer, slots start at zero. An
array parameter is a reference to the caller's slots, and the length is
part of the type, so `int[4]` and `int[8]` do not match. A const may be the
length (`const N = 8` then `int[N]`).

A struct parameter is a reference to the caller's struct, whether that
struct is a local (`hero := Player()`) or a global (`var hero: Player`). A
function returns one 16-bit value; it does not return a struct. Struct
data comes back by mutating a struct argument or by writing a global.

`Sprite` is a built-in 6-byte struct (`x_lo`, `x_hi`, `y`, `tile`, `attr`,
`ctrl`) and may be a global. `SPR_PAL`, `SPR_PRI`, `SPR_HFLIP`, `SPR_VFLIP`,
`SPR_ENABLE`, `SPR_SIZE_*`, `SPR_BLEND`, and `SPR_ALPHA` are real functions,
implemented next to the other sprite builtins.

## Code shape

Each function gets its own spill area for expression temporaries, allocated
before its parameters. A call may use registers, then spill, then evaluate
the next argument. Builtins are inlined. User calls pass arguments in
`R0`–`R5` and return in `R0`. `R6` and `R7` are scratch for far calls and
for address arithmetic. A nested expression clobbers every register, so a
live value is spilled first.

A function may continue into the next ROM bank. One statement still has to
fit in a single bank. Same-function branches are relative. Positive
non-negative integer division is inlined; a negative operand calls the
signed helper. That split is load-bearing for the overworld frame budget.

## ROM

Program banks are 1 through 125. Each bank stores 32KB, so the functional
ROM limit is 4,096,000 bytes. The file is that payload plus a 32-byte
header (4,096,032 bytes). Each bank is a 64KB address window, and only the
high half (`0x8000`–`0xFFFF`) is wired. Counting the unmapped half is where
the old "7.8MB" figure came from. Locals live in WRAM and do not shrink the
ROM.

## Compile options

`CompileOptions` merges onto defaults. A zero `EmitROMBytes`,
`EmitManifestJSON`, `EmitDiagnosticsJSON`, or `EmitBundleJSON` keeps the
default, which is on, so `&CompileOptions{OutputPath: path}` still emits a
ROM. Set `NoROMBytes`, `NoManifestJSON`, `NoDiagnosticsJSON`, or
`NoBundleJSON` to turn one of those off. `NoROMBytes` omits the bytes from
the result; an `OutputPath` still writes the file.
