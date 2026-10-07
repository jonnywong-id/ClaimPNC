package apierror

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/validation"
)

func TestErrorShapes(t *testing.T) {
	list := []validation.Violation{{Field: "a", Message: "x"}}
	for want, got := range map[string]any{
		`[{"field":"a","pesan":"x"}]`: FieldErrors(list),
		`[{"kolom":"a","pesan":"x"}]`: ColumnErrors(list),
		`[{"isian":"a","pesan":"x"}]`: InputErrors(list),
	} {
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		require.JSONEq(t, want, string(raw))
	}
}
