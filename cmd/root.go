package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "0.1"

var rootCmd = &cobra.Command{
	Use:   "pdbtk",
	Short: "PDB (and PDBx/mmCIF) structure file manipulation toolkit",
	Long: `pdbtk is a command-line toolkit for manipulating PDB and PDBx/mmCIF structure files.
It provides various operations for extracting, filtering, and transforming protein structure data.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(extractSeqCmd)
}

// checkFileExists checks if a file exists and returns an error if it doesn't
func checkFileExists(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}
	return nil
}
