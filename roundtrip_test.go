package cii_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cii "github.com/invopop/gobl.cii"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cdarXML declares neither qdt nor udt.
const cdarXML = `<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossDomainAcknowledgementAndResponse xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossDomainAcknowledgementAndResponse:100" xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
  <rsm:ExchangedDocument><ram:ID>CDAR-1</ram:ID></rsm:ExchangedDocument>
</rsm:CrossDomainAcknowledgementAndResponse>`

func TestDecodeEncodeRoundTrip(t *testing.T) {
	docs := map[string][]byte{"inline/cdar.xml": []byte(cdarXML)}
	err := filepath.WalkDir(getDataPath(), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".xml" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(getDataPath(), path)
		docs[rel] = data
		return nil
	})
	require.NoError(t, err)

	for name, data := range docs {
		t.Run(name, func(t *testing.T) {
			first, err := cii.Decode(data)
			require.NoError(t, err)
			out, err := cii.Encode(first)
			require.NoError(t, err)

			assertValidRoot(t, out)
			assertPrefixesBound(t, out)

			second, err := cii.Decode(out)
			require.NoError(t, err)
			assert.Equal(t, first, second, "decoding the output gives the same document")

			again, err := cii.Encode(second)
			require.NoError(t, err)
			assert.Equal(t, string(out), string(again), "encoding is stable")
		})
	}
}

// assertValidRoot checks that the root element repeats no attribute and
// declares no prefix as empty, which XML 1.0 forbids.
func assertValidRoot(t *testing.T, data []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.RawToken()
		require.NoError(t, err)
		root, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		seen := make(map[xml.Name]bool)
		for _, a := range root.Attr {
			assert.False(t, seen[a.Name], "attribute %s:%s is repeated", a.Name.Space, a.Name.Local)
			seen[a.Name] = true
			if a.Name.Space == "xmlns" {
				assert.NotEmpty(t, a.Value, "xmlns:%s is declared empty", a.Name.Local)
			}
		}
		return
	}
}

// assertPrefixesBound checks that every element prefix is declared. The
// decoder leaves an undeclared prefix as the name's space instead of a URI.
func assertPrefixesBound(t *testing.T, data []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return
		}
		require.NoError(t, err)
		if el, ok := tok.(xml.StartElement); ok {
			require.True(t, strings.Contains(el.Name.Space, ":"), "prefix %q of <%s> is not declared", el.Name.Space, el.Name.Local)
		}
	}
}
