package gui

import (
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
)

type ListContext struct {
	ViewName              string
	ContextKey            string
	GetItemsLength        func() int
	GetSelectedLineIdxPtr func() *int
	GetDisplayStrings     func() [][]string
	OnFocus               func() error
	OnFocusLost           func() error
	OnClickSelectedItem   func() error

	Gui               *Gui
	RendersToMainView bool
	Kind              int
}

// OnFocus assumes that the content of the context has already been rendered to the view. OnRender is the function which actually renders the content to the view
func (lc *ListContext) OnRender() error {
	view, err := lc.Gui.g.View(lc.ViewName)
	if err != nil {
		return nil
	}

	if lc.GetDisplayStrings != nil {
		lc.Gui.refreshSelectedLine(lc.GetSelectedLineIdxPtr(), lc.GetItemsLength())
		lc.Gui.renderDisplayStrings(view, lc.GetDisplayStrings())
	}

	return nil
}

func (lc *ListContext) GetKey() string {
	return lc.ContextKey
}

func (lc *ListContext) GetKind() int {
	return lc.Kind
}

func (lc *ListContext) GetViewName() string {
	return lc.ViewName
}

func (lc *ListContext) HandleFocusLost() error {
	if lc.OnFocusLost != nil {
		return lc.OnFocusLost()
	}

	return nil
}

func (lc *ListContext) HandleFocus() error {
	if lc.Gui.popupPanelFocused() {
		return nil
	}

	if lc.Gui.inDiffMode() {
		return lc.Gui.renderDiff()
	}

	if lc.OnFocus != nil {
		return lc.OnFocus()
	}

	return nil
}

func (lc *ListContext) HandleRender() error {
	return lc.OnRender()
}

func (lc *ListContext) handlePrevLine(g *gocui.Gui, v *gocui.View) error {
	return lc.handleLineChange(-1)
}

func (lc *ListContext) handleNextLine(g *gocui.Gui, v *gocui.View) error {
	return lc.handleLineChange(1)
}

func (lc *ListContext) handleLineChange(change int) error {
	if !lc.Gui.isPopupPanel(lc.ViewName) && lc.Gui.popupPanelFocused() {
		return nil
	}

	view, err := lc.Gui.g.View(lc.ViewName)
	if err != nil {
		return err
	}

	lc.Gui.changeSelectedLine(lc.GetSelectedLineIdxPtr(), lc.GetItemsLength(), change)
	view.FocusPoint(0, *lc.GetSelectedLineIdxPtr())

	if lc.RendersToMainView {
		if err := lc.Gui.resetOrigin(lc.Gui.getMainView()); err != nil {
			return err
		}
		if err := lc.Gui.resetOrigin(lc.Gui.getSecondaryView()); err != nil {
			return err
		}
	}

	return lc.HandleFocus()
}

func (lc *ListContext) handleNextPage(g *gocui.Gui, v *gocui.View) error {
	view, err := lc.Gui.g.View(lc.ViewName)
	if err != nil {
		return nil
	}
	_, height := view.Size()
	delta := height - 1
	if delta == 0 {
		delta = 1
	}
	return lc.handleLineChange(delta)
}

func (lc *ListContext) handleGotoTop(g *gocui.Gui, v *gocui.View) error {
	return lc.handleLineChange(-lc.GetItemsLength())
}

func (lc *ListContext) handleGotoBottom(g *gocui.Gui, v *gocui.View) error {
	return lc.handleLineChange(lc.GetItemsLength())
}

func (lc *ListContext) handlePrevPage(g *gocui.Gui, v *gocui.View) error {
	view, err := lc.Gui.g.View(lc.ViewName)
	if err != nil {
		return nil
	}
	_, height := view.Size()
	delta := height - 1
	if delta == 0 {
		delta = 1
	}
	return lc.handleLineChange(-delta)
}

