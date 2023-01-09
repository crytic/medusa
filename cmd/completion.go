package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use: "completion bash",
	Short: "Generate shell completion code for specified shell (bash) and evaluate it" +
		"to enable interactive completion of kubectl commands",
	Long: `To load completions:

Bash:

  $ source <(%[1]s completion bash), e.g. source <(medusa completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ %[1]s completion bash > /etc/bash_completion.d/%[1]s
  # macOS:
  $ %[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("Error: No shell specified")
			return
		}
		switch args[0] {
		case "bash":
			err := cmd.Root().GenBashCompletion(os.Stdout)
			if err != nil {
				fmt.Printf("Error: Unable to generate a bash completion")
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
