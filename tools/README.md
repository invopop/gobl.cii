# CII Tools

This directory contains various tools used for CII (Cross Industry Invoice) validation and processing. Below you'll find information about each tool and its components. Versions reffer to code release and only revert back to format version when no other option is available.

## Schema Tools

The schema directory contains XML Schema Definition (XSD) files for different CII implementations. These schemas define the structure and validation rules for CII documents.

### Schema Versions

| Implementation | Version | CII Version | Notes | Source |
|---------------|---------|-------------|-------|------|
| EN16931 | 1.3.14.1 | D16B | Current version used by Connecting Europe | [EN16931 Schema](https://github.com/ConnectingEurope/eInvoicing-EN16931/tree/master/cii/schema/D16B%20SCRDM%20(Subset)/uncoupled%20clm/CII/uncefact) |
| Factur-X (Extended) | 1.07.2 | D22B | We are checking against CII D16B for simplicity, only change is in BG-3 cardinality | [Factur-X Schema](https://fnfe-mpe.org/factur-x/factur-x-et-zugferd-2-2/) |
| XRechnung | - | D16B | Xrechnung uses standard CII | [XRechnung Info](https://github.com/itplr-kosit/validator-configuration-xrechnung) |
| Peppol | - | D16B | Peppol uses standard CII | [PEPPOL Info](https://docs.peppol.eu/poacc/billing/3.0/bis/#_cross_industry_invoice) |

## Schematron Tools

The schematron directory contains Schematron rules for validating CII documents according to different implementation guidelines. These rules provide additional validation beyond what's possible with XSD alone. 

### Schematron Versions

| Implementation | Version | CII Version | Notes | Source |
|---------------|---------|-------------|-------|------|
| EN16931 | 1.3.14.1 | D16B | Current version used by Connecting Europe | [EN16931 Schematron](https://github.com/ConnectingEurope/eInvoicing-EN16931/tree/master/cii/xslt) |
| Factur-X (Extended)| 1.07.2 | D22B | Adds rules like SIRET location for Chorus Pro | [Factur-X Schematron](https://fnfe-mpe.org/factur-x/factur-x-et-zugferd-2-2/) |
| XRechnung | 2.3.0 | D16B | German-specific rules, include some PEPPOL rules | [XRechnung Schematron](https://github.com/itplr-kosit/xrechnung-schematron/releases/tag/release-2.3.0) |
| PEPPOL | 3.0.18 | D16B | Adds PEPPOL specific rules | [PEPPOL Schematron](https://github.com/OpenPEPPOL/peppol-bis-invoice-3/tree/master/rules/sch) |

#### Compilation

We prefer downloading the precompiled XSLT rather than compiling SCH ourselves. Peppol is the only schematron that needs to be compiled by us to XSLT. A they focus on UBL they only provide that stylesheet. To compile the schematron to a stylesheet we need to follow these steps:

1. Clone https://github.com/OpenPEPPOL/peppol-bis-invoice-3.git
2. Download `schxslt-1.10.1-xslt-only.zip` from [schxslt](https://github.com/schxslt/schxslt/releases)
4. Run the following command:
```bash
xslt3 -s:peppol-bis-invoice-3/rules/sch/PEPPOL-EN16931-CII.sch -xsl:schxslt-1.10.1/2.0/pipeline-for-svrl.xsl -o:stylesheet.xslt	
```
*More information regarding XSLT3 is provided bellow*

To increase the speed at which tests are run we have precompiled XSLT to SEF files which can be used by xslt3. 
To do this we need to run this command:
```bash
xslt3  -xsl:stylesheet.xsl -export:compiled/stylesheet.sef -nogo -relocate:yes
```
`-relocate` is specially important for facturx

## xslt3

The validation process uses `xslt3`, an npm package that provides Saxon-JS functionality for XML transformations without needing to install Java.

To use xslt3 we edited lines 9590 and 9695 in facturx/stylesheet.xslt to fix number handling errors caused by using javascript. 

### Installation

To use the validation tools, you need to install `xslt3` globally:

```bash
npm install -g xslt3
```

### Usage Notes

1. Schema files are used for basic structural validation of CII documents
2. Schematron rules provide additional business rule validation
3. xslt3 is used for XML transformations and processing
4. All tools are designed to work together to provide comprehensive CII validation
5. The validation process follows these steps:
   - First validates against the base EN16931 schema
   - Then validates against format-specific schema if available
   - Finally runs schematron validation using xslt3
6. Temporary files are used during validation to handle the XML data
7. Error messages from validation are collected and formatted for easy debugging

### Usage

You can run xslt3 directly from the command line with the following basic command:

```bash
xslt3 -s:input.xml -xsl:transform.xsl -o:output.xml
```

Parameters:
- input.xml: The XML file you want to transform
- transform.xsl: The XSLT file to apply to the XML file
- output.xml: The output file where the transformed XML will be written to (optional)

### Calling xslt3 from Go

You can call xslt3 programmatically from Go using the exec package. Here's an example that captures the output of the command:

```go
package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func main() {
	// Define the xslt3 command with parameters
	cmd := exec.Command(
		"xslt3",
		"-s:input.xml",
		"-xsl:transform.xsl",
	)

	// Create buffers to capture stdout and stderr
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	// Execute the command
	err := cmd.Run()
	if err != nil {
		// Handle errors, print stderr if available
		fmt.Printf("Error running command: %v\n", err)
		fmt.Println("Stderr:", errOut.String())
		return
	}

	// Print the standard output captured from the command
	fmt.Println("Output:", out.String())
}
```

## Dependencies

- Node.js and npm for xslt3
- XML processing libraries for schema validation