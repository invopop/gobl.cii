package cii

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanString(t *testing.T) {
	t.Run("leaves clean text untouched", func(t *testing.T) {
		in := `<a>Première vérification</a>`
		assert.Equal(t, in, cleanString(in))
	})

	t.Run("drops the replacement character", func(t *testing.T) {
		assert.Equal(t, "<a>Premire</a>", cleanString("<a>Premi�re</a>"))
	})

	t.Run("drops invalid UTF-8", func(t *testing.T) {
		assert.Equal(t, "<a>Premire</a>", cleanString("<a>Premi\xe9re</a>"))
	})

	t.Run("handles both at once", func(t *testing.T) {
		assert.Equal(t, "<a>n et Premire</a>", cleanString("<a>n\xe9 et Premi�re</a>"))
	})

	t.Run("is idempotent", func(t *testing.T) {
		in := "<a>Premi�re</a>"
		assert.Equal(t, cleanString(in), cleanString(cleanString(in)))
	})

	t.Run("preserves valid multi-byte text", func(t *testing.T) {
		in := "<a>1000 m² – 20 €</a>"
		assert.Equal(t, in, cleanString(in))
	})
}
