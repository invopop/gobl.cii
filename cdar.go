package cii

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"github.com/invopop/xmlctx"
)

// CDAR namespaces
const (
	NamespaceCDARRSM = "urn:un:unece:uncefact:data:standard:CrossDomainAcknowledgementAndResponse:100"
)

// CDAR represents the root structure for Cross Domain Acknowledgement and Response
type CDAR struct {
	XMLName                  xml.Name               `xml:"rsm:CrossDomainAcknowledgementAndResponse"`
	RSMNamespace             string                 `xml:"xmlns:rsm,attr"`
	RAMNamespace             string                 `xml:"xmlns:ram,attr"`
	QDTNamespace             string                 `xml:"xmlns:qdt,attr"`
	UDTNamespace             string                 `xml:"xmlns:udt,attr"`
	ExchangedDocumentContext *CDARExchangedContext  `xml:"rsm:ExchangedDocumentContext,omitempty"`
	ExchangedDocument        *CDARExchangedDocument `xml:"rsm:ExchangedDocument"`
	AcknowledgementDocuments []*CDARAcknowledgement `xml:"rsm:AcknowledgementDocument"`
}

// NewCDAR creates a new CDAR document with the necessary namespaces
func NewCDAR() *CDAR {
	return &CDAR{
		RSMNamespace: NamespaceCDARRSM,
		RAMNamespace: NamespaceRAM,
		QDTNamespace: NamespaceQDT,
		UDTNamespace: NamespaceUDT,
	}
}

// decodeCDAR unmarshals a raw XML CDAR document into a CDAR struct
func decodeCDAR(data []byte) (*CDAR, error) {
	cdar := new(CDAR)
	if err := xmlctx.Unmarshal(data, cdar, xmlctx.WithNamespaces(
		map[string]string{
			nsPrefixRSM: NamespaceCDARRSM,
			nsPrefixRAM: NamespaceRAM,
			nsPrefixQDT: NamespaceQDT,
			nsPrefixUDT: NamespaceUDT,
		},
	)); err != nil {
		return nil, fmt.Errorf("error unmarshaling CDAR: %w", err)
	}

	return cdar, nil
}

// encode converts the CDAR document to XML bytes
func (c *CDAR) encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteString(xml.Header)

	encoder := xml.NewEncoder(buf)
	encoder.Indent("", "  ")

	if err := encoder.Encode(c); err != nil {
		return nil, fmt.Errorf("error encoding CDAR: %w", err)
	}

	return buf.Bytes(), nil
}
