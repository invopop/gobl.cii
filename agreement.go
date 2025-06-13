package cii

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
)

// Agreement defines the structure of the ApplicableHeaderTradeAgreement of the CII standard
type Agreement struct {
	BuyerReference     string                `xml:"ram:BuyerReference,omitempty"`
	Seller             *Party                `xml:"ram:SellerTradeParty,omitempty"`
	Buyer              *Party                `xml:"ram:BuyerTradeParty,omitempty"`
	TaxRepresentative  *Party                `xml:"ram:SellerTaxRepresentativeTradeParty,omitempty"`
	Sales              *IssuerID             `xml:"ram:SellerOrderReferencedDocument,omitempty"`
	Purchase           *IssuerID             `xml:"ram:BuyerOrderReferencedDocument,omitempty"`
	Contract           *IssuerID             `xml:"ram:ContractReferencedDocument,omitempty"`
	AdditionalDocument []*AdditionalDocument `xml:"ram:AdditionalReferencedDocument,omitempty"`
	Project            *Project              `xml:"ram:SpecifiedProcurringProject,omitempty"`
}

// Project defines common architecture of document reference fields in the CII standard
type Project struct {
	ID   string `xml:"ram:ID,omitempty"`
	Name string `xml:"ram:Name,omitempty"`
}

// AdditionalDocument defines the structure of AdditionalReferencedDocument of the CII standard
type AdditionalDocument struct {
	ID        string              `xml:"ram:IssuerAssignedID,omitempty"`
	TypeCode  string              `xml:"ram:TypeCode,omitempty"`
	Name      string              `xml:"ram:Name,omitempty"`
	IssueDate *FormattedIssueDate `xml:"ram:FormattedIssueDateTime,omitempty"`
}

// IssuerID defines the structure of IssuerAssignedID of the CII standard
type IssuerID struct {
	ID string `xml:"ram:IssuerAssignedID,omitempty"`
}

const (
	defaultBuyerReference = "NA"
)

// prepareAgreement creates the ApplicableHeaderTradeAgreement part of a EN 16931 compliant invoice
func (out *Invoice) addAgreement(inv *bill.Invoice) error {
	out.Transaction.Agreement = new(Agreement)
	agmt := out.Transaction.Agreement
	if inv.Ordering != nil && inv.Ordering.Code != "" {
		agmt.BuyerReference = inv.Ordering.Code.String()
	} else {
		agmt.BuyerReference = defaultBuyerReference
	}
	if supplier := inv.Supplier; supplier != nil {
		agmt.Seller = newParty(supplier)
	}
	if customer := inv.Customer; customer != nil {
		agmt.Buyer = newParty(customer)
	}
	if inv.Ordering != nil {
		if inv.Ordering.Seller != nil {
			// Reflects rules from CII-SR-282 to 291
			// These rules are warnings but have been added as they produce cleaner invoices
			agmt.TaxRepresentative = &Party{
				ID:                       agmt.Seller.ID,
				Name:                     agmt.Seller.Name,
				PostalTradeAddress:       agmt.Seller.PostalTradeAddress,
				SpecifiedTaxRegistration: agmt.Seller.SpecifiedTaxRegistration,
			}

			agmt.Seller = newParty(inv.Ordering.Seller)
		}
		if len(inv.Ordering.Contracts) > 0 {
			c := inv.Ordering.Contracts[0].Code.String()
			agmt.Contract = &IssuerID{
				ID: c,
			}
		}
		if len(inv.Ordering.Purchases) > 0 {
			p := inv.Ordering.Purchases[0].Code.String()
			agmt.Purchase = &IssuerID{
				ID: p,
			}
		}
		if len(inv.Ordering.Sales) > 0 {
			s := inv.Ordering.Sales[0].Code.String()
			agmt.Sales = &IssuerID{
				ID: s,
			}
		}
		if len(inv.Ordering.Projects) > 0 {
			agmt.Project = &Project{
				ID: inv.Ordering.Projects[0].Code.String(),
			}
			if inv.Ordering.Projects[0].Description != "" {
				agmt.Project.Name = inv.Ordering.Projects[0].Description
			}
		}
	}
	return nil
}

func newPostalTradeAddress(addresses []*org.Address) *PostalTradeAddress {
	if len(addresses) == 0 {
		return nil
	}
	address := addresses[0]

	a := &PostalTradeAddress{
		Postcode:  address.Code.String(),
		LineOne:   address.LineOne(),
		LineTwo:   address.LineTwo(),
		City:      address.Locality,
		CountryID: string(address.Country),
		Region:    address.Region,
	}

	return a
}
