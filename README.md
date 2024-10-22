# GOBL.CII

GOBL conversion into Cross Industry Invoice (CII) XML format and vice versa.

Copyright [Invopop Ltd.](https://invopop.com) 2023. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses, please contact the [dev team at invopop](mailto:dev@invopop.com). To accept contributions to this library, we require transferring copyrights to Invopop Ltd.

## Usage

### Go Package

Usage of the GOBL to CII conversion library is straightforward and supports bidirectional conversion:

1. GOBL to CII XML:
   You must first have a GOBL Envelope, including an invoice, ready to convert. There are some samples in the `test/data` directory.

2. CII XML to GOBL:
   You need to have a valid CII XML document that you want to convert to GOBL format.

Both conversion directions are supported, allowing you to seamlessly transform between GOBL and CII XML formats as needed.

#### GOBL to CII

```go
package main

import (
    "os"

    "github.com/invopop/gobl"
    cii "github.com/invopop/gobl.cii"
)

func main() {
    data, _ := os.ReadFile("./test/data/invoice-sample.json")

    env := new(gobl.Envelope)
    if err := json.Unmarshal(data, env); err != nil {
        panic(err)
    }

    // Prepare the CII document
    conversor := cii.NewConversor()
    doc, err := conversor.ConvertToCII(env)
    if err != nil {
        panic(err)
    }

    // Create the XML output
    out, err := doc.Bytes()
    if err != nil {
        panic(err)
    }

}
```

#### CII to GOBL

```go
package main

import (
    "io"

    cii "github.com/invopop/gobl.cii"
    ctog "github.com/invopop/gobl.cii/ctog"
    )

func main() {
    // Read the CII XML file
	inData, err := io.ReadAll("path/to/cii_invoice.xml")
	if err != nil {
		panic(err)
	}

    doc := new(ctog.Docuemnt)
    if err := xml.Unmarshal(inData, doc); err != nil {
        panic(err)
    }

    conversor := cii.NewConversor()
    env, err := conversor.ConvertToGOBL(inData)
    if err != nil {
        panic(err)
    }

    outputData, err = json.MarshalIndent(env, "", "  ")
    if err != nil {
        panic(err)
    }
}
```

## Command Line

The GOBL to CII tool includes a command-line helper. You can install it manually in your Go environment with:

```bash
go install ./cmd/gobl.cii
```

Once installed, usage is straightforward. The tool automatically detects the input file type and performs the appropriate conversion:

- If the input is a JSON file (GOBL format), it will convert it to CII XML.
- If the input is an XML file (CII format), it will convert it to GOBL JSON.

For example:

```bash
gobl.cii convert ./test/data/invoice-sample.json
```

## Testing
### testify
The library uses testify for testing. To run the tests, you can use the following command:

```bash
go test
```

## Considerations

There are certain assumptions and lost information in the conversion from CII to GOBL that should be considered:

1. GOBL does not currently support additional embedded documents, so the AdditionalReferencedDocument field (BG-24 in EN 16931) is not supported and lost in the conversion.
2. Payment advances do not include their own tax rate, they use the global tax rate of the invoice.
3. The field TotalPrepaidAmount (BT-113) in CII is not directly mapped to GOBL, so payment advances must be included in the SpecifiedAdvancePayment field in CII, or they will be lost in conversion.
4. Fields BusinessProcessSpecifiedDocumentContextParameter (BT-10) and GuidelineSpecifiedDocumentContextParameter (BT-11) in CII are not supported and lost in the conversion.
5. The fields ReceivableSpecifiedTradeAccountingAccount (BT-133) and DesignatedProductClassification (BT-158) are added as a note to the line, with the type code as the key.
6. The field ApplicableProductCharacteristic (BG-32) is currently not supported in GOBLand lost in the conversion.
7. The GlobalID (BT-29, BT-46, BT-60,BT-157) fields are currently not directly mapped to their [codelist](https://docs.peppol.eu/poacc/billing/3.0/codelist/ICD/) ISO 6523 ICD, with their reference code being mapped to the Label.

## Development

The main source of information for this project comes from the EN 16931 standard, developed by the EU for electronic invoicing. [Part 1](https://standards.iteh.ai/catalog/standards/cen/4f31d4a9-53eb-4f1a-835e-6f0583cad2bb/en-16931-1-2017) of the standard defines the semantic data model that forms an invoice, but does not provide a concrete implementation. [Part 3.3](https://standards.iteh.ai/catalog/standards/cen/5540f673-0224-44a3-8490-feaf51aa3200/cen-ts-16931-3-3-2020) defines the mappings from the semantic data model to the CII XML format covered in this repository. 

Useful links:


- [UN/CEFACT CII](https://unece.org/trade/documents/2023/10/executive-guide-einvoicing-cross-industry-invoice)
- [CII Schemas](https://unece.org/trade/uncefact/xml-schemas-2018-2012)
