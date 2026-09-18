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
		"statusprogres_daftar",
		"statusprogres_ambil",
		"statusprogres_daftar_id_terkunci",
		"statusprogres_sisip",
		"statusprogres_perbarui",
		"statusprogres_periksa_tabel",
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
		"statusprogres_ambil",
		"statusprogres_sisip",
		"statusprogres_perbarui",
	}
	for _, nama := range berparameter {
		require.Containsf(t, ambilKueri(nama), ":1",
			"kueri %q harus memakai parameter binding", nama)
	}
}

// ID_PROGRESS bertipe CHAR berlebar tetap (ditetapkan Work Owner 2026-09-17), sehingga
// nilainya dipadatkan spasi: "01" tersimpan sebagai "01 ".
//
// Oracle membandingkan CHAR dengan CHAR memakai blank-padded comparison — spasi di ujung
// diabaikan. Tetapi parameter binding bertipe VARCHAR2, dan CHAR lawan VARCHAR2 memakai
// NON-padded comparison: "01 " tidak sama dengan "01". Tanpa TRIM, kedua kueri ini tidak
// pernah menemukan barisnya.
//
// Uji ini menjaga TRIM tidak hilang saat seseorang "merapikan" kueri kelak, karena
// gejalanya sangat menyesatkan: tidak ada galat basis data sama sekali — penyuntingan
// hanya melaporkan "tidak ditemukan" untuk setiap baris yang sebenarnya ada.
func TestPenyaringIDMenanganiPemadatanCHAR(t *testing.T) {
	menyaringID := []string{
		"statusprogres_ambil",
		"statusprogres_perbarui",
	}
	for _, nama := range menyaringID {
		t.Run(nama, func(t *testing.T) {
			teks := strings.ToUpper(ambilKueri(nama))
			require.Contains(t, teks, "TRIM(ID_PROGRESS)",
				"penyaring ID wajib memangkas spasi padatan kolom CHAR")
			require.NotRegexp(t, `WHERE\s+ID_PROGRESS\s*=`, teks,
				"perbandingan langsung tanpa TRIM tidak akan menemukan baris apa pun")
		})
	}
}

// Kolom ID_PROGRESS tidak boleh ikut di-SET saat memperbarui: ia kunci baris, dirujuk
// GCNM_PROGRESS_CLAIM.STATUS_PROGRESS1 dan GCNM_MST_PROGRESS.ID_PROGRESS pada data klaim
// yang sudah berjalan.
func TestPerbaruiTidakMengubahKunciBaris(t *testing.T) {
	teks := strings.ToUpper(ambilKueri("statusprogres_perbarui"))
	bagianSet := teks[strings.Index(teks, "SET"):strings.Index(teks, "WHERE")]
	require.NotContains(t, bagianSet, "ID_PROGRESS",
		"ID_PROGRESS hanya boleh menyaring di WHERE, tidak pernah di-SET")
}

// Penurunan nomor baru harus menyerialkan diri; tanpa FOR UPDATE, dua penambahan
// bersamaan dapat menerima nomor yang sama.
func TestPengambilIDMenguncBaris(t *testing.T) {
	require.Contains(t, strings.ToUpper(ambilKueri("statusprogres_daftar_id_terkunci")), "FOR UPDATE")
}

// Kueri pemeriksa tabel tidak boleh mengambil satu baris pun — ia dijalankan terhadap
// produksi pada mode periksa.
func TestPeriksaTabelTidakMengambilBaris(t *testing.T) {
	require.Contains(t, ambilKueri("statusprogres_periksa_tabel"), "1 = 0")
}
