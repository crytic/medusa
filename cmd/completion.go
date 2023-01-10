package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const generalComDesc = `
Generate the autocompletion script for medusa for the specific shell.

Bash:
To load completions in the current shell session:

    source <(medusa completion bash)

To load completions for every new session, execute once:
- Linux:
      medusa completion bash > /etc/bash_completion.d/medusa

- macOS:
      medusa completion bash > /usr/local/etc/bash_completion.d/medusa

Zsh:
To load completions in the current shell session:

    source <(medusa completion zsh)

To load completions for every new session, execute once:

    medusa completion zsh > "${fpath[1]}/_medusa"

PowerShell:
To load completions in the current shell session:
PS> medusa completion powershell | Out-String | Invoke-Expression

To load completions for every new session, run:
PS> medusa completion powershell > medusa.ps1
and source this file from your PowerShell profile.
`

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "generate the autocompletion script for medusa for the specific shell",
	Long:  generalComDesc,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("Error: No shell specified")
			return
		}
		switch args[0] {
		case "bash":
			err := runCompletionBash(cmd)
			if err != nil {
				log.Fatalln("Error: unable to generate a bash completion")
			}
		case "zsh":
			err := runCompletionZsh(cmd)
			if err != nil {
				log.Fatalln("Error: unable to generate a zsh completion")
			}
		case "powershell":
			err := runCompletionPowerShell(cmd)
			if err != nil {
				log.Fatalln("Error: unable to generate a powershell completion")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// runCompletionBash generates and prints Bash completion code to the STDOUT
func runCompletionBash(cmd *cobra.Command) error {
	err := cmd.Root().GenBashCompletionV2(os.Stdout, true)
	if err != nil {
		return err
	}
	return nil
}

// runCompletionZsh generates and prints ZSH completion code to the STDOUT
func runCompletionZsh(cmd *cobra.Command) error {
	err := cmd.Root().GenZshCompletion(os.Stdout)
	if err != nil {
		return err
	}
	return nil
}

// runCompletionPowerShell generates and prints ZSH completion code to the STDOUT
func runCompletionPowerShell(cmd *cobra.Command) error {
	err := cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	if err != nil {
		return err
	}
	return nil
}
