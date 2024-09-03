/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.

Emojis definition https://github.com/yuin/goldmark-emoji/blob/master/definition/github.go
Color definition https://www.ditig.com/256-colors-cheat-sheet
*/
package provisionLogs

import (
	"fmt"
	"log"

	"github.com/charmbracelet/glamour"
	"github.com/konstructio/kubefirst/internal/progress"
)

func renderMessage(message string) string {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(progress.StyleConfig),
		glamour.WithEmoji(),
	)

	out, err := r.Render(message)
	if err != nil {
		s := fmt.Errorf("rendering message failed: %w", err)
		log.Println(s)
		return s.Error()
	}
	return out
}
