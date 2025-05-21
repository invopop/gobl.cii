package main

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
	context        string
	schemaPath     string
	schematronPath string
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
	cmd.Flags().StringVar(&c.schemaPath, "schema", "", "Path to the schema file")
	cmd.Flags().StringVar(&c.schematronPath, "schematron", "", "Path to the schematron file (xslt or sef)")

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
		ctx = cii.ContextFacturX
	case "xrechnung":
		ctx = cii.ContextXRechnung
	case "peppol":
		ctx = cii.ContextPeppolV3
	case "en16931":
		ctx = cii.ContextEN16931
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

		if c.schemaPath != "" {
			schemaPath, err := filepath.Abs(c.schemaPath)
			if err != nil {
				return fmt.Errorf("resolving schema path: %w", err)
			}
			if err := cii.ValidateAgainstSchema(outputData, schemaPath); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
		}

		if c.schematronPath != "" {
			schematronPath, err := filepath.Abs(c.schematronPath)
			if err != nil {
				return fmt.Errorf("resolving schematron path: %w", err)
			}
			if err := cii.ValidateWithSchematron(outputData, schematronPath); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
		}
	} else {
		// Assume XML if not JSON
		env, err := cii.ParseInvoice(inData)
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
