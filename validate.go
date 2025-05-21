package cii

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
)

// FailedAssert represents a failed assertion from schematron validation
type FailedAssert struct {
	ID       string `xml:"id,attr"`
	Text     string `xml:"text"`
	Role     string `xml:"role,attr"`
	Flag     flag   `xml:"flag,attr"`
	Location string `xml:"location,attr"`
}

// SVRL represents the schematron validation result
type SVRL struct {
	XMLName       xml.Name       `xml:"schematron-output"`
	FailedAsserts []FailedAssert `xml:"http://purl.oclc.org/dsdl/svrl failed-assert"`
}

type flag string

const (
	flagFatal flag = "fatal"
)

// ValidateAgainstSchema validates an XML document against the specified schema
func ValidateAgainstSchema(data []byte, schemaPath string) error {
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
func ValidateWithSchematron(xmlData []byte, stylesheetPath string) error {
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
