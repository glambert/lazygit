package gui

import (
	"math"
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (gui *Gui) nextScreenMode(g *gocui.Gui, v *gocui.View) error {
	gui.State.ScreenMode = utils.NextIntInCycle([]int{SCREEN_NORMAL, SCREEN_HALF, SCREEN_FULL}, gui.State.ScreenMode)
	// commits render differently depending on whether we're in fullscreen more or not
	if err := gui.refreshCommitsViewWithSelection(); err != nil {
		return err
	}
	// same with branches
	if err := gui.refreshBranchesViewWithSelection(); err != nil {
		return err
	}

	return nil
}

func (gui *Gui) prevScreenMode(g *gocui.Gui, v *gocui.View) error {
	gui.State.ScreenMode = utils.PrevIntInCycle([]int{SCREEN_NORMAL, SCREEN_HALF, SCREEN_FULL}, gui.State.ScreenMode)
	// commits render differently depending on whether we're in fullscreen more or not
	if err := gui.refreshCommitsViewWithSelection(); err != nil {
		return err
	}
	// same with branches
	if err := gui.refreshBranchesViewWithSelection(); err != nil {
		return err
	}

	return nil
}

func (gui *Gui) scrollUpView(viewName string) error {
	mainView, _ := gui.g.View(viewName)
	ox, oy := mainView.Origin()
	newOy := int(math.Max(0, float64(oy-gui.Config.GetUserConfig().GetInt("gui.scrollHeight"))))
	return mainView.SetOrigin(ox, newOy)
}

func (gui *Gui) scrollDownView(viewName string) error {
	mainView, _ := gui.g.View(viewName)
	ox, oy := mainView.Origin()
	y := oy
	if !gui.Config.GetUserConfig().GetBool("gui.scrollPastBottom") {
		_, sy := mainView.Size()
		y += sy
	}
	scrollHeight := gui.Config.GetUserConfig().GetInt("gui.scrollHeight")
	if y < mainView.LinesHeight() {
		if err := mainView.SetOrigin(ox, oy+scrollHeight); err != nil {
			return err
		}
	}
	if manager, ok := gui.viewBufferManagerMap[viewName]; ok {
		manager.ReadLines(scrollHeight)
	}
	return nil
}

func (gui *Gui) scrollUpMain(g *gocui.Gui, v *gocui.View) error {
	if gui.canScrollMergePanel() {
		gui.State.Panels.Merging.UserScrolling = true
	}

	return gui.scrollUpView("main")
}

func (gui *Gui) scrollDownMain(g *gocui.Gui, v *gocui.View) error {
	if gui.canScrollMergePanel() {
		gui.State.Panels.Merging.UserScrolling = true
	}

	return gui.scrollDownView("main")
}

func (gui *Gui) scrollUpSecondary(g *gocui.Gui, v *gocui.View) error {
	return gui.scrollUpView("secondary")
}

func (gui *Gui) scrollDownSecondary(g *gocui.Gui, v *gocui.View) error {
	return gui.scrollDownView("secondary")
}

func (gui *Gui) scrollUpConfirmationPanel(g *gocui.Gui, v *gocui.View) error {
	if v.Editable {
		return nil
	}
	return gui.scrollUpView("confirmation")
}

func (gui *Gui) scrollDownConfirmationPanel(g *gocui.Gui, v *gocui.View) error {
	if v.Editable {
		return nil
	}
	return gui.scrollDownView("confirmation")
}

func (gui *Gui) handleRefresh(g *gocui.Gui, v *gocui.View) error {
	return gui.refreshSidePanels(refreshOptions{mode: ASYNC})
}

func (gui *Gui) handleMouseDownMain(g *gocui.Gui, v *gocui.View) error {
	if gui.popupPanelFocused() {
		return nil
	}

	switch g.CurrentView().Name() {
	case "files":
		return gui.enterFile(false, v.SelectedLineIdx())
	case "commitFiles":
		return gui.enterCommitFile(v.SelectedLineIdx())
	}

	return nil
}

func (gui *Gui) handleMouseDownSecondary(g *gocui.Gui, v *gocui.View) error {
	if gui.popupPanelFocused() {
		return nil
	}

	switch g.CurrentView().Name() {
	case "files":
		return gui.enterFile(true, v.SelectedLineIdx())
	}

	return nil
}

func (gui *Gui) handleInfoClick(g *gocui.Gui, v *gocui.View) error {
	if !gui.g.Mouse {
		return nil
	}

	cx, _ := v.Cursor()
	width, _ := v.Size()

	// if we're in the normal context there will be a donate button here
	// if we have ('reset') at the end then
	if gui.inFilterMode() {
		if width-cx <= len(gui.Tr.SLocalize("(reset)")) {
			return gui.exitFilterMode()
		} else {
			return nil
		}
	}

	if gui.inDiffMode() {
		if width-cx <= len(gui.Tr.SLocalize("(reset)")) {
			return gui.exitDiffMode()
		} else {
			return nil
		}
	}

	if cx <= len(gui.Tr.SLocalize("Donate"))+len(INFO_SECTION_PADDING) {
		return gui.OSCommand.OpenLink("https://github.com/sponsors/jesseduffield")
	}
	return nil
}

func (gui *Gui) fetch(canPromptForCredentials bool) (err error) {
	fetchOpts := commands.FetchOptions{}
	if canPromptForCredentials {
		fetchOpts.PromptUserForCredential = gui.promptUserForCredential
	}

	err = gui.GitCommand.Fetch(fetchOpts)

	if canPromptForCredentials && err != nil && strings.Contains(err.Error(), "exit status 128") {
		gui.createErrorPanel(gui.Tr.SLocalize("PassUnameWrong"))
	}

	gui.refreshSidePanels(refreshOptions{scope: []int{BRANCHES, COMMITS, REMOTES, TAGS}, mode: ASYNC})

	return err
}
