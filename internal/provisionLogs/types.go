/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package provisionLogs //nolint:revive // allowed during refactoring

// Terminal model
type provisionLogsModel struct {
	logs []string
}

// Bubbletea messages
type logMessage struct {
	message string
}
