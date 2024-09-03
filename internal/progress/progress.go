/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package progress

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/spf13/viper"
)

var Progress *tea.Program
var CanRunBubbleTea bool = true

//nolint:revive // will be removed after refactoring
func NewModel() progressModel {
	return progressModel{
		isProvisioned: false,
	}
}

// Bubbletea functions
func InitializeProgressTerminal() {
	Progress = tea.NewProgram(NewModel())
}

func DisableBubbleTeaExecution() {
	CanRunBubbleTea = false
}

func (m progressModel) Init() tea.Cmd {
	return nil
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}

	case headerMsg:
		m.header = msg.message
		return m, nil

	case addStep:
		m.nextStep = msg.message
		return m, nil

	case completeStep:
		m.completedSteps = append(m.completedSteps, msg.message)
		m.nextStep = ""
		return m, nil

	case errorMsg:
		m.error = msg.message
		return m, tea.Quit

	case successMsg:
		m.successMessage = msg.message + "\n\n"
		return m, tea.Quit

	case startProvision:
		m.clusterName = msg.clusterName
		return m, GetClusterInterval(m.clusterName)

	case ClusterProvisioningMsg:
		m.provisioningCluster = types.Cluster(msg)
		completedSteps, nextStep := BuildCompletedSteps(types.Cluster(msg))
		m.completedSteps = append(m.completedSteps, completedSteps...)
		m.nextStep = renderMessage(fmt.Sprintf(":dizzy: %s", nextStep))

		if m.provisioningCluster.Status == "error" {
			errorMessage := createErrorLog(m.provisioningCluster.LastCondition)
			m.error = errorMessage.message
			return m, tea.Quit
		}

		if m.provisioningCluster.Status == "provisioned" {
			m.isProvisioned = true
			m.nextStep = ""
			viper.Set("kubefirst-checks.cluster-install-complete", true)
			viper.WriteConfig()

			return m, AddSuccesMessage(m.provisioningCluster)
		}

		return m, GetClusterInterval(m.clusterName)

	default:
		return m, nil
	}
}

func (m progressModel) View() string {
	if !m.isProvisioned && m.successMessage == "" {
		index := 0

		if len(m.completedSteps) > 5 {
			index = len(m.completedSteps) - 5
		}

		completedSteps := ""
		for i := index; i < len(m.completedSteps); i++ {
			completedSteps += renderMessage(fmt.Sprintf(":white_check_mark: %s", m.completedSteps[i]))
		}

		if m.header != "" {
			return m.header + "\n\n" +
				completedSteps +
				m.nextStep + "\n\n" +
				m.error + "\n\n"
		}
	}

	return m.successMessage
}
