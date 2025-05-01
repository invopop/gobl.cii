package cii

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
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
	flagFatal      flag   = "fatal"
	imageName      string = "saxon"
	dockerfilePath string = "/tools/saxon"
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

	// Build the Saxon Docker image if it doesn't exist
	if err := buildSaxon(); err != nil {
		return err
	}

	// Run the schematron validation
	cmd := exec.Command(
		"docker",
		"run",
		"-v",
		tmpFile.Name()+":"+tmpFile.Name(),
		"-v",
		stylesheetPath+":"+stylesheetPath, // Mount the stylesheet path and not the file as there may be extra files in the directory
		imageName,
		"-s:"+tmpFile.Name(),
		"-xsl:"+stylesheetPath,
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
			if assert.Flag == flagFatal {
				errors = append(errors, fmt.Sprintf("%s: %s (location: %s)", assert.ID, assert.Text, assert.Location))
			}
		}
		return fmt.Errorf("schematron validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func buildSaxon() error {
	// Check if the Saxon Docker image already exists
	cmd := exec.Command("docker", "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("checking for Saxon image: %w", err)
	}

	// If the image already exists, no need to build it
	if len(output) > 0 {
		return nil
	}

	// Get the path to the Dockerfile
	dockerfilePath := RootFolder() + "/tools/saxon"

	// Build the Docker image
	buildCmd := exec.Command(
		"docker",
		"build",
		"-t",
		imageName,
		dockerfilePath,
	)

	var stderr bytes.Buffer
	buildCmd.Stderr = &stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("building Saxon Docker image: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
