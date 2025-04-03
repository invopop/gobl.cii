package cii

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
)

// issueDateFormat is the issue date format in the form YYYYMMDD
const issueDateFormat = "102"

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

func documentDate(date *cal.Date) *Date {
	if date == nil {
		return nil
	}
	return &Date{
		Value:  formatIssueDate(*date),
		Format: issueDateFormat,
	}
}

func formatIssueDate(d cal.Date) string {
	if d.IsZero() {
		return ""
	}
	t := d.Time()
	return t.Format("20060102")
}

// parseDate converts a date string to a cal.Date
func parseDate(date string) (cal.Date, error) {
	t, err := time.Parse("20060102", date)
	if err != nil {
		return cal.Date{}, err
	}

	return cal.MakeDate(t.Year(), t.Month(), t.Day()), nil
}

func invoiceNumber(s cbc.Code, c cbc.Code) string {
	if s == "" {
		return c.String()
	}
	return fmt.Sprintf("%s-%s", s, c)
}
