package reports

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

type Model struct {
	Content  string
	ready    bool
	viewport viewport.Model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// allow key mapping for exit screen
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.Content)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m Model) headerView() string {
	title := titleStyle.Render("kubefirst platform")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CommandSummary receives a well-formatted buffer of bytes, and style it to the output.
func CommandSummary(cleanSummary bytes.Buffer) {

	style := getStyle()

	p := tea.NewProgram(
		Model{Content: style.Render(cleanSummary.String())},
	)

	if err := p.Start(); err != nil {
		log.Panicf("unable to load reports screen, error is: %s", err)
	}
}

func getStyle() lipgloss.Style {

	const kubefirstBoldPurple = "#5f00af"
	const kubefirstLightPurple = "#d0bae9"

	var style = lipgloss.NewStyle().
		Foreground(lipgloss.Color(kubefirstLightPurple)).
		Background(lipgloss.Color(kubefirstBoldPurple)).
		PaddingTop(2).
		PaddingBottom(2).
		PaddingLeft(2).
		PaddingRight(2).
		Width(75)
	return style
}

// StyleMessage receives a string and return a style string
func StyleMessage(message string) string {
	style := getStyle()
	return style.Render(message)
}

// StyleMessageBlackAndWhite receives a string and return a style black and white string
func StyleMessageBlackAndWhite(message string) string {
	style := getStyle()
	style.Background(lipgloss.Color("black"))
	style.Foreground(lipgloss.Color("white"))
	return style.Render(message)
}
