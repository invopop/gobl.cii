package cii_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathPatternXML  = "*.xml"
	pathPatternJSON = "*.json"
	pathConvert     = "convert"
	pathParse       = "parse"
	pathOut         = "out"

	staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"
)

// updateOut is a flag that can be set to update example files
var updateOut = flag.Bool("update", false, "Update the example files in test/data")

func TestConvertInvoice(t *testing.T) {
	convertPath := filepath.Join(dataPath(), pathConvert)

	// Get all folders inside convert
	folders, err := os.ReadDir(convertPath)
	require.NoError(t, err)

	for _, folder := range folders {
		if !folder.IsDir() {
			continue
		}

		format := folder.Name()
		var ctx cii.Context

		// Set the context based on the folder name
		switch format {
		case "facturx":
			ctx = cii.ContextFacturX
		case "xrechnung":
			ctx = cii.ContextXRechnung
		case "peppol":
			ctx = cii.ContextPeppolV3
		case "cii":
			ctx = cii.ContextEN16931
		default:
			t.Logf("Skipping unknown format folder: %s", format)
			continue
		}

		t.Run(format, func(t *testing.T) {
			// Find all JSON files in this format's folder
			examples := findSourceFiles(t, filepath.Join(pathConvert, format), pathPatternJSON)

			for _, example := range examples {
				inName := filepath.Base(example)
				outName := strings.Replace(inName, ".json", ".xml", 1)

				t.Run(inName, func(t *testing.T) {
					// Load and convert using the format-specific context
					env := loadEnvelope(t, filepath.Join(format, inName))
					out, err := cii.ConvertInvoice(env, cii.WithContext(ctx))
					require.NoError(t, err)

					data, err := out.Bytes()
					require.NoError(t, err)

					if *updateOut {
						// Create the output directory if it doesn't exist
						outDir := filepath.Join(dataPath(), pathConvert, format, pathOut)
						require.NoError(t, os.MkdirAll(outDir, 0755))

						err = os.WriteFile(filepath.Join(outDir, outName), data, 0644)
						require.NoError(t, err)
						err = cii.ValidateXML(data, format)
						require.NoError(t, err)
					}

					// Load the expected output
					output := loadOutputFile(t, filepath.Join(pathConvert, format), outName)
					assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
				})
			}
		})
	}
}

func TestParseInvoice(t *testing.T) {
	examples := findSourceFiles(t, pathParse, pathPatternXML)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			// Load XML data
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			// Convert CII XML to GOBL
			env, err := cii.ParseInvoice(xmlData)
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			if inv, ok := env.Extract().(*bill.Invoice); ok {
				inv.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			writeEnvelope(dataPath(pathParse, pathOut, outName), env)

			// Extract the invoice from the envelope
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok, "Document should be an invoice")

			// Marshal only the invoice
			data, err := json.MarshalIndent(inv, "", "\t")
			require.NoError(t, err)

			// Load the expected output
			output := loadOutputFile(t, pathParse, outName)

			// Parse the expected output to extract the invoice
			var expectedEnv gobl.Envelope
			err = json.Unmarshal(output, &expectedEnv)
			require.NoError(t, err)

			expectedInvoice, ok := expectedEnv.Extract().(*bill.Invoice)
			require.True(t, ok, "Expected document should be an invoice")

			// Marshal the expected invoice
			expectedData, err := json.MarshalIndent(expectedInvoice, "", "  ")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Invoice should match the expected JSON. Update with --update flag.")
		})
	}
}

// newInvoiceFrom creates a cii Document from a GOBL file in the `test/data` folder
func newInvoiceFrom(t *testing.T, name string) (*cii.Invoice, error) {
	t.Helper()
	env := loadEnvelope(t, name)
	return cii.ConvertInvoice(env)
}

func parseInvoiceFrom(t *testing.T, name string) (*gobl.Envelope, error) {
	t.Helper()
	path := dataPath(pathParse, name)
	src, err := os.Open(path)
	require.NoError(t, err)
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	data, err := io.ReadAll(src)
	if err != nil {
		require.NoError(t, err)
	}
	return cii.ParseInvoice(data)
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

func outputFilepath(path, name string) string {
	return filepath.Join(dataPath(path, pathOut, name))
}

func loadOutputFile(t *testing.T, path, name string) []byte {
	t.Helper()
	src, err := os.Open(outputFilepath(path, name))
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		require.NoError(t, err)
	}
	return buf.Bytes()
}

func findSourceFiles(t *testing.T, path, pattern string) []string {
	path = filepath.Join(dataPath(), path, pattern)
	files, err := filepath.Glob(path)
	require.NoError(t, err)
	return files
}

func dataPath(files ...string) string {
	files = append([]string{cii.RootFolder(), "test", "data"}, files...)
	return filepath.Join(files...)
}
