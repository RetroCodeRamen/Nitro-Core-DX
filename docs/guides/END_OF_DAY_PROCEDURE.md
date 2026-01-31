# End of Day Procedure

**Purpose**: Ensure all changes are documented, project is clean, and everything is pushed to GitHub.

**Trigger**: When user says "Run the End of Day procedure" or similar, read this file and execute all steps.

---

## Step 1: Review All Changes Made Today

### 1.1 Check Git Status
```bash
git status
git diff --stat
```

### 1.2 Identify Changed Files
- Note all modified files
- Note all new files
- Note all deleted files
- Identify any untracked files that should be committed

### 1.3 Review Code Changes
- Read through modified source files to understand what changed
- Identify new features, bug fixes, or improvements
- Note any breaking changes or API modifications

---

## Step 2: Update Documentation

### 2.1 Programming Manual (`NITRO_CORE_DX_PROGRAMMING_MANUAL.md`)

**Check and update:**
- [ ] **New Instructions**: If any CPU instructions were added/modified, document them
- [ ] **I/O Registers**: If any new I/O registers were added, add to register map
- [ ] **Memory Map**: If memory layout changed, update memory map section
- [ ] **Addressing Modes**: If addressing modes changed, update documentation
- [ ] **Examples**: Add code examples for new features
- [ ] **Version Number**: Update version and last updated date at top

**Key sections to review:**
- Instruction Set (Section 3)
- Memory Map (Section 4)
- I/O Register Map (Section 4.2)
- Programming Examples (Section 9)

### 2.2 System Manual (`SYSTEM_MANUAL.md`)

**Check and update:**
- [ ] **Architecture Changes**: Document any architectural changes
- [ ] **Component Updates**: Update CPU, PPU, APU, Memory sections if changed
- [ ] **Development Tools**: Document new debugging tools or features
- [ ] **Testing**: Update test coverage if new tests were added
- [ ] **FPGA Compatibility**: Note any FPGA compatibility changes
- [ ] **Development Status**: Update "Completed Components" section

**Key sections to review:**
- System Architecture (Section 2)
- Component Details (Sections 3-7)
- Development Tools (Section 8)
- Development Status (Section 9)

### 2.3 README.md

**Check and update:**
- [ ] **Features List**: Add new features to "Currently Implemented"
- [ ] **Status**: Update project status if major milestones reached
- [ ] **Quick Start**: Update if build/run instructions changed
- [ ] **Documentation Links**: Ensure all doc links are correct

### 2.4 CHANGELOG.md

**Always update:**
- [ ] **Add new entry** under [Unreleased] or create new version section
- [ ] **Categorize changes**: Added, Changed, Fixed, Removed, Deprecated
- [ ] **Include details**: What changed, why, where (file locations)
- [ ] **Link to issues**: If applicable, reference issue numbers
- [ ] **Date**: Use current date for new entries

**Format:**
```markdown
## [Unreleased] or ## [X.Y.Z] - YYYY-MM-DD

### Added
- Feature description

### Changed
- Change description

### Fixed
- Bug fix description (include file location)
```

### 2.5 MASTER_PLAN.md (docs/planning/MASTER_PLAN.md)

**Check and update:**
- [ ] **Completed Items**: Move completed items from "In Progress" to "Completed"
- [ ] **New Issues**: Document any new issues found
- [ ] **Status Updates**: Update implementation status for features
- [ ] **Next Steps**: Update planned items if priorities changed

---

## Step 3: Code Quality Checks

### 3.1 Linter Errors
```bash
# Check for linter errors in modified files
read_lints tool on all modified files
```

**Action**: Fix any linter errors before proceeding.

### 3.2 Build Verification
```bash
# Ensure project still builds
go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator
```

**Action**: If build fails, fix issues before proceeding.

### 3.3 Test Status
```bash
# Run tests (if applicable)
go test ./... -v 2>&1 | grep -E "(PASS|FAIL|ok|FAIL)"
```

