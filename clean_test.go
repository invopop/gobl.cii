package cii

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanXML(t *testing.T) {
	t.Run("leaves clean documents untouched", func(t *testing.T) {
		in := []byte(`<a>Première vérification</a>`)
		out := cleanXML(in)
		assert.Equal(t, in, out)
		assert.Same(t, &in[0], &out[0], "should not copy when there is nothing to clean")
	})

	t.Run("drops the replacement character", func(t *testing.T) {
		in := []byte("<a>Premi�re</a>")
		assert.Equal(t, []byte("<a>Premire</a>"), cleanXML(in))
	})

	t.Run("drops invalid UTF-8", func(t *testing.T) {
		in := []byte("<a>Premi\xe9re</a>")
		assert.Equal(t, []byte("<a>Premire</a>"), cleanXML(in))
	})

	t.Run("handles both at once", func(t *testing.T) {
		in := []byte("<a>n\xe9 et Premi�re</a>")
		assert.Equal(t, []byte("<a>n et Premire</a>"), cleanXML(in))
	})

	t.Run("is idempotent", func(t *testing.T) {
		in := []byte("<a>Premi�re</a>")
		assert.Equal(t, cleanXML(in), cleanXML(cleanXML(in)))
	})

	t.Run("preserves valid multi-byte text", func(t *testing.T) {
		in := []byte("<a>1000 m² – 20 €</a>")
		assert.Equal(t, in, cleanXML(in))
	})
}
