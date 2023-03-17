package k3d

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

type recap struct {
	viewport viewport.Model
}

func presentRecap(gitProvider string, gitDestDescriptor string, gitOwner string) (*recap, error) {
	const width = 78

	vp := viewport.New(width, 10)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	content := fmt.Sprintf(`
# kubefirst platform settings
## Make sure these look correct before proceeding!

GIT PROVIDER: %s

%s %s: %s

	`, gitProvider, strings.ToUpper(gitProvider), strings.ToUpper(gitDestDescriptor), gitOwner)

	str, err := renderer.Render(content)
	if err != nil {
		return nil, err
	}

	vp.SetContent(str)

	return &recap{
		viewport: vp,
	}, nil
}

func (r recap) Init() tea.Cmd {
	return nil
}

func (r recap) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			fmt.Println("Install canceled at user request.")
			os.Exit(0)
			return r, tea.Quit
		case tea.KeyEnter:
			return r, tea.Quit
		default:
			var cmd tea.Cmd
			r.viewport, cmd = r.viewport.Update(msg)
			return r, cmd
		}
	default:
		return r, nil
	}
}

func (r recap) View() string {
	return r.viewport.View() + r.helpView()
}

func (r recap) helpView() string {
	return helpStyle("\n  ↑/↓: Navigate • Ctrl+C: Quit • enter: Proceed\n")
}
