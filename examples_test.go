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
	"github.com/invopop/gobl/addons/fr/ctc/flow6"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
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

// cdarConvertContexts pins each fixture directory to the CDAR Context it
// represents. The test loop and the fixture regenerator both read context
// from this map — never from bill.Status.Type.
var cdarConvertContexts = []struct {
	name    string
	dir     string
	context cii.Context
}{
	{"CDARFlow6", "cdar", cii.ContextCDARFlow6},
	{"CDARFlow6PPF", "cdar-PPF", cii.ContextCDARFlow6PPF},
}

// cdarConvertFixtures lists the synthetic CDAR status fixtures used to
// drive TestConvertCDAR. Each entry names the destination directory; that
// directory's Context (in cdarConvertContexts) is the only signal for how
// the fixture is shaped — process code and bill.Status.Type are irrelevant.
var cdarConvertFixtures = []struct {
	processCode string
	fixtureName string
	dir         string
}{
	// Buyer-issued ack 23 — Issuer = Customer, Recipient = Supplier
	{"204", "status-204-prise-en-charge.json", "cdar"},
	{"205", "status-205-approuvee.json", "cdar"},
	{"206", "status-206-approuvee-partiellement.json", "cdar"},
	{"207", "status-207-litige.json", "cdar"},
	{"208", "status-208-suspendue.json", "cdar"},
	{"210", "status-210-refusee.json", "cdar"},
	// Seller-issued ack 23 — Issuer = Supplier, Recipient = Customer
	{"209", "status-209-completee.json", "cdar"},
	{"212", "status-212-encaissee.json", "cdar"},
	// Platform-issued ack 23 — caller-supplied Issuer/Recipient
	{"213", "status-213-rejetee-semantique.json", "cdar"},
	// PPF transmissions (305)
	{"200", "status-200-deposee.json", "cdar-PPF"},
	{"211", "status-211-paiement.json", "cdar-PPF"},
}

// contextForDir returns the CDAR Context bound to a fixture directory.
func contextForDir(dir string) (cii.Context, bool) {
	for _, c := range cdarConvertContexts {
		if c.dir == dir {
			return c.context, true
		}
	}
	return cii.Context{}, false
}

// regenerateCDARFixtures rebuilds the JSON fixtures for TestConvertCDAR from
// the synthetic-status factory. Only runs under -update.
//
// For PPF-context fixtures the Issuer is rewritten as the WK platform
// and the Recipient is replaced with the canonical PPF party — those
// are the only spec-conformant shapes for ack TypeCode 305.
func regenerateCDARFixtures(t *testing.T) {
	t.Helper()
	for _, f := range cdarConvertFixtures {
		ctx, ok := contextForDir(f.dir)
		require.True(t, ok, "no context bound to dir %q", f.dir)

		st := buildSyntheticStatus(t, f.processCode)
		if ctx.GuidelineID == cii.CDARGuidelinePPF {
			// 305 / PPF: Issuer is the WK platform; Recipient is the PPF.
			st.Issuer = &org.Party{
				Name:       "PA-FR",
				Identities: st.Issuer.Identities, // keep some SIREN for traceability
				Inboxes:    []*org.Inbox{{Scheme: "0225", Code: "9998_PEP"}},
				Ext:        tax.MakeExtensions().Set(flow6.ExtKeyRole, flow6.RoleWK),
			}
			st.Recipient = ppfRecipient()
		}

		env, err := gobl.Envelop(st)
		require.NoError(t, err, "envelope for %s", f.processCode)
		env.Head.UUID = staticUUID
		st.UUID = staticUUID
		require.NoError(t, env.Calculate())
		require.NoError(t, env.Validate())

		dir := filepath.Join(getConvertPath(), f.dir)
		require.NoError(t, os.MkdirAll(dir, 0755))
		data, err := json.MarshalIndent(env, "", "\t")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, f.fixtureName), data, 0644))
	}
}

func TestConvertCDAR(t *testing.T) {
	if *updateOut {
		regenerateCDARFixtures(t)
	}

	var pc phive.ValidationServiceClient
	if *validate {
		conn, err := grpc.NewClient(
			"localhost:9091",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close() //nolint:errcheck
		pc = phive.NewValidationServiceClient(conn)
	}

	for _, ctx := range cdarConvertContexts {
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
					env := loadEnvelope(t, filepath.Join(ctx.dir, inName))
					out, err := cii.Convert(env, cii.WithContext(ctx.context))
					require.NoError(t, err)

					cdar, ok := out.(*cii.CDAR)
					require.True(t, ok, "expected *cii.CDAR, got %T", out)

					data, err := cdar.Bytes()
					require.NoError(t, err)

					outPath := filepath.Join(getConvertPath(), ctx.dir, pathOut, outName)
					if *updateOut {
						outDir := filepath.Join(getConvertPath(), ctx.dir, pathOut)
						require.NoError(t, os.MkdirAll(outDir, 0755))
						require.NoError(t, os.WriteFile(outPath, data, 0644))
					}

					if *validate && ctx.context.VESID != "" {
						resp, err := pc.ValidateXml(context.Background(), &phive.ValidateXmlRequest{
							Vesid:      ctx.context.VESID,
							XmlContent: data,
						})
						require.NoError(t, err)
						results, err := json.MarshalIndent(resp.Results, "", "  ")
						require.NoError(t, err)
						require.True(t, resp.Success, "Generated CDAR XML should be valid for %s: %s", ctx.context.VESID, string(results))
					}

					expected, err := os.ReadFile(outPath)
					assert.NoError(t, err)
					assert.Equal(t, string(expected), string(data), "Output should match the expected XML. Update with --update flag.")
				})
			}
		})
	}
}

func TestParseCDAR(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join(getParsePath(), "UC*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, examples, "expected UC*.xml CDAR fixtures")

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			env, err := cii.Parse(xmlData)
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			if st, ok := env.Extract().(*bill.Status); ok {
				st.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			outPath := filepath.Join(getParsePath(), pathOut, outName)
			if *updateOut {
				outDir := filepath.Join(getParsePath(), pathOut)
				require.NoError(t, os.MkdirAll(outDir, 0755))
				data, err := json.MarshalIndent(env, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			st, ok := env.Extract().(*bill.Status)
			require.True(t, ok, "Document should be a bill.Status")

			data, err := json.MarshalIndent(st, "", "\t")
			require.NoError(t, err)

			expectedRaw, err := os.ReadFile(outPath)
			assert.NoError(t, err)

			var expectedEnv gobl.Envelope
			require.NoError(t, json.Unmarshal(expectedRaw, &expectedEnv))
			expectedStatus, ok := expectedEnv.Extract().(*bill.Status)
			require.True(t, ok, "Expected document should be a bill.Status")

			expectedData, err := json.MarshalIndent(expectedStatus, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Status should match the expected JSON. Update with --update flag.")
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
