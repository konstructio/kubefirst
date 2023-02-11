package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strconv"
	"strings"
)

type CommandMenu struct {
	Key   string
	Value string
}

func Interactive(commands []*cobra.Command) {
	command, _ := buildCommand(commands)

	if command.Use == "None" {
		return
	}

	parent := command.Parent()
	finalCommand := ""

	if parent != nil {
		finalCommand = fmt.Sprintf("%s %s", finalCommand, parent.Use)
	} else {
		finalCommand = "kubefirst"
	}

	finalCommand = fmt.Sprintf("%s %s", finalCommand, command.Use)
	finalCommand = printFlags(command, finalCommand)

	fmt.Println("Executing command")
	fmt.Println(color.GreenString(finalCommand))

	promptConfirm := promptui.Prompt{
		Label:     fmt.Sprintf("Are you sure, to execute %s", finalCommand),
		IsConfirm: true,
	}
	result, err := promptConfirm.Run()

	if result != "y" {
		return
	}

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	if command.Run != nil {
		command.Run(command, []string{})
		return
	}

	errorExecution := command.RunE(command, []string{})
	if errorExecution != nil {
		fmt.Printf("Error executing command %v\n", errorExecution)
		return
	}
}

func printFlags(command *cobra.Command, finalCommand string) string {
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Value.Type() == "bool" {
			if flag.Value.String() == "true" {
				finalCommand = fmt.Sprintf("%s --%s", finalCommand, flag.Name)
			}
		}
	})
	return finalCommand
}

func buildCommand(commands []*cobra.Command) (*cobra.Command, error) {
	var commandsList []CommandMenu
	commands = append([]*cobra.Command{{Use: "None", Short: "-----"}}, commands...)

	for _, value := range commands {
		commandsList = append(commandsList, CommandMenu{Key: value.Use, Value: value.Short})
	}

	options := make([]string, len(commandsList))
	for i, command := range commandsList {
		commandValue := fmt.Sprintf("%s: %s",
			color.BlueString(command.Key),
			color.GreenString(command.Value))
		options[i] = commandValue
	}

	prompt := promptui.Select{
		Label:             "Select your command",
		Items:             options,
		StartInSearchMode: true,
		Searcher: func(input string, index int) bool {
			if found := strings.Index(options[index], input); found != -1 {
				return true
			}
			return false
		},
	}

	index, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, err
	}

	command := commands[index]

	// do a slice of strings
	var flags []*item

	command.Flags().VisitAll(func(flag *pflag.Flag) {
		flagValue, _ := strconv.ParseBool(flag.Value.String())
		flags = append(flags, &item{flag.Name, flagValue})
	})

	if len(flags) > 0 {
		flagsSelected, err := selectItems(0, flags)
		if err != nil {
			return nil, err
		}

		command.Flags().VisitAll(func(flag *pflag.Flag) {
			command.Flags().Set(flag.Name, "false")
		})

		for _, flag := range flagsSelected {
			command.Flags().Set(flag.ID, "true")
		}
	}

	subcommands := command.Commands()

	if len(subcommands) > 0 {
		subcommand, err := buildCommand(subcommands)
		if err != nil {
			return nil, err
		}
		if subcommand.Use != "None" {
			return subcommand, nil
		}

		return command, nil
	}

	return command, nil
}

type item struct {
	ID         string
	IsSelected bool
}

// selectItems() prompts user to select one or more items in the given slice
func selectItems(selectedPos int, allItems []*item) ([]*item, error) {
	const doneID = "Confirm ?"
	if len(allItems) > 0 && allItems[0].ID != doneID {
		var items = []*item{
			{
				ID: doneID,
			},
		}
		allItems = append(items, allItems...)
	}

	templates := &promptui.SelectTemplates{
		Label: `{{if .IsSelected}}
                    ✔
                {{end}} {{ .ID }} - label`,
		Active:   "→ {{if .IsSelected}}✔ {{end}}{{ .ID | cyan }}",
		Inactive: "{{if .IsSelected}}✔ {{end}}{{ .ID | cyan }}",
	}

	prompt := promptui.Select{
		Label:     "Flags...",
		Items:     allItems,
		Templates: templates,
		Size:      5,
		// Start the cursor at the currently selected index
		CursorPos:    selectedPos,
		HideSelected: true,
	}

	selectionIdx, _, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	chosenItem := allItems[selectionIdx]

	if chosenItem.ID != doneID {
		chosenItem.IsSelected = !chosenItem.IsSelected
		return selectItems(selectionIdx, allItems)
	}

	var selectedItems []*item
	for _, i := range allItems {
		if i.IsSelected {
			selectedItems = append(selectedItems, i)
		}
	}
	return selectedItems, nil
}
