package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji di sini berjalan TANPA basis data. Yang diperiksa adalah teks SQL-nya sendiri.

var namaKueri = []string{
	"acceptance_count",
	"acceptance_list",
	"acceptance_get",
	"acceptance_decide",
}

func TestSeluruhKueriTermuat(t *testing.T) {
	for _, nama := range namaKueri {
		require.NotEmpty(t, strings.TrimSpace(query(nama)), "kueri %q kosong atau tidak ada", nama)
	}
}

// TestKueriMengikutiDisiplinSQLPortabel menjaga `D-20`.
//
// Modul ini TIDAK punya pengecualian: ia tidak menerbitkan nomor, sehingga tidak ada alasan
// memakai konstruksi khas Oracle mana pun.
func TestKueriMengikutiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []string{
		"NVL(", "SYSDATE", "ROWNUM", "DECODE(", "TO_CHAR(", "TO_NUMBER(",
		"TRUNC(", "FROM DUAL", "SELECT *", "INSTR(", "LISTAGG(",
	}

	for _, nama := range namaKueri {
		teks := strings.ToUpper(query(nama))
		for _, pola := range terlarang {
			require.NotContains(t, teks, pola,
				"kueri %q memakai %s yang dilarang 09-DATABASE-STRATEGY.md §4", nama, pola)
		}
	}
}

// TestAntreanMenyaringTigaSyaratDefinisiLayar menjaga apa yang membedakan layar ini dari
// layar pemohon.
//
// `InboxOpenProtection2_RD` menyaring `CaseID IS NOT NULL AND PolicyNo IS NOT NULL AND
// AcceptStatus IS NULL`. Menghilangkan syarat pertama akan menarik permintaan rancangan —
// yang belum tertaut klaim — ke meja petugas akseptasi.
func TestAntreanMenyaringTigaSyaratDefinisiLayar(t *testing.T) {
	for _, nama := range []string{"acceptance_count", "acceptance_list", "acceptance_decide"} {
		teks := strings.ToUpper(query(nama))
		require.Contains(t, teks, "APPROVAL_STATUS IS NULL", "kueri %q", nama)
		require.Contains(t, teks, "CLAIM_NO IS NOT NULL", "kueri %q", nama)
		require.Contains(t, teks, "POLICY_NO IS NOT NULL", "kueri %q", nama)
	}
}

// TestGetTidakMenyaringStatusKeputusan menjaga kebalikannya.
//
// Antrean ini BERSAMA: sebuah baris dapat diputuskan petugas lain kapan saja. Form harus
// tetap terbuka supaya pesannya dapat menyatakan keputusan siapa dan kapan.
func TestGetTidakMenyaringStatusKeputusan(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("acceptance_get")), "APPROVAL_STATUS IS NULL")
}

// TestKeputusanHanyaMenulisTigaKolom menjaga `P-1`.
//
// Kolom pembuatan dimiliki modul inputreqprotection. Menuliskannya dari sini berarti dua
// modul menulis kolom yang sama — dan akibatnya bukan galat, melainkan keterangan pemohon
// yang tertimpa saat petugas mengakseptasi.
func TestKeputusanHanyaMenulisTigaKolom(t *testing.T) {
	teks := strings.ToUpper(query("acceptance_decide"))
	bagianSet := strings.SplitN(teks, "WHERE", 2)[0]

	for _, kolom := range []string{
		"POLICY_NO", "CLAIM_NO", "ID_CLAIM", "PROTECTION_TYPE", "CREATE_DATE", "CREATED_BY",
		"NOTES", "OLD_DATA", "NEW_DATA", "OBJECT_NAME", "BRANCH_NAME", "STATUS_ACTIVE",
	} {
		require.NotContains(t, bagianSet, kolom,
			"keputusan akseptasi tidak boleh menulis %s (P-1)", kolom)
	}

	for _, kolom := range []string{"APPROVAL_STATUS", "RESOLVED_BY", "RESOLVED_DATE_TIME"} {
		require.Contains(t, bagianSet, kolom, "keputusan akseptasi harus menulis %s", kolom)
	}
}

// TestSeluruhKueriMenyaringSoftDelete menjaga `D-66`.
func TestSeluruhKueriMenyaringSoftDelete(t *testing.T) {
	for _, nama := range namaKueri {
		require.Contains(t, strings.ToUpper(query(nama)), "STATUS_ACTIVE",
			"kueri %q tidak menyaring penanda hapus (D-66)", nama)
	}
}

// TestKeduaAntreanMemakaiSatuKueri menjaga pemisahan PREMI / NON PREMI tetap satu sumber.
//
// Dua kueri yang nyaris sama akan berbeda isinya cepat atau lambat, dan yang berbeda akan
// menampilkan antrean yang salah tanpa satu pun gejala. Karena itu pembedanya PARAMETER,
// bukan kueri kedua — dan uji ini menggagalkan penambahan kueri antrean terpisah.
func TestKeduaAntreanMemakaiSatuKueri(t *testing.T) {
	// Yang diperiksa CABANGNYA, bukan nomor bind-nya: nomor berubah begitu sebuah penanda
	// ditambahkan, dan uji yang mengunci nomor akan gagal pada perubahan yang benar.
	for _, nama := range []string{"acceptance_count", "acceptance_list"} {
		teks := strings.ToUpper(query(nama))
		require.Regexp(t, `:\d+ = 1 AND`, teks, "kueri %q kehilangan cabang antrean PREMI", nama)
		require.Regexp(t, `:\d+ = 0 AND`, teks, "kueri %q kehilangan cabang antrean NON PREMI", nama)
	}

	for _, nama := range namaKueri {
		require.NotContains(t, strings.ToLower(nama), "premi",
			"antrean dibedakan parameter, bukan kueri terpisah")
	}
}

// TestKueriMemakaiParameterBinding menjaga penutup celah `{ASIS:…}` warisan.
func TestKueriMemakaiParameterBinding(t *testing.T) {
	for _, nama := range namaKueri {
		teks := query(nama)
		// Satu-satunya literal teks yang diizinkan: '1' pada STATUS_ACTIVE dan '%' pada LIKE.
		for _, literal := range []string{"'1'", "'%'"} {
			teks = strings.ReplaceAll(teks, literal, "")
		}
		require.NotContains(t, teks, "'",
			"kueri %q memuat literal teks selain yang diizinkan; nilai harus lewat bind", nama)
	}
}
