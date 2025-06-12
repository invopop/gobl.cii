package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
	context string
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

	cmd.Flags().StringVar(&c.context, "context", "en16931", "Output format (en16931, facturx, xrechnung, peppol)")

	return cmd
}

func (c *convertOpts) runE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("expected one or two arguments, the command usage is `gobl.cii convert <infile> [outfile]`")
	}

	// Get the context based on the format
	var ctx cii.Context
	switch c.context {
	case "facturx":
		ctx = cii.ContextFacturXV1
	case "zugferd":
		ctx = cii.ContextZUGFeRDV2
	case "xrechnung":
		ctx = cii.ContextXRechnungV3
	case "peppol":
		ctx = cii.ContextPeppolV3
	case "en16931":
		ctx = cii.ContextEN16931V2017
	case "choruspro":
		ctx = cii.ContextChorusProV1
	default:
		return fmt.Errorf("unsupported context: %s", c.context)
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

		// Convert using the selected format's context
		doc, err := cii.ConvertInvoice(env, cii.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("building %s document: %w", c.context, err)
		}

		outputData, err = doc.Bytes()
		if err != nil {
			return fmt.Errorf("generating %s xml: %w", c.context, err)
		}

	} else {
		// Assume XML if not JSON
		env, err := cii.Parse(inData)
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
