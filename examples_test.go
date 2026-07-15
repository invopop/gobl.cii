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
	"github.com/invopop/gobl/num"
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

	// Convert fixture directory / context names, shared across the
	// conversion tests (also used in invoice_probe_test.go).
	dirCDAR      = "cdar"
	dirCDARPPF   = "cdar-PPF"
	dirFRFacturX = "peppol-france-facturx"
	dirFRCIUS    = "peppol-france-cius"

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
		{"PeppolFranceFacturX", cii.ContextPeppolFranceFacturXV1, dirFRFacturX},
		{"PeppolFranceCIUS", cii.ContextPeppolFranceCIUSV1, dirFRCIUS},
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
		{"PeppolFranceFacturX", dirFRFacturX},
		{"PeppolFranceCIUS", dirFRCIUS},
		{"PeppolFranceExtended", "peppol-france-extended"},
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
	{"CDARFlow6", dirCDAR, cii.ContextCDARFlow6},
	{"CDARFlow6PPF", dirCDARPPF, cii.ContextCDARFlow6PPF},
}

// cdarConvertFixtures lists the synthetic CDAR status fixtures used to
// drive TestConvertCDAR. Each entry names the destination directory; that
// directory's Context (in cdarConvertContexts) is the only signal for how
// the fixture is shaped — process code and bill.Status.Type are irrelevant.
var cdarConvertFixtures = []struct {
	processCode string
	fixtureName string
	dir         string
	payment     bool
}{
	// Buyer-issued ack 23 — Issuer = Customer, Recipient = Supplier
	{"204", "status-204-prise-en-charge.json", dirCDAR, false},
	{"205", "status-205-approuvee.json", dirCDAR, false},
	{"206", "status-206-approuvee-partiellement.json", dirCDAR, false},
	{"207", "status-207-litige.json", dirCDAR, false},
	{"208", "status-208-suspendue.json", dirCDAR, false},
	{"210", "status-210-refusee.json", dirCDAR, false},
	// Platform-issued ack 23
	{"213", "status-213-rejetee-semantique.json", dirCDAR, false},
	// Seller-issued payment receipt (212 Encaissée) — bill.Payment
	{"212", "payment-212-encaissee.json", dirCDAR, true},
	// PPF transmissions (305)
	{"200", "status-200-deposee.json", dirCDARPPF, false},
	{"212", "payment-212-encaissee.json", dirCDARPPF, true},
	// NOTE: 209 (Complétée) has no flow6 (Type, Key) mapping yet and 211
	// (Paiement transmis) is out of the mandatory-status scope; both are
	// omitted until needed.
}

