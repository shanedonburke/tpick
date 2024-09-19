package explorer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"tpick/filter"
	"tpick/text"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
)

type DisplayMode uint

const (
	DisplayNormal DisplayMode = iota
	DisplayKeybinds
)

type InputMode uint

const (
	InputNormal InputMode = iota
	InputFilterEntry
	InputKeybindsPage
)

type BarMode uint

const (
	BarCurrentDir BarMode = iota
	BarFilterEntry
	BarFilterApplied
)

type Item struct {
	Text  string
	IsDir bool
}

const QUICK_SELECT_AMT = 5

var dirStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
var selectedStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)
var normalBarStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorAqua)
var filterBarStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorYellow)
var filterMatchStyle = tcell.StyleDefault.Foreground(tcell.ColorRed)

type Keybind struct {
	key  string
	desc string
}

func newKeybind(key string, desc string) *Keybind {
	return &Keybind{
		key:  key,
		desc: desc,
	}
}

var keybinds = []*Keybind{
	newKeybind("Ctrl+C", "Quit"),
	newKeybind("?", "Open this help page"),
	newKeybind("[Shift] ↑/↓", "Change selection"),
	newKeybind("T/B", "Jump to top/bottom"),
	newKeybind("Space", "Open selected directory"),
	newKeybind("Backspace", "Open parent directory"),
	newKeybind("/", "Filter items"),
	newKeybind("Enter", "Print selected path and copy to clipboard"),
}

var dotItems = []*Item{
	{
		Text:  ".",
		IsDir: true,
	},
	{
		Text:  "..",
		IsDir: true,
	},
}

type Explorer struct {
	screen         tcell.Screen
	currentDir     string
	selectedIdx    int
	screenStartIdx int
	displayMode    DisplayMode
	inputMode      InputMode
	barMode        BarMode
	items          []*Item
	filterState    *filter.FilterState
}

func NewExplorer(s tcell.Screen, directory string) *Explorer {
	e := &Explorer{
		screen:         s,
		currentDir:     directory,
		selectedIdx:    0,
		screenStartIdx: 0,
		displayMode:    DisplayNormal,
		inputMode:      InputNormal,
		barMode:        BarCurrentDir,
		items:          []*Item{},
		filterState:    filter.NewFilterState(),
	}
	e.update()
	return e
}

func (e *Explorer) HandleKeyEvent(ev *tcell.EventKey) {
	switch e.inputMode {
	case InputNormal:
		e.handleNormalKeyEvent(ev)
	case InputFilterEntry:
		e.handleFilterEntryKeyEvent(ev)
	case InputKeybindsPage:
		e.handleKeybindsKeyEvent(ev)
	}
	e.update()
}

func (e *Explorer) handleNormalKeyEvent(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyCtrlC:
		e.quit()
	case tcell.KeyUp:
		e.handleNormalKeyUp(ev)
	case tcell.KeyDown:
		e.handleNormalKeyDown(ev)
	case tcell.KeyBackspace:
		e.navToParent()
	case tcell.KeyEnter:
		e.copyAndExit()
	case tcell.KeyEsc:
		e.resetFilter()
	default:
		switch ev.Rune() {
		case 't':
			e.selectFirst()
		case 'b':
			e.selectLast()
		case ' ':
			e.navToCurrentItem()
		case '/':
			e.beginFilterEntry()
		case '?':
			e.openKeybindsPage()
		}
	}
	e.update()
}

func (e *Explorer) handleNormalKeyUp(ev *tcell.EventKey) {
	if ev.Modifiers()&tcell.ModShift > 0 {
		e.sel(e.selectedIdx - QUICK_SELECT_AMT)
	} else {
		e.sel(e.selectedIdx - 1)
	}
}

func (e *Explorer) handleNormalKeyDown(ev *tcell.EventKey) {
	if ev.Modifiers()&tcell.ModShift > 0 {
		e.sel(e.selectedIdx + QUICK_SELECT_AMT)
	} else {
		e.sel(e.selectedIdx + 1)
	}
}

func (e *Explorer) selectFirst() {
	e.sel(0)
}

func (e *Explorer) selectLast() {
	e.sel(len(e.items) - 1)
}

func (e *Explorer) getParentPath() string {
	return filepath.Join(e.currentDir, "..")
}

func (e *Explorer) navToDirectory(path string) {
	e.currentDir = path
	e.selectedIdx = 0
	e.resetFilter()
}

func (e *Explorer) navToCurrentItem() {
	if e.getSelectedItem().IsDir {
		e.navToDirectory(e.getSelectedItemPath())
	}
}

func (e *Explorer) navToParent() {
	e.navToDirectory(e.getParentPath())
}

