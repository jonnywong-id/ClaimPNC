package reportkpihttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi/repo/memory"
)

// Parameter PERSIS dari layar yang dilaporkan gagal pada 2026-10-08.
//
// # Kenapa uji ini ada
//
// Uji tab ini yang sudah ada memakai rentang satu bulan (2026-03-01..2026-03-31) dan
// lulus. Layar yang dilaporkan memakai rentang NYARIS DUA TAHUN — 2025-01-01 sampai
// 2026-08-10 — dan gagal dengan "Terjadi kesalahan pada sistem".
//
// Perbedaan satu-satunya yang terlihat adalah rentangnya. Uji ini memakai angka yang sama
// persis seperti di layar, karena uji yang memakai angka "yang mirip" tidak membuktikan
// apa pun tentang angka yang benar-benar dipakai.
func TestPICTeknikRentangPanjangSeperiLayarNyata(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet,
		"/report-kpi/pic-teknik?lini_bisnis=NONMBU&dari=2025-01-01&sampai=2026-08-10")

	require.Equal(t, http.StatusOK, rec.Code,
		"badan respons: %s", rec.Body.String())
}

// Rentang yang melintasi pergantian tahun, lebih pendek — untuk memisahkan "panjangnya
// rentang" dari "melintasi tahun" bila yang di atas gagal.
func TestPICTeknikRentangLintasTahun(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet,
		"/report-kpi/pic-teknik?lini_bisnis=NONMBU&dari=2025-12-01&sampai=2026-01-31")

	require.Equal(t, http.StatusOK, rec.Code,
		"badan respons: %s", rec.Body.String())
}
