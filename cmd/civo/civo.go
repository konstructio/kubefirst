package civo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runCivo(cmd *cobra.Command, args []string) error {

	log.Println("civo run command now ")
	var userInput string
	printConfirmationScreen()
	go counter()
	fmt.Println("to proceed, type 'yes' any other answer will exit")
	fmt.Scanln(&userInput)
	fmt.Println("proceeding with cluster create")
	os.Exit(1)

	// fmt.Fprintf(w, "%s to open %s in your browser... ", cs.Bold("Press Enter"), oauthHost)
	// https://github.com/cli/cli/blob/trunk/internal/authflow/flow.go#L37
	// to do consider if we can credit github on theirs

	return nil
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}

// todo move below functions? pkg? rename?
func counter() {
	i := 0
	for {
		time.Sleep(time.Second * 1)
		i++
	}
}

func printConfirmationScreen() {
	var createKubefirstSummary bytes.Buffer
	createKubefirstSummary.WriteString(strings.Repeat("-", 70))
	createKubefirstSummary.WriteString("\nCreate Kubefirst Cluster?\n")
	createKubefirstSummary.WriteString(strings.Repeat("-", 70))
	createKubefirstSummary.WriteString("\nCivo Details:\n\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("DNS:    %s\n", viper.GetString("civo.dns")))
	createKubefirstSummary.WriteString(fmt.Sprintf("Region: %s\n", viper.GetString("civo.region")))
	createKubefirstSummary.WriteString("\nGithub Organization Details:\n\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("Organization: %s\n", viper.GetString("github.owner")))
	createKubefirstSummary.WriteString(fmt.Sprintf("User:         %s\n", viper.GetString("github.user")))
	createKubefirstSummary.WriteString("New Github Repository URL's:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor-frontend.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("github.repo.metaphor-go.url")))

	createKubefirstSummary.WriteString("\nTemplate Repository URL's:\n")
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.gitops.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.gitops.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor-frontend.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor-frontend.branch")))
	createKubefirstSummary.WriteString(fmt.Sprintf("  %s\n", viper.GetString("template-repo.metaphor-go.url")))
	createKubefirstSummary.WriteString(fmt.Sprintf("    branch:  %s\n", viper.GetString("template-repo.metaphor-go.branch")))

	fmt.Println(reports.StyleMessage(createKubefirstSummary.String()))
}
