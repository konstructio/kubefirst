package cmd

import (
	"bytes"
	"log"
	"os/exec"
)

func execShellReturnStrings(command string, args ...string) (string, string, error) {
	var outb, errb bytes.Buffer
	k := exec.Command(command, args...)
	k.Stdout = &outb
	k.Stderr = &errb
	log.Printf("Command Execution:  %s", command)
	log.Printf("Command Execution STDOUT: %s", outb.String())
	log.Printf("Command Execution STDERR: %s", errb.String())
	err := k.Run()
	if err != nil {
		log.Panic("Error executing command: ", err)
	}
	return outb.String(), errb.String(), err
}
