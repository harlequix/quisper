package cmd

 import (
     "github.com/harlequix/quisper/quisper"
     // "github.com/harlequix/quisper/internal/encoding"
     "bufio"
     "os"
     "context"
     "github.com/spf13/cobra"
 )

 var writeCmd = &cobra.Command{
 	Use:   "write",
 	Short: "A brief description of your application",
 	Long: `A longer description that spans multiple lines and likely contains
 examples and usage of using your application. For example:

 Cobra is a CLI library for Go that empowers applications.
 This application is a tool to generate the needed files
 to quickly create a Cobra application.`,
    Run: write,
    Args: cobra.MinimumNArgs(2),
 	// Uncomment the following line if your bare application
 	// has an action associated with it:
 	//      Run: func(cmd *cobra.Command, args []string) { },
 }

 var heavywriteCmd = &cobra.Command{
 	Use:   "heavywrite",
 	Short: "A brief description of your application",
 	Long: `A longer description that spans multiple lines and likely contains
 examples and usage of using your application. For example:

 Cobra is a CLI library for Go that empowers applications.
 This application is a tool to generate the needed files
 to quickly create a Cobra application.`,
    Run: Heavywrite,
    Args: cobra.MinimumNArgs(2),
 	// Uncomment the following line if your bare application
 	// has an action associated with it:
 	//      Run: func(cmd *cobra.Command, args []string) { },
 }

 func init() {
 	rootCmd.AddCommand(writeCmd)
    rootCmd.AddCommand(heavywriteCmd)
 }

func write(cmd *cobra.Command, args []string) {
    Writer := quisper.NewWriter(args[0], args[1])
    Writer.Connect()
    reader := bufio.NewReader(os.Stdin)
    for {
        text, _ := reader.ReadString('\n')
        messagebits := []byte(text)
        Writer.Write(messagebits)
    }
}
func Heavywrite(cmd *cobra.Command, args []string) {
    Writer := quisper.NewWriter(args[0], args[1])
    pipeline := make(chan byte, 64)
    waitFor := context.Background()
    go Writer.MainLoop(waitFor, pipeline)
    for {
        char := byte(49)
        pipeline <- char
    }
}
