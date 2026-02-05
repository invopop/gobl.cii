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
	NamespaceCDARRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	NamespaceCDARQDT = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	NamespaceCDARUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
)

var (
	// ErrUnknownCDARDocumentType is returned when the document type
	// is not recognized during CDAR parsing.
	ErrUnknownCDARDocumentType = fmt.Errorf("unknown CDAR document type")
)

// CDAR represents the root structure for Cross Domain Acknowledgement and Response
type CDAR struct {
	XMLName                  xml.Name                  `xml:"rsm:CrossDomainAcknowledgementAndResponse"`
	RSMNamespace             string                    `xml:"xmlns:rsm,attr"`
	RAMNamespace             string                    `xml:"xmlns:ram,attr"`
	QDTNamespace             string                    `xml:"xmlns:qdt,attr"`
	UDTNamespace             string                    `xml:"xmlns:udt,attr"`
	ExchangedDocumentContext *CDARExchangedContext     `xml:"rsm:ExchangedDocumentContext,omitempty"`
	ExchangedDocument        *CDARExchangedDocument    `xml:"rsm:ExchangedDocument"`
	AcknowledgementDocuments []*CDARAcknowledgement    `xml:"rsm:AcknowledgementDocument"`
}

// NewCDAR creates a new CDAR document with the necessary namespaces
func NewCDAR() *CDAR {
	return &CDAR{
		RSMNamespace: NamespaceCDARRSM,
		RAMNamespace: NamespaceCDARRAM,
		QDTNamespace: NamespaceCDARQDT,
		UDTNamespace: NamespaceCDARUDT,
	}
}

// UnmarshalCDAR unmarshals a raw XML CDAR document into a CDAR struct
func UnmarshalCDAR(data []byte) (*CDAR, error) {
	ns, err := extractRootNamespace(data)
	if err != nil {
		return nil, err
	}

	if ns != NamespaceCDARRSM {
		return nil, ErrUnknownCDARDocumentType
	}

	cdar := new(CDAR)
	if err := xmlctx.Unmarshal(data, cdar, xmlctx.WithNamespaces(
		map[string]string{
			"rsm": NamespaceCDARRSM,
			"ram": NamespaceCDARRAM,
			"qdt": NamespaceCDARQDT,
			"udt": NamespaceCDARUDT,
		},
	)); err != nil {
		return nil, fmt.Errorf("error unmarshaling CDAR: %w", err)
	}

	return cdar, nil
}

// Bytes converts the CDAR document to XML bytes
func (c *CDAR) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteString(xml.Header)

	encoder := xml.NewEncoder(buf)
	encoder.Indent("", "  ")

	if err := encoder.Encode(c); err != nil {
		return nil, fmt.Errorf("error encoding CDAR: %w", err)
	}

	return buf.Bytes(), nil
}

// String converts the CDAR document to an XML string
func (c *CDAR) String() (string, error) {
	data, err := c.Bytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}
