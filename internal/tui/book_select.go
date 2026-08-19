package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BookSearchResult represents a search result from OpenLibrary.
// Defined here to be used by both tui and enrichment packages.
type BookSearchResult struct {
	Title            string
	Authors          []string
	FirstPublishYear int
	EditionCount     int
	CoverID          int
	ISBN13           string
	ISBN             string
}

// BookSelectionResult holds the result of a book TUI selection.
type BookSelectionResult struct {
	Action        SelectionAction
	BookSelection *BookSearchResult
}

type bookItem struct {
	BookSearchResult
}

func (i bookItem) Title() string {
	return strings.ToUpper(i.BookSearchResult.Title)
}

func (i bookItem) FilterValue() string {
	return i.BookSearchResult.Title
}

func (i bookItem) Description() string {
	var parts []string
	if len(i.Authors) > 0 {
		parts = append(parts, strings.Join(i.Authors, ", "))
	}
	if i.FirstPublishYear > 0 {
		parts = append(parts, fmt.Sprintf("%d", i.FirstPublishYear))
	}
	if i.EditionCount > 0 {
		parts = append(parts, fmt.Sprintf("%d editions", i.EditionCount))
	}
	return strings.Join(parts, " | ")
}

type bookDelegate struct {
	styles itemStyles
}

func newBookDelegate() bookDelegate {
	return bookDelegate{styles: newItemStyles()}
}

func (d bookDelegate) Height() int                         { return 3 }
func (d bookDelegate) Spacing() int                        { return 1 }
func (d bookDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d bookDelegate) Render(w io.Writer, m list.Model, idx int, item list.Item) {
	result, ok := item.(bookItem)
	if !ok {
		return
	}

	titleLine := d.styles.titleStyle.Render(strings.ToUpper(result.BookSearchResult.Title))
	metadataLine := d.styles.metadataStyle.Render(result.Description())

	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, metadataLine)

	container := d.styles.normal
	if idx == m.Index() {
		container = d.styles.selected
	}
	_, _ = fmt.Fprint(w, container.Render(content))
}

type bookModel struct {
	list        list.Model
	searchTitle string
	result      BookSelectionResult
}

func newBookModel(title string, items []bookItem) *bookModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := newBookDelegate()
	l := list.New(listItems, delegate, defaultListWidth, defaultListHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle()

	return &bookModel{
		list:        l,
		searchTitle: title,
		result: BookSelectionResult{
			Action: ActionNone,
		},
	}
}

func (m *bookModel) Init() tea.Cmd { return nil }

func (m *bookModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if selected, ok := m.list.SelectedItem().(bookItem); ok {
				result := selected.BookSearchResult
				m.result = BookSelectionResult{
					Action:        ActionSelected,
					BookSelection: &result,
				}
				return m, tea.Quit
			}
		case "s":
			m.result = BookSelectionResult{Action: ActionSkipped}
			return m, tea.Quit
		case "ctrl+c", "q":
			m.result = BookSelectionResult{Action: ActionStopped}
			return m, tea.Quit
		case "esc":
			m.result = BookSelectionResult{Action: ActionSkipped}
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

func (m *bookModel) View() string {
	header := headerStyle.Render(fmt.Sprintf("Multiple OpenLibrary results for: %s", m.searchTitle))

	elements := []string{header, m.list.View()}

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

// BookSelectOptions holds optional parameters for the SelectBook function.
type BookSelectOptions struct {
	// SourceURL is an optional URL to display
	SourceURL string
}

// SelectBook presents an interactive selection UI for OpenLibrary search results.
func SelectBook(title string, results []BookSearchResult, _ *BookSelectOptions) (BookSelectionResult, error) {
	if len(results) == 0 {
		return BookSelectionResult{Action: ActionSkipped}, nil
	}

	items := make([]bookItem, len(results))
	for i, result := range results {
		items[i] = bookItem{BookSearchResult: result}
	}

	m := newBookModel(title, items)
	finalModel, err := runProgram(m)
	if err != nil {
		return BookSelectionResult{}, err
	}

	if typed, ok := finalModel.(*bookModel); ok {
		return typed.result, nil
	}

	return BookSelectionResult{}, fmt.Errorf("unexpected program result")
}
