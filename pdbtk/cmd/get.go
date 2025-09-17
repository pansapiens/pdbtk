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
The file will be downloaded from https://files.rcsb.org/download/{pdb_code}.{format}

By default, the file is saved as {pdb_code}.pdb in the current directory.
Use --output to specify a different filename or "-" to output to stdout.
Use --format to specify the file format (pdb, pdb.gz, cif, cif.gz).

Examples:
  # Download 1A02 as PDB file
  pdbtk get 1A02

  # Download as compressed PDB file
  pdbtk get --format pdb.gz 1A02

  # Download as mmCIF file
  pdbtk get --format cif 1A02

  # Download to stdout
  pdbtk get --output - 1A02

  # Download to specific file
  pdbtk get --output my_structure.pdb 1A02`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func init() {
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "", "Output file (default: {pdb_code}.{format}, use '-' for stdout)")
	getCmd.Flags().StringVarP(&getFormat, "format", "f", "pdb", "File format: pdb, pdb.gz, cif, cif.gz (default: pdb)")
}

func runGet(cmd *cobra.Command, args []string) error {
	pdbCode := strings.ToUpper(args[0])

	// Validate PDB code format (4 characters, alphanumeric)
	if len(pdbCode) != 4 {
		return fmt.Errorf("PDB code must be exactly 4 characters: %s", pdbCode)
	}

	// Validate format
	validFormats := map[string]bool{
		"pdb":    true,
		"pdb.gz": true,
		"cif":    true,
		"cif.gz": true,
	}
	if !validFormats[getFormat] {
		return fmt.Errorf("invalid format '%s', must be one of: pdb, pdb.gz, cif, cif.gz", getFormat)
	}

	// Construct download URL
	url := fmt.Sprintf("https://files.rcsb.org/download/%s.%s", pdbCode, getFormat)

	// Determine output filename
	var outputFile string
	if getOutput == "" {
		// Default filename based on PDB code and format
		outputFile = fmt.Sprintf("%s.%s", pdbCode, getFormat)
	} else if getOutput == "-" {
		// Output to stdout
		outputFile = ""
	} else {
		// Use specified filename
		outputFile = getOutput
	}

	// Download the file
	return downloadFile(url, outputFile, pdbCode)
}

func downloadFile(url, outputFile, pdbCode string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make HTTP request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d - %s", resp.StatusCode, resp.Status)
	}

	// Check if file already exists (only for file output, not stdout)
	if outputFile != "" {
		if _, err := os.Stat(outputFile); err == nil {
			return fmt.Errorf("file already exists: %s", outputFile)
		}
	}

	// Determine output destination
	var writer io.Writer
	if outputFile == "" {
		// Output to stdout
		writer = os.Stdout
	} else {
		// Create output file
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	// Copy response body to output
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// Print success message to stderr (so it doesn't interfere with stdout output)
	if outputFile == "" {
		fmt.Fprintf(os.Stderr, "Downloaded %s to stdout\n", pdbCode)
	} else {
		// Get file size for confirmation
		if stat, err := os.Stat(outputFile); err == nil {
			fmt.Fprintf(os.Stderr, "Downloaded %s (%d bytes) to %s\n", pdbCode, stat.Size(), outputFile)
		} else {
			fmt.Fprintf(os.Stderr, "Downloaded %s to %s\n", pdbCode, outputFile)
		}
	}

	return nil
}
