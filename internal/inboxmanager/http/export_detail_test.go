package inboxmanagerhttp_test

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/repo/memory"

	inboxmanagerhttp "claim-pnc/internal/inboxmanager/http"
	portalhttp "claim-pnc/internal/portal/http"
)

func TestExportDetailMenulisBerkasPega(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, _ := call(t, server, http.MethodGet,
		route+"/ekspor-detail?dari=2026-01-01&sampai=2026-12-31", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="Data Detail Klaim.csv"`,
		rec.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	records, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3, "judul + dua baris contoh")
	require.Len(t, records[0], 90)
	require.Equal(t, "No Klaim", records[0][0])
	require.Equal(t, " Kanwil", records[0][5], "spasi di depan judul dibawa apa adanya")
	require.Equal(t, "PNC-900001", records[1][0])
	require.Equal(t, "1500000,5", records[2][31], "kolom Total OS")
}

// Tanpa satu baris pun, berkasnya tetap berjudul — bukan badan kosong.
func TestExportDetailTanpaBarisTetapBerjudul(t *testing.T) {
	server, _ := newServer(t, serverOptions{
		login: "MGR", repo: scriptedRepo{Store: newEmptyStore()},
	})

	rec, _ := call(t, server, http.MethodGet, route+"/ekspor-detail", "")
	require.Equal(t, http.StatusOK, rec.Code)
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	require.Len(t, lines, 1)
	require.True(t, strings.HasPrefix(lines[0], "No Klaim,Status Work,COB"))
}

// Penolakan tiba sebagai JSON yang dibaca layar, BUKAN sebagai berkas CSV.
func TestExportDetailMenolakSatuTanggal(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"/ekspor-detail?dari=2026-09-01", "")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeValidationFail, body["kode"])
	require.NotContains(t, rec.Header().Get("Content-Type"), "text/csv")
	detail := body["detail"].([]any)
	require.Contains(t, detail[0].(map[string]any)["pesan"], "keduanya")
}

func TestExportDetailMenuntutPortal(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	req := httptest.NewRequest(http.MethodGet, route+"/ekspor-detail", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Header().Get("Content-Type"), "text/csv")
}

func TestExportDetailGagalDiTengahDicatat(t *testing.T) {
	logs := &bytes.Buffer{}
	server, _ := newServer(t, serverOptions{login: "MGR", logs: logs})

	req := httptest.NewRequest(http.MethodGet, route+"/ekspor-detail", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	writer := &brokenWriter{header: http.Header{}}
	server.ServeHTTP(writer, req)

	require.Equal(t, http.StatusOK, writer.status)
	require.Contains(t, logs.String(), "ekspor inbox manager gagal di tengah berkas")
}

// newEmptyStore adalah penyimpanan contoh tanpa satu baris ekspor pun.
func newEmptyStore() *memory.Store {
	store := memory.NewSampleStore()
	store.SetDetailExport([]inboxmanager.DetailExportRow{})
	return store
}
