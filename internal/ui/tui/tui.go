package tui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	"github.com/xeyossr/anitr-cli/internal"
)

var ErrQuit = errors.New("quit requested")

// Stil ve renkler
var (
	highlightFgColor = "#e45cc0"
	normalFgColor    = "#aabbcc"
	highlightColor   = "#e45cc0"
	filterInputFg    = "#8bb27f"
	filterCursorFg   = "#c4b48b"
	inputPromptFg    = "#c4b48b"
	inputTextFg      = "#aabbcc"
	inputCursorFg    = "#c4b48b"
	selectionMark    = "▸ "

	pinkHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color(highlightColor))

	filterInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(filterInputFg)).
				Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(highlightFgColor)).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(normalFgColor)).
			Padding(0, 1)
)

// Hatayı pastel kırmızı kutu içinde gösterir ve programı sonlandırır
func ShowErrorBox(message string) {
	// Pastel kırmızı kutu stili
	errorBox := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff7f7f")). // pastel kırmızı
		Background(lipgloss.Color("#1c1c1c")). // koyu arka plan
		Bold(true).
		Padding(1, 2).
		Margin(1, 0).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#ff5f5f")) // kutu sınırı

	// Tam hata mesajını göster
	fullMessage := "❌ Hata: " + message

	// Kutunun içine render et
	fmt.Println(errorBox.Render(fullMessage))
}

// Tek seçimli list item
type listItem string

func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return string(i) }

// Çoklu seçim için checkbox item
type checkboxItem struct {
	TitleStr string
	Selected bool
}

func (i checkboxItem) Title() string       { return i.TitleStr }
func (i checkboxItem) Description() string { return "" }
func (i checkboxItem) FilterValue() string { return i.TitleStr }

// Render delegate
type slimDelegate struct {
	list.DefaultDelegate
}

func (d slimDelegate) Height() int  { return 1 }
func (d slimDelegate) Spacing() int { return 0 }
func (d slimDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	title := ""
	if li, ok := item.(listItem); ok {
		// Tek seçimli item
		title = li.Title()
	} else if ci, ok := item.(checkboxItem); ok {
		// Çoklu seçimli checkbox item
		check := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666")). // seçilmemiş gri
			Italic(true).                       // italik
			Render("[ ] ")

		if ci.Selected {
			check = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#e45cc0")). // seçili pembe
				Bold(true).                            // kalın
				Italic(true).                          // italik
				Render("[x] ")
		}

		title = check + ci.Title()
	} else {
		title = "???"
	}

	// Seçili item için prefix
	isSelected := index == m.Index()
	prefix := "  "
	if isSelected {
		prefix = selectionMark
	}

	// Başlığı truncate et
	availableWidth := m.Width() - lipgloss.Width(prefix) - 4
	displayTitle := truncate.StringWithTail(title, uint(availableWidth), "...")

	// Satır stili
	line := prefix + displayTitle
	if isSelected {
		line = highlightStyle.Render(line)
	} else {
		line = normalStyle.Render(line)
	}

	fmt.Fprint(w, line)
}

// Tek seçimli model
type SelectionListModel struct {
	list     list.Model
	quitting bool
	selected []string
	err      error
	width    int
}

func NewSelectionListModel(params internal.UiParams) SelectionListModel {
	items := make([]list.Item, len(*params.List))
	for i, v := range *params.List {
		items[i] = listItem(v)
	}

	const defaultWidth, defaultHeight = 48, 20
	l := list.New(items, slimDelegate{}, defaultWidth, defaultHeight)

	titleStyle := lipgloss.NewStyle().Align(lipgloss.Center).Bold(true)
	l.Title = titleStyle.Render(params.Label)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.FilterInput.Prompt = pinkHighlight.Render("🔍 Search: ")
	l.FilterInput.Placeholder = "Ara..."
	l.FilterInput.TextStyle = filterInputStyle
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(filterCursorFg))

	return SelectionListModel{list: l}
}

