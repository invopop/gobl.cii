package gtoc

import (
	"encoding/xml"
)

// CFDI schema constants
const (
	RSM              = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	RAM              = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	QDT              = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	UDT              = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
	BusinessProcess  = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
	GuidelineContext = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
)

// Document is a pseudo-model for containing the XML document being created
type Document struct {
	XMLName                xml.Name     `xml:"rsm:CrossIndustryInvoice"`
	RSMNamespace           string       `xml:"xmlns:rsm,attr"`
	RAMNamespace           string       `xml:"xmlns:ram,attr"`
	QDTNamespace           string       `xml:"xmlns:qdt,attr"`
	UDTNamespace           string       `xml:"xmlns:udt,attr"`
	BusinessProcessContext string       `xml:"rsm:ExchangedDocumentContext>ram:BusinessProcessSpecifiedDocumentContextParameter>ram:ID"`
	GuidelineContext       string       `xml:"rsm:ExchangedDocumentContext>ram:GuidelineSpecifiedDocumentContextParameter>ram:ID"`
	ExchangedDocument      *Header      `xml:"rsm:ExchangedDocument"`
	Transaction            *Transaction `xml:"rsm:SupplyChainTradeTransaction"`
}
