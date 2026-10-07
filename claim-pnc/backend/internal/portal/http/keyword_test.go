package portalhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

func TestActiveKeyword(t *testing.T) {
	var failed error
	tooLong := false
	fail := func(_ http.ResponseWriter, _ *http.Request, err error) { failed = err }
	long := func(http.ResponseWriter, *http.Request) { tooLong = true }

	_, _, ok := portalhttp.ActiveKeyword(nil, httptest.NewRequest(http.MethodGet, "/x?cari=a", nil), 5, fail, long)
	require.False(t, ok)
	require.ErrorIs(t, failed, portal.ErrNotStated)

	ctx := portalhttp.WithActivePortal(context.Background(), portal.Portal{Alias: "SPK"})
	req := httptest.NewRequest(http.MethodGet, "/x?cari="+strings.Repeat("a", 6), nil).WithContext(ctx)
	_, _, ok = portalhttp.ActiveKeyword(nil, req, 5, fail, long)
	require.False(t, ok)
	require.True(t, tooLong)

	req = httptest.NewRequest(http.MethodGet, "/x?cari=+ab+", nil).WithContext(ctx)
	active, keyword, ok := portalhttp.ActiveKeyword(nil, req, 5, fail, long)
	require.True(t, ok)
	require.Equal(t, "SPK", active.Alias)
	require.Equal(t, "ab", keyword)
}