func (e *Explorer) copyAndExit() {
	itemPath := e.getSelectedItemPath()

	if err := clipboard.WriteAll(itemPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to copy selection to clipboard: %v\n", err)
		os.Exit(1)
	}

	e.closeScreen()
	fmt.Print(itemPath)
	os.Exit(0)
}

func (e *Explorer) clearSelection() {
	e.selectedIdx = -1
}

func (e *Explorer) getSelectedItem() *Item {
	return e.items[e.selectedIdx]
}

func (e *Explorer) getSelectedItemPath() string {
	return filepath.Join(e.currentDir, e.getSelectedItem().Text)
}

func (e *Explorer) handleFilterEntryKeyEvent(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyCtrlC:
		e.quit()
	case tcell.KeyEsc:
		e.resetFilter()
	case tcell.KeyEnter:
		e.applyFilter()
	case tcell.KeyLeft:
		e.filterState.MoveCursorLeft()
	case tcell.KeyRight:
		e.filterState.MoveCursorRight()
	case tcell.KeyUp:
	case tcell.KeyDown:
	case tcell.KeyBackspace:
		e.filterState.DeleteCharacter()
	default:
		e.filterState.InsertCharacter(ev.Rune())
	}
}

func (e *Explorer) beginFilterEntry() {
	e.filterState.PrevSelectionText = e.getSelectedItem().Text
	e.filterState.CursorLoc = text.Width(e.filterState.Text)
	e.inputMode = InputFilterEntry
	e.barMode = BarFilterEntry
	e.clearSelection()
}

func (e *Explorer) resetFilter() {
	e.filterState.CursorLoc = 0
	e.filterState.Text = ""
	e.inputMode = InputNormal
	e.barMode = BarCurrentDir
	e.update()
	e.selByText(e.filterState.PrevSelectionText)
}

func (e *Explorer) applyFilter() {
	if e.filterState.Text == "" {
		e.resetFilter()
	} else {
		e.inputMode = InputNormal
		e.barMode = BarFilterApplied
	}
	e.update()
	if len(e.items) > 2 {
		e.sel(2)
	} else {
		e.sel(0)
	}
}

func (e *Explorer) handleKeybindsKeyEvent(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyCtrlC:
		e.quit()
	default:
		e.closeKeybindsPage()
	}
	e.update()
}

func (e *Explorer) openKeybindsPage() {
	e.displayMode = DisplayKeybinds
	e.inputMode = InputKeybindsPage
}

func (e *Explorer) closeKeybindsPage() {
	e.displayMode = DisplayNormal
	// Return to correct mode
	if e.barMode == BarFilterEntry {
		// Opening keybinds page doesn't modify bar mode.
		// So if we were entering a filter before, we are now.
		e.inputMode = InputFilterEntry
	} else {
		e.inputMode = InputNormal
	}
}

func (e *Explorer) HandleResize() {
	e.update()
	e.screen.Sync()
}

func (e *Explorer) closeScreen() {
	e.screen.Clear()
	e.screen.Fini()
}

func (e *Explorer) quit() {
	e.closeScreen()
	os.Exit(0)
}

func (e *Explorer) update() {
	switch e.displayMode {
	case DisplayNormal:
		e.updateNormal()
		e.updateBar()
	case DisplayKeybinds:
		e.updateKeybinds()
	}
	e.screen.Sync()
}

func (e *Explorer) updateNormal() {
	e.screen.Clear()
	e.screen.HideCursor()

	entries := e.readCurrentDir()

	// Build items from contents of current directory
	items := []*Item{}
	for _, entry := range entries {
		isDir := entry.IsDir()
		text := entry.Name()
		if isDir {
			text += "/" // Trailing slash to indicate directory
		}
		if strings.Contains(text, e.filterState.Text) {
			items = append(items, &Item{
				Text:  text,
				IsDir: isDir,
			})
		}
	}
	slices.SortFunc(items, e.itemSortFunc)

	// Append ever-present '.' and '..' items
	e.items = append(
		dotItems,
		items...,
	)

	e.updateScreenStartIdx()

	for y, item := range e.items[e.screenStartIdx:] {
		if y+e.screenStartIdx == e.selectedIdx {
			// Item is selected
			e.drawText(item.Text, 0, y, selectedStyle)
		} else {
			e.drawUnselectedItem(item, y)
		}
	}
}

