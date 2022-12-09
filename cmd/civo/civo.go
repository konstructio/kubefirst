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

	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/progressPrinter"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/internal/wrappers"
	"github.com/kubefirst/kubefirst/pkg"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runCivo(cmd *cobra.Command, args []string) error {

	config := configs.ReadConfig()
	log.Println("runCivo command is starting ")
	// var userInput string
	// printConfirmationScreen()
	// go counter()
	// fmt.Println("to proceed, type 'yes' any other answer will exit")
	// fmt.Scanln(&userInput)
	// fmt.Println("proceeding with cluster create")

	// fmt.Fprintf(w, "%s to open %s in your browser... ", cs.Bold("Press Enter"), oauthHost)
	// https://github.com/cli/cli/blob/trunk/internal/authflow/flow.go#L37
	// to do consider if we can credit github on theirs

	printConfirmationScreen()
	fmt.Println("proceeding with cluster create")

	civoDnsName := viper.GetString("civo.dns")
	gitopsTemplateBranch := viper.GetString("template-repo.gitops.branch")
	gitopsTemplateUrl := viper.GetString("template-repo.gitops.url")
	silentMode := false // todo fix
	dryRun := false     // todo fix

	//* emit cluster install started
	if useTelemetryFlag {
		if err := wrappers.SendSegmentIoTelemetry(civoDnsName, pkg.MetricMgmtClusterInstallStarted); err != nil {
			log.Println(err)
		}
	}

	//* download dependencies `$HOME/.k1/tools`
	if !viper.GetBool("kubefirst.dependency-download.complete") {
		log.Println("installing kubefirst dependencies")

		err := downloadManager.DownloadTools(config)
		if err != nil {
			return err
		}

		log.Println("download dependencies `$HOME/.k1/tools` complete")
		viper.Set("kubefirst.dependency-download.complete", true)
		viper.WriteConfig()
	} else {
		log.Println("already completed download of dependencies to `$HOME/.k1/tools` - continuing")
	}

	//* git clone and detokenize the gitops repository
	if !viper.GetBool("template-repo.gitops.cloned") {

		//* step 1
		pkg.InformUser("generating your new gitops repository", silentMode)
		gitClient.CloneBranchSetMain(gitopsTemplateUrl, config.GitOpsRepoPath, gitopsTemplateBranch)
		log.Println("gitops repository creation complete")

		//* step 2
		// adjust content in gitops repository
		opt := cp.Options{
			Skip: func(src string) (bool, error) {
				if strings.HasSuffix(src, ".git") {
					return true, nil
				} else if strings.Index(src, "/.terraform") > 0 {
					return true, nil
				}
				//Add more stuff to be ignored here
				return false, nil

			},
		}

		// clear out the root of `gitops-template` once we move
		// all the content we only remove the different root folders
		os.RemoveAll(config.GitOpsRepoPath + "/components")
		os.RemoveAll(config.GitOpsRepoPath + "/localhost")
		os.RemoveAll(config.GitOpsRepoPath + "/registry")
		os.RemoveAll(config.GitOpsRepoPath + "/validation")
		os.RemoveAll(config.GitOpsRepoPath + "/terraform")
		os.RemoveAll(config.GitOpsRepoPath + "/.gitignore")
		os.RemoveAll(config.GitOpsRepoPath + "/LICENSE")
		os.RemoveAll(config.GitOpsRepoPath + "/README.md")
		os.RemoveAll(config.GitOpsRepoPath + "/atlantis.yaml")
		os.RemoveAll(config.GitOpsRepoPath + "/logo.png")

		driverContent := fmt.Sprintf("%s/%s-%s", config.GitOpsRepoPath, viper.GetString("cloud-provider"), viper.GetString("git-provider"))
		err := cp.Copy(driverContent, config.GitOpsRepoPath, opt)
		if err != nil {
			log.Println("Error populating gitops with local setup:", err)
			return err
		}
		os.RemoveAll(driverContent)

		//* step 3 -- gitClient.CommitAndPush -- warning origin is github
		pkg.DetokenizeCivoGithub(config.GitOpsRepoPath)

		//* step 4 add a new remote of the github user who's token we have
		repo, err := git.PlainOpen(config.GitOpsRepoPath)
		if err != nil {
			log.Print("error opening repo at:", config.GitOpsRepoPath)
		}
		destinationGitopsRepoURL := viper.GetString("github.repo.gitops.giturl")
		log.Printf("git remote add github %s", destinationGitopsRepoURL)
		_, err = repo.CreateRemote(&gitConfig.RemoteConfig{
			Name: "github",
			URLs: []string{destinationGitopsRepoURL},
		})
		if err != nil {
			log.Panicf("Error creating remote %s at: %s - %s", viper.GetString("git-provider"), destinationGitopsRepoURL, err)
		}

		//* step 5 commit newly detokenized content
		w, _ := repo.Worktree()

		log.Printf("committing detokenized %s content", "gitops")
		status, err := w.Status()
		if err != nil {
			log.Println("error getting worktree status", err)
		}

		for file, _ := range status {
			_, err = w.Add(file)
			if err != nil {
				log.Println("error getting worktree status", err)
			}
		}
		w.Commit(fmt.Sprintf("[ci skip] committing initial detokenized %s content", destinationGitopsRepoURL), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "kubefirst-bot",
				Email: "kubefirst-bot@kubefirst.com",
				When:  time.Now(),
			},
		}) // todo emit init telemetry end

		viper.Set("template-repo.gitops.cloned", true)
		viper.WriteConfig()
	} else {
		log.Println("already completed gitops repo generation - continuing")
	}

	log.Println("yep - done init and appply")
	os.Exit(1)

	// todo move this above the cloud, its fast and easy
	executionControl := viper.GetBool("terraform.github.apply.complete")
	// create github teams in the org and gitops repo
	if !executionControl {
		pkg.InformUser("Creating github resources with terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
		terraform.InitApplyAutoApprove(dryRun, tfEntrypoint)

		pkg.InformUser(fmt.Sprintf("Created gitops Repo in github.com/%s", viper.GetString("github.owner")), silentMode)
		progressPrinter.IncrementTracker("step-github", 1)
	} else {
		log.Println("already created github terraform resources")
	}

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
