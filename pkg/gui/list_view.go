package gui

import "github.com/jesseduffield/gocui"

type listView struct {
	viewName                string
	context                 string
	getItemsLength          func() int
	getSelectedLineIdxPtr   func() *int
	handleFocus             func() error
	handleItemSelect        func() error
	handleClickSelectedItem func() error
	gui                     *Gui
	rendersToMainView       bool
}

func (lv *listView) handlePrevLine(g *gocui.Gui, v *gocui.View) error {
	return lv.handleLineChange(-1)
}

func (lv *listView) handleNextLine(g *gocui.Gui, v *gocui.View) error {
	return lv.handleLineChange(1)
}

func (lv *listView) handleLineChange(change int) error {
	if !lv.gui.isPopupPanel(lv.viewName) && lv.gui.popupPanelFocused() {
		return nil
	}

	lv.gui.changeSelectedLine(lv.getSelectedLineIdxPtr(), lv.getItemsLength(), change)

	if lv.rendersToMainView {
		if err := lv.gui.resetOrigin(lv.gui.getMainView()); err != nil {
			return err
		}
	}

	return lv.handleItemSelect()
}

func (lv *listView) handleNextPage(g *gocui.Gui, v *gocui.View) error {
	view, err := lv.gui.g.View(lv.viewName)
	if err != nil {
		return nil
	}
	_, height := view.Size()
	delta := height - 1
	if delta == 0 {
		delta = 1
	}
	return lv.handleLineChange(delta)
}

func (lv *listView) handleGotoTop(g *gocui.Gui, v *gocui.View) error {
	return lv.handleLineChange(-lv.getItemsLength())
}

func (lv *listView) handleGotoBottom(g *gocui.Gui, v *gocui.View) error {
	return lv.handleLineChange(lv.getItemsLength())
}

func (lv *listView) handlePrevPage(g *gocui.Gui, v *gocui.View) error {
	view, err := lv.gui.g.View(lv.viewName)
	if err != nil {
		return nil
	}
	_, height := view.Size()
	delta := height - 1
	if delta == 0 {
		delta = 1
	}
	return lv.handleLineChange(-delta)
}

func (lv *listView) handleClick(g *gocui.Gui, v *gocui.View) error {
	if !lv.gui.isPopupPanel(lv.viewName) && lv.gui.popupPanelFocused() {
		return nil
	}

	selectedLineIdxPtr := lv.getSelectedLineIdxPtr()
	prevSelectedLineIdx := *selectedLineIdxPtr
	newSelectedLineIdx := v.SelectedLineIdx()

	if newSelectedLineIdx > lv.getItemsLength()-1 {
		return lv.handleFocus()
	}

	*selectedLineIdxPtr = newSelectedLineIdx

	if lv.rendersToMainView {
		if err := lv.gui.resetOrigin(lv.gui.getMainView()); err != nil {
			return err
		}
	}

	prevViewName := lv.gui.currentViewName()
	if prevSelectedLineIdx == newSelectedLineIdx && prevViewName == lv.viewName && lv.handleClickSelectedItem != nil {
		return lv.handleClickSelectedItem()
	}
	return lv.handleItemSelect()
}

func (gui *Gui) getListViews() []*listView {
	return []*listView{
		{
			viewName:              "menu",
			getItemsLength:        func() int { return gui.getMenuView().LinesHeight() },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Menu.SelectedLine },
			handleFocus:           gui.handleMenuSelect,
			handleItemSelect:      gui.handleMenuSelect,
			// need to add a layer of indirection here because the callback changes during runtime
			handleClickSelectedItem: func() error { return gui.State.Panels.Menu.OnPress(gui.g, nil) },
			gui:                     gui,
			rendersToMainView:       false,
		},
		{
			viewName:                "files",
			getItemsLength:          func() int { return len(gui.State.Files) },
			getSelectedLineIdxPtr:   func() *int { return &gui.State.Panels.Files.SelectedLine },
			handleFocus:             gui.focusAndSelectFile,
			handleItemSelect:        gui.focusAndSelectFile,
			handleClickSelectedItem: gui.handleFilePress,
			gui:                     gui,
			rendersToMainView:       true,
		},
		{
			viewName:              "branches",
			context:               "local-branches",
			getItemsLength:        func() int { return len(gui.State.Branches) },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Branches.SelectedLine },
			handleFocus:           gui.handleBranchSelect,
			handleItemSelect:      gui.handleBranchSelect,
			gui:                   gui,
			rendersToMainView:     true,
		},
		{
			viewName:                "branches",
			context:                 "remotes",
			getItemsLength:          func() int { return len(gui.State.Remotes) },
			getSelectedLineIdxPtr:   func() *int { return &gui.State.Panels.Remotes.SelectedLine },
			handleFocus:             gui.renderRemotesWithSelection,
			handleItemSelect:        gui.handleRemoteSelect,
			handleClickSelectedItem: gui.handleRemoteEnter,
			gui:                     gui,
			rendersToMainView:       true,
		},
		{
			viewName:              "branches",
			context:               "remote-branches",
			getItemsLength:        func() int { return len(gui.State.RemoteBranches) },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.RemoteBranches.SelectedLine },
			handleFocus:           gui.handleRemoteBranchSelect,
			handleItemSelect:      gui.handleRemoteBranchSelect,
			gui:                   gui,
			rendersToMainView:     true,
		},
		{
			viewName:              "branches",
			context:               "tags",
			getItemsLength:        func() int { return len(gui.State.Tags) },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Tags.SelectedLine },
			handleFocus:           gui.handleTagSelect,
			handleItemSelect:      gui.handleTagSelect,
			gui:                   gui,
			rendersToMainView:     true,
		},

		{
			viewName:                "commits",
			context:                 "branch-commits",
			getItemsLength:          func() int { return len(gui.State.Commits) },
			getSelectedLineIdxPtr:   func() *int { return &gui.State.Panels.Commits.SelectedLine },
			handleFocus:             gui.handleCommitSelect,
			handleItemSelect:        gui.handleCommitSelect,
			handleClickSelectedItem: gui.handleSwitchToCommitFilesPanel,
			gui:                     gui,
			rendersToMainView:       true,
		},
		{
			viewName:              "commits",
			context:               "reflog-commits",
			getItemsLength:        func() int { return len(gui.State.FilteredReflogCommits) },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.ReflogCommits.SelectedLine },
			handleFocus:           gui.handleReflogCommitSelect,
			handleItemSelect:      gui.handleReflogCommitSelect,
			gui:                   gui,
			rendersToMainView:     true,
		},
		{
			viewName:              "stash",
			getItemsLength:        func() int { return len(gui.State.StashEntries) },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Stash.SelectedLine },
			handleFocus:           gui.handleStashEntrySelect,
			handleItemSelect:      gui.handleStashEntrySelect,
			gui:                   gui,
			rendersToMainView:     true,
		},
		{
			viewName:              "commitFiles",
			getItemsLength:        func() int { return len(gui.State.CommitFiles) },
			getSelectedLineIdxPtr: func() *int { return &gui.State.Panels.CommitFiles.SelectedLine },
			handleFocus:           gui.handleCommitFileSelect,
			handleItemSelect:      gui.handleCommitFileSelect,
			gui:                   gui,
			rendersToMainView:     true,
		},
	}
}
