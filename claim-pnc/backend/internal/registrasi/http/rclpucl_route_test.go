package registrasihttp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	registrasihttp "claim-pnc/internal/registrasi/http"
)

// GET /rclpucl/dokter menyerahkan `pyPromptTableList` property `NamaDokterRCL` apa adanya.
//
// # Kenapa diuji lewat router, bukan lewat service saja
//
// Cacat yang dilaporkan dua kali oleh Work Owner adalah DROPDOWN YANG KOSONG DI LAYAR.
// Uji pada tingkat service sudah hijau sejak daftarnya pindah ke `registrasi`, dan tetap
// hijau seandainya rute ini tidak terdaftar, atau handler-nya mengisi `nama` dengan nilai
// kosong, atau amplopnya berganti nama field. Ketiganya menghasilkan layar yang persis
// sama kosongnya, dan tidak satu pun tertangkap di sana.
//
// Yang diperiksa di sini karena itu bentuk yang benar-benar dilihat layar: status, nama
// field amplop (`pilihan`), dan kedua field tiap barisnya.
func TestRouteDokterRCLMenyerahkanPromptList(t *testing.T) {
	e := newHTTPEnv(t)

	w := e.do(t, http.MethodGet, "/registrasi/rclpucl/dokter", nil)
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())

	var got registrasihttp.RCLDoctorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	// Urutan prompt list, bukan abjad: `WAHYUKRISTANTI` lebih dulu.
	require.Equal(t, []registrasihttp.RCLDoctorDTO{
		{ID: "WAHYUKRISTANTI", Nama: "WAHYUKRISTANTI"},
		{ID: "MARGARETHAROSAGUNAWAN", Nama: "MARGARETHA ROSA GUNAWAN"},
	}, got.Pilihan)

	// Tidak kosong, dan tidak bergantung pada basis data — inilah inti perbaikannya.
	// Sebelumnya daftarnya kueri ke `T_ACCESS_GROUP_PNC` yang tidak mengembalikan satu
	// baris pun, sehingga layar hanya menggambar "----- PILIH -----".
	require.NotEmpty(t, got.Pilihan)
}
