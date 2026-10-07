package sqlvalue

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeOrNil(t *testing.T) {
	require.Nil(t, TimeOrNil(sql.NullTime{}))
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	require.Equal(t, at, *TimeOrNil(sql.NullTime{Time: at, Valid: true}))
}

func TestNilIfBlankAndEmpty(t *testing.T) {
	require.Nil(t, NilIfBlank("  "))
	require.Equal(t, "a", NilIfBlank("a"))
	require.Nil(t, NilIfEmpty(""))
	require.Equal(t, " ", NilIfEmpty(" "))
}

func TestLike(t *testing.T) {
	require.Equal(t, `%A\_B\%C\\%`, Like(" a_b%c\\ "))
	require.Equal(t, "%", LikeOrAll("  "))
	require.Equal(t, `%A\_B%`, LikeOrAll("a_b"))
}

func TestTidyNumberAndEscapeLike(t *testing.T) {
	require.Equal(t, "", TidyNumber(""))
	require.Equal(t, "12", TidyNumber("12.0"))
	require.Equal(t, "7", TidyNumber("007"))
	require.Equal(t, "A1", TidyNumber("A1"))
	require.Equal(t, `50\%\_A\\B`, EscapeLike(`50%_A\B`))
}
