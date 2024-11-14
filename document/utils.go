package document

import "encoding/xml"

// Bytes returns the XML representation of the document in bytes
func (d *Invoice) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

// HasPayment returns true if the settlement has payment. Helper function for CtoG readability
func (ah *Settlement) HasPayment() bool {
	return ah.Payee != nil ||
		(len(ah.PaymentTerms) > 0 && ah.PaymentTerms[0].DueDate != nil) ||
		(len(ah.PaymentMeans) > 0 && ah.PaymentMeans[0].TypeCode != "1")
}
