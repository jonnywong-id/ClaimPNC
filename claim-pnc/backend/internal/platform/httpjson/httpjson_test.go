package httpjson

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type body struct {
	Name string `json:"nama"`
}

type sink struct {
	status int
	body   any
}

func (s *sink) write(_ http.ResponseWriter, _ *http.Request, status int, b any) { s.status, s.body = status, b }

func run(t *testing.T, loose bool, payload string, max int64) (bool, body, *sink) {
	t.Helper()
	s := &sink{}
	var target body
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	decode := Decode
	if loose {
		decode = DecodeLoose
	}
	ok := decode(httptest.NewRecorder(), r, max, &target, s.write, "cacat")
	return ok, target, s
}

func TestDecodeStrict(t *testing.T) {
	ok, got, _ := run(t, false, `{"nama":"a"}`, 1<<10)
	require.True(t, ok)
	require.Equal(t, "a", got.Name)

	for _, payload := range []string{`{`, `{"lain":1}`, `{"nama":"a"}{}`, `{"nama":"` + strings.Repeat("x", 64) + `"}`} {
		ok, _, s := run(t, false, payload, 32)
		require.False(t, ok, payload)
		require.Equal(t, http.StatusBadRequest, s.status)
		require.Equal(t, "cacat", s.body)
	}
}

func TestDecodeLooseIgnoresUnknownAndTrailing(t *testing.T) {
	ok, got, _ := run(t, true, `{"nama":"a","lain":1}{}`, 1<<10)
	require.True(t, ok)
	require.Equal(t, "a", got.Name)

	ok, _, s := run(t, true, `{`, 1<<10)
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, s.status)
}
