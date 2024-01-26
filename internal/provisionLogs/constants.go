/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provisionLogs

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

const debounceDuration = time.Second * 10

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	checkMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
	StatusStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render
	spinnerStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)
