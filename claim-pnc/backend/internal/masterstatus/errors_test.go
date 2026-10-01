package masterstatus_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
)

func TestNewValidationErrorNilWithoutViolation(t *testing.T) {
	require.NoError(t, masterstatus.NewValidationError(nil))
}

func TestValidationErrorJoinsEveryViolation(t *testing.T) {
	err := masterstatus.NewValidationError([]masterstatus.Violation{{Field: "a", Message: "x"}, {Field: "b", Message: "y"}})
	require.EqualError(t, err, "masterstatus: validasi gagal — a: x; b: y")
}