func (lc *ListContext) handleClick(g *gocui.Gui, v *gocui.View) error {
	if !lc.Gui.isPopupPanel(lc.ViewName) && lc.Gui.popupPanelFocused() {
		return nil
	}

	selectedLineIdxPtr := lc.GetSelectedLineIdxPtr()
	prevSelectedLineIdx := *selectedLineIdxPtr
	newSelectedLineIdx := v.SelectedLineIdx()

	// we need to focus the view
	if err := lc.Gui.switchContext(lc); err != nil {
		return err
	}

	if newSelectedLineIdx > lc.GetItemsLength()-1 {
		return nil
	}

	*selectedLineIdxPtr = newSelectedLineIdx

	prevViewName := lc.Gui.currentViewName()
	if prevSelectedLineIdx == newSelectedLineIdx && prevViewName == lc.ViewName && lc.OnClickSelectedItem != nil {
		return lc.OnClickSelectedItem()
	}
	return lc.HandleFocus()
}

func (lc *ListContext) onSearchSelect(selectedLineIdx int) error {
	*lc.GetSelectedLineIdxPtr() = selectedLineIdx
	return lc.HandleFocus()
}

func (gui *Gui) menuListContext() *ListContext {
	return &ListContext{
		ViewName:              "menu",
		ContextKey:            "menu",
		GetItemsLength:        func() int { return gui.getMenuView().LinesHeight() },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Menu.SelectedLine },
		OnFocus:               gui.handleMenuSelect,
		// need to add a layer of indirection here because the callback changes during runtime
		OnClickSelectedItem: func() error { return gui.State.Panels.Menu.OnPress(gui.g, nil) },
		Gui:                 gui,
		RendersToMainView:   false,
		Kind:                PERSISTENT_POPUP,

		// no GetDisplayStrings field because we do a custom render on menu creation
	}
}

