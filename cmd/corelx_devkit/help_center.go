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
	{"Start Here", "Programming Manual", "PROGRAMMING_MANUAL.md", "Beginner-friendly guide for building software with CoreLX and Nitro-Core-DX."},
	{"Start Here", "Documentation Index", "docs/README.md", "Organized entry point to specs, guides, testing, and planning docs."},

	{"CoreLX", "CoreLX Language Guide", "docs/CORELX.md", "Language syntax, built-ins, and current compiler-supported features."},
	{"CoreLX", "CoreLX Data Model Plan", "docs/CORELX_DATA_MODEL_PLAN.md", "Asset/data model and compiler packaging plan for the Dev Kit."},
	{"CoreLX", "NC8 CoreLX Compiler Design", "docs/specifications/CORELX_NITRO_CORE_8_COMPILER_DESIGN.md", "Target-aware CoreLX compiler design for Nitro-Core-8."},

	{"Dev Kit", "Dev Kit Architecture", "docs/DEVKIT_ARCHITECTURE.md", "Backend/frontend split and integration contract for Nitro-Core-DX."},
	{"Dev Kit", "Release Binaries", "docs/guides/RELEASE_BINARIES.md", "How Linux/Windows release packages are built and distributed."},
	{"Dev Kit", "Creating a Release", "docs/guides/CREATING_A_RELEASE.md", "Tag-and-publish release workflow using GitHub Actions."},
	{"Dev Kit", "End of Day Procedure", "docs/guides/END_OF_DAY_PROCEDURE.md", "Project end-of-day checklist and maintenance flow."},

	{"Hardware & Audio", "Hardware Spec v2.1", "docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md", "Current source-of-truth hardware specification."},
	{"Hardware & Audio", "APU FM OPM Extension Spec", "docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md", "FM extension design and current implementation status."},
	{"Hardware & Audio", "Hardware Features Status", "docs/HARDWARE_FEATURES_STATUS.md", "Implementation status snapshot across major subsystems."},

	{"Testing & Project", "Testing Guide", "docs/testing/README.md", "Current test commands, tiers, and practical test workflows."},
	{"Testing & Project", "Guides Index", "docs/guides/README.md", "Guides and operational workflows."},
	{"Testing & Project", "Specifications Index", "docs/specifications/README.md", "Current and historical hardware/compiler specs."},
	{"Testing & Project", "Planning Index", "docs/planning/README.md", "Planning docs and future feature parking lot."},
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

	editMenu := fyne.NewMenu("Edit",
		disabledMenuItem("Undo"),
		disabledMenuItem("Redo"),
		fyne.NewMenuItemSeparator(),
		disabledMenuItem("Find"),
		disabledMenuItem("Find Next"),
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
		fyne.NewMenuItem("Maximize Window (F11)", func() {
			s.maximizeWindow()
		}),
		fyne.NewMenuItem("Restore Window", func() {
			s.restoreWindow()
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
				"Nitro-Core-DX is a project-centric SDK with an integrated emulator subsystem.\n\nUse Build + Run for the primary workflow. Code Only hides the emulator for focused development. Split View shows code and hardware output side by side. Emulator Focus isolates hardware output testing.\n\nPress F11 to maximize/restore the window.",
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
	docByID := make(map[string]helpDocItem)
	categories := make([]string, 0)
	categorySet := make(map[string]bool)
	for _, item := range helpDocCatalog {
		indexByCategory[item.Category] = append(indexByCategory[item.Category], item)
		if !categorySet[item.Category] {
			categorySet[item.Category] = true
			categories = append(categories, item.Category)
		}
		docByID["doc:"+item.Path] = item
	}
	sort.Strings(categories)
	for _, cat := range categories {
		sort.Slice(indexByCategory[cat], func(i, j int) bool {
			return indexByCategory[cat][i].Title < indexByCategory[cat][j].Title
		})
	}

	titleLbl := widget.NewLabelWithStyle("Nitro-Core-DX Help Center", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	pathLbl := widget.NewLabel("Select a document from the list")
	pathLbl.Wrapping = fyne.TextWrapBreak
	descLbl := widget.NewLabel("")
	descLbl.Wrapping = fyne.TextWrapWord
	statusLbl := widget.NewLabel("")
	statusLbl.Wrapping = fyne.TextWrapWord

	docViewer := widget.NewRichTextFromMarkdown("# Nitro-Core-DX Help Center\n\nSelect a document from the left panel.")
	docViewer.Wrapping = fyne.TextWrapWord
	docScroll := container.NewScroll(docViewer)

	var currentDoc helpDocItem
	loadDoc := func(item helpDocItem) {
		currentDoc = item
		titleLbl.SetText(item.Title)
		pathLbl.SetText(item.Path)
		descLbl.SetText(item.Description)

		content, loadSource, err := loadHelpDocContent(projectRoot, item.Path)
		if err != nil {
			statusLbl.SetText(fmt.Sprintf("Could not load local doc (%v). Use 'Open on GitHub'.", err))
			docViewer.ParseMarkdown(fmt.Sprintf("# %s\n\n%s\n\n**Local file not found:** `%s`\n\nUse the toolbar button to open the GitHub version.", item.Title, item.Description, item.Path))
			return
		}
		statusLbl.SetText("Loaded from " + loadSource)
		docViewer.ParseMarkdown(content)
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

	tree := widget.NewTree(
		func(uid string) []string {
			switch {
			case uid == "":
				return []string{"root"}
			case uid == "root":
				ids := make([]string, 0, len(categories))
				for _, cat := range categories {
					ids = append(ids, "cat:"+cat)
				}
				return ids
			case strings.HasPrefix(uid, "cat:"):
				cat := strings.TrimPrefix(uid, "cat:")
				items := indexByCategory[cat]
				ids := make([]string, 0, len(items))
				for _, item := range items {
					ids = append(ids, "doc:"+item.Path)
				}
				return ids
			default:
				return nil
			}
		},
		func(uid string) bool {
			return uid == "root" || strings.HasPrefix(uid, "cat:")
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(uid string, branch bool, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			switch {
			case uid == "root":
				lbl.SetText("Documentation")
			case strings.HasPrefix(uid, "cat:"):
				lbl.SetText(strings.TrimPrefix(uid, "cat:"))
			case strings.HasPrefix(uid, "doc:"):
				if item, ok := docByID[uid]; ok {
					lbl.SetText(item.Title)
				} else {
					lbl.SetText(strings.TrimPrefix(uid, "doc:"))
				}
			default:
				lbl.SetText(uid)
			}
		},
	)
	tree.OpenAllBranches()
	tree.OnSelected = func(uid widget.TreeNodeID) {
		if !strings.HasPrefix(uid, "doc:") {
			return
		}
		if item, ok := docByID[uid]; ok {
			loadDoc(item)
		}
	}

	leftPane := container.NewBorder(
		widget.NewLabelWithStyle("Docs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		tree,
	)

	contentTop := container.NewVBox(
		titleLbl,
		pathLbl,
		descLbl,
		container.NewHBox(openGitHubBtn, refreshBtn),
		statusLbl,
		widget.NewSeparator(),
	)
	rightPane := container.NewBorder(contentTop, nil, nil, nil, docScroll)

	split := container.NewHSplit(leftPane, rightPane)
	split.Offset = 0.28
	w.SetContent(split)

	// Default selection: Programming Manual if present, else first catalog entry.
	var defaultItem *helpDocItem
	for i := range helpDocCatalog {
		if helpDocCatalog[i].Path == "PROGRAMMING_MANUAL.md" {
			defaultItem = &helpDocCatalog[i]
			break
		}
	}
	if defaultItem == nil && len(helpDocCatalog) > 0 {
		defaultItem = &helpDocCatalog[0]
	}
	if defaultItem != nil {
		id := "doc:" + defaultItem.Path
		tree.Select(id)
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
	if _, err := os.Stat(filepath.Join(root, "PROGRAMMING_MANUAL.md")); err != nil {
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
