package cii

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// issueDateFormat is the issue date format in the form YYYYMMDD
const issueDateFormat = "102"

// Bytes returns the XML representation of the document in bytes
func (out *Invoice) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), bytes...), nil
}

// untdidUnit returns the UN/ECE code for a unit and its extensions. The unit
// takes priority, as it does in GOBL, and the extension answers for the codes
// it has no key for.
func untdidUnit(ext tax.Extensions, unit cbc.Key) cbc.Code {
	if code := untdid.UnitCode(unit); code != cbc.CodeEmpty {
		return code
	}
	return ext.Get(untdid.ExtKeyUnit)
}

// unitLabel describes a unit for presentation, falling back to the UN/ECE code
// when the unit has no GOBL key.
func unitLabel(unit cbc.Key, code cbc.Code) string {
	if unit != cbc.KeyEmpty {
		return unit.String()
	}
	return code.String()
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
