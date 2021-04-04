package cmd

 import (
     "github.com/harlequix/quisper/quisper"
     "context"
     "github.com/spf13/cobra"
 )

 var readCmd = &cobra.Command{
 	Use:   "read",
 	Short: "A brief description of your application",
 	Long: `A longer description that spans multiple lines and likely contains
 examples and usage of using your application. For example:

 Cobra is a CLI library for Go that empowers applications.
 This application is a tool to generate the needed files
 to quickly create a Cobra application.`,
    Run: read,
    Args: cobra.MinimumNArgs(2),
 	// Uncomment the following line if your bare application
 	// has an action associated with it:
 	//      Run: func(cmd *cobra.Command, args []string) { },
 }

 func init() {
 	rootCmd.AddCommand(readCmd)
 }

func read(cmd *cobra.Command, args []string) {
    Reader := quisper.NewReader(args[0], args[1])
    waitFor := context.Background()
    go Reader.MainLoop(waitFor, nil)
    for {

    }
}
