package inboxmanagerhttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
	inboxmanagerhttp "claim-pnc/internal/inboxmanager/http"
)

const paymentRoute = route + "/payment-akseptasi/AKS-2026-0001"

func TestPaymentDetailRoute(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, paymentRoute, "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "AKS-2026-0001", body["no_akseptasi"])
	require.Equal(t, "Tertanggung Contoh", body["nama_tertanggung"])
	require.Equal(t, "1250000", body["gross_value"], "nilai dikirim sebagai teks presisi penuh")
	require.Len(t, body["pilihan_status"], 3)
	require.Len(t, body["riwayat"], 1)

	documents := body["dokumen"].([]any)
	require.Len(t, documents, 1)
	first := documents[0].(map[string]any)
	require.Equal(t, "1", first["kunci"])
	require.Equal(t, false, first["tertaut"])
	require.Equal(t, true, first["dapat_dibuka"])

	rec, body = call(t, server, http.MethodGet, route+"/payment-akseptasi/TIDAK-ADA", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodePaymentGone, body["kode"])
}

func TestPaymentDocumentRoute(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, _ := call(t, server, http.MethodGet, paymentRoute+"/dokumen/1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	// text/plain bukan jenis yang aman digambar: diunduh, dan diumumkan sebagai biner.
	require.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="lampiran-contoh.txt"`, rec.Header().Get("Content-Disposition"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	require.Equal(t, "Lampiran contoh mode memori.", rec.Body.String())

	rec, body := call(t, server, http.MethodGet, paymentRoute+"/dokumen/2", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeDocumentGone, body["kode"])

	// Lampiran lama tidak punya tautan.
	rec, body = call(t, server, http.MethodGet, paymentRoute+"/dokumen/1/tautan", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeDocumentGone, body["kode"])
}

func TestSavePaymentRoute(t *testing.T) {
	t.Run("tidak setuju tanpa remark", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, paymentRoute+"/simpan", `{"status":"2","catatan":" "}`)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		details := body["detail"].([]any)
		require.Equal(t, inboxmanager.FieldReason, details[0].(map[string]any)["field"])
	})

	t.Run("setuju ditahan", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, paymentRoute+"/simpan", `{"status":"1"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, inboxmanagerhttp.CodeApproveBlocked, body["kode"])
	})

	t.Run("badan rusak", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, _ := call(t, server, http.MethodPost, paymentRoute+"/simpan", `{`)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("tidak setuju tersimpan", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, paymentRoute+"/simpan",
			`{"status":"2","catatan":"nilai belum sesuai"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, float64(1), body["berubah"])
		require.Equal(t, true, body["penanda_adjustment"])
		require.Equal(t, false, body["surel_terkirim"], "server uji tanpa pengirim surel")

		// Nomornya tidak lagi menunggu.
		rec, _ = call(t, server, http.MethodGet, paymentRoute, "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
