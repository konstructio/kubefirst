/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provisionLogs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var quitStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

var ProvisionLogs *tea.Program

func NewModel() provisionLogsModel {
	return provisionLogsModel{}
}

// Bubbletea functions
func InitializeProvisionLogsTerminal() {
	ProvisionLogs = tea.NewProgram(NewModel())
}

func (m provisionLogsModel) Init() tea.Cmd {
	return nil
}

func (m provisionLogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}

	case logMessage:
		m.logs = append(m.logs, msg.message)
		return m, nil

	default:
		return m, nil
	}
}

func (m provisionLogsModel) View() string {
	logs := ""
	for i := 0; i < len(m.logs); i++ {
		logs = logs + m.logs[i] + "\n"
	}

	return logs + "\n" + quitStyle("ctrl+c to quit") + "\n"
}
