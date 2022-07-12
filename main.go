/*
Copyright © 2022 Kubefirst Inc. devops@kubefirst.com

*/
package main

import (
	"fmt"
	"github.com/kubefirst/nebulous/cmd"
	"log"
	"os"
	"time"
)

func main() {
	now := time.Now()
	epoch := now.Unix()
	logsdir := "log"
	os.Mkdir(logsdir, 0700)
	logfile := fmt.Sprintf("./%s/log_%d.log", logsdir, epoch)
	fmt.Printf("Result will be logged at: %s \n", logfile)
	file, err := openLogFile(logfile)
	defer file.Close()
	log.SetOutput(file)
	if err != nil {
		log.Fatal(err)
	}
	log.SetPrefix("LOG: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)

	cmd.Execute()
}

func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}
