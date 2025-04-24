package cii

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
)

const (
	schemaPath     = "test/tools/schema"
	schematronPath = "test/tools/schematron"
	transformPath  = "test/tools/saxon/transform"

	schemaFile = "schema.xsd"
	styleFile  = "stylesheet.xslt"
)

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

// ValidateXML validates an XML document against the specified format's rules
func ValidateXML(xmlData []byte, format string) error {

	// First validate against base CII schema
	if format != "cii" {
		baseSchema, err := xsd.ParseFromFile(filepath.Join(RootFolder(), schemaPath, "cii", schemaFile))
		if err != nil {
			return fmt.Errorf("loading base CII schema: %w", err)
		}
		if err := validateAgainstSchema(baseSchema, xmlData); err != nil {
			return fmt.Errorf("base CII validation failed: %w", err)
		}
	}

	// Then validate against format-specific schema if the schema exists
	path := filepath.Join(RootFolder(), schemaPath, format, schemaFile)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		formatSchema, err := xsd.ParseFromFile(path)
		if err != nil {
			return fmt.Errorf("loading format schema: %w", err)
		}
		if err := validateAgainstSchema(formatSchema, xmlData); err != nil {
			return fmt.Errorf("format-specific validation failed: %w", err)
		}
	}

	// Finally run schematron validation
	if err := validateWithSchematron(xmlData, filepath.Join(RootFolder(), schematronPath, format)); err != nil {
		return fmt.Errorf("schematron validation failed: %w", err)
	}

	return nil
}

func validateAgainstSchema(schema *xsd.Schema, data []byte) error {
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

func validateWithSchematron(xmlData []byte, stylesheetPath string) error {
	// Create a temporary file for the XML data
	tmpFile, err := os.CreateTemp("", "validation-*.xml")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(xmlData); err != nil {
		return fmt.Errorf("writing to temporary file: %w", err)
	}
	tmpFile.Close()

	// Run the schematron validation
	cmd := exec.Command(
		"docker",
		"run",
		"-v",
		tmpFile.Name()+":"+tmpFile.Name(),
		"-v",
		stylesheetPath+":"+stylesheetPath, // Mount the stylesheet path and not the file as there may be extra files in the directory
		"saxon",                           // TODO: update this when we publish the docker image
		"-s:"+tmpFile.Name(),
		"-xsl:"+filepath.Join(stylesheetPath, styleFile),
	)

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
			errors = append(errors, fmt.Sprintf("%s: %s (location: %s)", assert.ID, assert.Text, assert.Location))
		}
		return fmt.Errorf("schematron validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}
