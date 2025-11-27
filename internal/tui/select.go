// Package tui provides interactive terminal UI components.
package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lepinkainen/hermes/internal/tmdb"
)

const (
	defaultListWidth  = 72
	defaultListHeight = 20
)

var runProgram = func(m tea.Model) (tea.Model, error) {
	return tea.NewProgram(m).Run()
}

// SelectionAction represents the user's action in the selection UI.
type SelectionAction int

const (
	// ActionNone indicates no action was taken.
	ActionNone SelectionAction = iota
	// ActionSelected indicates the user selected an item.
	ActionSelected
	// ActionSkipped indicates the user skipped the selection.
	ActionSkipped
	// ActionStopped indicates the user stopped processing entirely.
	ActionStopped
)

// SelectionResult holds the result of a TUI selection.
type SelectionResult struct {
	Action    SelectionAction
	Selection *tmdb.SearchResult
}

type tmdbItem struct {
	tmdb.SearchResult
}

func (i tmdbItem) Title() string {
	name := i.DisplayTitle()
	year := i.Year()
	return fmt.Sprintf("%s (%s)", strings.ToUpper(name), year)
}

func (i tmdbItem) FilterValue() string {
	return i.DisplayTitle()
}

func (i tmdbItem) Description() string {
	return i.Overview
}

type itemStyles struct {
	normal        lipgloss.Style
	selected      lipgloss.Style
	typeStyle     lipgloss.Style
	titleStyle    lipgloss.Style
	ratingStyle   lipgloss.Style
	metadataStyle lipgloss.Style
	overviewStyle lipgloss.Style
}

func newItemStyles() itemStyles {
	asciiBorder := lipgloss.Border{
		Top:         "-",
		Bottom:      "-",
		Left:        "|",
		Right:       "|",
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
	}

	container := lipgloss.NewStyle().
		Border(asciiBorder).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Foreground(lipgloss.Color("252"))

	selected := container.Copy().
		BorderForeground(lipgloss.Color("214")).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("237"))

	return itemStyles{
		normal:   container,
		selected: selected,
		typeStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("110")),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("254")),
		ratingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("178")),
		metadataStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("247")).
			Faint(true),
		overviewStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")),
	}
}

type tmdbDelegate struct {
	styles itemStyles
}

func newDelegate() tmdbDelegate {
	return tmdbDelegate{styles: newItemStyles()}
}

func (d tmdbDelegate) Height() int                         { return 5 }
func (d tmdbDelegate) Spacing() int                        { return 1 }
func (d tmdbDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d tmdbDelegate) Render(w io.Writer, m list.Model, idx int, item list.Item) {
	result, ok := item.(tmdbItem)
	if !ok {
		return
	}

	typeLabel := result.MediaType
	title := result.DisplayTitle()
	year := result.Year()
	rating := result.VoteAverage
	overview := result.Overview
	if len(overview) > 0 {
		overview = truncate(overview, m.Width()-4)
	}

	typeLine := d.styles.typeStyle.Render(fmt.Sprintf("[%s]", strings.ToUpper(typeLabel)))
	metadataLine := d.styles.metadataStyle.Render(formatMetadata(result.SearchResult, m.Width()-4))
	titleLine := d.styles.titleStyle.Render(fmt.Sprintf("%s (%s)", strings.ToUpper(title), year))
	ratingLine := d.styles.ratingStyle.Render(fmt.Sprintf("%.1f/10", rating))
	overviewLine := d.styles.overviewStyle.Render(overview)

	// Build content with metadata line after type
	content := lipgloss.JoinVertical(lipgloss.Left, typeLine, metadataLine, titleLine, ratingLine, overviewLine)

	container := d.styles.normal
	if idx == m.Index() {
		container = d.styles.selected
	}
	_, _ = fmt.Fprint(w, container.Render(content))
}

type model struct {
	list        list.Model
	searchTitle string
	sourceURL   string
	result      SelectionResult
}

