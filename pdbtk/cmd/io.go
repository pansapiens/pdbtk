package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/perry/pdbtk/pdbtk/structure"
)

var (
	inFormatFlag  string
	outFormatFlag string
	forceLossyPDB bool
)

// addInFormatFlag registers --in-format on commands that read a structure.
func addInFormatFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&inFormatFlag, "in-format", "",
		"Input format: pdb or cif (default: inferred from the file extension or contents)")
}

// addOutFormatFlag registers --out-format on commands that write a structure.
func addOutFormatFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&outFormatFlag, "out-format", "",
		"Output format: pdb or cif (default: inferred from --output, else same as the input)")
}

// readStructure resolves the input source and parses it. An empty path means
// stdin, which is only valid when stdin is not a terminal.
func readStructure(path string) (*structure.Structure, error) {
	want := structure.FormatUnknown
	if inFormatFlag != "" {
		f, err := structure.ParseFormat(inFormatFlag)
		if err != nil {
			return nil, err
		}
		want = f
	}

	if path != "" {
		s, err := structure.ReadFile(path, want)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", path, err)
		}
		return s, nil
	}

	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %v", err)
	}
	s, err := structure.ReadContent(content, want)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %v", err)
	}
	return s, nil
}

// resolveInputPath validates a positional input file argument, returning ""
// when the command should read from stdin instead.
func resolveInputPath(args []string) (string, error) {
	if len(args) > 0 {
		if err := CheckFileExists(args[0]); err != nil {
			return "", err
		}
		return args[0], nil
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to check stdin: %v", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no input file specified and stdin is not available")
	}
	return "", nil
}

// resolveOutputFormat picks the output format from --out-format, else the
// --output extension, else the format the input was read in.
func resolveOutputFormat(in *structure.Structure, outputPath string) (structure.Format, error) {
	if outFormatFlag != "" {
		return structure.ParseFormat(outFormatFlag)
	}
	if outputPath != "" && outputPath != "-" {
		if f := structure.FormatFromPath(outputPath); f != structure.FormatUnknown {
			return f, nil
		}
	}
	if in.Source != structure.FormatUnknown {
		return in.Source, nil
	}
	return structure.FormatPDB, nil
}

// writeStructure serialises to --output, or stdout when it is empty or "-".
func writeStructure(s *structure.Structure, outputPath string, f structure.Format, commandLine string) error {
	opt := structure.WriteOptions{
		CommandLine: commandLine,
		Version:     Version,
		ForceLossy:  forceLossyPDB,
		Warn:        func(msg string) { fmt.Fprintf(os.Stderr, "Warning: %s\n", msg) },
	}

	if outputPath == "" || outputPath == "-" {
		return structure.Write(os.Stdout, s, f, opt)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()
	return structure.Write(file, s, f, opt)
}

// buildCommandLine reconstructs the invocation for the output provenance
// record, using the flags the user actually set.
func buildCommandLine(cmd *cobra.Command, args []string) string {
	parts := []string{"pdbtk", cmd.Name()}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Value.Type() == "bool" {
			parts = append(parts, "--"+f.Name)
			return
		}
		parts = append(parts, "--"+f.Name, f.Value.String())
	})
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}

// splitChainList parses a comma-separated chain list. Chain IDs are not
// restricted to one character, since mmCIF permits longer identifiers.
func splitChainList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, c := range strings.Split(s, ",") {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}
