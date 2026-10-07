package apierror

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/validation"
)

type body struct{ Code string }

var errKnown = errors.New("dikenal")
var errBroken = errors.New("rusak")

func mapping(err error) (int, body, bool) {
	switch {
	case errors.Is(err, errKnown):
		return http.StatusNotFound, body{"tidak_ada"}, true
	case errors.Is(err, errBroken):
		return http.StatusInternalServerError, body{"rusak"}, true
	}
	return 0, body{}, false
}

type recorder struct {
	status   int
	body     any
	fallback error
}

func (rec *recorder) json(_ http.ResponseWriter, _ *http.Request, status int, b any) {
	rec.status, rec.body = status, b
}
func (rec *recorder) fall(_ http.ResponseWriter, _ *http.Request, err error) { rec.fallback = err }

func TestWriteMapsKnownAndLogsOnly5xx(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	r := httptest.NewRequest(http.MethodGet, "/x", nil)

	rec := &recorder{}
	Write(nil, r, errKnown, mapping, logger, rec.json, rec.fall)
	require.Equal(t, http.StatusNotFound, rec.status)
	require.Equal(t, body{"tidak_ada"}, rec.body)
	require.Empty(t, logs.String())

	Write(nil, r, errBroken, mapping, logger, rec.json, rec.fall)
	require.Equal(t, http.StatusInternalServerError, rec.status)
	require.Contains(t, logs.String(), "permintaan gagal")

	other := errors.New("lain")
	Write(nil, r, other, mapping, logger, rec.json, rec.fall)
	require.Same(t, other, rec.fallback)
}

func TestWriterFallsBackOrAnswersInternal(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := &recorder{}
	internal := body{"galat_internal"}

	withFallback := Writer(nil, rec.json, rec.fall, mapping, internal)
	other := errors.New("lain")
	withFallback(nil, r, other)
	require.Same(t, other, rec.fallback)

	var logs bytes.Buffer
	plain := Writer(slog.New(slog.NewTextHandler(&logs, nil)), rec.json, nil, mapping, internal)
	plain(nil, r, errors.New("lain"))
	require.Equal(t, http.StatusInternalServerError, rec.status)
	require.Equal(t, internal, rec.body)
	require.Contains(t, logs.String(), "permintaan gagal")

	// Tanpa logger: tetap menulis respons, tidak panik.
	Writer(nil, rec.json, nil, mapping, internal)(nil, r, errKnown)
	require.Equal(t, http.StatusNotFound, rec.status)
}

func TestContextWriter(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := &recorder{}
	internal := body{"galat_internal"}

	other := errors.New("lain")
	ContextWriter(nil, rec.json, rec.fall, mapping, internal)(nil, r, other)
	require.Same(t, other, rec.fallback)

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	ContextWriter(logger, rec.json, nil, mapping, internal)(nil, r, other)
	require.Equal(t, http.StatusInternalServerError, rec.status)
	require.Equal(t, internal, rec.body)
	require.Contains(t, logs.String(), "permintaan gagal")

	logs.Reset()
	ContextWriter(logger, rec.json, nil, mapping, internal)(nil, r, errKnown)
	require.Equal(t, http.StatusNotFound, rec.status)
	require.Empty(t, logs.String())

	ContextWriter(logger, rec.json, nil, mapping, internal)(nil, r, errBroken)
	require.Equal(t, body{"rusak"}, rec.body)
	require.Contains(t, logs.String(), "permintaan gagal")
}

func TestDetails(t *testing.T) {
	got := Details([]validation.Violation{{Field: "a", Message: "x"}, {Field: "b", Message: "y"}},
		func(field, message string) string { return field + "=" + message })
	require.Equal(t, []string{"a=x", "b=y"}, got)
	require.Empty(t, Details(nil, func(field, message string) string { return "" }))
}
