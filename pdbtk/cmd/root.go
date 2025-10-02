package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "0.1.1"

var rootCmd = &cobra.Command{
	Use:   "pdbtk",
	Short: "PDB structure file manipulation toolkit",
	Long: fmt.Sprintf(`pdbtk is a command-line toolkit for manipulating PDB structure files.
It provides various operations for extracting, filtering, and transforming protein structure data.

Version: %s`, Version),
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of pdbtk.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(extractSeqCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(renameChainCmd)
	rootCmd.AddCommand(renumberResiduesCmd)
	rootCmd.AddCommand(versionCmd)
}

// CheckFileExists checks if a file exists and returns an error if it doesn't
func CheckFileExists(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}
	return nil
}
