package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/cbc"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
	format string
}

func convert(o *rootOpts) *convertOpts {
	return &convertOpts{rootOpts: o}
}

func (c *convertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert <infile> <outfile>",
		Short: "Convert a GOBL JSON into a Cross Industry Invoice (CII) document and vice versa",
		RunE:  c.runE,
	}

	cmd.Flags().StringVar(&c.format, "format", cii.FormatEN16931.Key.String(), "Output format key, e.g. cii+en16931 or cii+peppol")

	return cmd
}

func (c *convertOpts) runE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("expected one or two arguments, the command usage is `gobl.cii convert <infile> [outfile]`")
	}

	f := cii.FormatFor(cbc.Key(c.format))
	if f == nil {
		return fmt.Errorf("unsupported format: %s", c.format)
	}

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	out, err := c.openOutput(cmd, args)
	if err != nil {
		return err
	}
	defer out.Close() // nolint:errcheck

	inData, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	// Check if input is JSON or XML
	isJSON := json.Valid(inData)

	var outputData []byte

	if isJSON {
		env := new(gobl.Envelope)
		if err := json.Unmarshal(inData, env); err != nil {
			return fmt.Errorf("parsing input as GOBL Envelope: %w", err)
		}

		doc, err := cii.Export(env, cii.WithFormat(*f))
		if err != nil {
			return fmt.Errorf("building %s document: %w", c.format, err)
		}

		outputData, err = cii.Encode(doc)
		if err != nil {
			return fmt.Errorf("generating %s xml: %w", c.format, err)
		}

	} else {
		// Assume XML if not JSON
		doc, err := cii.Decode(inData)
		if err != nil {
			return fmt.Errorf("converting CII to GOBL: %w", err)
		}
		env, err := cii.Import(doc)
		if err != nil {
			return fmt.Errorf("converting CII to GOBL: %w", err)
		}

		outputData, err = json.MarshalIndent(env, "", "  ")
		if err != nil {
			return fmt.Errorf("generating JSON output: %w", err)
		}
	}

	if _, err = out.Write(outputData); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	return nil
}