func (m SelectionListModel) Init() tea.Cmd { return nil }
func (m SelectionListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(listItem); ok {
				m.selected = []string{string(i)}
			}
			m.quitting = true
			return m, tea.Quit
		case "ctrl+c", "esc", "q":
			m.err = ErrQuit
			m.quitting = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SelectionListModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func SelectionList(params internal.UiParams) (string, error) {
	p := tea.NewProgram(NewSelectionListModel(params), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	model := m.(SelectionListModel)
	if model.err != nil {
		return "", model.err
	}
	if len(model.selected) > 0 {
		return model.selected[0], nil
	}
	return "", nil
}

// Çoklu seçimli model
type MultiSelectionListModel struct {
	list     list.Model
	selected []string
	quitting bool
	err      error
	width    int
}

func NewMultiSelectionListModel(params internal.UiParams) MultiSelectionListModel {
	items := make([]list.Item, len(*params.List))
	for i, v := range *params.List {
		items[i] = checkboxItem{TitleStr: v}
	}

	const defaultWidth, defaultHeight = 48, 20
	l := list.New(items, slimDelegate{}, defaultWidth, defaultHeight)
	titleStyle := lipgloss.NewStyle().Align(lipgloss.Center).Bold(true)
	l.Title = titleStyle.Render(params.Label)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.FilterInput.Prompt = pinkHighlight.Render("🔍 Search: ")
	l.FilterInput.Placeholder = "Ara..."
	l.FilterInput.TextStyle = filterInputStyle
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(filterCursorFg))

	return MultiSelectionListModel{list: l}
}

func (m MultiSelectionListModel) Init() tea.Cmd { return nil }
func (m MultiSelectionListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", " ":
			items := m.list.Items()
			if ci, ok := items[m.list.Index()].(checkboxItem); ok {
				ci.Selected = !ci.Selected
				items[m.list.Index()] = ci
				m.list.SetItems(items)

				if m.list.Index() == len(items)-1 {
					m.list.Select(0) // ilk iteme al
				} else {
					m.list.CursorDown() // normalde bir alta in
				}
			}
		case "enter":
			selected := []string{}
			for _, it := range m.list.Items() {
				if ci, ok := it.(checkboxItem); ok && ci.Selected {
					selected = append(selected, ci.Title())
				}
			}
			m.selected = selected
			if len(selected) == 0 {
				if ci, ok := m.list.SelectedItem().(checkboxItem); ok {
					m.selected = []string{ci.Title()}
				}
			}
			m.quitting = true
			return m, tea.Quit
		case "ctrl+c", "esc", "q":
			m.err = ErrQuit
			m.quitting = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MultiSelectionListModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func MultiSelectList(params internal.UiParams) ([]string, error) {
	p := tea.NewProgram(NewMultiSelectionListModel(params), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	model := m.(MultiSelectionListModel)
	if model.err != nil {
		return nil, model.err
	}
	return model.selected, nil
}

// Kullanıcıdan giriş almak
type InputFromUserModel struct {
	textInput textinput.Model
	err       error
	quitting  bool
}

func NewInputFromUserModel(params internal.UiParams) InputFromUserModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = "🔍 " + params.Label + ": "
	ti.CharLimit = 256
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(inputPromptFg)).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(inputTextFg))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(inputCursorFg))
	return InputFromUserModel{textInput: ti}
}

func (m InputFromUserModel) Init() tea.Cmd { return textinput.Blink }
func (m InputFromUserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(strings.TrimSpace(m.textInput.Value())) == 0 {
				m.err = errors.New("boş bırakılamaz")
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.err = ErrQuit
			m.quitting = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InputFromUserModel) View() string {
	if m.quitting {
		return ""
	}
	return lipgloss.NewStyle().Padding(0, 2).Render(m.textInput.View())
}

func InputFromUser(params internal.UiParams) (string, error) {
	p := tea.NewProgram(NewInputFromUserModel(params))
	m, err := p.Run()
	if err != nil {
		return "", err
	}
	model := m.(InputFromUserModel)
	if model.err != nil {
		return "", model.err
	}
	return model.textInput.Value(), nil
}
