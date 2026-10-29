package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type helpDocItem struct {
	Category    string
	Title       string
	Path        string
	Description string
}

var helpDocCatalog = []helpDocItem{
	{"Start Here", "README", "README.md", "Project overview, downloads, quick start, and current status."},
	{"Start Here", "Nitro-Core-DX Manual", "NITRO_CORE_DX_MANUAL.md", "The console, the hardware, and the Dev Kit."},
	{"Start Here", "CoreLX Programming Manual", "CORELX_PROGRAMMING_MANUAL.md", "The CoreLX language, builtins, and how studio assets land in source."},
	{"Start Here", "Documentation Index", "docs/README.md", "The two manuals, the demo programs, and the archive."},
}

func (s *devKitState) buildMainMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New Project", func() {
			s.showTemplateDialog()
		}),
		fyne.NewMenuItem("Open Project...", func() {
			s.showOpenProjectDialog()
		}),
		fyne.NewMenuItem("Load ROM...", func() {
			s.openROMDialog()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Save", func() {
			if err := s.save(); err != nil {
				dialog.ShowError(err, s.window)
			}
		}),
		fyne.NewMenuItem("Save As...", func() {
			s.saveAsDialog()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Recover Autosave", func() {
			s.tryRecoverAutosave()
		}),
		fyne.NewMenuItemSeparator(),
		s.buildRecentFilesMenuItem(),
	)

	findItem := fyne.NewMenuItem("Find", func() { s.showEditorFind(false) })
	findItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}
	replaceItem := fyne.NewMenuItem("Replace", func() { s.showEditorFind(true) })
	replaceItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyH, Modifier: fyne.KeyModifierControl}
	gotoItem := fyne.NewMenuItem("Go to Line", func() { s.showEditorGoto() })
	gotoItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyG, Modifier: fyne.KeyModifierControl}
	editMenu := fyne.NewMenu("Edit",
		disabledMenuItem("Undo"),
		disabledMenuItem("Redo"),
		fyne.NewMenuItemSeparator(),
		findItem,
		fyne.NewMenuItem("Find Next", func() { s.editorFind(true) }),
		fyne.NewMenuItem("Find Previous", func() { s.editorFind(false) }),
		replaceItem,
		fyne.NewMenuItemSeparator(),
		gotoItem,
	)

	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Code Only", func() {
			s.setViewMode(viewModeCodeOnly)
		}),
		fyne.NewMenuItem("Split View", func() {
			s.setViewMode(viewModeFull)
		}),
		fyne.NewMenuItem("Emulator Focus", func() {
			s.setViewMode(viewModeEmulatorOnly)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Focus Active Panel", func() {
			s.togglePanelFocus()
		}),
	)

	buildMenu := fyne.NewMenu("Build",
		fyne.NewMenuItem("Build", func() {
			s.runBuild(false)
		}),
		fyne.NewMenuItem("Build + Run", func() {
			s.runBuild(true)
		}),
	)

	debugMenu := fyne.NewMenu("Debug",
		fyne.NewMenuItem("Run", func() {
			s.runEmulator()
		}),
		fyne.NewMenuItem("Pause", func() {
			s.pauseEmulator()
		}),
		fyne.NewMenuItem("Stop", func() {
			s.stopEmulator()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Step Frame", func() {
			s.stepFrame()
		}),
		fyne.NewMenuItem("Step CPU", func() {
			s.stepCPU()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Hardware Reset", func() {
			s.hardwareReset()
		}),
	)

	toolsMenu := fyne.NewMenu("Tools",
		fyne.NewMenuItem("Layout: Balanced", func() {
			s.applyLayoutPreset(layoutPresetBalanced)
		}),
		fyne.NewMenuItem("Layout: Code Focus", func() {
			s.applyLayoutPreset(layoutPresetCodeFocus)
		}),
		fyne.NewMenuItem("Layout: Art Mode", func() {
			s.applyLayoutPreset(layoutPresetArtMode)
		}),
		fyne.NewMenuItem("Layout: Debug Mode", func() {
			s.applyLayoutPreset(layoutPresetDebugMode)
		}),
		fyne.NewMenuItem("Layout: Emulator Focus", func() {
			s.applyLayoutPreset(layoutPresetEmulatorFocus)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Theme: Nitro Industrial", func() {
			s.setThemeChoice(themeIndustrial)
		}),
		fyne.NewMenuItem("Theme: Nitro Classic", func() {
			s.setThemeChoice(themeClassic)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("UI Density: Compact", func() {
			s.setUIDensity("compact")
		}),
		fyne.NewMenuItem("UI Density: Standard", func() {
			s.setUIDensity("standard")
		}),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Help Center", func() {
			s.showHelpCenter()
		}),
		fyne.NewMenuItem("Open Docs on GitHub", func() {
			s.openExternalURL("https://github.com/RetroCodeRamen/Nitro-Core-DX/tree/main/docs")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("About Nitro-Core-DX", func() {
			dialog.ShowInformation(
				"About Nitro-Core-DX",
				"Nitro-Core-DX is a project-centric SDK with an integrated emulator subsystem.\n\nUse Build + Run for the primary workflow. Code Only hides the emulator for focused development. Split View shows code and hardware output side by side. Emulator Focus isolates hardware output testing.\n\nWindow maximize/restore is handled by your operating system title bar controls.",
				s.window,
			)
		}),
	)
	return fyne.NewMainMenu(fileMenu, editMenu, viewMenu, buildMenu, debugMenu, toolsMenu, helpMenu)
}

func disabledMenuItem(label string) *fyne.MenuItem {
	item := fyne.NewMenuItem(label, nil)
	item.Disabled = true
	return item
}

func (s *devKitState) buildRecentFilesMenuItem() *fyne.MenuItem {
	item := fyne.NewMenuItem("Open Recent", nil)
	recentMenu := fyne.NewMenu("Open Recent")
	if len(s.settings.RecentFiles) == 0 {
		empty := fyne.NewMenuItem("(none)", nil)
		empty.Disabled = true
		recentMenu.Items = []*fyne.MenuItem{empty}
		item.ChildMenu = recentMenu
		return item
	}

	recentMenu.Items = make([]*fyne.MenuItem, 0, len(s.settings.RecentFiles))
	for _, path := range s.settings.RecentFiles {
		p := path
		label := baseNameOr(p, p)
		recentMenu.Items = append(recentMenu.Items, fyne.NewMenuItem(label, func() {
			if err := s.loadFile(p, true); err != nil {
				dialog.ShowError(err, s.window)
				s.appendBuildOutput("Open recent failed: " + err.Error())
				s.setStatus("Open recent failed")
				return
			}
			s.setStatus("Opened " + baseNameOr(p, p))
		}))
	}
	item.ChildMenu = recentMenu
	return item
}

func (s *devKitState) showHelpCenter() {
	w := fyne.CurrentApp().NewWindow("Nitro-Core-DX Help Center")
	w.Resize(fyne.NewSize(1200, 820))

	projectRoot := findProjectRoot()
	githubBase := "https://github.com/RetroCodeRamen/Nitro-Core-DX/blob/main/"

	indexByCategory := make(map[string][]helpDocItem)
	categories := make([]string, 0)
	categorySet := make(map[string]bool)
	for _, item := range helpDocCatalog {
		indexByCategory[item.Category] = append(indexByCategory[item.Category], item)
		if !categorySet[item.Category] {
			categorySet[item.Category] = true
			categories = append(categories, item.Category)
		}
	}
	sort.Strings(categories)
	for _, cat := range categories {
		sort.Slice(indexByCategory[cat], func(i, j int) bool {
			return indexByCategory[cat][i].Title < indexByCategory[cat][j].Title
		})
	}

	titleLbl := widget.NewLabelWithStyle("Nitro-Core-DX Help Center", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	pathLbl := widget.NewLabel("Select a document from the list")
	pathLbl.Wrapping = fyne.TextWrapWord
	descLbl := widget.NewLabel("")
	descLbl.Wrapping = fyne.TextWrapWord
	statusLbl := widget.NewLabel("")
	statusLbl.Wrapping = fyne.TextWrapWord

	docHolder := container.NewMax(renderHelpDocument("Select a document from the list.", nil))
	var headings []helpHeading
	var sections []helpSection
	var sectionList *widget.List
	showMarkdown := func(markdown string) {
		docHolder.Objects = []fyne.CanvasObject{container.NewScroll(renderHelpDocument(markdown, func(anchor string) {
			for i, heading := range headings {
				if headingMatchesAnchor(heading.Title, anchor) {
					if sectionList != nil {
						sectionList.Select(i + 1)
					}
					return
				}
			}
		}))}
		docHolder.Refresh()
	}

	var currentDoc helpDocItem
	var currentMarkdown string

	showSection := func(index int) {
		showMarkdown(helpSectionMarkdown(currentMarkdown, headings, index))
	}
	loadDoc := func(item helpDocItem) {
		currentDoc = item
		titleLbl.SetText(item.Title)
		pathLbl.SetText(item.Path)
		descLbl.SetText(item.Description)

		content, loadSource, err := loadHelpDocContent(projectRoot, item.Path)
		if err != nil {
			statusLbl.SetText(fmt.Sprintf("Could not load local doc (%v). Use Open on GitHub.", err))
			currentMarkdown = fmt.Sprintf("# %s\n\n%s\n\n**Local file not found:** `%s`\n\nUse the toolbar button to open the GitHub version.", item.Title, item.Description, item.Path)
			headings = nil
		} else {
			statusLbl.SetText("Loaded from " + loadSource)
			currentMarkdown = content
			headings = helpHeadings(content)
		}
		sections = helpSectionList(headings)
		if sectionList != nil {
			sectionList.UnselectAll()
			sectionList.Refresh()
			sectionList.Select(0)
		}
		showSection(-1)
	}

	openGitHubBtn := widget.NewButton("Open on GitHub", func() {
		if currentDoc.Path == "" {
			s.openExternalURL("https://github.com/RetroCodeRamen/Nitro-Core-DX/tree/main/docs")
			return
		}
		s.openExternalURL(githubBase + currentDoc.Path)
	})
	refreshBtn := widget.NewButton("Reload", func() {
		if currentDoc.Path == "" {
			return
		}
		loadDoc(currentDoc)
	})

	type helpNavRow struct {
		header bool
		item   helpDocItem
	}
	navRows := make([]helpNavRow, 0, len(helpDocCatalog)+len(categories))
	for _, cat := range categories {
		navRows = append(navRows, helpNavRow{header: true, item: helpDocItem{Title: cat}})
		for _, item := range indexByCategory[cat] {
			navRows = append(navRows, helpNavRow{item: item})
		}
	}
	docList := widget.NewList(
		func() int { return len(navRows) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("Document")
			lbl.Wrapping = fyne.TextWrapWord
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id < 0 || id >= len(navRows) {
				lbl.SetText("")
				return
			}
			row := navRows[id]
			if row.header {
				lbl.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
				lbl.SetText(strings.ToUpper(row.item.Title))
				return
			}
			lbl.TextStyle = fyne.TextStyle{}
			lbl.SetText(row.item.Title)
		},
	)
	docList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(navRows) || navRows[id].header {
			return
		}
		loadDoc(navRows[id].item)
	}

	sectionList = widget.NewList(
		func() int { return len(sections) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Wrapping = fyne.TextWrapWord
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id < 0 || id >= len(sections) {
				lbl.SetText("")
				return
			}
			lbl.SetText(sections[id].Title)
		},
	)
	sectionList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(sections) {
			return
		}
		showSection(sections[id].Index)
	}

	docsPane := container.NewBorder(
		widget.NewLabelWithStyle("DOCUMENTS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true}),
		nil, nil, nil, docList,
	)
	outlinePane := container.NewBorder(
		widget.NewLabelWithStyle("OUTLINE", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true}),
		nil, nil, nil, sectionList,
	)
	navSplit := container.NewVSplit(docsPane, outlinePane)
	navSplit.Offset = 0.38
	leftPane := navSplit

	contentTop := container.NewVBox(
		titleLbl,
		pathLbl,
		descLbl,
		container.NewHBox(openGitHubBtn, refreshBtn),
		statusLbl,
		widget.NewSeparator(),
	)
	rightPane := container.NewBorder(contentTop, nil, nil, nil, docHolder)

	split := container.NewHSplit(leftPane, rightPane)
	split.Offset = 0.30
	w.SetContent(container.NewPadded(split))

	// Default selection: Programming Manual if present, else first catalog entry.
	var defaultItem *helpDocItem
	for i := range helpDocCatalog {
		if helpDocCatalog[i].Path == "CORELX_PROGRAMMING_MANUAL.md" {
			defaultItem = &helpDocCatalog[i]
			break
		}
	}
	if defaultItem == nil && len(helpDocCatalog) > 0 {
		defaultItem = &helpDocCatalog[0]
	}
	if defaultItem != nil {
		for i, row := range navRows {
			if !row.header && row.item.Path == defaultItem.Path {
				docList.Select(i)
				break
			}
		}
		loadDoc(*defaultItem)
	}

	w.Show()
}

