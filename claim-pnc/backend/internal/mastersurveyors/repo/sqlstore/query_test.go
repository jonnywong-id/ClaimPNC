package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// namaKueri adalah seluruh kueri yang WAJIB ada.
//
// Daftarnya ditulis eksplisit, bukan dibaca dari petanya sendiri: kueri yang terhapus
// karena salah sunting harus membuat uji ini MERAH, bukan lolos karena daftarnya ikut
// menyusut.
var namaKueri = []string{
	"surveyor_list",
	"surveyor_count",
	"surveyor_get",
	"surveyor_by_name_key",
	"surveyor_by_login",
	"surveyor_site",
	"surveyor_next_sequence",
	"surveyor_insert",
	"surveyor_update",
	"surveyor_check_table",
}

func TestSeluruhKueriTerbaca(t *testing.T) {
	for _, name := range namaKueri {
		require.NotEmpty(t, strings.TrimSpace(getQuery(name)), "kueri %q kosong", name)
	}
}

// TestTidakAdaPolaSQLTerlarang menegakkan `D-20` — satu set SQL yang berjalan di Oracle
// 19c DAN PostgreSQL 17+.
//
// Ia penjaga yang disebut `docs/Steering/08-TECHNICAL-STRATEGY.md` §6: aturan yang hanya
// ada di dokumen akan dilanggar pada bulan ketiga, ketika tekanan jadwal membuat orang
// menempuh jalan pintas. Uji inilah yang gagal ketika seseorang menulis `NVL`.
func TestTidakAdaPolaSQLTerlarang(t *testing.T) {
	terlarang := map[string]string{
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET … FETCH NEXT",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "format tanggal dan angka dilakukan di Go",
		"SELECT *":  "sebutkan nama kolomnya",
		"SELECT  *": "sebutkan nama kolomnya",
	}

	for _, name := range namaKueri {
		// FROM DUAL dikecualikan HANYA pada kueri nomor urut: NEXTVAL menuntutnya, dan ia
		// sengaja diisolasi di kueri tersendiri supaya pengecualiannya sesempit mungkin.
		text := strings.ToUpper(getQuery(name))

		for pattern, saran := range terlarang {
			require.NotContains(t, text, pattern,
				"kueri %q memuat pola tidak portabel %q — %s", name, pattern, saran)
		}
	}
}

// TestFromDualHanyaDiKueriNomorUrut memastikan pengecualian dialek tidak melebar.
//
// FROM DUAL adalah satu-satunya bentuk khas Oracle di modul ini. Membiarkannya menyebar
// ke kueri lain akan membuat perpindahan ke PostgreSQL menyentuh lebih dari satu tempat —
// dan itu persis yang `D-20` cegah.
func TestFromDualHanyaDiKueriNomorUrut(t *testing.T) {
	for _, name := range namaKueri {
		text := strings.ToUpper(getQuery(name))
		if name == "surveyor_next_sequence" {
			require.Contains(t, text, "FROM DUAL")
			continue
		}
		require.NotContains(t, text, "FROM DUAL", "kueri %q tidak boleh memakai FROM DUAL", name)
	}
}

// TestKueriTulisMemakaiParameterBinding menutup celah yang ada pada pola `{ASIS:...}`
// warisan — 538 kemunculan di sistem lama.
func TestKueriTulisMemakaiParameterBinding(t *testing.T) {
	for _, name := range []string{"surveyor_insert", "surveyor_update"} {
		require.Contains(t, getQuery(name), ":1", "kueri %q wajib memakai parameter binding", name)
	}
}

// TestEkspresiKeunikanNamaSamaDenganNameKey adalah uji terpenting di berkas ini.
//
// Ekspresi `UPPER(REPLACE(NAME, ' ', ”))` di SQL harus sama artinya dengan
// mastersurveyors.NameKey di Go DAN dengan indeks unik pada migrasi 0004. Bila ketiganya
// tidak sepakat, aplikasi akan menerima nama yang kemudian ditolak basis data — dan
// pengguna melihat galat 500 alih-alih pesan yang dapat ditindaklanjuti.
//
// REPLACE ditulis dengan TIGA argumen: bentuk dua argumen sah di Oracle dan lebih
// ringkas, tetapi tidak ada di PostgreSQL.
func TestEkspresiKeunikanNamaSamaDenganNameKey(t *testing.T) {
	text := strings.ToUpper(getQuery("surveyor_by_name_key"))
	require.Contains(t, text, "UPPER(REPLACE(D.NAME, ' ', ''))")
}

