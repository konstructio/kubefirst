package teawrapper

import (
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

// WrapBubbleTea wraps the main user's function with the progress terminal
// so its errors can be handled while still allowing the main command function
// to handle additional, outside-of-the-progress-terminal errors.
func WrapBubbleTea(fn func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Initialize the progress terminal
		progress.InitializeProgressTerminal()

		// Run the progress terminal, and listen for errors
		chTeaError := make(chan error, 1)
		go func() {
			_, err := progress.Progress.Run()
			chTeaError <- err
		}()

		// Run the main user's function
		if err := fn(cmd, args); err != nil {
			// print the error and send it to the progress terminal, but don't
			// return here, we want the error to be handled by bubbletea
			progress.Error(err.Error())
		}

		// Quit the progress terminal if the execution succeeded
		// so it can stop `progress.Run()`
		progress.Progress.Quit()

		// Receive the error from the progress terminal, and check
		// if it's not nil, then return it
		if err := <-chTeaError; err != nil {
			return err
		}

		return nil
	}
}
