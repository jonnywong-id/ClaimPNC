package csvexport

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBeginDownload(t *testing.T) {
	rec := httptest.NewRecorder()
	BeginDownload(rec, "a.csv")
	require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="a.csv"`, rec.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

func TestTruncationNotice(t *testing.T) {
	require.Empty(t, TruncationNotice(0, 5, 9, ""))
	require.Equal(t, []string{"-- Terpotong pada 5 baris dari 9 yang cocok. --", ""}, TruncationNotice(2, 5, 9, ""))
	require.Equal(t, []string{"-- Terpotong pada 5 baris dari 9 yang cocok. Persempit. --"}, TruncationNotice(1, 5, 9, " Persempit."))
}

func TestLogFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	LogFailure(httptest.NewRequest(http.MethodGet, "/x", nil), logger, "putus", errors.New("rusak"))
	require.Contains(t, buf.String(), "putus")
	require.Contains(t, buf.String(), "jalur=/x")
	require.Contains(t, buf.String(), "galat=rusak")
}

func paged(next func(int) ([]string, int, error), fail *[]error) Paged[string] {
	return Paged[string]{
		Header: []string{"h"},
		Limit:  3,
		Notice: func(width, total int) []string { return TruncationNotice(width, 3, total, "") },
		Row:    func(s string) []string { return []string{s} },
		Next:   next,
		Fail:   func(err error) { *fail = append(*fail, err) },
	}
}

func TestPagedWritesAllChunks(t *testing.T) {
	var fail []error
	rec := httptest.NewRecorder()
	paged(func(page int) ([]string, int, error) {
		require.Equal(t, 2, page)
		return []string{"c"}, 3, nil
	}, &fail).Write(rec, []string{"a", "b"}, 3)
	require.Empty(t, fail)
	require.Equal(t, "h\na\nb\nc\n", rec.Body.String())
}

func TestPagedTruncatesAtLimit(t *testing.T) {
	var fail []error
	rec := httptest.NewRecorder()
	paged(func(page int) ([]string, int, error) { return []string{"d", "e"}, 9, nil }, &fail).Write(rec, []string{"a", "b"}, 9)
	require.Equal(t, "h\na\nb\nd\n-- Terpotong pada 3 baris dari 9 yang cocok. --\n", rec.Body.String())
}

func TestPagedStopsOnEmptyChunkAndReportsErrors(t *testing.T) {
	var fail []error
	rec := httptest.NewRecorder()
	paged(func(page int) ([]string, int, error) { return nil, 5, nil }, &fail).Write(rec, []string{"a"}, 5)
	require.Equal(t, "h\na\n", rec.Body.String())

	paged(func(page int) ([]string, int, error) { return nil, 0, errors.New("mati") }, &fail).Write(httptest.NewRecorder(), []string{"a"}, 5)
	require.EqualError(t, fail[0], "mati")
}

type brokenWriter struct{ http.ResponseWriter }

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("putus") }

func TestPagedReportsWriteFailures(t *testing.T) {
	var fail []error
	p := paged(func(int) ([]string, int, error) { return nil, 0, nil }, &fail)
	p.Header = []string{strings.Repeat("x", 5000)}
	p.Write(brokenWriter{httptest.NewRecorder()}, []string{"a"}, 1)
	require.Len(t, fail, 1)

	fail = nil
	p.Header = []string{"h"}
	p.Write(brokenWriter{httptest.NewRecorder()}, []string{"a"}, 1)
	require.Len(t, fail, 1)
}
