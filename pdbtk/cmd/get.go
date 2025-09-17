package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	getOutput string
	getFormat string
)

var getCmd = &cobra.Command{
	Use:   "get [flags] <pdb_code>",
	Short: "Download a PDB file from the RCSB PDB database",
	Long: `Download a PDB file from the RCSB PDB database using the PDB code.
The file will be downloaded from https://files.rcsb.org/download/{pdb_code}.pdb

By default, the file is saved as {pdb_code}.pdb in the current directory.
Use --output to specify a different filename or "-" to output to stdout.
Use --format to specify the file format (pdb, pdb.gz).

Examples:
  # Download 1A02 as PDB file
  pdbtk get 1A02

  # Download as compressed PDB file
  pdbtk get --format pdb.gz 1A02

  # Download to stdout
  pdbtk get --output - 1A02

  # Download to specific file
  pdbtk get --output my_structure.pdb 1A02`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "", "Output file (default: {pdb_code}.pdb, use '-' for stdout)")
	getCmd.Flags().StringVarP(&getFormat, "format", "f", "pdb", "File format: pdb, pdb.gz (default: pdb)")
}

func runGet(cmd *cobra.Command, args []string) error {
	pdbCode := strings.ToUpper(args[0])

	// Validate PDB code format (4 characters, alphanumeric)
	if len(pdbCode) != 4 {
		return fmt.Errorf("PDB code must be exactly 4 characters, got: %s", pdbCode)
	}

	// Validate format
	validFormats := map[string]bool{
		"pdb":    true,
		"pdb.gz": true,
	}
	if !validFormats[getFormat] {
		return fmt.Errorf("unsupported format: %s (supported: pdb, pdb.gz)", getFormat)
	}

	// Construct download URL
	url := fmt.Sprintf("https://files.rcsb.org/download/%s.%s", pdbCode, getFormat)

	// Determine output filename
	var outputFile string
	if getOutput == "" {
		outputFile = fmt.Sprintf("%s.%s", pdbCode, getFormat)
	} else if getOutput == "-" {
		outputFile = ""
	} else {
		outputFile = getOutput
	}

	// Download the file
	fmt.Fprintf(os.Stderr, "Downloading %s from %s...\n", pdbCode, url)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	// Write to output
	if outputFile == "" {
		// Write to stdout
		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write to stdout: %v", err)
		}
	} else {
		// Write to file
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write to file: %v", err)
		}

		fmt.Fprintf(os.Stderr, "Downloaded %s to %s\n", pdbCode, outputFile)
	}

	return nil
}
