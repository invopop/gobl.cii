package gtoc

import (
	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
)

// SchemeIDEmail represents the Scheme ID for email addresses
const SchemeIDEmail = "EM"

const (
	defaultBuyerReference = "N/A"
)

// prepareAgreement creates the ApplicableHeaderTradeAgreement part of a EN 16931 compliant invoice
func (c *Converter) prepareAgreement(inv *bill.Invoice) error {
	c.doc.Transaction.Agreement = new(document.Agreement)
	agmt := c.doc.Transaction.Agreement
	if inv.Ordering != nil && inv.Ordering.Code != "" {
		agmt.BuyerReference = inv.Ordering.Code.String()
	} else {
		agmt.BuyerReference = defaultBuyerReference
	}
	if supplier := inv.Supplier; supplier != nil {
		agmt.Seller = NewParty(supplier)
	}
	if customer := inv.Customer; customer != nil {
		agmt.Buyer = NewParty(customer)
	}
	if inv.Ordering != nil {
		if inv.Ordering.Seller != nil {
			agmt.TaxRepresentative = agmt.Seller
			agmt.Seller = NewParty(inv.Ordering.Seller)
		}
		if len(inv.Ordering.Contracts) > 0 {
			c := inv.Ordering.Contracts[0].Code.String()
			agmt.Contract = &document.IssuerID{
				ID: c,
			}
		}
		if len(inv.Ordering.Purchases) > 0 {
			p := inv.Ordering.Purchases[0].Code.String()
			agmt.Purchase = &document.IssuerID{
				ID: p,
			}
		}
		if len(inv.Ordering.Sales) > 0 {
			s := inv.Ordering.Sales[0].Code.String()
			agmt.Sales = &document.IssuerID{
				ID: s,
			}
		}
		if len(inv.Ordering.Projects) > 0 {
			agmt.Project = &document.Project{
				ID: inv.Ordering.Projects[0].Code.String(),
			}
			if inv.Ordering.Projects[0].Description != "" {
				agmt.Project.Name = inv.Ordering.Projects[0].Description
			}
		}
	}
	return nil
}

func newPostalTradeAddress(addresses []*org.Address) *document.PostalTradeAddress {
	if len(addresses) == 0 {
		return nil
	}
	address := addresses[0]

	a := &document.PostalTradeAddress{
		Postcode:  address.Code.String(),
		LineOne:   address.LineOne(),
		LineTwo:   address.LineTwo(),
		City:      address.Locality,
		CountryID: string(address.Country),
		Region:    address.Region,
	}

	return a
}

func newEmail(emails []*org.Email) *document.URIUniversalCommunication {
	if len(emails) == 0 {
		return nil
	}

	e := &document.URIUniversalCommunication{
		ID: &document.PartyID{
			Value:    emails[0].Address,
			SchemeID: SchemeIDEmail,
		},
	}

	return e
}
