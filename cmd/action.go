package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Panic(err)
		}

		metaphorDir := fmt.Sprintf("%s/.k1/metaphor", homeDir)
		// os.Mkdir(metaphorDir, 0700)
		// init
		// _, err = git.PlainInit(metaphorDir, false)
		// if err != nil {
		// 	return err
		// }

		metaphorRepo, err := git.PlainOpen(metaphorDir)

		w, _ := metaphorRepo.Worktree()
		branchName := plumbing.NewBranchReferenceName("main")
		headRef, err := metaphorRepo.Head()
		if err != nil {
			return fmt.Errorf("Error Setting reference: %s", err)
		}

		ref := plumbing.NewHashReference(branchName, headRef.Hash())
		err = metaphorRepo.Storer.SetReference(ref)
		if err != nil {
			return fmt.Errorf("error Storing reference: %s", err)
		}

		err = w.Checkout(&git.CheckoutOptions{Branch: ref.Name()})
		if err != nil {
			return fmt.Errorf("error checking out main: %s", err)
		}

		// metaphorRepo, err = gitClient.SetRefToMainBranch(metaphorRepo)
		// if err != nil {
		// 	return err
		// }

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
