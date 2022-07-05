package cmd

import (
	"log"
	"bytes"
	"os/exec"
)

func execShellReturnStrings(command string, args ...string) (string, string, error) {
	var outb, errb bytes.Buffer	
	k := exec.Command(command, args...)
	k.Stdout = &outb
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Println("Error executing command: %v", err)
	}
	return outb.String(), errb.String(), err
}
