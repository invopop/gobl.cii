package cii_test

import (
	"bytes"
	"context"
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
	"github.com/invopop/phive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

// validate is a flag that enables Phive validation
var validate = flag.Bool("validate", false, "Run Phive validation on generated XML")

func TestConvertToInvoice(t *testing.T) {
	var pc phive.ValidationServiceClient

	// Only connect to Phive if validation is requested
	if *validate {
		conn, err := grpc.NewClient(
			"localhost:9091",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close() //nolint:errcheck
		pc = phive.NewValidationServiceClient(conn)
	}

	// Define contexts to test
	contexts := []struct {
		name    string
		context cii.Context
		dir     string
	}{
		{"EN16931", cii.ContextEN16931V2017, "en16931"},
		{"Peppol", cii.ContextPeppolV3, "peppol"},
		{"FacturX", cii.ContextFacturXV1, "facturx"},
		{"XRechnung", cii.ContextXRechnungV3, "xrechnung"},
		{"ChorusPro", cii.ContextChorusProV1, "choruspro"},
		{"PeppolFranceFacturX", cii.ContextPeppolFranceFacturXV1, "peppol-france-facturx"},
		{"PeppolFranceCIUS", cii.ContextPeppolFranceCIUSV1, "peppol-france-cius"},
		{"ZUGFeRD", cii.ContextZUGFeRDV2, "zugferd"},
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
					out, err := cii.ConvertInvoice(env, cii.WithContext(ctx.context))
					require.NoError(t, err)

					data, err := out.Bytes()
					require.NoError(t, err)

					outPath := filepath.Join(getConvertPath(), ctx.dir, pathOut, outName)
					if *updateOut {
						// Create the output directory if it doesn't exist
						outDir := filepath.Join(getConvertPath(), ctx.dir, pathOut)
						require.NoError(t, os.MkdirAll(outDir, 0755))

						err = os.WriteFile(outPath, data, 0644)
						require.NoError(t, err)
					}

					// Run Phive validation if requested
					if *validate && ctx.context.VESID != "" {
						resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
							Vesid:      ctx.context.VESID,
							XmlContent: data,
						})
						require.NoError(t, err)
						results, err := json.MarshalIndent(resp.Results, "", "  ")
						require.NoError(t, err)
						require.True(t, resp.Success, "Generated XML should be valid for %s: %s", ctx.context.VESID, string(results))
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
		{"EN16931", "en16931"},
		{"Peppol", "peppol"},
		{"FacturX", "facturx"},
		{"XRechnung", "xrechnung"},
		{"ChorusPro", "choruspro"},
		{"PeppolFranceFacturX", "peppol-france-facturx"},
		{"PeppolFranceCIUS", "peppol-france-cius"},
		{"PeppolFranceExtended", "peppol-france-extended"},
		{"ZUGFeRD", "zugferd"},
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
					env, err := cii.Parse(xmlData)
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
	return cii.ConvertInvoice(env)
}

// parseInvoiceFrom parses a CII XML file from the `test/data/parse` folder
func parseInvoiceFrom(t *testing.T, name string) (*gobl.Envelope, error) {
	t.Helper()
	path := dataPath(pathParse, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return cii.Parse(data)
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
