package cii_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/invopop/phorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathPatternXML  = "*.xml"
	pathPatternJSON = "*.json"
	pathConvert     = "convert"
	pathParse       = "parse"
	pathOut         = "out"

	// rateZero is a zero amount, as written in totals.
	rateZero = "0.00"

	// Format names shared by the conversion and parse tables.
	ctxEN16931 = "EN16931"
	ctxPeppol  = "Peppol"

	// Item attribute labels and values shared by the attribute tests.
	attrLabelColor  = "Color"
	attrValueBlack  = "Black"
	attrLabelWeight = "Weight"

	staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"
)

// updateOut is a flag that can be set to update example files
var updateOut = flag.Bool("update", false, "Update the example files in test/data")

// validate is a flag that enables schematron validation against phorm
var validate = flag.Bool("validate", false, "Run phorm schematron validation on generated XML")

func TestConvertToInvoice(t *testing.T) {
	var pc *phorm.Client

	// Only connect to phorm if validation is requested
	if *validate {
		pc = phormClient(t)
	}

	// Define contexts to test
	contexts := []struct {
		name    string
		context cii.Format
		dir     string
	}{
		{ctxEN16931, cii.FormatEN16931, "en16931"},
		{ctxPeppol, cii.FormatPeppol, "peppol"},
	}

	for _, ctx := range contexts {
		t.Run(ctx.name, func(t *testing.T) {
			examples, err := filepath.Glob(filepath.Join(getConvertPath(), ctx.dir, pathPatternJSON))
			require.NoError(t, err)

			if len(examples) == 0 {
				t.Skip("No examples found for context")
			}

			for _, example := range examples {
				inName := filepath.Base(example)
				outName := strings.Replace(inName, ".json", ".xml", 1)

				t.Run(inName, func(t *testing.T) {
					// Load and convert using the format-specific context
					env := loadEnvelope(t, filepath.Join(ctx.dir, inName))
					out, err := cii.ExportInvoice(env, cii.WithFormat(ctx.context))
					require.NoError(t, err)

					data, err := cii.Encode(out)
					require.NoError(t, err)

					outPath := filepath.Join(getConvertPath(), ctx.dir, pathOut, outName)
					if *updateOut {
						// Create the output directory if it doesn't exist
						outDir := filepath.Join(getConvertPath(), ctx.dir, pathOut)
						require.NoError(t, os.MkdirAll(outDir, 0755))

						err = os.WriteFile(outPath, data, 0644)
						require.NoError(t, err)
					}

					// Run schematron validation if requested
					if *validate && ctx.context.VESID != "" {
						validateXML(t, pc, ctx.context.VESID, data)
					}

					// Load the expected output
					output, err := os.ReadFile(outPath)
					assert.NoError(t, err)
					assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
				})
			}
		})
	}
}

func TestParseInvoice(t *testing.T) {
	// Define contexts to test
	contexts := []struct {
		name string
		dir  string
	}{
		{ctxEN16931, "en16931"},
		{ctxPeppol, "peppol"},
	}

	for _, ctx := range contexts {
		t.Run(ctx.name, func(t *testing.T) {
			examples, err := filepath.Glob(filepath.Join(getParsePath(), ctx.dir, pathPatternXML))
			require.NoError(t, err)

			if len(examples) == 0 {
				t.Skip("No examples found for context")
			}

			for _, example := range examples {
				inName := filepath.Base(example)
				outName := strings.Replace(inName, ".xml", ".json", 1)

				t.Run(inName, func(t *testing.T) {
					// Load XML data
					xmlData, err := os.ReadFile(example)
					require.NoError(t, err)

					// Convert CII XML to GOBL
					env, err := parseCII(xmlData)
					require.NoError(t, err)

					env.Head.UUID = staticUUID
					if inv, ok := env.Extract().(*bill.Invoice); ok {
						inv.UUID = staticUUID
					}
					require.NoError(t, env.Calculate())

					outPath := filepath.Join(getParsePath(), ctx.dir, pathOut, outName)
					if *updateOut {
						// Create the output directory if it doesn't exist
						outDir := filepath.Join(getParsePath(), ctx.dir, pathOut)
						require.NoError(t, os.MkdirAll(outDir, 0755))

						data, err := json.MarshalIndent(env, "", "\t")
						require.NoError(t, err)
						err = os.WriteFile(outPath, data, 0644)
						require.NoError(t, err)
					}

					// Extract the invoice from the envelope
					inv, ok := env.Extract().(*bill.Invoice)
					require.True(t, ok, "Document should be an invoice")

					// Marshal only the invoice
					data, err := json.MarshalIndent(inv, "", "\t")
					require.NoError(t, err)

					// Load the expected output
					output, err := os.ReadFile(outPath)
					assert.NoError(t, err)

					// Parse the expected output to extract the invoice
					var expectedEnv gobl.Envelope
					err = json.Unmarshal(output, &expectedEnv)
					require.NoError(t, err)

					expectedInvoice, ok := expectedEnv.Extract().(*bill.Invoice)
					require.True(t, ok, "Expected document should be an invoice")

					// Marshal the expected invoice
					expectedData, err := json.MarshalIndent(expectedInvoice, "", "\t")
					require.NoError(t, err)

					assert.JSONEq(t, string(expectedData), string(data), "Invoice should match the expected JSON. Update with --update flag.")
				})
			}
		})
	}
}

