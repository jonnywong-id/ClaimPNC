package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji di sini berjalan TANPA basis data. Yang diperiksa adalah teks SQL-nya sendiri —
// disiplin yang `09-DATABASE-STRATEGY.md` §4 tetapkan, dan yang hanya benar-benar terjaga
// bila ada yang menggagalkannya saat dilanggar.

// namaKueri adalah seluruh kueri yang dipakai adapter ini.
//
// Daftar ini menggagalkan uji bila sebuah kueri hilang dari berkas .sql — kegagalan yang
// tanpa uji baru muncul sebagai panic saat layar dibuka.
var namaKueri = []string{
	"protection_count",
	"protection_list",
	"protection_get",
	"protection_duplicate",
	"protection_next_sequence",
	"protection_insert",
	"protection_update",
}

func TestSeluruhKueriTermuat(t *testing.T) {
	for _, nama := range namaKueri {
		require.NotEmpty(t, strings.TrimSpace(query(nama)), "kueri %q kosong atau tidak ada", nama)
	}
}

// TestKueriMengikutiDisiplinSQLPortabel menjaga `D-20`.
//
// Satu pengecualian diakui: `TO_NUMBER` pada generator nomor. `D-22` dan `D-71` menetapkan
// generator nomor sebagai SATU-SATUNYA tempat dengan sakelar dialek (`ADR-0005`), dan
// pengecualian itu sudah berlaku bagi nomor klaim.
func TestKueriMengikutiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []string{
		"NVL(",
		"SYSDATE",
		"ROWNUM",
		"DECODE(",
		"TO_CHAR(",
		"TRUNC(",
		"FROM DUAL",
		"SELECT *",
		"INSTR(",
		"LISTAGG(",
	}

	for _, nama := range namaKueri {
		teks := strings.ToUpper(query(nama))
		for _, pola := range terlarang {
			require.NotContains(t, teks, pola,
				"kueri %q memakai %s yang dilarang 09-DATABASE-STRATEGY.md §4", nama, pola)
		}
	}
}

// TestHanyaGeneratorNomorYangMemakaiTONUMBER memagari pengecualian itu supaya ia tidak
// menyebar diam-diam ke kueri lain.
func TestHanyaGeneratorNomorYangMemakaiTONUMBER(t *testing.T) {
	for _, nama := range namaKueri {
		if nama == "protection_next_sequence" {
			require.Contains(t, strings.ToUpper(query(nama)), "TO_NUMBER(")
			continue
		}
		require.NotContains(t, strings.ToUpper(query(nama)), "TO_NUMBER(",
			"hanya generator nomor yang boleh memakai TO_NUMBER (ADR-0005)")
	}
}

// TestSeluruhKueriBacaMenyaringSoftDelete menjaga `D-66`.
//
// Satu kueri yang lupa menyaringnya akan menampilkan baris yang seharusnya hilang — kelas
// cacat baru yang tidak ada di sistem lama (`09-DATABASE-STRATEGY.md` §8.1). Kegagalannya
// tidak punya gejala: layar tampil normal, hanya isinya lebih banyak dari seharusnya.
func TestSeluruhKueriBacaMenyaringSoftDelete(t *testing.T) {
	for _, nama := range []string{
		"protection_count", "protection_list", "protection_get",
		"protection_duplicate", "protection_update",
	} {
		require.Contains(t, strings.ToUpper(query(nama)), "STATUS_ACTIVE",
			"kueri %q tidak menyaring penanda hapus (D-66)", nama)
	}
}

// TestKueriDaftarMenyaringBelumDiakseptasi menjaga definisi layar ini.
//
// `Report Definition/InboxReqOpenProtection_RD-RD.xml` menyaring `.AcceptStatus IS NULL`,
// dan itu BUKAN pilihan pengguna melainkan definisi layarnya. Menghapusnya akan menampilkan
// permintaan yang sudah selesai di layar yang bukan tempatnya.
func TestKueriDaftarMenyaringBelumDiakseptasi(t *testing.T) {
	for _, nama := range []string{"protection_count", "protection_list"} {
		require.Contains(t, strings.ToUpper(query(nama)), "APPROVAL_STATUS IS NULL",
			"kueri %q harus menyaring permintaan yang belum diakseptasi", nama)
	}
}

// TestKueriGetTidakMenyaringStatusAkseptasi menjaga kebalikannya.
//
// Form harus tetap dapat dibuka untuk permintaan yang baru saja diakseptasi orang lain,
// supaya pesannya dapat menjelaskan apa yang terjadi — bukan sekadar "tidak ditemukan".
func TestKueriGetTidakMenyaringStatusAkseptasi(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("protection_get")), "APPROVAL_STATUS IS NULL")
}

// TestPenyisipanTidakMenyentuhKolomAkseptasi menjaga `P-1`.
//
// Ketiga kolom itu milik modul inboxacceptopenprotection. Menuliskannya dari sini berarti
// dua modul menulis kolom yang sama, dan akibatnya bukan galat melainkan keputusan
// akseptasi yang tertimpa.
func TestPenyisipanTidakMenyentuhKolomAkseptasi(t *testing.T) {
	for _, nama := range []string{"protection_insert", "protection_update"} {
		teks := strings.ToUpper(query(nama))
		for _, kolom := range []string{"APPROVAL_STATUS", "RESOLVED_BY", "RESOLVED_DATE_TIME"} {
			if nama == "protection_update" && kolom == "APPROVAL_STATUS" {
				// Pada UPDATE ia muncul di WHERE sebagai penjaga, bukan di SET.
				require.NotContains(t, strings.SplitN(teks, "WHERE", 2)[0], kolom,
					"kueri %q tidak boleh MENULIS %s", nama, kolom)
				continue
			}
			require.NotContains(t, teks, kolom, "kueri %q tidak boleh menyentuh %s", nama, kolom)
		}
	}
}

// TestPenyuntinganMenolakBarisYangSudahTertautKlaim menjaga aturan penguncian.
//
// `Section/InboxReqProtection_Section-Section.xml:8657` menonaktifkan tautan baris ketika
// nomor klaimnya terisi. Syaratnya ada DI DALAM WHERE, bukan hanya diperiksa di Go: yang
// menahan dua permintaan bersamaan adalah basis data.
func TestPenyuntinganMenolakBarisYangSudahTertautKlaim(t *testing.T) {
	teks := strings.ToUpper(query("protection_update"))
	require.Contains(t, teks, "CLAIM_NO IS NULL")
	require.Contains(t, teks, "APPROVAL_STATUS IS NULL")
}

// TestKueriMemakaiParameterBinding menjaga penutup celah `{ASIS:…}` warisan.
//
// Tidak ada satu pun nilai yang dirangkai ke dalam teks SQL; seluruhnya lewat penanda
// bind Oracle (`:1`, `:2`, …).
func TestKueriMemakaiParameterBinding(t *testing.T) {
	for _, nama := range namaKueri {
		teks := query(nama)
		// Tanda kutip tunggal hanya boleh muncul pada literal yang memang tetap —
		// '1' pada STATUS_ACTIVE dan '%' pada LIKE.
		for _, literal := range []string{"'1'", "'%'"} {
			teks = strings.ReplaceAll(teks, literal, "")
		}
		require.NotContains(t, teks, "'",
			"kueri %q memuat literal teks selain yang diizinkan; nilai harus lewat bind", nama)
	}
}
