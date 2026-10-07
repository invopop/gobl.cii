# GOBL.CII

GOBL conversion into Cross Industry Invoice (CII) XML format and vice versa.

[![codecov](https://codecov.io/gh/invopop/gobl.cii/graph/badge.svg?token=H2POAHNRT1)](https://codecov.io/gh/invopop/gobl.cii)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/invopop/gobl.cii)

Copyright [Invopop Ltd.](https://invopop.com) 2025. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses, please contact the [dev team at invopop](mailto:dev@invopop.com). To accept contributions to this library, we require transferring copyrights to Invopop Ltd.

## Usage

### Go Package

Usage of the GOBL to CII conversion library is straightforward and supports bidirectional conversion:

1. Convert GOBL to CII XML:
   You must first have a GOBL Envelope, including an invoice, ready to convert. There are some samples in the `test/data` directory.

2. Parse CII XML into GOBL:
   You need to have a valid CII XML document that you want to convert to GOBL format.

The package uses the same terms as GOBL's `convert` package: **export** maps a GOBL envelope into a CII document and **import** maps it back, while **encode** and **decode** turn CII documents into XML bytes and back.

#### Export GOBL to CII

```go
package main

import (
    "encoding/json"
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

    // Export the CII document
    doc, err := cii.Export(env)
    if err != nil {
        panic(err)
    }

    // Encode the XML output
    out, err := cii.Encode(doc)
    if err != nil {
        panic(err)
    }
}
```

To export into a format other than the default EN 16931, add it as an option. `ExportInvoice` does the same when an invoice is expected:

```go
doc, err := cii.ExportInvoice(env, cii.WithFormat(cii.FormatPeppol))
```

#### Formats

This package provides the base CII import and export, and the formats that apply in any country:

| Key | Format |
| --- | --- |
| `cii+en16931` | `FormatEN16931` |
| `cii+peppol` | `FormatPeppol` |

CII invoices that declare no known specification are imported under the `cii` key, but cannot be exported.

The base import and export have no format-specific behavior. A format adds the rules of its specification with `ExportFuncs` and `ImportFuncs`, which adjust the finished document in order: an export function receives the GOBL envelope and the exported CII document, and an import function the CII document and the imported envelope, before it is calculated.

Regional formats live in their own modules, which register them with `cii.RegisterFormats` when imported, making them available to `FindFormat`, `Import`, and the GOBL `convert` register:

| Key | Module |
| --- | --- |
| `cii+peppol+fr-cius-v1`, `cii+peppol+fr-extended-v1`, `cii+peppol+fr-facturx-v1` | [gobl.fr.ctc](https://github.com/invopop/gobl.fr.ctc) (`_ "github.com/invopop/gobl.fr.ctc/cii"`) |
| `cii+fr-facturx-v1`, `cii+fr-facturx-v1+basic`, `cii+fr-facturx-v1+extended` | [gobl.fr.ctc](https://github.com/invopop/gobl.fr.ctc) (`_ "github.com/invopop/gobl.fr.ctc/cii"`) |
| `cii+fr-choruspro-v1` | [gobl.fr.ctc](https://github.com/invopop/gobl.fr.ctc) (`_ "github.com/invopop/gobl.fr.ctc/cii"`) |
| `cii+de-xrechnung-v3`, `cii+de-zugferd-v2`, `cii+de-zugferd-v2+basic`, `cii+de-zugferd-v2+extended` | [gobl.de.xinvoice](https://github.com/invopop/gobl.de.xinvoice) |

#### CDAR

`Decode` and `Encode` also handle UN/CEFACT Cross Domain Acknowledgement and Response (CDAR) documents, as a `*CDAR`. Their mapping to GOBL is specific to each specification, so it lives in the regional modules: the French CTC Flow 6 lifecycle statuses and payments are in [gobl.fr.ctc](https://github.com/invopop/gobl.fr.ctc) (`_ "github.com/invopop/gobl.fr.ctc/cdar"`), under the `cdar+peppol+fr-cdv-v1` and `cdar+fr-ppf-cdv-v1` keys.

#### Import CII to GOBL

```go
package main

import (
    "encoding/json"
    "os"

    cii "github.com/invopop/gobl.cii"
)

func main() {
    // Read the CII XML file
    data, err := os.ReadFile("path/to/cii_invoice.xml")
    if err != nil {
        panic(err)
    }

    // Decode the CII document
    doc, err := cii.Decode(data)
    if err != nil {
        panic(err)
    }

    // Import into a GOBL envelope, with the format found from the document
    env, err := cii.Import(doc)
    if err != nil {
        panic(err)
    }

    out, err := json.MarshalIndent(env, "", "  ")
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
gobl.cii convert <input> <output> [--format <key>]
```

The tool automatically detects the input file type (JSON/XML) and performs the appropriate conversion. Optionally specify the key of the export format:

```bash
gobl.cii convert invoice.json invoice.xml --format cii+peppol
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
real EN 16931 and Peppol schematron rule sets. Validation runs against [phorm](https://github.com/phax/phorm), the
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

All fixtures currently pass schematron across every format.

#### Notes

- **A failed validation is not an error.** phorm answers a document that breaks
  a rule with an HTTP 400 carrying the report, which it also uses for a request
  it rejects outright, so `invopop/phorm` separates the two by whether the body
  is a validation report. An error from `ValidateXml` therefore means the
  validation never ran — unreachable service, rejected token, unresolvable
  VESID, or a body that is not XML — and the tests treat it as fatal, since
  nothing was checked.
- **phorm normalises VESID versions**, so a `1.4.0-03` spelling resolves to its
  published `1.4-03` rule set. The
  resolved id comes back as `ves.vesid`, worth checking when a rule set behaves
  unexpectedly.
- **phive-rules keeps only a rolling window of releases**, so the VESIDs in `format.go`
  need periodic updating; `GET /api/get/vesids?include-deprecated=true` lists what a
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
