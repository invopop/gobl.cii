package cii_test

import (
	"os"
	"path/filepath"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalCDAR(t *testing.T) {
	// Get all CDAR XML files from the test data directory
	cdarFiles, err := filepath.Glob(filepath.Join(getParsePath(), "UC*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, cdarFiles, "No CDAR test files found")

	for _, file := range cdarFiles {
		fileName := filepath.Base(file)
		t.Run(fileName, func(t *testing.T) {
			// Read the XML file
			data, err := os.ReadFile(file)
			require.NoError(t, err, "Failed to read file %s", fileName)

			// Unmarshal using the generic Unmarshal function
			result, err := cii.Unmarshal(data)
			require.NoError(t, err, "Failed to unmarshal %s", fileName)

			// Verify it's a CDAR document
			cdar, ok := result.(*cii.CDAR)
			require.True(t, ok, "Expected CDAR document, got %T", result)
			require.NotNil(t, cdar, "CDAR should not be nil")

			// Basic validation - check required fields exist
			assert.NotNil(t, cdar.ExchangedDocument, "ExchangedDocument should not be nil")
			assert.NotEmpty(t, cdar.AcknowledgementDocuments, "Should have at least one AcknowledgementDocument")

			// Set namespaces for marshaling back (they're not captured during unmarshaling)
			cdar.RSMNamespace = cii.NamespaceCDARRSM
			cdar.RAMNamespace = cii.NamespaceCDARRAM
			cdar.QDTNamespace = cii.NamespaceCDARQDT
			cdar.UDTNamespace = cii.NamespaceCDARUDT

			// Verify we can marshal it back to XML
			xmlData, err := cdar.Bytes()
			require.NoError(t, err, "Failed to marshal CDAR back to XML")
			assert.NotEmpty(t, xmlData, "Marshaled XML should not be empty")
		})
	}
}

func TestUnmarshalCDARSpecific(t *testing.T) {
	// Test a specific CDAR file in detail
	testFile := filepath.Join(getParsePath(), "UC1_F202500003_01-CDV-200_Deposee.xml")

	data, err := os.ReadFile(testFile)
	require.NoError(t, err)

	cdar, err := cii.UnmarshalCDAR(data)
	require.NoError(t, err)
	require.NotNil(t, cdar)

	// Verify the structure
	require.NotNil(t, cdar.ExchangedDocument)
	require.NotEmpty(t, cdar.AcknowledgementDocuments)

	// Check ExchangedDocument details
	assert.Equal(t, "UC1_F202500003_01-CDV-200_Deposee", cdar.ExchangedDocument.Name)

	// Check context
	if cdar.ExchangedDocumentContext != nil {
		assert.NotNil(t, cdar.ExchangedDocumentContext.GuidelineParameter)
		if cdar.ExchangedDocumentContext.GuidelineParameter != nil {
			assert.Contains(t, cdar.ExchangedDocumentContext.GuidelineParameter.ID, "CDV")
		}
	}
}

func TestUnmarshalInvoice(t *testing.T) {
	// Test that UnmarshalInvoice correctly unmarshals a CII invoice
	testFile := filepath.Join(getParsePath(), "CII_example1.xml")

	data, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Use the generic Unmarshal function which should detect it's an invoice
	result, err := cii.Unmarshal(data)
	require.NoError(t, err)

	// Verify it's an Invoice document
	inv, ok := result.(*cii.Invoice)
	require.True(t, ok, "Expected Invoice document, got %T", result)
	require.NotNil(t, inv, "Invoice should not be nil")

	// Basic validation
	assert.NotNil(t, inv.ExchangedDocument, "ExchangedDocument should not be nil")
	assert.NotNil(t, inv.Transaction, "Transaction should not be nil")
}

func TestCDARRoundtrip(t *testing.T) {
	// Get all CDAR XML files for roundtrip testing
	cdarFiles, err := filepath.Glob(filepath.Join(getParsePath(), "UC*.xml"))
	require.NoError(t, err)
	require.NotEmpty(t, cdarFiles, "No CDAR test files found")

	for _, file := range cdarFiles {
		fileName := filepath.Base(file)
		t.Run(fileName, func(t *testing.T) {
			// Read original XML
			originalData, err := os.ReadFile(file)
			require.NoError(t, err, "Failed to read file %s", fileName)

			// Unmarshal
			cdar, err := cii.UnmarshalCDAR(originalData)
			require.NoError(t, err, "Failed to unmarshal %s", fileName)
			require.NotNil(t, cdar, "CDAR should not be nil")

			// Set namespaces for marshaling (they're used for output)
			cdar.RSMNamespace = cii.NamespaceCDARRSM
			cdar.RAMNamespace = cii.NamespaceCDARRAM
			cdar.QDTNamespace = cii.NamespaceCDARQDT
			cdar.UDTNamespace = cii.NamespaceCDARUDT

			// Marshal back to XML
			marshaledData, err := cdar.Bytes()
			require.NoError(t, err, "Failed to marshal CDAR back to XML")
			require.NotEmpty(t, marshaledData, "Marshaled XML should not be empty")

			// Unmarshal again to verify data integrity
			cdar2, err := cii.UnmarshalCDAR(marshaledData)
			require.NoError(t, err, "Failed to unmarshal marshaled XML")
			require.NotNil(t, cdar2, "Second CDAR should not be nil")

			// Verify key fields are preserved
			assert.Equal(t, cdar.ExchangedDocument.Name, cdar2.ExchangedDocument.Name, "Document name should match")
			assert.Equal(t, cdar.ExchangedDocument.ID, cdar2.ExchangedDocument.ID, "Document ID should match")
			assert.Equal(t, len(cdar.AcknowledgementDocuments), len(cdar2.AcknowledgementDocuments), "Number of acknowledgements should match")

			if len(cdar.AcknowledgementDocuments) > 0 && len(cdar2.AcknowledgementDocuments) > 0 {
				assert.Equal(t, cdar.AcknowledgementDocuments[0].TypeCode, cdar2.AcknowledgementDocuments[0].TypeCode, "Acknowledgement type code should match")
			}
		})
	}
}