func (e *Explorer) drawUnselectedItem(item *Item, y int) {
	var style tcell.Style

	// Show directories in a different color
	if item.IsDir {
		style = dirStyle
	} else {
		style = tcell.StyleDefault
	}

	// Filter doesn't apply to '.' and '..
	if e.filterState.IsActive() && y >= 2 {
		// Highlight part of text matching filter
		matchIdx := strings.Index(item.Text, e.filterState.Text)
		pre := item.Text[:matchIdx]
		post := item.Text[matchIdx+len(e.filterState.Text):]
		e.drawText(pre, 0, y, style)
		e.drawText(e.filterState.Text, text.Width(pre), y, filterMatchStyle)
		e.drawText(post, text.Width(pre+e.filterState.Text), y, style)
	} else {
		e.drawText(item.Text, 0, y, style)
	}
}

func (e *Explorer) itemSortFunc(a, b *Item) int {
	// Directories alphabetically, then files alphabetically
	if a.IsDir && !b.IsDir {
		return -1
	} else if b.IsDir && !a.IsDir {
		return 1
	} else {
		return strings.Compare(a.Text, b.Text)
	}
}

func (e *Explorer) readCurrentDir() []fs.DirEntry {
	entries, err := os.ReadDir(e.currentDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read directory: %v\n", err)
		os.Exit(1)
	}
	return entries
}

func (e *Explorer) updateKeybinds() {
	y := 0

	drawLine := func(line string) {
		e.drawText(line, 0, y, tcell.StyleDefault)
		y++
	}

	e.screen.Clear()

	drawLine("(Press any key to close.)")
	drawLine("")
	drawLine("Keybinds:")
	drawLine("──────────")

	// Find longest "key" to align descriptions together
	maxKeyLen := 0
	for _, keybind := range keybinds {
		keyLen := text.Width(keybind.key)
		if keyLen > maxKeyLen {
			maxKeyLen = keyLen
		}
	}
	padTo := maxKeyLen

	for _, keybind := range keybinds {
		keyText := fmt.Sprintf("%-*s  ", padTo, keybind.key)
		line := keyText + keybind.desc
		drawLine(line)
	}
}

func (e *Explorer) updateBar() {
	switch e.barMode {
	case BarCurrentDir:
		e.updateBarCurrentDir()
	case BarFilterEntry:
		e.updateBarFilterEntry()
	case BarFilterApplied:
		e.updateBarFilterApplied()
	}
}

func (e *Explorer) updateBarCurrentDir() {
	style := normalBarStyle
	y := e.getUsableHeight()
	e.drawBarBase(style)
	barText := e.currentDir
	e.drawText(barText, 0, y, style.Bold(true))
	e.drawText(" (? for help)", text.Width(barText), y, style)
}

func (e *Explorer) updateBarFilterEntry() {
	style := filterBarStyle
	y := e.getUsableHeight()
	e.drawBarBase(style)
	barText := "(Esc/Enter) Enter filter: "
	e.drawText(barText, 0, y, style.Bold(true))
	e.screen.ShowCursor(text.Width(barText)+e.filterState.CursorLoc, y)
	e.drawText(e.filterState.Text, text.Width(barText), y, filterBarStyle)
}

func (e *Explorer) updateBarFilterApplied() {
	style := filterBarStyle
	y := e.getUsableHeight()
	e.drawBarBase(style)
	barText := "(Esc) Filter: "
	e.drawText(barText, 0, y, style.Bold(true))
	e.drawText(e.filterState.Text, text.Width(barText), y, filterBarStyle)
}

func (e *Explorer) sel(idx int) {
	if idx >= 0 && idx < len(e.items) {
		e.selectedIdx = idx
		e.filterState.PrevSelectionText = e.getSelectedItem().Text
	}
}

func (e *Explorer) selByText(text string) {
	for i, item := range e.items {
		if item.Text == text {
			e.sel(i)
			return
		}
	}
	e.sel(0)
}

func (e *Explorer) drawText(text string, x, y int, style tcell.Style) {
	for _, r := range text {
		e.screen.SetContent(x, y, r, nil, style)
		x++
	}
}

func (e *Explorer) updateScreenStartIdx() {
	usableHeight := e.getUsableHeight()
	if e.selectedIdx >= e.screenStartIdx+usableHeight {
		// Must scroll down
		e.screenStartIdx = e.selectedIdx - e.getUsableHeight() + 1
	} else if e.selectedIdx < e.screenStartIdx {
		// Must scroll up
		// Prevent starting at -1 if we are entering filter (no selection)
		e.screenStartIdx = max(0, e.selectedIdx)
	}
}

func (e *Explorer) getUsableHeight() int {
	_, height := e.screen.Size()
	return height - 1 // Leave room for bottom bar
}

func (e *Explorer) drawBarBase(style tcell.Style) {
	width, height := e.screen.Size()
	e.drawText(strings.Repeat(" ", width), 0, height-1, style)
}
