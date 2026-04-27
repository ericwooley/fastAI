package cli

import (
	"github.com/spf13/cobra"
)

func newLoginCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GitHub Copilot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			account, err := deps.Authenticator.Login(cmd.Context(), deps.Out)
			if err != nil {
				return WrapError(ExitAuth, "GitHub Copilot login failed", "Check network access and retry `fastAI login`.", err)
			}
			if err := deps.AuthStore.Save(cmd.Context(), account); err != nil {
				return WrapError(ExitAuth, "could not persist GitHub Copilot login", "Check config directory permissions and retry.", err)
			}
			FormatLoginSuccess(deps.Out, account)
			return nil
		},
	}
}