func loadHelpDocContent(projectRoot, relPath string) (content string, source string, err error) {
	if projectRoot != "" {
		full := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		if b, readErr := os.ReadFile(full); readErr == nil {
			return string(b), full, nil
		}
	}
	return "", "", fmt.Errorf("file not found: %s", relPath)
}

func findProjectRoot() string {
	candidates := make([]string, 0, 4)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates, exeDir)
		candidates = append(candidates, filepath.Dir(exeDir))
	}

	seen := make(map[string]bool)
	for _, c := range candidates {
		if c == "" {
			continue
		}
		abs, err := filepath.Abs(c)
		if err != nil || seen[abs] {
			continue
		}
		seen[abs] = true
		if hasDocsTree(abs) {
			return abs
		}
	}
	return ""
}

func hasDocsTree(root string) bool {
	if _, err := os.Stat(filepath.Join(root, "README.md")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "CORELX_PROGRAMMING_MANUAL.md")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "NITRO_CORE_DX_MANUAL.md")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "README.md")); err != nil {
		return false
	}
	return true
}

func (s *devKitState) openExternalURL(raw string) {
	u, err := url.Parse(raw)
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}
	if err := fyne.CurrentApp().OpenURL(u); err != nil {
		dialog.ShowError(err, s.window)
	}
}