func newModel(title string, items []tmdbItem, sourceURL string) *model {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := newDelegate()
	l := list.New(listItems, delegate, defaultListWidth, defaultListHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle()

	return &model{
		list:        l,
		searchTitle: title,
		sourceURL:   sourceURL,
		result: SelectionResult{
			Action: ActionNone,
		},
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if selected, ok := m.list.SelectedItem().(tmdbItem); ok {
				result := selected.SearchResult
				m.result = SelectionResult{
					Action:    ActionSelected,
					Selection: &result,
				}
				return m, tea.Quit
			}
		case "s":
			m.result = SelectionResult{Action: ActionSkipped}
			return m, tea.Quit
		case "ctrl+c", "q":
			m.result = SelectionResult{Action: ActionStopped}
			return m, tea.Quit
		case "esc":
			m.result = SelectionResult{Action: ActionSkipped}
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

func (m *model) View() string {
	header := headerStyle.Render(fmt.Sprintf("Multiple results found for: %s", m.searchTitle))

	var elements []string
	elements = append(elements, header)

	// Add source URL if provided
	if m.sourceURL != "" {
		sourceStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("110")).
			Faint(true)
		sourceURL := sourceStyle.Render(fmt.Sprintf("Source: %s", m.sourceURL))
		elements = append(elements, sourceURL)
	}

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

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")).
			MarginBottom(1)

	skipButtonStyle = lipgloss.NewStyle().
			MarginTop(1).
			Padding(0, 2).
			Background(lipgloss.Color("178")).
			Foreground(lipgloss.Color("0")).
			Bold(true)

	stopButtonStyle = lipgloss.NewStyle().
			MarginTop(1).
			Padding(0, 2).
			Background(lipgloss.Color("161")).
			Foreground(lipgloss.Color("230")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			MarginTop(1).
			Foreground(lipgloss.Color("244"))
)

// SelectOptions holds optional parameters for the Select function
type SelectOptions struct {
	// SourceURL is an optional URL to display (e.g., Letterboxd URL)
	// to help users verify which exact item they're selecting
	SourceURL string
}

// Select presents an interactive selection UI for TMDB search results.
// The caller is responsible for filtering results before calling this function.
func Select(title string, results []tmdb.SearchResult, opts *SelectOptions) (SelectionResult, error) {
	if len(results) == 0 {
		return SelectionResult{Action: ActionSkipped}, nil
	}

	items := make([]tmdbItem, len(results))
	for i, result := range results {
		items[i] = tmdbItem{SearchResult: result}
	}

	sourceURL := ""
	if opts != nil {
		sourceURL = opts.SourceURL
	}

	m := newModel(title, items, sourceURL)
	finalModel, err := runProgram(m)
	if err != nil {
		return SelectionResult{}, err
	}

	if typed, ok := finalModel.(*model); ok {
		return typed.result, nil
	}

	return SelectionResult{}, fmt.Errorf("unexpected program result")
}

func truncate(value string, width int) string {
	value = strings.Join(strings.Fields(value), " ")
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}

// formatMetadata creates the metadata line with runtime, language, vote count, and popularity
func formatMetadata(result tmdb.SearchResult, availableWidth int) string {
	var parts []string

	// Runtime (if available)
	if result.Runtime > 0 {
		parts = append(parts, fmt.Sprintf("%dm", result.Runtime))
	}

	// Language (if available)
	if result.OriginalLang != "" {
		lang := strings.ToUpper(result.OriginalLang)
		parts = append(parts, lang)
	}

	// Vote count (if available)
	if result.VoteCount > 0 {
		votes := formatVoteCount(result.VoteCount)
		parts = append(parts, votes)
	}

	// Popularity (if available)
	if result.Popularity > 0 {
		pop := fmt.Sprintf("📊%.1f", result.Popularity)
		parts = append(parts, pop)
	}

	if len(parts) == 0 {
		return "No metadata available"
	}

	metadata := strings.Join(parts, " | ")
	if availableWidth > 0 && len(metadata) > availableWidth {
		metadata = truncate(metadata, availableWidth)
	}

	return metadata
}

// formatVoteCount formats vote count in a compact way
func formatVoteCount(count int) string {
	if count >= 1000 {
		return fmt.Sprintf("%.1fK votes", float64(count)/1000)
	}
	return fmt.Sprintf("%d votes", count)
}

func clamp(defaultValue, available, minimum int) int {
	width := defaultValue
	if available > 0 && available < defaultValue {
		width = available
	}
	if width < minimum {
		width = minimum
	}
	return width
}
