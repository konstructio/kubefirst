/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provisionLogs

// Terminal model
type provisionLogsModel struct {
	logs []string
}

// Bubbletea messages
type logMessage struct {
	message string
}
