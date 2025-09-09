package cii_test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"
	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"

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

const (
	schemaPath     = "tools/schema"
	schematronPath = "tools/schematron"
	compiledPath   = "compiled"

	schemaFile    = "schema.xsd"
	styleFile     = "stylesheet.sef"
	defaultFormat = "en16931"
)

// updateOut is a flag that can be set to update example files
var updateOut = flag.Bool("update", false, "Update the example files in test/data")

// validateOut is a flag that can be set to validate the XML output
var validateOut = flag.Bool("validate", false, "Validate the XML output")

func TestConvertFacturXInvoice(t *testing.T) {
	testConvertInvoiceFormat(t, "facturx", cii.ContextFacturXV1)
}

func TestConvertXRechnungInvoice(t *testing.T) {
	testConvertInvoiceFormat(t, "xrechnung", cii.ContextXRechnungV3)
}

func TestConvertPeppolInvoice(t *testing.T) {
	testConvertInvoiceFormat(t, "peppol", cii.ContextPeppolV3)
}

func TestConvertEN16931Invoice(t *testing.T) {
	testConvertInvoiceFormat(t, "en16931", cii.ContextEN16931V2017)
}

func TestConvertChorusProInvoice(t *testing.T) {
	testConvertInvoiceFormat(t, "choruspro", cii.ContextChorusProV1)
}

func testConvertInvoiceFormat(t *testing.T, folder string, ctx cii.Context) {
	// Find all JSON files in this format's folder
	examples := findSourceFiles(t, filepath.Join(pathConvert, folder), pathPatternJSON)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			// Load and convert using the format-specific context
			env := loadEnvelope(t, filepath.Join(folder, inName))
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok)
			require.NoError(t, inv.RemoveIncludedTaxes())
			out, err := cii.ConvertInvoice(env, cii.WithContext(ctx))
			require.NoError(t, err)

			data, err := out.Bytes()
			require.NoError(t, err)

			if *validateOut {
				err = validateXML(data, folder)
				require.NoError(t, err)
			}

			if *updateOut {
				// Create the output directory if it doesn't exist
				outDir := filepath.Join(dataPath(), pathConvert, folder, pathOut)
				require.NoError(t, os.MkdirAll(outDir, 0755))

				err = os.WriteFile(filepath.Join(outDir, outName), data, 0644)
				require.NoError(t, err)
			}

			// Load the expected output
			output := loadOutputFile(t, filepath.Join(pathConvert, folder), outName)
			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
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
			env, err := cii.Parse(xmlData)
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
	files = append([]string{rootFolder(), "test", "data"}, files...)
	return filepath.Join(files...)
}

// ValidateXML validates an XML document against the specified format's rules
func validateXML(xmlData []byte, format string) error {
	// First validate against base EN16931 schema
	if format != defaultFormat {
		if err := validateAgainstSchema(xmlData, filepath.Join(rootFolder(), schemaPath, defaultFormat, schemaFile)); err != nil {
			return fmt.Errorf("base EN16931 validation failed: %w", err)
		}
	}

	// Then validate against format-specific schema if the schema exist
	schemaPath := filepath.Join(rootFolder(), schemaPath, format, schemaFile)
	if _, err := os.Stat(schemaPath); !errors.Is(err, os.ErrNotExist) {
		if err := validateAgainstSchema(xmlData, schemaPath); err != nil {
			return fmt.Errorf("format-specific validation failed: %w", err)
		}
	}

	// Finally run schematron validatio
	schematronPath := filepath.Join(rootFolder(), schematronPath, format, compiledPath, styleFile)
	if _, err := os.Stat(schematronPath); !errors.Is(err, os.ErrNotExist) {
		if err := validateAgainstSchematron(xmlData, schematronPath); err != nil {
			return fmt.Errorf("schematron validation failed: %w", err)
		}
	}

	return nil
}

// rootFolder returns the root folder of the project
func rootFolder() string {
	cwd, _ := os.Getwd()
	for !isrootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}
	return cwd
}

func isrootFolder(dir string) bool {
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

// FailedAssert represents a failed assertion from schematron validation
type FailedAssert struct {
	ID       string `xml:"id,attr"`
	Text     string `xml:"text"`
	Role     string `xml:"role,attr"`
	Flag     string `xml:"flag,attr"`
	Location string `xml:"location,attr"`
}

// SVRL represents the schematron validation result
type SVRL struct {
	XMLName       xml.Name       `xml:"schematron-output"`
	FailedAsserts []FailedAssert `xml:"http://purl.oclc.org/dsdl/svrl failed-assert"`
}

const (
	flagFatal = "fatal"
)

// ValidateAgainstSchema validates an XML document against the specified schema
func validateAgainstSchema(data []byte, schemaPath string) error {
	schema, err := xsd.ParseFromFile(schemaPath)
	if err != nil {
		return fmt.Errorf("loading format schema: %w", err)
	}
	xmlDoc, err := libxml2.Parse(data)
	if err != nil {
		return err
	}

	err = schema.Validate(xmlDoc)
	if err != nil {
		// Collect all errors into a single error message
		errors := err.(xsd.SchemaValidationError).Errors()
		var errorMessages []string
		for _, e := range errors {
			errorMessages = append(errorMessages, e.Error())
		}
		return fmt.Errorf("validation errors: %s", strings.Join(errorMessages, ",\n "))
	}

	return nil
}

// ValidateWithSchematron validates an XML document against the specified stylesheet
func validateAgainstSchematron(xmlData []byte, stylesheetPath string) error {
	// Create a temporary file for the XML data
	tmpFile, err := os.CreateTemp("", "validation-*.xml")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	if _, err := tmpFile.Write(xmlData); err != nil {
		return fmt.Errorf("writing to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}

	// Get the directory containing the stylesheet
	stylesheetDir := filepath.Dir(stylesheetPath)

	// Run the schematron validation using xslt3
	cmd := exec.Command(
		"xslt3",
		"-s:"+tmpFile.Name(),
		"-xsl:"+stylesheetPath,
	)

	// Set the working directory to the stylesheet directory so relative paths work
	cmd.Dir = stylesheetDir

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	// Execute the command
	err = cmd.Run()
	if err != nil {
		// Handle errors, print stderr if available
		fmt.Println("Stderr:", errOut.String())
		return fmt.Errorf("running schematron validation: %w", err)
	}

	// Parse the validation results
	var result SVRL
	if err := xml.Unmarshal(out.Bytes(), &result); err != nil {
		return fmt.Errorf("parsing schematron output: %w", err)
	}

	// Check for failed assertions
	if len(result.FailedAsserts) > 0 {
		var errors []string
		for _, assert := range result.FailedAsserts {
			if assert.Flag == flagFatal {
				errors = append(errors, fmt.Sprintf("%s: %s (location: %s)", assert.ID, assert.Text, assert.Location))
			}
		}
		if len(errors) > 0 {
			return fmt.Errorf("schematron validation failed:\n%s", strings.Join(errors, "\n"))
		}
	}

	return nil
}