// TestKueriUpdateTidakMenyentuhKolomYangDimilikiSistem menjaga empat kolom yang sengaja
// tidak boleh berubah lewat penyuntingan biasa.
//
// KOMITE yang paling penting: prasyarat aslinya `TempDetailSurveyors.KOMITE==""` pada
// langkah 6 `CNMInsertDetailSurveyors_act` berarti komite ditetapkan SEKALI. Menetapkannya
// ulang pada setiap penyuntingan akan memindahkan kewenangan memutuskan ke orang lain di
// tengah jalan.
func TestKueriUpdateTidakMenyentuhKolomYangDimilikiSistem(t *testing.T) {
	assigned := assignedColumns(getQuery("surveyor_update"))

	// Dicocokkan sebagai NAMA KOLOM UTUH, bukan substring.
	//
	// Pencocokan substring pernah dipakai di sini dan menghasilkan positif palsu:
	// "KOMITE =" ikut cocok dengan "TGL_APPROVE_KOMITE = :16", sehingga uji ini gagal
	// atas kueri yang sebenarnya benar. Itu pelajaran yang sama dengan yang dicatat
	// sistem lama — `@contains` pada toleransi spreading meloloskan `199.99`.
	for _, column := range []string{"KOMITE", "OLD_D_SURVEY_ID", "USER_INPUT", "D_SURVEY_ID"} {
		require.NotContains(t, assigned, column,
			"kolom %s tidak boleh diubah lewat surveyor_update", column)
	}

	// TRFKOMITE justru HARUS ada — ia ditandai saat keputusan komite diambil.
	require.Contains(t, assigned, "TRFKOMITE")
	// Begitu pula ketiga kolom jejak keputusan.
	require.Contains(t, assigned, "APPROVAL")
	require.Contains(t, assigned, "TGL_APPROVE_KOMITE")
	require.Contains(t, assigned, "CATATAN_KOMITE")
	require.Contains(t, assigned, "USER_UPDATE")
}

// assignedColumns mengembalikan nama kolom yang benar-benar di-SET sebuah UPDATE.
//
// Ia mengurai klausa SET menjadi daftar nama, bukan memperlakukannya sebagai teks —
// sehingga "KOMITE" tidak pernah cocok dengan "TGL_APPROVE_KOMITE".
func assignedColumns(query string) []string {
	upper := strings.ToUpper(query)

	start := strings.Index(upper, " SET ")
	if start < 0 {
		return nil
	}
	body := upper[start+len(" SET "):]
	if end := strings.Index(body, "WHERE"); end >= 0 {
		body = body[:end]
	}

	var name []string
	for _, assignment := range strings.Split(body, ",") {
		before, _, found := strings.Cut(assignment, "=")
		if !found {
			continue
		}
		if column := strings.TrimSpace(before); column != "" {
			name = append(name, column)
		}
	}
	return name
}

// TestKueriBacaMemakaiLeftJoin menjaga agar surveyor yang tipenya sudah tidak ada di
// master TETAP terbaca.
//
// INNER JOIN akan membuat barisnya HILANG dari layar tanpa satu pun pesan — cacat yang
// paling sulit disadari, karena yang salah adalah barisnya tidak muncul.
func TestKueriBacaMemakaiLeftJoin(t *testing.T) {
	for _, name := range []string{"surveyor_list", "surveyor_get", "surveyor_by_name_key", "surveyor_by_login"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "LEFT JOIN", "kueri %q wajib memakai LEFT JOIN ke M_SURVEYORS", name)
		require.NotContains(t, text, "INNER JOIN", "kueri %q tidak boleh memakai INNER JOIN", name)
	}
}

// TestKueriDaftarDanHitungMemakaiSaringanYangSama menjaga agar jumlah halaman yang
// ditampilkan cocok dengan isinya.
func TestKueriDaftarDanHitungMemakaiSaringanYangSama(t *testing.T) {
	daftar := whereOf(getQuery("surveyor_list"))
	hitung := whereOf(getQuery("surveyor_count"))
	require.Equal(t, hitung, daftar,
		"saringan surveyor_list dan surveyor_count wajib sama persis")
}

// whereOf mengambil klausa WHERE sebuah kueri, tanpa ORDER BY dan paginasinya.
func whereOf(text string) string {
	upper := strings.ToUpper(text)
	start := strings.Index(upper, "WHERE")
	if start < 0 {
		return ""
	}
	body := upper[start:]
	if end := strings.Index(body, "ORDER BY"); end >= 0 {
		body = body[:end]
	}
	return strings.Join(strings.Fields(body), " ")
}

// TestSelectorUrutanSequenceBenar menjaga agar urutan yang dipakai TIDAK tertukar dengan
// milik master tipe surveyor.
//
// Keduanya ada, keduanya bernama mirip, dan keduanya membentuk kode dengan jumlah digit
// BERBEDA — enam di sini, tiga di master tipe. Tertukar berarti kode yang terbit salah
// panjang dan bertabrakan dengan deret milik master lain.
func TestSelectorUrutanSequenceBenar(t *testing.T) {
	text := strings.ToUpper(getQuery("surveyor_next_sequence"))
	require.Contains(t, text, "D_SURVEYORS_SEQ")
	require.NotContains(t, text, "M_SURVEYORS_SEQ")
}

// TestPad6MengisiNolDiDepan menguji pembentukan bagian nomor urut pada D_SURVEY_ID.
func TestPad6MengisiNolDiDepan(t *testing.T) {
	require.Equal(t, "000001", pad6(1))
	require.Equal(t, "000042", pad6(42))
	require.Equal(t, "123456", pad6(123456))

	// Nomor yang MELEBIHI enam digit ditulis apa adanya, tidak dipotong. Memotongnya akan
	// menghasilkan ID yang bertabrakan dengan ID lama tanpa satu pun pesan.
	require.Equal(t, "1234567", pad6(1234567))
}
