/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.

Emojis definition https://github.com/yuin/goldmark-emoji/blob/master/definition/github.go
*/
package progress

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/kubefirst/runtime/pkg/types"
	"github.com/spf13/viper"
)

const (
	padding  = 2
	maxWidth = 200
)

const debounceDuration = time.Second * 10

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	checkMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	dizzyIcon           = lipgloss.NewStyle().Foreground(lipgloss.NoColor{}).SetString("💫")
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
	StatusStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render
	spinnerStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

var CompletedStepsLabels = ProvisionSteps{
	install_tools_check:           "Installing tools",
	domain_liveness_check:         "Domain liveness check",
	kbot_setup_check:              "Kbot setup",
	git_init_check:                "Initializing Git",
	gitops_ready_check:            "Initializing gitops",
	git_terraform_apply_check:     "Git Terraform apply",
	gitops_pushed_check:           "Gitops repos pushed",
	cloud_terraform_apply_check:   "Cloud Terraform apply",
	cluster_secrets_created_check: "Creating cluster secrets",
	argocd_install_check:          "Installing Argo CD",
	argocd_initialize_check:       "Initializing Argo CD",
	vault_initialized_check:       "Initializing Vault",
	vault_terraform_apply_check:   "Vault Terraform apply",
	users_terraform_apply_check:   "Users Terraform apply",
}

var Progress *tea.Program

type CusterProvisioningMsg types.Cluster

type startProvision struct {
	clusterName string
}

type addMsg struct {
	message string
}

type successMsg struct {
	message string
}

type errorMsg struct {
	message string
}

// Terminal model
type progressModel struct {
	// Terminal
	logs           []addMsg
	error          string
	isProvisioning bool

	// Provisioning fields
	clusterName         string
	provisionProgress   float64
	provisioningCluster types.Cluster
	completedSteps      []string
	nextStep            string
	progress            progress.Model
	spinner             spinner.Model
	successMessage      string
}

func NewModel() progressModel {
	p := progress.New(
		progress.WithScaledGradient("#8851C8", "#81E2B4"),
	)
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	s.Style.Width(200)
	return progressModel{
		spinner:        s,
		progress:       p,
		isProvisioning: false,
	}
}

func DisplayLogHints() {
	logFile := viper.GetString("k1-paths.log-file")
	logInfo := fmt.Sprintf("```tail -f -n +1 %s```", logFile)
	Log("\n\n", "")
	Log("# Welcome to Kubefirst", "")
	Log("Follow your logs in a new terminal with:", "")
	Log(logInfo, "")
}

// Custom actions to update terminal
func renderMessage(message string) string {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONFile("styles.json"),
		glamour.WithEmoji(),
	)

	out, _ := r.Render(message)
	return out
}

func createLog(message string) addMsg {
	out := renderMessage(message)

	return addMsg{
		message: out,
	}
}

func createErrorLog(message string) errorMsg {
	out := renderMessage(fmt.Sprintf("`Error: %s`", message))

	return errorMsg{
		message: out,
	}
}

func Log(message string, logType string) {
	icon := ""

	if logType == "info" {
		icon = ":dizzy: "
	}

	renderedMessage := createLog(fmt.Sprintf("%s %s", icon, message))
	Progress.Send(renderedMessage)
}

func Error(message string) {
	renderedMessage := createErrorLog(message)
	Progress.Send(renderedMessage)
}

func Success(message string) {
	out := renderMessage(message)
	successMessage := successMsg{
		message: out,
	}
	Progress.Send(successMessage)
}

func StartProvisioning(clusterName string, estimatedTime int) {
	provisioningMessage := startProvision{
		clusterName: clusterName,
	}

	Progress.Send(provisioningMessage)
	Progress.Send(createLog("---"))
	Progress.Send(createLog(fmt.Sprintf("## **Estimated time: %s minutes**", strconv.Itoa(estimatedTime))))
}

// Bubbletea functions
func InitializeProgressTerminal() {
	Progress = tea.NewProgram(NewModel())
}

