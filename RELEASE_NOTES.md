# Nitro-Core-DX v0.1.7

## What's New

### Sprite Editor
The IDE now includes a built-in **Sprite Lab** — a pixel-art editor for creating game sprites directly inside the development environment. Draw sprites from 8x8 up to 64x64, manage 16 color palettes, and export your artwork straight into CoreLX code with one click. Supports undo/redo, mirror painting, grid overlays, and `.clxsprite` file import/export.

### IDE Redesign
The entire interface has been restructured to feel like a professional development environment rather than a collection of features bolted together. There's now a proper menu bar, the toolbar is organized into logical groups (Project, Build, Run/Debug, View), and the layout supports three view modes — **Split View**, **Emulator Focus**, and a new **Code Only** mode that hides the emulator entirely when you just want to write code.

### Project Templates
Starting a new project is much easier now. Choose from six built-in templates (Blank Game, Minimal Loop, Sprite Demo, Tilemap Demo, Shmup Starter, Matrix Mode Demo) instead of starting from scratch every time.

### Quality of Life
- **UI density modes** — switch between Compact and Standard spacing to fit more on screen
- **Autosave** — your work is automatically saved so you don't lose progress on a crash
- **Settings persistence** — view mode, split positions, recent files, and preferences are remembered between sessions
- **Load ROM button** — quickly test ROM files without having to rebuild every time
- **F11 maximize/restore** — proper window management on Linux

### Bug Fixes
- Fixed pixel alignment and grid line consistency in the Sprite Lab
- Fixed split view not rendering correctly after switching view modes
- Fixed window maximize not working on Linux (title bar double-click and right-click menu)
- Improved Sprite Lab painting performance — no more lag while drawing

---

**Downloads:**
- **Linux:** `nitrocoredx-v0.1.7-linux-amd64.tar.gz`
- **Windows:** `nitrocoredx-v0.1.7-windows-amd64.zip`
