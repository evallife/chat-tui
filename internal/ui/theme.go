package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Theme struct {
	Name      string
	Primary   tcell.Color
	Secondary tcell.Color
	Tertiary  tcell.Color
	Border    tcell.Color
	Title     tcell.Color
	Accent    tcell.Color
	InputBg   tcell.Color
	InputFg   tcell.Color
}

var themes = map[string]Theme{
	"night": {
		Name:      "Night",
		Primary:   tcell.ColorWhite,
		Secondary: tcell.ColorGray,
		Tertiary:  tcell.ColorLightGray,
		Border:    tcell.ColorDarkSlateGray,
		Title:     tcell.ColorLightSkyBlue,
		Accent:    tcell.ColorYellow,
		InputBg:   tcell.ColorBlack,
		InputFg:   tcell.ColorWhite,
	},
	"nord": {
		Name:      "Nord",
		Primary:   tcell.NewRGBColor(216, 222, 233),
		Secondary: tcell.NewRGBColor(129, 161, 193),
		Tertiary:  tcell.NewRGBColor(136, 192, 208),
		Border:    tcell.NewRGBColor(76, 86, 106),
		Title:     tcell.NewRGBColor(94, 129, 172),
		Accent:    tcell.NewRGBColor(163, 190, 140),
		InputBg:   tcell.NewRGBColor(46, 52, 64),
		InputFg:   tcell.NewRGBColor(216, 222, 233),
	},
	"gruvbox": {
		Name:      "Gruvbox",
		Primary:   tcell.NewRGBColor(235, 219, 178),
		Secondary: tcell.NewRGBColor(168, 153, 132),
		Tertiary:  tcell.NewRGBColor(189, 174, 147),
		Border:    tcell.NewRGBColor(80, 73, 69),
		Title:     tcell.NewRGBColor(215, 153, 33),
		Accent:    tcell.NewRGBColor(184, 187, 38),
		InputBg:   tcell.NewRGBColor(40, 40, 40),
		InputFg:   tcell.NewRGBColor(235, 219, 178),
	},
	"solarized-dark": {
		Name:      "Solarized Dark",
		Primary:   tcell.NewRGBColor(147, 161, 161),
		Secondary: tcell.NewRGBColor(133, 153, 0),
		Tertiary:  tcell.NewRGBColor(42, 161, 152),
		Border:    tcell.NewRGBColor(0, 43, 54),
		Title:     tcell.NewRGBColor(38, 139, 210),
		Accent:    tcell.NewRGBColor(203, 75, 22),
		InputBg:   tcell.NewRGBColor(0, 43, 54),
		InputFg:   tcell.NewRGBColor(131, 148, 150),
	},
	"light": {
		Name:      "Light",
		Primary:   tcell.ColorBlack,
		Secondary: tcell.ColorDarkSlateGray,
		Tertiary:  tcell.ColorGray,
		Border:    tcell.ColorSilver,
		Title:     tcell.ColorDarkCyan,
		Accent:    tcell.ColorDarkGreen,
		InputBg:   tcell.ColorWhite,
		InputFg:   tcell.ColorBlack,
	},
}

func (ui *TViewUI) applyTheme(themeKey string) {
	theme, ok := themes[themeKey]
	if !ok {
		theme = themes["night"]
	}
	tview.Styles.PrimitiveBackgroundColor = theme.InputBg
	tview.Styles.ContrastBackgroundColor = theme.Border
	tview.Styles.BorderColor = theme.Border
	tview.Styles.TitleColor = theme.Title
	tview.Styles.PrimaryTextColor = theme.Primary
	tview.Styles.SecondaryTextColor = theme.Secondary
	tview.Styles.TertiaryTextColor = theme.Tertiary

	if ui.Sidebar != nil {
		ui.Sidebar.SetTitleColor(theme.Accent)
	}
	if ui.ChatView != nil {
		ui.ChatView.SetTitleColor(theme.Title)
	}
	if ui.InputField != nil {
		ui.InputField.SetTitleColor(theme.Title)
		ui.InputField.SetTextStyle(tcell.StyleDefault.Foreground(theme.InputFg).Background(theme.InputBg))
		ui.InputField.SetLabelStyle(tcell.StyleDefault.Foreground(theme.Secondary).Background(theme.InputBg))
		ui.InputField.SetPlaceholderStyle(tcell.StyleDefault.Foreground(theme.Tertiary).Background(theme.InputBg))
	}
	if ui.CopyView != nil {
		ui.CopyView.SetTitleColor(theme.Title)
		ui.CopyView.SetTextStyle(tcell.StyleDefault.Foreground(theme.InputFg).Background(theme.InputBg))
		ui.CopyView.SetLabelStyle(tcell.StyleDefault.Foreground(theme.Secondary).Background(theme.InputBg))
	}
	if ui.SettingsForm != nil {
		ui.SettingsForm.SetTitleColor(theme.Title)
	}
	if ui.HistoryList != nil {
		ui.HistoryList.SetTitleColor(theme.Title)
	}
	if ui.HistoryPreview != nil {
		ui.HistoryPreview.SetTitleColor(theme.Title)
	}
	if ui.StatusBar != nil {
		ui.StatusBar.SetTextColor(theme.Tertiary)
		ui.StatusBar.SetBackgroundColor(theme.InputBg)
	}
}
