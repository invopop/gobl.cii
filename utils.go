package cii

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// cleanString strips the Unicode replacement character (U+FFFD) which can
// appear in badly-encoded XML documents and causes canonical JSON
// serialization to fail.
func cleanString(s string) string {
	return strings.ReplaceAll(s, "\uFFFD", "")
}

// cleanDocument strips replacement characters from every text field a parsed
// document carries. Senders whose own encoding broke upstream ship U+FFFD in
// place of the character they lost, and GOBL's canonical JSON refuses any
// document holding one, so a single field is enough to lose the whole invoice.
// Cleaning the parsed fields rather than the raw payload leaves the document's
// own bytes alone, and reaches the fields no explicit cleanString call covers.
func cleanDocument(doc any) {
	cleanValue(reflect.ValueOf(doc))
}

func cleanValue(v reflect.Value) {
	switch v.Kind() { //nolint:exhaustive // only the kinds a document can hold
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			cleanValue(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			cleanValue(v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			cleanValue(v.Index(i))
		}
	case reflect.String:
		// Unexported fields cannot be set, and need no cleaning.
		if !v.CanSet() {
			return
		}
		if s := v.String(); strings.Contains(s, "\uFFFD") {
			v.SetString(cleanString(s))
		}
	}
}

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
