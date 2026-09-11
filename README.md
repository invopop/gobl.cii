# GOBL.CII

GOBL conversion into Cross Industry Invoice (CII) XML format and vice versa.

[![codecov](https://codecov.io/gh/invopop/gobl.cii/graph/badge.svg?token=H2POAHNRT1)](https://codecov.io/gh/invopop/gobl.cii)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/invopop/gobl.cii)

Copyright [Invopop Ltd.](https://invopop.com) 2025. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses, please contact the [dev team at invopop](mailto:dev@invopop.com). To accept contributions to this library, we require transferring copyrights to Invopop Ltd.

## Usage

### Go Package

Usage of the GOBL to CII conversion library is straightforward and supports two key actions

1. Conversion of GOBL to CII XML:
   You must first have a GOBL Envelope, including an invoice, ready to convert. There are some samples in the `test/data` directory.

2. Parsing of CII XML to GOBL:
   You need to have a valid CII XML document that you want to convert to GOBL format.

Both conversion directions are supported, allowing you to seamlessly transform between GOBL and CII XML formats as needed.

#### Converting GOBL to CII Invoice

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
    doc, err := cii.ConvertInvoice(env)
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

Contexts are supported to include specific Guideline and Business rules. Available contexts include:

- `ContextEN16931V2017` (default)
- `ContextPeppolV3`
- `ContextXRechnungV3`
- `ContextChorusProV1`
- `ContextPeppolFranceFacturXV1`
- `ContextPeppolFranceCIUSV1`
- `ContextPeppolFranceExtendedV1`
- `ContextCDARFlow6`, `ContextCDARFlow6PPF`

Factur-X and ZUGFeRD have one context per profile, since BT-24 is checked
against a closed codelist per profile:

| Profile  | Factur-X                   | ZUGFeRD                    |
| -------- | -------------------------- | -------------------------- |
| BASIC    | `ContextFacturXBasicV1`    | `ContextZUGFeRDBasicV2`    |
| EN 16931 | `ContextFacturXV1`         | `ContextZUGFeRDV2`         |
| EXTENDED | `ContextFacturXExtendedV1` | `ContextZUGFeRDExtendedV2` |

MINIMUM and BASIC WL are not covered: they are not EN 16931 conformant.

Example:

```go
doc, err := cii.ConvertInvoice(env, cii.WithContext(cii.ContextXRechnungV3))
```

#### Parsing CII Invoice into GOBL

```go
package main

import (
    "io"

    cii "github.com/invopop/gobl.cii"
    )

func main() {
    // Read the CII XML file
	data, err := io.ReadAll("path/to/cii_invoice.xml")
	if err != nil {
		panic(err)
	}

    env, err := cii.Parse(data)
    if err != nil {
        panic(err)
    }

    out, err = json.MarshalIndent(env, "", "  ")
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

Usage:

```bash
gobl.cii convert <input> <output> [--context <format>]
```

The tool automatically detects the input file type (JSON/XML) and performs the appropriate conversion. Optionally specify a context format:

```bash
gobl.cii convert invoice.json invoice.xml --context xrechnung
```

## Testing

Run tests with:

```bash
go test ./...
```

To update test fixtures:

```bash
go test ./... -update
```

### Schematron validation

Beyond the golden-file comparisons, the generated XML can be pushed through the
real EN 16931 / Factur-X / XRechnung / ZUGFeRD / French CTC schematron rule
sets. Validation runs against [phorm](https://github.com/phax/phorm), the
standalone validation service that replaced the now-archived `invopop/phive`
gRPC wrapper, using the [`invopop/phorm`](https://github.com/invopop/phorm)
HTTP client.

Start a service locally:

```bash
docker run -d --name phorm -p 8080:8080 phelger/phorm
```

Use `phelger/phorm-arm64` on Apple Silicon. Note the image is `phelger/phorm`,
**not** `phax/phorm` — the latter does not exist.

It takes a few seconds to boot. It is ready once this returns HTTP 200:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  -H 'X-Token: phorm-dev-token' \
  'http://localhost:8080/api/get/vesids?include-deprecated=true'
```

Then run the suite with `-validate`:

```bash
go test ./... -validate
```

Without `-validate` the validating tests are skipped, so the plain `go test
./...` never needs a service and CI stays offline.

`PHORM_URL` and `PHORM_TOKEN` default to `http://localhost:8080` and phorm's
stock development token. Override them for a shared instance, or when port 8080
is already taken locally:

```bash
docker run -d --name phorm -p 8085:8080 phelger/phorm
PHORM_URL=http://localhost:8085 go test ./... -validate
```

All fixtures currently pass schematron across every context.

#### Notes

- **A failed validation is not an error.** phorm answers a document that breaks
  a rule with an HTTP 400 carrying the report, which it also uses for a request
  it rejects outright, so `invopop/phorm` separates the two by whether the body
  is a validation report. An error from `ValidateXml` therefore means the
  validation never ran — unreachable service, rejected token, unresolvable
  VESID, or a body that is not XML — and the tests treat it as fatal, since
  nothing was checked.
- **phorm normalises VESID versions**, so the `fr.ctc:cii:1.4.0-03` spelling in
  `context.go` resolves to its published `fr.ctc:cii:1.4-03` rule set. The
  resolved id comes back as `ves.vesid`, worth checking when a rule set behaves
  unexpectedly. The French `1.4-03` sets are already deprecated in favour of
  `1.4-04`.
- **phive-rules keeps only a rolling window of releases**, so `context.go` needs
  periodic updating; `GET /api/get/vesids?include-deprecated=true` lists what a
  given phorm build carries, along with a `deprecated` flag.

## Considerations

There are certain assumptions and lost information in the conversion from CII to GOBL that should be considered:

1. GOBL does not currently support additional embedded documents, so the AdditionalReferencedDocument field (BG-24 in EN 16931) is not supported and lost in the conversion.
2. Payment advances do not include their own tax rate, they use the global tax rate of the invoice.
3. The fields ReceivableSpecifiedTradeAccountingAccount (BT-133) and DesignatedProductClassification (BT-158) are added as a note to the line, with the type code as the key.

## Development

The main source of information for this project comes from the EN 16931 standard, developed by the EU for electronic invoicing. [Part 1](https://standards.iteh.ai/catalog/standards/cen/4f31d4a9-53eb-4f1a-835e-6f0583cad2bb/en-16931-1-2017) of the standard defines the semantic data model that forms an invoice, but does not provide a concrete implementation. [Part 3.3](https://standards.iteh.ai/catalog/standards/cen/5540f673-0224-44a3-8490-feaf51aa3200/cen-ts-16931-3-3-2020) defines the mappings from the semantic data model to the CII XML format covered in this repository.

Useful links:

- [UN/CEFACT CII](https://unece.org/trade/documents/2023/10/executive-guide-einvoicing-cross-industry-invoice)
- [CII Schemas](https://unece.org/trade/uncefact/xml-schemas-2018-2012)
