package pkg

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"strings"
)

// ExecShellReturnStrings Exec shell actions returning a string for use by the caller.
func ExecShellReturnStrings(command string, args ...string) (string, string, error) {
	var outb, errb bytes.Buffer
	k := exec.Command(command, args...)
	//  log.Info()().Msg()("Command:", k.String()) //Do not remove this line used for some debugging, will be wrapped by debug log some day.
	k.Stdout = &outb
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Printf("Error executing command: %v\n", err)
	}
	log.Printf("Command Execution: %s\n", command)
	log.Printf("OUT: %s\n", outb.String())
	log.Printf("ERR: %s\n", errb.String())
	return outb.String(), errb.String(), err
}

// ExecShellWithVars Exec shell actions supporting:
//   - On-the-fly logging of result
//   - Map of Vars loaded
func ExecShellWithVars(osvars map[string]string, command string, args ...string) error {

	log.Printf("INFO: Running %s", command)
	for k, v := range osvars {
		os.Setenv(k, v)
		suppressedValue := strings.Repeat("*", len(v))
		log.Printf(" export %s = %s", k, suppressedValue)
	}
	cmd := exec.Command(command, args...)
	cmdReaderOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("error: %s failed creating out pipe for: %v", command, err))
		return err
	}
	cmdReaderErr, err := cmd.StderrPipe()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("error: %s failed creating out pipe for:  %v", command, err))
		return err
	}

	scannerOut := bufio.NewScanner(cmdReaderOut)
	stdOut := make(chan string)
	go reader(scannerOut, stdOut)
	doneOut := make(chan bool)

	scannerErr := bufio.NewScanner(cmdReaderErr)
	stdErr := make(chan string)
	go reader(scannerErr, stdErr)
	doneErr := make(chan bool)
	go func() {
		for msg := range stdOut {
			log.Info().Msgf("OUT: %s", msg)
		}
		doneOut <- true
	}()
	go func() {
		for msg := range stdErr {
			log.Info().Msgf("ERR: %s", msg)
		}
		doneErr <- true
	}()

	err = cmd.Run()
	if err != nil {
		log.Info().Msg(fmt.Sprintf("error: %s failed %v", command, err))
		return err
	} else {
		close(stdOut)
		close(stdErr)
	}
	<-doneOut
	<-doneErr
	return nil

}

// Not meant to be exported, for internal use only.
func reader(scanner *bufio.Scanner, out chan string) {
	defer func() {
		if r := recover(); r != nil {
			log.Info().Msgf("Error processing logs from command. Error: %s", r)
		}
	}()
	for scanner.Scan() {
		out <- scanner.Text()
	}
}
