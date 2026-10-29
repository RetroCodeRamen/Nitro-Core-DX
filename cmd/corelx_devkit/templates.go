package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type projectTemplate struct {
	Name        string
	Description string
	Content     string
}

var devKitTemplates = []projectTemplate{
	{
		Name:        "Blank Game",
		Description: "Start from a clean project with display enabled and a vblank loop.",
		Content: `function Start()
    ppu.enable_display()

    while true
        wait_vblank()
`,
	},
	{
		Name:        "Minimal Loop",
		Description: "Small frame-cadence scaffold for gameplay update loops.",
		Content: `function Start()
    ppu.enable_display()
    frame := frame_counter()

    while true
        while frame_counter() == frame
            wait_vblank()
        frame = frame_counter()
        // update systems here
`,
	},
	{
		Name:        "Sprite Demo",
		Description: "Starter for sprite placement and OAM flush testing.",
		Content: `function Start()
    ppu.enable_display()
    gfx.init_default_palettes()

    player := Sprite()
    player.tile = 0
    player.attr = SPR_PAL(1) | SPR_PRI(0)
    player.ctrl = SPR_ENABLE() | SPR_SIZE_8()
    sprite.set_pos(&player, 128, 96)

    while true
        wait_vblank()
        oam.write(0, &player)
        oam.flush()
`,
	},
	{
		Name:        "Tilemap Demo",
		Description: "Starter for tilemap upload and camera experiments.",
		Content: `function Start()
    ppu.enable_display()
    gfx.init_default_palettes()

    while true
        wait_vblank()
`,
	},
	{
		Name:        "Shmup Starter",
		Description: "Vertical scrolling shoot-em-up scaffold with player movement and bullet firing.",
		Content: `function Start()
    ppu.enable_display()

    gfx.set_palette(1, 1, 0x7FFF)
    gfx.set_palette(1, 2, 0x03FF)
    gfx.set_palette(2, 1, 0x7C00)

    player := Sprite()
    player.tile = 0
    player.attr = SPR_PAL(1) | SPR_PRI(0)
    player.ctrl = SPR_ENABLE() | SPR_SIZE_16()
    px := 152
    py := 170

    bullet := Sprite()
    bullet.tile = 0
    bullet.attr = SPR_PAL(2) | SPR_PRI(0)
    bullet.ctrl = SPR_SIZE_8()
    bx := 0
    by := 0
    bullet_active := 0

    prev_buttons := 0
    frame := frame_counter()

    while true
        while frame_counter() == frame
            wait_vblank()
        frame = frame_counter()

        buttons := input.read(0)

        -- player movement
        if (buttons & 0x01) != 0
            if py > 8
                py = py - 2
        if (buttons & 0x02) != 0
            if py < 190
                py = py + 2
        if (buttons & 0x04) != 0
            if px > 8
                px = px - 2
        if (buttons & 0x08) != 0
            if px < 300
                px = px + 2

        -- fire bullet on A press (edge trigger)
        if (buttons & 0x10) != 0 and (prev_buttons & 0x10) == 0
            if bullet_active == 0
                bx = px
                by = py
                bullet_active = 1
                bullet.ctrl = SPR_ENABLE() | SPR_SIZE_8()

        -- update bullet
        if bullet_active != 0
            by = by - 4
            if by < 2
                bullet_active = 0
                bullet.ctrl = SPR_SIZE_8()

        prev_buttons = buttons

        sprite.set_pos(&player, px, py)
        oam.write(0, &player)

        if bullet_active != 0
            sprite.set_pos(&bullet, bx, by)
        oam.write(1, &bullet)
        oam.flush()
`,
	},
	{
		Name:        "Matrix Mode Demo",
		Description: "Starter scene for matrix transform experiments.",
		Content: `function Start()
    ppu.enable_display()
    gfx.init_default_palettes()

    while true
        wait_vblank()
`,
	},
}

func (s *devKitState) showTemplateDialog() {
	if len(devKitTemplates) == 0 {
		dialog.ShowInformation("Templates", "No templates are currently available.", s.window)
		return
	}

	var d dialog.Dialog
	descLabel := widget.NewLabel("Select a template to see its description.")
	descLabel.Wrapping = fyne.TextWrapWord
	preview := widget.NewMultiLineEntry()
	preview.Disable()
	preview.Wrapping = fyne.TextWrapOff
	preview.SetMinRowsVisible(12)
	preview.SetText("Source preview appears here.")

	selectedTemplate := ""
	selectTemplate := func(tpl projectTemplate) {
		selectedTemplate = tpl.Name
		descLabel.SetText(tpl.Description)
		preview.Enable()
		preview.SetText(tpl.Content)
		preview.Disable()
	}

	templateList := widget.NewList(
		func() int { return len(devKitTemplates) },
		func() fyne.CanvasObject {
			name := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			desc := widget.NewLabel("Short description stays on two lines\nso the row does not clip it.")
			desc.Wrapping = fyne.TextWrapWord
			return container.NewVBox(name, desc)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			box := obj.(*fyne.Container)
			name := box.Objects[0].(*widget.Label)
			desc := box.Objects[1].(*widget.Label)
			if id < 0 || id >= len(devKitTemplates) {
				name.SetText("")
				desc.SetText("")
				return
			}
			name.SetText(devKitTemplates[id].Name)
			desc.SetText(devKitTemplates[id].Description)
		},
	)
	templateList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(devKitTemplates) {
			return
		}
		selectTemplate(devKitTemplates[id])
	}

	createProject := func() {
		if selectedTemplate == "" {
			dialog.ShowInformation("New Project", "Select a project template first.", s.window)
			return
		}
		tpl, exists := templateByName(selectedTemplate)
		if !exists {
			dialog.ShowError(fmt.Errorf("template not found: %s", selectedTemplate), s.window)
			return
		}
		apply := func() {
			s.currentPath = ""
			s.pathLabel.SetText(displayPath(s.currentPath))
			s.setSourceContent(tpl.Content, true, false)
			s.setStatus("New project created: " + tpl.Name)
			s.appendBuildOutput("Loaded project template: " + tpl.Name)
			if d != nil {
				d.Hide()
			}
		}
		if s.dirty {
			dialog.NewConfirm("Discard current changes?", "Current unsaved editor content will be replaced by the selected project template.", func(confirm bool) {
				if confirm {
					apply()
				}
			}, s.window).Show()
			return
		}
		apply()
	}

	createBtn := widget.NewButton("Create Project", createProject)
	createBtn.Importance = widget.HighImportance

	previewPane := container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("PREVIEW", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), descLabel),
		nil, nil, nil,
		preview,
	)
	listPane := container.NewBorder(
		widget.NewLabelWithStyle("TEMPLATES", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		templateList,
	)
	content := container.NewBorder(
		nil,
		createBtn,
		nil, nil,
		container.NewHSplit(listPane, previewPane),
	)
	d = dialog.NewCustom("New Project", "Close", content, s.window)
	d.Resize(fyne.NewSize(820, 480))
	if len(devKitTemplates) > 0 {
		templateList.Select(0)
	}
	d.Show()
}

func templateByName(name string) (projectTemplate, bool) {
	for _, t := range devKitTemplates {
		if t.Name == name {
			return t, true
		}
	}
	return projectTemplate{}, false
}