// regenerateCDARFixtures rebuilds the JSON fixtures for TestConvertCDAR from
// the synthetic-status / synthetic-payment factories. Only runs under
// -update. PPF (305) wire shape — WK issuer, PPF recipient — is derived
// by the converter from the Context, so the fixtures carry only the
// business parties.
func regenerateCDARFixtures(t *testing.T) {
	t.Helper()
	for _, f := range cdarConvertFixtures {
		var env *gobl.Envelope
		var err error
		if f.payment {
			pmt := buildSyntheticPayment(t, num.MakeAmount(25000, 2))
			env, err = gobl.Envelop(pmt)
			require.NoError(t, err, "envelope for %s", f.processCode)
			pmt.UUID = staticUUID
		} else {
			st := buildSyntheticStatus(t, f.processCode)
			env, err = gobl.Envelop(st)
			require.NoError(t, err, "envelope for %s", f.processCode)
			st.UUID = staticUUID
		}
		env.Head.UUID = staticUUID
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

// cdarFixturesNeedingRouting names the CDV fixtures whose issuer party omits
// its electronic address from the document body (as a conformant CDV may): the
// parser must hydrate the missing inbox from the transport routing. The other
// fixtures already carry both party inboxes.
var cdarFixturesNeedingRouting = map[string]bool{
	"UC1_F202500003_04-CDV-204_Prise_en_charge.xml":    true,
	"UC1_F202500003_05-CDV-205_Approuvee.xml":          true,
	"UC1_F202500003_06-CDV-211_Paiement_transmis.xml":  true,
	"UC1_F202500003_07-CDV-212_Encaissee.xml":          true,
	"UC1_F202500003_07-CDV-212_Encaissee_POUR_PPF.xml": true,
	"UC4_F202500006_04-CDV-207_En_litige.xml":          true,
	"UC5_F202500007_04-CDV-207_En_litige.xml":          true,
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

			// A conformant CDV need not repeat a party's electronic address in
			// the body (it travels at the SBD layer). For the fixtures whose
			// issuer omits it, supply the transport routing the Peppol layer
			// would provide — the two business participants — so the parser
			// hydrates the missing party inbox (BR-FR-CDV-08). Fixtures that
			// already carry both inboxes are parsed without routing.
			var parseOpts []cii.ParseOption
			if cdarFixturesNeedingRouting[inName] {
				parseOpts = append(parseOpts, cii.WithRouting(
					"0225:100000009_STATUTS",
					"0225:200000008_STATUTS",
				))
			}
			env, err := cii.Parse(xmlData, parseOpts...)
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			// Parse dispatches on the ProcessConditionCode: statuses and
			// payments (211 / 212) come back as different documents.
			switch doc := env.Extract().(type) {
			case *bill.Status:
				doc.UUID = staticUUID
			case *bill.Payment:
				doc.UUID = staticUUID
			default:
				t.Fatalf("parsed document should be a status or payment, got %T", doc)
			}
			require.NoError(t, env.Calculate())
			// PPF transmission copies (305) only carry the platform-level
			// parties (WK / DFH) plus the seller SIREN — they cannot
			// satisfy the full B2B document rules and are never parsed
			// into bill documents in production (the PPF leg stays raw
			// CDAR). The B2B fixtures must validate cleanly.
			if !strings.Contains(inName, "POUR_PPF") {
				require.NoError(t, env.Validate(), "parsed envelope must satisfy GOBL validation")
			}

			outPath := filepath.Join(getParsePath(), pathOut, outName)
			if *updateOut {
				outDir := filepath.Join(getParsePath(), pathOut)
				require.NoError(t, os.MkdirAll(outDir, 0755))
				data, err := json.MarshalIndent(env, "", "\t")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(outPath, data, 0644))
			}

			data, err := json.MarshalIndent(env.Extract(), "", "\t")
			require.NoError(t, err)

			expectedRaw, err := os.ReadFile(outPath)
			assert.NoError(t, err)

			var expectedEnv gobl.Envelope
			require.NoError(t, json.Unmarshal(expectedRaw, &expectedEnv))
			expectedData, err := json.MarshalIndent(expectedEnv.Extract(), "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Document should match the expected JSON. Update with --update flag.")
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

// TestParseRoutingFromArgs verifies that a received CDAR takes its transport
// Head.From/To from the routing args (who routed it to us), overriding GOBL's
// document-derived, outgoing (supplier->customer) assumption. The 212's
// outgoing From would be the supplier (100000009); the args here are the
// REVERSE, so seeing 200000008 as From proves the args win.
func TestParseRoutingFromArgs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(getParsePath(), "UC1_F202500003_07-CDV-212_Encaissee.xml"))
	require.NoError(t, err)

	t.Run("short-form args are canonicalized onto Head.From/To", func(t *testing.T) {
		env, err := cii.Parse(data, cii.WithRouting("0225:200000008_STATUTS", "0225:100000009_STATUTS"))
		require.NoError(t, err)
		assert.Equal(t, "iso6523-actorid-upis::0225:200000008_STATUTS", string(env.Head.From))
		assert.Equal(t, "iso6523-actorid-upis::0225:100000009_STATUTS", string(env.Head.To))
	})

	t.Run("already-qualified args are kept verbatim", func(t *testing.T) {
		env, err := cii.Parse(data, cii.WithRouting("iso6523-actorid-upis::0225:200000008_STATUTS", ""))
		require.NoError(t, err)
		assert.Equal(t, "iso6523-actorid-upis::0225:200000008_STATUTS", string(env.Head.From))
	})

}
