package filter

import "tpick/text"

type FilterState struct {
	Text              string
	CursorLoc         int
	PrevSelectionText string
}

func (fs *FilterState) IsActive() bool {
	return fs.Text != ""
}

func (fs *FilterState) MoveCursorLeft() {
	if fs.CursorLoc > 0 {
		fs.CursorLoc--
	}
}

func (fs *FilterState) MoveCursorRight() {
	if fs.CursorLoc < text.Width(fs.Text) {
		fs.CursorLoc++
	}
}

func (fs *FilterState) DeleteCharacter() {
	if fs.CursorLoc > 0 {
		ft := fs.Text
		cl := fs.CursorLoc
		fs.Text = string([]rune(ft)[:cl-1]) + string([]rune(ft)[cl:])
		fs.CursorLoc--
	}
}

func (fs *FilterState) InsertCharacter(r rune) {
	ft := fs.Text
	cl := fs.CursorLoc
	fs.Text = string([]rune(ft)[:cl]) + string(r) + string([]rune(ft)[cl:])
	fs.CursorLoc++
}

func NewFilterState() *FilterState {
	return &FilterState{
		Text:              "",
		CursorLoc:         0,
		PrevSelectionText: "",
	}
}
