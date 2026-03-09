package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	awsprovider "github.com/havenapp/haven/internal/provider/aws"
)

func newLoginCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with your cloud provider",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompter := newTerminalPrompter()
			switch *providerName {
			case "aws":
				_, _, err := awsprovider.Login(cmd.Context(), prompter, os.Stdout)
				if err != nil {
					return err
				}
				fmt.Println("\nLogged in successfully.")
				return nil
			default:
				return fmt.Errorf("unknown provider %q", *providerName)
			}
		},
	}
}