// newInvoiceFrom creates a CII Invoice from a GOBL file in the `test/data/convert` folder
func newInvoiceFrom(t *testing.T, name string) (*cii.Invoice, error) {
	t.Helper()
	env := loadEnvelope(t, name)
	return cii.ExportInvoice(env)
}

// parseInvoiceFrom parses a CII XML file from the `test/data/parse` folder
func parseInvoiceFrom(t *testing.T, name string) (*gobl.Envelope, error) {
	t.Helper()
	path := dataPath(pathParse, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseCII(data)
}

// parseCII decodes and imports CII XML into a GOBL envelope.
func parseCII(data []byte, opts ...cii.Option) (*gobl.Envelope, error) {
	doc, err := cii.Decode(data)
	if err != nil {
		return nil, err
	}
	return cii.Import(doc, opts...)
}

// loadEnvelope returns a GOBL Envelope from a file in the `test/data/convert` folder
func loadEnvelope(t *testing.T, name string) *gobl.Envelope {
	t.Helper()
	path := dataPath(pathConvert, name)

	src, _ := os.Open(path)
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(src)
	require.NoError(t, err)

	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(buf.Bytes(), env))

	// Clear the IDs
	env.Head.UUID = staticUUID
	if inv, ok := env.Extract().(*bill.Invoice); ok {
		inv.UUID = staticUUID
	}
	require.NoError(t, env.Calculate())
	require.NoError(t, env.Validate())

	writeEnvelope(path, env)

	return env
}

func writeEnvelope(path string, env *gobl.Envelope) {
	if !*updateOut {
		return
	}
	data, err := json.MarshalIndent(env, "", "\t")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func dataPath(files ...string) string {
	files = append([]string{rootFolder(), "test", "data"}, files...)
	return filepath.Join(files...)
}

func getConvertPath() string {
	return filepath.Join(getDataPath(), pathConvert)
}

func getParsePath() string {
	return filepath.Join(getDataPath(), pathParse)
}

func getDataPath() string {
	return filepath.Join(getTestPath(), "data")
}

func getTestPath() string {
	return filepath.Join(rootFolder(), "test")
}

// rootFolder returns the root folder of the project
func rootFolder() string {
	cwd, _ := os.Getwd()
	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}
	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}
	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}

// TestImportRouting verifies that a received document takes its transport
// Head.From/To from the routing options, overriding GOBL's document-derived,
// outgoing (supplier->customer) assumption.
func TestImportRouting(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(getParsePath(), "CII_example1.xml"))
	require.NoError(t, err)

	env, err := parseCII(data, cii.WithRouting(
		"iso6523-actorid-upis::0088:receiver",
		"iso6523-actorid-upis::0088:sender",
	))
	require.NoError(t, err)
	assert.Equal(t, "iso6523-actorid-upis::0088:receiver", string(env.Head.From))
	assert.Equal(t, "iso6523-actorid-upis::0088:sender", string(env.Head.To))
}