func (m progressModel) Init() tea.Cmd {
	return m.spinner.Tick
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

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case addMsg:
		m.logs = append(m.logs, msg)

		return m, nil

	case errorMsg:
		m.error = msg.message
		return m, tea.Quit

	case successMsg:
		m.successMessage = msg.message
		return m, nil

	case startProvision:
		m.clusterName = msg.clusterName
		m.isProvisioning = true

		return m, GetClusterInterval(m.clusterName)

	case CusterProvisioningMsg:
		m.provisioningCluster = types.Cluster(msg)
		completedSteps, nextStep := BuildCompletedSteps(types.Cluster(msg), m)
		m.completedSteps = completedSteps
		m.nextStep = fmt.Sprintf("%s %s", dizzyIcon, nextStep)

		m.provisionProgress = float64(len(m.completedSteps)) / float64(14)

		if m.provisionProgress == 1 && m.provisioningCluster.Status == "provisioning" {
			m.provisionProgress = 0.98
		}

		if m.provisioningCluster.Status == "error" {
			errorMessage := createErrorLog(m.provisioningCluster.LastCondition)
			m.error = errorMessage.message
			return m, tea.Quit
		}

		if m.provisioningCluster.Status != "provisioning" {
			m.provisionProgress = 100
			m.nextStep = ""
			return m, AddSuccesMessage(fmt.Sprintf(":tada: Cluster **%s** is now up and running.", m.clusterName))
		}

		return m, GetClusterInterval(m.clusterName)

	default:
		return m, nil
	}
}

func (m progressModel) View() string {
	pad := strings.Repeat(" ", padding)
	spin := m.spinner.View() + " "

	logs := ""
	for _, logValue := range m.logs {
		logs = logs + pad + logValue.message
	}

	provisioning := ""
	if m.isProvisioning {
		if m.provisioningCluster.Status == "" {
			provisioning = "\n" + pad + spin
		} else {

			completedSteps := ""
			for _, v := range m.completedSteps {
				completedSteps = completedSteps + pad + fmt.Sprintf("%s %s", checkMark, v) + "\n\n"
			}

			provisioning = "\n" +
				completedSteps +
				pad + m.nextStep + "\n\n" +
				pad + m.progress.ViewAs(m.provisionProgress) + "\n\n"
		}
	}

	return logs + provisioning + pad + m.successMessage + m.error + "\n\n" + helpStyle("Press ctrl + c to quit")
}

// Commands
func GetClusterInterval(clusterName string) tea.Cmd {
	return tea.Every(time.Second*10, func(t time.Time) tea.Msg {
		provisioningCluster, err := utilities.GetCluster(clusterName)

		if err != nil {

		}

		return CusterProvisioningMsg(provisioningCluster)
	})
}

func AddSuccesMessage(message string) tea.Cmd {
	return tea.Tick(0, func(t time.Time) tea.Msg {
		out := renderMessage(message)
		successMessage := successMsg{
			message: out,
		}

		return successMsg(successMessage)
	})
}

func BuildCompletedSteps(cluster types.Cluster, model progressModel) ([]string, string) {
	completedSteps := []string{}
	nextStep := ""
	if cluster.InstallToolsCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.install_tools_check)
		nextStep = CompletedStepsLabels.domain_liveness_check
	}
	if cluster.DomainLivenessCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.domain_liveness_check)
		nextStep = CompletedStepsLabels.kbot_setup_check
	}
	if cluster.KbotSetupCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.kbot_setup_check)
		nextStep = CompletedStepsLabels.git_init_check
	}
	if cluster.GitInitCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.git_init_check)
		nextStep = CompletedStepsLabels.gitops_ready_check
	}
	if cluster.GitopsReadyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitops_ready_check)
		nextStep = CompletedStepsLabels.git_terraform_apply_check
	}
	if cluster.GitTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.git_terraform_apply_check)
		nextStep = CompletedStepsLabels.gitops_pushed_check
	}
	if cluster.GitopsPushedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.gitops_pushed_check)
		nextStep = CompletedStepsLabels.cloud_terraform_apply_check
	}
	if cluster.CloudTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.cloud_terraform_apply_check)
		nextStep = CompletedStepsLabels.cluster_secrets_created_check
	}
	if cluster.ClusterSecretsCreatedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.cluster_secrets_created_check)
		nextStep = CompletedStepsLabels.argocd_install_check
	}
	if cluster.ArgoCDInstallCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.argocd_install_check)
		nextStep = CompletedStepsLabels.argocd_initialize_check
	}
	if cluster.ArgoCDInitializeCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.argocd_initialize_check)
		nextStep = CompletedStepsLabels.vault_initialized_check
	}
	if cluster.VaultInitializedCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.vault_initialized_check)
		nextStep = CompletedStepsLabels.vault_terraform_apply_check
	}
	if cluster.VaultTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.vault_terraform_apply_check)
		nextStep = CompletedStepsLabels.users_terraform_apply_check
	}
	if cluster.UsersTerraformApplyCheck {
		completedSteps = append(completedSteps, CompletedStepsLabels.users_terraform_apply_check)
		nextStep = "Wrapping up"
	}

	return completedSteps, nextStep
}