func (gui *Gui) filesListContext() *ListContext {
	return &ListContext{
		ViewName:              "files",
		ContextKey:            "files",
		GetItemsLength:        func() int { return len(gui.State.Files) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Files.SelectedLine },
		OnFocus:               gui.focusAndSelectFile,
		OnClickSelectedItem:   gui.handleFilePress,
		Gui:                   gui,
		RendersToMainView:     false,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetFileListDisplayStrings(gui.State.Files, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) branchesListContext() *ListContext {
	return &ListContext{
		ViewName:              "branches",
		ContextKey:            "local-branches",
		GetItemsLength:        func() int { return len(gui.State.Branches) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Branches.SelectedLine },
		OnFocus:               gui.handleBranchSelect,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetBranchListDisplayStrings(gui.State.Branches, gui.State.ScreenMode != SCREEN_NORMAL, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) remotesListContext() *ListContext {
	return &ListContext{
		ViewName:              "branches",
		ContextKey:            "remotes",
		GetItemsLength:        func() int { return len(gui.State.Remotes) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Remotes.SelectedLine },
		OnFocus:               gui.handleRemoteSelect,
		OnClickSelectedItem:   gui.handleRemoteEnter,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetRemoteListDisplayStrings(gui.State.Remotes, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) remoteBranchesListContext() *ListContext {
	return &ListContext{
		ViewName:              "branches",
		ContextKey:            "remote-branches",
		GetItemsLength:        func() int { return len(gui.State.RemoteBranches) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.RemoteBranches.SelectedLine },
		OnFocus:               gui.handleRemoteBranchSelect,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetRemoteBranchListDisplayStrings(gui.State.RemoteBranches, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) tagsListContext() *ListContext {
	return &ListContext{
		ViewName:              "branches",
		ContextKey:            "tags",
		GetItemsLength:        func() int { return len(gui.State.Tags) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Tags.SelectedLine },
		OnFocus:               gui.handleTagSelect,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetTagListDisplayStrings(gui.State.Tags, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) branchCommitsListContext() *ListContext {
	return &ListContext{
		ViewName:              "commits",
		ContextKey:            "branch-commits",
		GetItemsLength:        func() int { return len(gui.State.Commits) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Commits.SelectedLine },
		OnFocus:               gui.handleCommitSelect,
		OnClickSelectedItem:   gui.handleSwitchToCommitFilesPanel,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetCommitListDisplayStrings(gui.State.Commits, gui.State.ScreenMode != SCREEN_NORMAL, gui.cherryPickedCommitShaMap(), gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) reflogCommitsListContext() *ListContext {
	return &ListContext{
		ViewName:              "commits",
		ContextKey:            "reflog-commits",
		GetItemsLength:        func() int { return len(gui.State.FilteredReflogCommits) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.ReflogCommits.SelectedLine },
		OnFocus:               gui.handleReflogCommitSelect,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetReflogCommitListDisplayStrings(gui.State.FilteredReflogCommits, gui.State.ScreenMode != SCREEN_NORMAL, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) stashListContext() *ListContext {
	return &ListContext{
		ViewName:              "stash",
		ContextKey:            "stash",
		GetItemsLength:        func() int { return len(gui.State.StashEntries) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.Stash.SelectedLine },
		OnFocus:               gui.handleStashEntrySelect,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			// TODO :see if we still need to reset the origin here
			return presentation.GetStashEntryListDisplayStrings(gui.State.StashEntries, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) commitFilesListContext() *ListContext {
	return &ListContext{
		ViewName:              "commitFiles",
		ContextKey:            "commit-files",
		GetItemsLength:        func() int { return len(gui.State.CommitFiles) },
		GetSelectedLineIdxPtr: func() *int { return &gui.State.Panels.CommitFiles.SelectedLine },
		OnFocus:               gui.handleCommitFileSelect,
		Gui:                   gui,
		RendersToMainView:     true,
		Kind:                  SIDE_CONTEXT,
		GetDisplayStrings: func() [][]string {
			return presentation.GetCommitFileListDisplayStrings(gui.State.CommitFiles, gui.State.Diff.Ref)
		},
	}
}

func (gui *Gui) getListContexts() []*ListContext {
	return []*ListContext{
		gui.menuListContext(),
		gui.filesListContext(),
		gui.branchesListContext(),
		gui.remotesListContext(),
		gui.remoteBranchesListContext(),
		gui.tagsListContext(),
		gui.branchCommitsListContext(),
		gui.reflogCommitsListContext(),
		gui.stashListContext(),
		gui.commitFilesListContext(),
	}
}

func (gui *Gui) getListContextKeyBindings() []*Binding {
	bindings := make([]*Binding, 0)

	for _, listContext := range gui.getListContexts() {
		bindings = append(bindings, []*Binding{
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.prevItem-alt"), Modifier: gocui.ModNone, Handler: listContext.handlePrevLine},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.prevItem"), Modifier: gocui.ModNone, Handler: listContext.handlePrevLine},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gocui.MouseWheelUp, Modifier: gocui.ModNone, Handler: listContext.handlePrevLine},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.nextItem-alt"), Modifier: gocui.ModNone, Handler: listContext.handleNextLine},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.nextItem"), Modifier: gocui.ModNone, Handler: listContext.handleNextLine},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.prevPage"), Modifier: gocui.ModNone, Handler: listContext.handlePrevPage, Description: gui.Tr.SLocalize("prevPage")},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.nextPage"), Modifier: gocui.ModNone, Handler: listContext.handleNextPage, Description: gui.Tr.SLocalize("nextPage")},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gui.getKey("universal.gotoTop"), Modifier: gocui.ModNone, Handler: listContext.handleGotoTop, Description: gui.Tr.SLocalize("gotoTop")},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gocui.MouseWheelDown, Modifier: gocui.ModNone, Handler: listContext.handleNextLine},
			{ViewName: listContext.ViewName, Contexts: []string{listContext.ContextKey}, Key: gocui.MouseLeft, Modifier: gocui.ModNone, Handler: listContext.handleClick},
		}...)

		// the commits panel needs to lazyload things so it has a couple of its own handlers
		openSearchHandler := gui.handleOpenSearch
		gotoBottomHandler := listContext.handleGotoBottom
		if listContext.ViewName == "commits" {
			openSearchHandler = gui.handleOpenSearchForCommitsPanel
			gotoBottomHandler = gui.handleGotoBottomForCommitsPanel
		}

		bindings = append(bindings, []*Binding{
			{
				ViewName:    listContext.ViewName,
				Contexts:    []string{listContext.ContextKey},
				Key:         gui.getKey("universal.startSearch"),
				Handler:     openSearchHandler,
				Description: gui.Tr.SLocalize("startSearch"),
			},
			{
				ViewName:    listContext.ViewName,
				Contexts:    []string{listContext.ContextKey},
				Key:         gui.getKey("universal.gotoBottom"),
				Handler:     gotoBottomHandler,
				Description: gui.Tr.SLocalize("gotoBottom"),
			},
		}...)
	}

	return bindings
}
