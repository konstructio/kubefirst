/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

// podSessionOptions provides a struct to assign parameters to an exec session
type PodSessionOptions struct {
	Command    []string
	Namespace  string
	PodName    string
	Stdin      bool
	Stdout     bool
	Stderr     bool
	TtyEnabled bool
}
