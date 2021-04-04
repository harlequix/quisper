package cmd
import (
    "github.com/harlequix/quisper/quisper"
    // "github.com/harlequix/quisper/internal/encoding"
    "github.com/spf13/cobra"
)
var checkCmd = &cobra.Command{
   Use:   "check",
   Short: "A brief description of your application",
   Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
   Run: check,
   Args: cobra.MinimumNArgs(2),
   // Uncomment the following line if your bare application
   // has an action associated with it:
   //      Run: func(cmd *cobra.Command, args []string) { },
}

func init() {
   rootCmd.AddCommand(checkCmd)
}

func check(cmd *cobra.Command, args []string) {
    Writer := quisper.NewWriter(args[0], args[1])
    Writer.TestServer()
    Writer.TestLongServer()
}
