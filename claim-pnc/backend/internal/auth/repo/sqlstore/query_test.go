package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji
// ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	used := []string{
		"pengguna_ambil_by_identitas",
		"login_lokal_cari_aktif",
		"layanan_alamat",
		"pengguna_perbarui",
		"pengguna_sisip",
		"sesi_sisip",
		"sesi_ambil_by_sidik",
		"sesi_cabut",
		"sesi_perpanjang",
	}
	for _, name := range used {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = loadQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(loadQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = loadQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestQueriesFollowPortableSQLRules(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":  "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan tanggal dan angka dilakukan di Go",
		"FROM DUAL": "tidak ada padanannya di PostgreSQL",
	}

	for name, text := range queries {
		uppercase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, uppercase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, uppercase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterized := []string{
		"pengguna_ambil_by_identitas",
		"login_lokal_cari_aktif",
		"layanan_alamat",
		"pengguna_perbarui",
		"pengguna_sisip",
		"sesi_sisip",
		"sesi_ambil_by_sidik",
		"sesi_cabut",
		"sesi_perpanjang",
	}
	for _, name := range parameterized {
		require.Containsf(t, loadQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}
