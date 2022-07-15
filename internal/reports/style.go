package reports

import (
	"github.com/charmbracelet/lipgloss"
)

// StyleMessage receives a string and return a style string
func StyleMessage(message string) string {

	const kubefirstBoldPurple = "#d0bae9"
	const kubefirstLightPurple = "#3c356c"

	var style = lipgloss.NewStyle().
		Foreground(lipgloss.Color(kubefirstBoldPurple)).
		Background(lipgloss.Color(kubefirstLightPurple)).
		PaddingTop(2).
		PaddingBottom(2).
		PaddingLeft(2).
		PaddingRight(2).
		Width(75)

	return style.Render(message)
}
