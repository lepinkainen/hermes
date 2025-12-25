package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SteamSearchResult represents a search result from Steam.
// Defined here to be used by both tui and enrichment packages.
type SteamSearchResult struct {
	AppID       int
	Name        string
	HeaderImage string
}

// SteamSelectionResult holds the result of a Steam TUI selection.
type SteamSelectionResult struct {
	Action         SelectionAction
	SteamSelection *SteamSearchResult
}

type steamItem struct {
	SteamSearchResult
}

func (i steamItem) Title() string {
	return strings.ToUpper(i.Name)
}

func (i steamItem) FilterValue() string {
	return i.Name
}

func (i steamItem) Description() string {
	return fmt.Sprintf("AppID: %d", i.AppID)
}

type steamDelegate struct {
	styles itemStyles
}

func newSteamDelegate() steamDelegate {
	return steamDelegate{styles: newItemStyles()}
}

func (d steamDelegate) Height() int                         { return 3 }
func (d steamDelegate) Spacing() int                        { return 1 }
func (d steamDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d steamDelegate) Render(w io.Writer, m list.Model, idx int, item list.Item) {
	result, ok := item.(steamItem)
	if !ok {
		return
	}

	titleLine := d.styles.titleStyle.Render(strings.ToUpper(result.Name))
	appIDLine := d.styles.metadataStyle.Render(fmt.Sprintf("Steam AppID: %d", result.AppID))

	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, appIDLine)

	container := d.styles.normal
	if idx == m.Index() {
		container = d.styles.selected
	}
	_, _ = fmt.Fprint(w, container.Render(content))
}

type steamModel struct {
	list        list.Model
	searchTitle string
	result      SteamSelectionResult
}

func newSteamModel(title string, items []steamItem) *steamModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := newSteamDelegate()
	l := list.New(listItems, delegate, defaultListWidth, defaultListHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle()

	return &steamModel{
		list:        l,
		searchTitle: title,
		result: SteamSelectionResult{
			Action: ActionNone,
		},
	}
}

func (m *steamModel) Init() tea.Cmd { return nil }

func (m *steamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if selected, ok := m.list.SelectedItem().(steamItem); ok {
				result := selected.SteamSearchResult
				m.result = SteamSelectionResult{
					Action:         ActionSelected,
					SteamSelection: &result,
				}
				return m, tea.Quit
			}
		case "s":
			m.result = SteamSelectionResult{Action: ActionSkipped}
			return m, tea.Quit
		case "ctrl+c", "q":
			m.result = SteamSelectionResult{Action: ActionStopped}
			return m, tea.Quit
		case "esc":
			m.result = SteamSelectionResult{Action: ActionSkipped}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		width := clamp(defaultListWidth, msg.Width-4, 40)
		height := clamp(defaultListHeight, msg.Height-6, 5)
		m.list.SetSize(width, height)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *steamModel) View() string {
	header := headerStyle.Render(fmt.Sprintf("Multiple Steam results for: %s", m.searchTitle))

	var elements []string
	elements = append(elements, header)
	elements = append(elements, m.list.View())

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		skipButtonStyle.Render(" Skip "),
		lipgloss.NewStyle().Padding(0, 2).Render(""),
		stopButtonStyle.Render(" Stop Processing "),
	)
	elements = append(elements, buttons)

	help := helpStyle.Render("Up/Down navigate | Enter select | s skip | q stop")
	elements = append(elements, help)

	return lipgloss.JoinVertical(lipgloss.Left, elements...)
}

// SteamSelectOptions holds optional parameters for the SelectSteam function.
type SteamSelectOptions struct {
	// SourceURL is an optional URL to display
	SourceURL string
}

// SelectSteam presents an interactive selection UI for Steam search results.
func SelectSteam(title string, results []SteamSearchResult, _ *SteamSelectOptions) (SteamSelectionResult, error) {
	if len(results) == 0 {
		return SteamSelectionResult{Action: ActionSkipped}, nil
	}

	items := make([]steamItem, len(results))
	for i, result := range results {
		items[i] = steamItem{SteamSearchResult: result}
	}

	m := newSteamModel(title, items)
	finalModel, err := runProgram(m)
	if err != nil {
		return SteamSelectionResult{}, err
	}

	if typed, ok := finalModel.(*steamModel); ok {
		return typed.result, nil
	}

	return SteamSelectionResult{}, fmt.Errorf("unexpected program result")
}
