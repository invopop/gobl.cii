package cii_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	cii "github.com/invopop/gobl.cii"
)

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
	env, err := cii.Parse(raw)
	if err != nil {
		return nil, err
	}
	return env.Document, nil
}