**Action**: Note any test failures (don't block, but document).

---

## Step 4: Cleanup Non-Essential Files

### 4.1 Check .gitignore Compliance

**Files that should NOT be committed (verify they're ignored):**
- [ ] Binary files: `nitro-core-dx`, `emulator`, `*.exe`, `*.dll`, `*.so`, `*.dylib`
- [ ] ROM files: `*.rom`, `test/roms/*.rom`
- [ ] Log files: `*.log`, `emulator_log_*.txt`, `sprite_debug.log`
- [ ] Debug dumps: `register_state_*.txt`
- [ ] Build artifacts: `*.test`, `*.out`
- [ ] IDE files: `.vscode/`, `.idea/`, `*.swp`, `*.swo`, `*~`
- [ ] OS files: `.DS_Store`, `Thumbs.db`
- [ ] Temporary files: `*.tmp`, `*.bak`, `*.orig`

**Action**: If any of these are tracked, remove them:
```bash
git rm --cached <file>
# Then ensure .gitignore includes them
```

### 4.2 Remove Temporary Files

**Check for and remove:**
- [ ] Temporary test files in project root
- [ ] Old backup files (`*.bak`, `*.orig`)
- [ ] Debug output files that shouldn't be committed
- [ ] Any files created during testing that aren't needed

**Action**: Delete files that shouldn't be in repo:
```bash
# Review untracked files
git status --untracked-files=all

# Remove unwanted files
rm <unwanted-file>
```

### 4.3 Verify Project Structure

**Ensure:**
- [ ] No duplicate files
- [ ] No orphaned files (files not referenced anywhere)
- [ ] Documentation is in correct locations
- [ ] Test files are in `test/` directory
- [ ] Source files are in `internal/` or `cmd/` directories

---

## Step 5: Update Version Numbers

### 5.1 Check if Version Bump Needed

**Bump version if:**
- Major feature added → Increment minor version (0.2.0 → 0.3.0)
- Breaking change → Increment major version (0.2.0 → 1.0.0)
- Bug fix only → Increment patch version (0.2.0 → 0.2.1)

### 5.2 Update Version References

**Files that may contain version numbers:**
- [ ] `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` (header)
- [ ] `CHANGELOG.md` (new version section)
- [ ] `README.md` (if version mentioned)
- [ ] `go.mod` (if module version tracked)

---

## Step 6: Final Documentation Review

### 6.1 Cross-Reference Check

**Verify consistency:**
- [ ] Programming Manual examples match actual implementation
- [ ] System Manual architecture matches code structure
- [ ] README status matches actual implementation status
- [ ] CHANGELOG entries match actual changes
- [ ] docs/planning/MASTER_PLAN.md status matches reality

### 6.2 Documentation Completeness

**Ensure:**
- [ ] All new features are documented
- [ ] All API changes are documented
- [ ] All breaking changes are clearly marked
- [ ] Examples are accurate and tested
- [ ] Links between documents work correctly

---

## Step 7: Git Operations

### 7.1 Stage All Changes
```bash
git add -A
```

### 7.2 Review Staged Changes
```bash
git status --short
git diff --cached --stat
```

**Verify:**
- [ ] Only intended files are staged
- [ ] No binary files are staged
- [ ] No temporary files are staged
- [ ] Documentation updates are included

### 7.3 Create Commit

**Commit message format:**
```
<Brief summary> (< 50 chars)

<Detailed description>
- What changed
- Why it changed
- Impact/benefits

<Optional: Breaking changes>
<Optional: Related issues>
```

**Example:**
```bash
git commit -m "Add cycle logger and fix sprite movement

- Implemented cycle-by-cycle debug logger with PPU/APU state
- Fixed MOV mode 2 I/O register bug (sprite movement)
- Added register viewer with copy/save functionality
- Updated all documentation

Breaking changes: None
Fixes: Sprite movement issue"
```

### 7.4 Push to GitHub
```bash
git push
```

**If upstream not set:**
```bash
git push --set-upstream origin main
```

---

## Step 8: Verification

### 8.1 Post-Push Verification

**Verify:**
- [ ] Push succeeded (check output)
- [ ] All files are on remote (optional: check GitHub)
- [ ] No errors during push

### 8.2 Final Status Check
```bash
git status
```

**Should show:**
```
On branch main
Your branch is up to date with 'origin/main'.
nothing to commit, working tree clean
```

---

## Step 9: Summary Report

**Generate summary:**
- List all changes made
- List all documentation updated
- List all files cleaned up
- Note any issues encountered
- Note any items deferred to next session

**Present to user:**
```
End of Day Procedure Complete ✅

Changes Committed:
- [list of changes]

Documentation Updated:
- [list of docs]

Files Cleaned:
- [list of files]

Pushed to GitHub: ✅

Status: Working tree clean
```

---

## Quick Reference Checklist

**Before running procedure, ensure:**
- [ ] All code changes are complete
- [ ] All tests pass (or failures are documented)
- [ ] All features are working

**During procedure:**
- [ ] Review all changes
- [ ] Update Programming Manual
- [ ] Update System Manual
- [ ] Update README
- [ ] Update CHANGELOG
- [ ] Update docs/planning/MASTER_PLAN.md
- [ ] Fix linter errors
- [ ] Verify build
- [ ] Clean up files
- [ ] Stage changes
- [ ] Commit changes
- [ ] Push to GitHub
- [ ] Verify success

**After procedure:**
- [ ] Confirm working tree clean
- [ ] Confirm push successful
- [ ] Present summary to user

---

## Notes

- **Don't skip steps**: Each step is important for maintaining project quality
- **When in doubt**: Document it - better to over-document than under-document
- **Breaking changes**: Always clearly mark and document breaking changes
- **Version bumps**: Be conservative - only bump when necessary
- **Git commits**: Make meaningful commits with clear messages
- **Time estimate**: This procedure typically takes 10-15 minutes

---

## Troubleshooting

### If build fails:
- Fix build errors before proceeding
- Document any known issues in docs/planning/MASTER_PLAN.md

### If tests fail:
- Document failures but don't block
- Add to TODO list for next session

### If documentation is inconsistent:
- Prioritize accuracy over speed
- Cross-reference with actual code

### If git push fails:
- Check network connection
- Verify GitHub credentials
- Check for conflicts (git pull first if needed)

---

**End of Procedure**
