package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/reports"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// todo ask for user input to verify deletion?
// todo ask for user input to verify?
// cleanCmd removes all kubefirst resources locally for new execution.
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "removes all kubefirst resources locally for new execution",
	Long: `Kubefirst creates files, folders and cloud buckets during installation at your environment. This command removes and 
re-create all Kubefirst files. To destroy cloud resources you need to specify aditional flags (--destroy-buckets)`,
	Run: func(cmd *cobra.Command, args []string) {

		config := configs.ReadConfig()

		destroyBuckets, err := cmd.Flags().GetBool("destroy-buckets")
		if err != nil {
			log.Println(err)
		}
		destroyConfirm, err := cmd.Flags().GetBool("destroy-confirm")
		if err != nil {
			log.Println(err)
		}
		if destroyBuckets && !destroyConfirm {
			destroyConfirm = pkg.AskForConfirmation("This process will delete cloud buckets and all files inside, do you really want to proceed?")
			if !destroyConfirm {
				os.Exit(130)
			}
		}

		aws.DestroyBucketsInUse(destroyBuckets && destroyConfirm)

		// command line flags
		rmLogsFolder, err := cmd.Flags().GetBool("rm-logs")
		if err != nil {
			log.Panic(err)
		}

		// delete files and folders
		err = os.RemoveAll(config.K1FolderPath)
		if err != nil {
			log.Panicf("unable to delete %q folder, error is: %s", config.K1FolderPath, err)
		}

		err = os.Remove(config.KubefirstConfigFilePath)
		if err != nil {
			log.Panicf("unable to delete %q file, error is: ", err)
		}

		// remove logs folder if flag is enabled
		var logFolderLocation string
		if rmLogsFolder {
			logFolderLocation = viper.GetString("log.folder.location")
			err := os.RemoveAll(logFolderLocation)
			if err != nil {
				log.Panicf("unable to delete logs folder at %q", config.KubefirstLogPath)
			}
		}

		// re-create folder
		if err := os.Mkdir(fmt.Sprintf("%s", config.K1FolderPath), os.ModePerm); err != nil {
			log.Panicf("error: could not create directory %q - it must exist to continue. error is: %s", config.K1FolderPath, err)
		}

		// re-create base
		log.Printf("%q config file and %q folder were deleted and re-created", config.KubefirstConfigFilePath, config.K1FolderPath)

		var cleanSummary bytes.Buffer
		cleanSummary.WriteString(strings.Repeat("-", 70))
		cleanSummary.WriteString("\nclean summary:\n")
		cleanSummary.WriteString(strings.Repeat("-", 70))
		cleanSummary.WriteString("\n\nFiles and folders deleted:\n\n")

		cleanSummary.WriteString(fmt.Sprintf("   %q\n", config.KubefirstConfigFilePath))
		cleanSummary.WriteString(fmt.Sprintf("   %q\n", config.K1FolderPath))

		if rmLogsFolder {
			cleanSummary.WriteString(fmt.Sprintf("   %q\n", logFolderLocation))
		}

		cleanSummary.WriteString("\nRe-created empty folder: \n\n")
		cleanSummary.WriteString(fmt.Sprintf("   %q\n\n", config.K1FolderPath))

		cleanSummary.WriteString("Re-created empty config file: \n\n")
		cleanSummary.WriteString(fmt.Sprintf("   %q", config.KubefirstConfigFilePath))

		fmt.Println(reports.StyleMessage(cleanSummary.String()))
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().Bool("rm-logs", false, "remove logs folder")
	cleanCmd.Flags().Bool("destroy-buckets", false, "destroy buckets created by init cmd")
	cleanCmd.Flags().Bool("destroy-confirm", false, "confirm destroy operation (to be used during automation")
}
