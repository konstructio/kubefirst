package pkg

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"bufio"
)

func ExecShellReturnStrings(command string, args ...string) (string, string, error) {
	var outb, errb bytes.Buffer
	k := exec.Command(command, args...)
	k.Stdout = &outb
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Println("Error executing command: %v", err)
	}
	log.Println("Commad Execution: %s", command)
	log.Println("Commad Execution STDOUT: %s", outb.String())
	log.Println("Commad Execution STDERR: %s", errb.String())
	return outb.String(), errb.String(), err
}

func ExecShellReturnStringsWithVars(osvars map[string]string, command string, args ...string) (string, string, error) {
	
	log.Printf("INFO: Running %s",command)
	for k, v := range osvars {		
		os.Setenv(k, v)
		log.Printf(" export %s = %s", k, v)
	}
	var outb, errb bytes.Buffer
	k := exec.Command(command, args...)
	k.Stdout = &outb
	k.Stderr = &errb
	err := k.Run()
	if err != nil {
		log.Println("Error executing command: %v", err)
	}
	log.Println("Commad Execution: %s", command)
	log.Println("Commad Execution STDOUT: %s", outb.String())
	log.Println("Commad Execution STDERR: %s", errb.String())
	return outb.String(), errb.String(), err
}

// Exec shell actions supporting:
// - On-the-fly logging of result
// - Map of Vars loaded
func ExecShellWithVars(osvars map[string]string, command string, args ...string) (error) {
	
	
	log.Printf("INFO: Running %s",command)
	for k, v := range osvars {		
		os.Setenv(k, v)
		log.Printf(" export %s = %s", k, v)
	}
	cmd := exec.Command(command, args...)
	cmdReaderOut, _ := cmd.StdoutPipe()
	cmdReaderErr, _ := cmd.StderrPipe()

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
			log.Println("OUT: ",msg)
		}
		doneOut <- true
	}()
	go func() {
		for msg := range stdErr {
			log.Println("ERR: ",msg)
		}
		doneErr <- true
	}()

	errApply := cmd.Run()
	if errApply != nil {
		log.Println(fmt.Sprintf("error: %s init failed %v",command, errApply))
		return errApply
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
    for scanner.Scan() {
        out <- scanner.Text()
    }
}
