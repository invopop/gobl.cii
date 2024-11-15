package gtoc

import (
	"fmt"

	"github.com/invopop/gobl.cii/document"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
)

// issueDateFormat is the issue date format in the form YYYYMMDD
const issueDateFormat = "102"

// newHeader creates the ExchangedDocument part of a EN 16931 compliant invoice
func (c *Converter) newHeader(inv *bill.Invoice) error {
	h := &document.Header{
		ID:       invoiceNumber(inv.Series, inv.Code),
		TypeCode: inv.Tax.Ext[untdid.ExtKeyDocumentType].String(),
		IssueDate: &document.IssueDate{
			DateFormat: &document.Date{
				Value:  formatIssueDate(inv.IssueDate),
				Format: issueDateFormat,
			},
		},
	}
	if len(inv.Notes) > 0 {
		notes := make([]*document.Note, 0, len(inv.Notes))
		for _, n := range inv.Notes {
			notes = append(notes, &document.Note{
				Content: n.Text,
				SubjectCode: n.Code,
			})
		}
		h.IncludedNote = notes
	}
	c.doc.ExchangedDocument = h
	return nil
}

func formatIssueDate(d cal.Date) string {
	if d.IsZero() {
		return ""
	}
	t := d.Time()
	return t.Format("20060102")
}

func invoiceNumber(s cbc.Code, c cbc.Code) string {
	if s == "" {
		return s.String()
	}
	return fmt.Sprintf("%s-%s", s, c)
}
