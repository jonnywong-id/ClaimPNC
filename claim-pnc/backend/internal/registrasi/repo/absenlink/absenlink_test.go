package absenlink_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/absenlink"
)

// TestAttendanceFollowsConnectRest menjaga bentuk permintaan rule ServiceGetDataAbsenPIC.
func TestAttendanceFollowsConnectRest(t *testing.T) {
	var path, query, user string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		user, _, _ = r.BasicAuth()
		_, _ = w.Write([]byte(`[{"Day":"SENIN","Holiday":0,"RuleTimeIn":"080000","TimeIn":"075500"}]`))
	}))
	defer srv.Close()

	h := absenlink.New(srv.URL+"/", "svc", "rahasia", nil)
	got, err := h.Attendance(context.Background(), " TONY ", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC), "PNCN.26.1")
	require.NoError(t, err)
	require.Equal(t, "/prweb/PRRestService/HCC/Absen/attendance/TONY/20261008", path)
	require.Equal(t, "caseId=PNCN.26.1", query)
	require.Equal(t, "svc", user)
	require.Equal(t, registrasi.Attendance{Day: "SENIN", Holiday: "0", RuleTimeIn: "080000", TimeIn: "075500"}, got)
}

func TestAttendanceFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := absenlink.New(srv.URL, "", "", nil).Attendance(context.Background(), "X", time.Now(), "K")
	require.ErrorContains(t, err, "HTTP 500")

	_, err = absenlink.Parse([]byte("bukan json"))
	require.ErrorContains(t, err, "bukan JSON")

	empty, err := absenlink.Parse([]byte("[]"))
	require.NoError(t, err)
	require.Equal(t, registrasi.Attendance{}, empty)
}
