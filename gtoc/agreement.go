package gtoc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
)

// SchemeIDEmail represents the Scheme ID for email addresses
const SchemeIDEmail = "EM"

const (
	defaultBuyerReference = "N/A"
)

// NewAgreement creates the ApplicableHeaderTradeAgreement part of a EN 16931 compliant invoice
func NewAgreement(inv *bill.Invoice) (*Agreement, error) {
	agreement := new(Agreement)
	if inv.Ordering != nil && inv.Ordering.Code != "" {
		agreement.BuyerReference = inv.Ordering.Code.String()
	} else {
		agreement.BuyerReference = defaultBuyerReference
	}
	if supplier := inv.Supplier; supplier != nil {
		agreement.Seller = NewSeller(supplier)
	}
	if customer := inv.Customer; customer != nil {
		agreement.Buyer = NewBuyer(customer)
	}
	return agreement, nil
}

// NewPostalTradeAddress creates the PostalTradeAddress part of a EN 16931 compliant invoice
func NewPostalTradeAddress(addresses []*org.Address) *PostalTradeAddress {
	if len(addresses) == 0 {
		return nil
	}
	address := addresses[0]

	postalTradeAddress := &PostalTradeAddress{
		Postcode:  address.Code,
		LineOne:   address.Street,
		LineTwo:   address.Number,
		City:      address.Locality,
		Region:    address.Region,
		CountryID: string(address.Country),
	}

	return postalTradeAddress
}

// NewEmail creates the URIUniversalCommunication part of a EN 16931 compliant invoice
func NewEmail(emails []*org.Email) *URIUniversalCommunication {
	if len(emails) == 0 {
		return nil
	}

	email := &URIUniversalCommunication{
		URIID:    emails[0].Address,
		SchemeID: SchemeIDEmail,
	}

	return email
}
