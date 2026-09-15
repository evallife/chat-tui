package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/evallife/chat-tui/internal/api"
	"github.com/evallife/chat-tui/internal/config"
	"github.com/rivo/tview"
)

func (ui *TViewUI) setupSettingsView() {
	ui.rebuildSettingsForm()
}

func (ui *TViewUI) rebuildSettingsForm() {
	ui.config.Normalize()
	providers := api.SupportedProviders()
	providerLabels := make([]string, len(providers))
	providerIndex := 0
	for i, p := range providers {
		providerLabels[i] = p.Label
		if p.Key == ui.config.CanonicalProvider() {
			providerIndex = i
		}
	}

	themeKeys := make([]string, 0, len(themes))
	for key := range themes {
		themeKeys = append(themeKeys, key)
	}
	sort.Strings(themeKeys)
	themeNames := make([]string, 0, len(themeKeys))
	currentThemeIndex := 0
	if ui.config.Theme == "" {
		ui.config.Theme = "night"
	}
	for i, key := range themeKeys {
		themeNames = append(themeNames, themes[key].Name)
		if key == ui.config.Theme {
			currentThemeIndex = i
		}
	}

	form := tview.NewForm()
	form.AddDropDown("Provider", providerLabels, providerIndex, nil)
	form.AddInputField("API Key", ui.config.APIKey, 48, nil, nil)
	form.AddInputField("Base URL", ui.config.BaseURL, 48, nil, nil)
	form.AddInputField("Model", ui.config.Model, 40, nil, nil)
	form.AddInputField("Region", ui.config.Region, 24, nil, nil)
	form.AddInputField("Access Key", ui.config.AccessKey, 40, nil, nil)
	form.AddInputField("Secret Key", ui.config.SecretKey, 40, nil, nil)
	form.AddDropDown("Theme", themeNames, currentThemeIndex, nil)
	form.AddInputField("Max Iterations", fmt.Sprintf("%d", ui.config.MaxIterations), 8, nil, nil)
	form.AddInputField("Workspace Root", ui.config.WorkspaceRoot, 48, nil, nil)
	form.AddCheckbox("Enable Write File", ui.config.WriteFileEnabled(), nil)
	form.AddCheckbox("Enable Run Command", ui.config.RunCommandEnabled(), nil)
	form.AddInputField("Agent Name", ui.config.AgentName, 24, nil, nil)
	form.AddCheckbox("Use Default Instruction", ui.config.DefaultInstructionEnabled(), nil)

	form.GetFormItemByLabel("Provider").(*tview.DropDown).SetSelectedFunc(func(_ string, optionIndex int) {
		if optionIndex < 0 || optionIndex >= len(providers) {
			return
		}
		p := providers[optionIndex].Key
		if field, ok := form.GetFormItemByLabel("Base URL").(*tview.InputField); ok {
			if api.IsKnownDefaultBaseURL(field.GetText()) {
				field.SetText(api.DefaultBaseURL(p))
			}
		}
		if field, ok := form.GetFormItemByLabel("Model").(*tview.InputField); ok && strings.TrimSpace(field.GetText()) == "" {
			field.SetText(api.DefaultModel(p))
		}
	})

	form.AddButton("Save", func() {
		_, providerLabel := form.GetFormItemByLabel("Provider").(*tview.DropDown).GetCurrentOption()
		chosen := ui.config.CanonicalProvider()
		for _, p := range providers {
			if p.Label == providerLabel {
				chosen = p.Key
				break
			}
		}
		ui.config.Provider = chosen
		ui.config.APIKey = form.GetFormItemByLabel("API Key").(*tview.InputField).GetText()
		ui.config.BaseURL = form.GetFormItemByLabel("Base URL").(*tview.InputField).GetText()
		ui.config.Model = form.GetFormItemByLabel("Model").(*tview.InputField).GetText()
		ui.config.Region = form.GetFormItemByLabel("Region").(*tview.InputField).GetText()
		ui.config.AccessKey = form.GetFormItemByLabel("Access Key").(*tview.InputField).GetText()
		ui.config.SecretKey = form.GetFormItemByLabel("Secret Key").(*tview.InputField).GetText()
		themeIndex, _ := form.GetFormItemByLabel("Theme").(*tview.DropDown).GetCurrentOption()
		if themeIndex >= 0 && themeIndex < len(themeKeys) {
			ui.config.Theme = themeKeys[themeIndex]
		}
		maxStr := strings.TrimSpace(form.GetFormItemByLabel("Max Iterations").(*tview.InputField).GetText())
		var maxIter int
		if _, err := fmt.Sscanf(maxStr, "%d", &maxIter); err == nil {
			ui.config.MaxIterations = maxIter
		} else if maxStr == "" {
			ui.config.MaxIterations = 0
		}
		ui.config.WorkspaceRoot = strings.TrimSpace(form.GetFormItemByLabel("Workspace Root").(*tview.InputField).GetText())
		ui.config.DisableWriteFile = !form.GetFormItemByLabel("Enable Write File").(*tview.Checkbox).IsChecked()
		ui.config.DisableRunCommand = !form.GetFormItemByLabel("Enable Run Command").(*tview.Checkbox).IsChecked()
		ui.config.AgentName = strings.TrimSpace(form.GetFormItemByLabel("Agent Name").(*tview.InputField).GetText())
		ui.config.DisableDefaultInstruction = !form.GetFormItemByLabel("Use Default Instruction").(*tview.Checkbox).IsChecked()
		ui.config.Normalize()
		if err := config.SaveConfig(ui.config); err != nil {
			ui.reportError("Failed to save config", err)
		}
		ui.apiClient.UpdateConfig(ui.config)
		ui.applyTheme(ui.config.Theme)
		ui.refreshChat()
		ui.Pages.SwitchToPage("chat")
		ui.App.SetFocus(ui.InputField)
	}).
		AddButton("Cancel", func() {
			ui.Pages.SwitchToPage("chat")
		})
	form.SetBorder(true).SetTitle(" Settings — Eino multi-provider agent ")
	form.SetTitleColor(tview.Styles.TitleColor)
	ui.SettingsForm = form

	hint := tview.NewTextView().SetDynamicColors(true)
	fmt.Fprint(hint, "[gray]Providers: openai · ark · ollama · claude · gemini · qwen · deepseek  |  workspace sandbox + tool policy  |  /agent for status[-]")

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hint, 1, 0, false).
		AddItem(form, 0, 1, true)
	if ui.Pages.HasPage("settings") {
		ui.Pages.RemovePage("settings")
	}
	ui.Pages.AddPage("settings", flex, true, false)
}

func (ui *TViewUI) showSettings() {
	ui.rebuildSettingsForm()
	ui.Pages.SwitchToPage("settings")
}
