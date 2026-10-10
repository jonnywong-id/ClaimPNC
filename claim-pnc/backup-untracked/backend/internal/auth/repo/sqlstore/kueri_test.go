package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji
// ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestSeluruhKueriYangDipakaiAda(t *testing.T) {
	dipakai := []string{
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
	for _, nama := range dipakai {
		t.Run(nama, func(t *testing.T) {
			require.NotPanics(t, func() { _ = ambilKueri(nama) })
			require.NotEmpty(t, strings.TrimSpace(ambilKueri(nama)))
		})
	}
}

func TestKueriYangTidakAdaMenimbulkanPanik(t *testing.T) {
	require.Panics(t, func() { _ = ambilKueri("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestKueriMematuhiDisiplinSQLPortabel(t *testing.T) {
	terlarang := map[string]string{
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

	for nama, teks := range kueri {
		hurufBesar := strings.ToUpper(teks)
		for pola, alasan := range terlarang {
			require.NotContainsf(t, hurufBesar, pola,
				"kueri %q memakai %q — %s", nama, pola, alasan)
		}
		require.NotContainsf(t, hurufBesar, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", nama)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}.
func TestKueriMemakaiParameterBinding(t *testing.T) {
	berparameter := []string{
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
	for _, nama := range berparameter {
		require.Containsf(t, ambilKueri(nama), ":1",
			"kueri %q harus memakai parameter binding", nama)
	}
}
