package cii

import (
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Define tests for the ParseDate function
func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid date", "20230515", "2023-05-15", false},
		{"Invalid date", "20231345", "", true},
		{"Empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

// Define tests for the TypeCodeParse function
func TestTypeCodeParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Standard invoice", "380", "standard"},
		{"Credit note", "381", "credit-note"},
		{"Corrective invoice", "384", "corrective"},
		{"Proforma invoice", "325", "proforma"},
		{"Debit note", "383", "debit-note"},
		{"Unknown type code", "999", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeCodeParse(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestCleanString(t *testing.T) {
	t.Run("leaves clean text untouched", func(t *testing.T) {
		in := "Première vérification"
		assert.Equal(t, in, cleanString(in))
	})

	t.Run("drops the replacement character", func(t *testing.T) {
		// A sender emits U+FFFD when its own encoding conversion has already
		// given up. It is valid UTF-8, so it reaches gobl, where canonical
		// JSON refuses it and the document fails to digest.
		assert.Equal(t, "Premire", cleanString("Premi�re"))
	})

	t.Run("drops a decoded character reference", func(t *testing.T) {
		// &#xFFFD; in the document arrives here already decoded.
		assert.Equal(t, "bad  char", cleanString("bad � char"))
	})

	t.Run("is idempotent", func(t *testing.T) {
		in := "Premi�re"
		assert.Equal(t, cleanString(in), cleanString(cleanString(in)))
	})

	t.Run("preserves valid multi-byte text", func(t *testing.T) {
		in := "1000 m² – 20 €"
		assert.Equal(t, in, cleanString(in))
	})
}

// elementText matches an element holding non-empty text, capturing its local
// name so the injection can be done one element name at a time.
var elementText = regexp.MustCompile(`<([a-zA-Z]+:)?([A-Za-z]+)>([^<>]*[A-Za-z][^<>]*)</`)

// freeText names the CII elements that carry text a human typed, which is
// where a sender's broken encoding shows up. Codes, identifiers and other
// controlled vocabularies are deliberately excluded: they cannot carry an
// accent, so they cannot arrive mangled, and wrapping them would be noise.
var freeText = map[string]bool{
	"Content":                true, // ram:IncludedNote
	"Name":                   true, // party, item, account holder, project
	"Description":            true, // item, period, payment terms
	"Reason":                 true, // allowance / charge
	"ExemptionReason":        true, // tax exemption text (not the code)
	"Information":            true, // payment instruction detail
	"LineOne":                true, // address
	"LineTwo":                true,
	"CityName":               true,
	"CountrySubDivisionName": true,
	"PersonName":             true,
	"RequestedAction":        true, // CDAR free text
	"Location":               true,
}

// TestReplacementCharCoverage guards the cleanString calls. It injects U+FFFD
// into one free-text element at a time across every fixture and fails if the
// marker reaches GOBL, either surviving into a field or failing the digest.
//
// A new free-text field that nobody remembered to wrap shows up here rather
// than on a customer invoice.
func TestReplacementCharCoverage(t *testing.T) {
	files := fixtures(t)
	if len(files) == 0 {
		t.Fatal("no fixtures found; the guard would pass vacuously")
	}

	var gaps []gap
	seen := map[string]bool{}
	checked := 0

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if _, err := parseFixture(raw); err != nil {
			continue // fixture is not a clean baseline; it proves nothing
		}
		for _, elem := range textElements(raw) {
			if !freeText[elem] {
				continue
			}
			one := regexp.MustCompile(`(<([a-zA-Z]+:)?` + elem + `>)([^<>]*[A-Za-z][^<>]*)(</)`)
			dirty := one.ReplaceAllString(string(raw), "${1}${3}�${4}")
			if dirty == string(raw) {
				continue
			}
			checked++
			doc, err := parseFixture([]byte(dirty))
			if err != nil {
				if v := failingValue(err); v != "" {
					record(&gaps, seen, gap{elem, v, filepath.Base(f)})
				}
				continue
			}
			for _, hit := range findReplacementChar(reflect.ValueOf(doc)) {
				record(&gaps, seen, gap{elem, hit, filepath.Base(f)})
			}
		}
	}

	t.Logf("checked %d element injections across %d fixtures", checked, len(files))
	if len(gaps) == 0 {
		return
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i].elem < gaps[j].elem })
	var b strings.Builder
	fmt.Fprintf(&b, "%d element(s) carry U+FFFD into GOBL; each needs a cleanString call:\n", len(gaps))
	for _, g := range gaps {
		fmt.Fprintf(&b, "  %-28s %-46s [%s]\n", g.elem, g.value, g.src)
	}
	t.Error(b.String())
}

type gap struct{ elem, value, src string }

func record(gaps *[]gap, seen map[string]bool, g gap) {
	k := g.elem + "|" + g.value
	if seen[k] {
		return
	}
	seen[k] = true
	*gaps = append(*gaps, g)
}

func textElements(raw []byte) []string {
	set := map[string]bool{}
	for _, m := range elementText.FindAllStringSubmatch(string(raw), -1) {
		if strings.TrimSpace(m[3]) != "" {
			set[m[2]] = true
		}
	}
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// failingValue pulls the offending string out of a canonical JSON error, which
// is what names the field that went unwrapped.
func failingValue(err error) string {
	const marker = "json: unsupported value: "
	i := strings.Index(err.Error(), marker)
	if i < 0 {
		return ""
	}
	v := err.Error()[i+len(marker):]
	if len(v) > 44 {
		v = v[:44] + "…"
	}
	return v
}

func findReplacementChar(v reflect.Value) []string {
	var out []string
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, path string) {
		switch v.Kind() {
		case reflect.Pointer, reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem(), path)
			}
		case reflect.Struct:
			for i := range v.NumField() {
				if f := v.Type().Field(i); f.IsExported() {
					walk(v.Field(i), path+"."+f.Name)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := range v.Len() {
				walk(v.Index(i), path+"[]")
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				walk(v.MapIndex(k), path+"["+fmt.Sprint(k.Interface())+"]")
			}
		case reflect.String:
			if strings.ContainsRune(v.String(), '�') {
				out = append(out, path)
			}
		}
	}
	walk(v, "")
	return out
}

func fixtures(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pat := range []string{"test/data/parse/*.xml", "test/data/parse/*/*.xml"} {
		m, _ := filepath.Glob(pat)
		out = append(out, m...)
	}
	return out
}

func parseFixture(raw []byte) (any, error) {
	env, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	return env.Document, nil
}
