package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeluruhKueriTermuat(t *testing.T) {
	for _, nama := range namaKueri {
		require.NotEmpty(t, strings.TrimSpace(query(nama)),
			"kueri %q kosong atau tidak ada", nama)
	}
}

// TestKueriMengikutiDisiplinSQLPortabel menjaga `D-20`.
//
// Modul ini TIDAK punya pengecualian dialek: generator nomor — satu-satunya pengecualian
// yang `ADR-0005` akui — tidak ada di sini.
func TestKueriMengikutiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []string{
		"NVL(", "SYSDATE", "ROWNUM", "DECODE(", "TO_CHAR(", "TO_NUMBER(",
		"TRUNC(", "FROM DUAL", "NEXTVAL", "SELECT *", "INSTR(", "LISTAGG(",
	}

	for _, nama := range namaKueri {
		teks := strings.ToUpper(query(nama))
		for _, pola := range terlarang {
			require.NotContains(t, teks, pola,
				"kueri %q memakai %s yang dilarang 09-DATABASE-STRATEGY.md §4", nama, pola)
		}
	}
}

// TestTidakAdaPerangkaianNilai menjaga larangan `{ASIS:...}` warisan.
//
// Dua literal diizinkan dan keduanya bukan nilai pengguna: tidak ada satu pun di modul ini
// hari ini, jadi daftarnya kosong dan setiap kutip tunggal yang muncul akan menggagalkan
// uji. Itu disengaja — nilai yang masuk lewat kutip adalah nilai yang tidak melewati bind.
func TestTidakAdaPerangkaianNilai(t *testing.T) {
	for _, nama := range namaKueri {
		require.NotContains(t, query(nama), "'",
			"kueri %q memuat literal teks; nilai harus lewat parameter binding", nama)
	}
}

// TestSeluruhTabelPenyimpananLewatDBLink menjaga temuan 2026-09-26.
//
// Skema `GENERAL` **tidak ada** di basis data Claim PNC — `ALL_USERS` nol baris. Menulis
// `GENERAL.T_STORAGE_IMAGE` tanpa DB link menghasilkan `ORA-00942`, dan galat itu baru
// muncul saat pengguna mengunggah, bukan saat kode ditulis.
func TestSeluruhTabelPenyimpananLewatDBLink(t *testing.T) {
	const link = "@ASMD.SINARMAS.CO.ID"

	for _, nama := range namaKueri {
		teks := strings.ToUpper(query(nama))
		for _, tabel := range []string{
			"GENERAL.T_FOLDER_STORAGE", "GENERAL.T_STORAGE_IMAGE", "GENERAL.GCP_IMAGE",
		} {
			idx := strings.Index(teks, tabel)
			if idx < 0 {
				continue
			}
			sisa := teks[idx+len(tabel):]
			require.True(t, strings.HasPrefix(sisa, link),
				"kueri %q menyebut %s tanpa DB link — skema GENERAL tidak ada di basis data ini",
				nama, tabel)
		}
	}
}

// TestDaftarMenyaringPenandaHapusDanArsip menjaga `D-66` §8.1.
//
// Satu kueri pembaca yang lupa menyaring akan menampilkan dokumen yang seharusnya sudah
// hilang — dan tidak ada galat yang menandainya.
func TestDaftarMenyaringPenandaHapusDanArsip(t *testing.T) {
	teks := strings.ToUpper(query("dokumen_per_klaim"))
	require.Contains(t, teks, "DELETE_DATE IS NULL")
	require.Contains(t, teks, "TGL_ARCHIVE IS NULL")
}

// TestDaftarMenyaringAplikasiJuga menutup kebocoran antar aplikasi.
//
// Tabelnya dipakai bersama sembilan aplikasi — `klaimasmtest` sendiri 113.449 baris. Tanpa
// saringan `APPNAME`, daftar sebuah klaim dapat memuat dokumen aplikasi lain yang kebetulan
// bernomor sama.
func TestDaftarMenyaringAplikasiJuga(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("dokumen_per_klaim")), "APPNAME",
		"daftar wajib disaring APPNAME, bukan NO_CLAIM saja")
}

// TestTidakAdaCommitDiDalamPernyataan menjaga `D-68`.
//
// `InsertDataPNCStorage` versi Pega berbentuk blok `BEGIN ... COMMIT; END;`. Kepemilikan
// transaksi pindah ke Go, dan pernyataan yang commit sendiri adalah pola yang ditinggalkan.
func TestTidakAdaCommitDiDalamPernyataan(t *testing.T) {
	for _, nama := range namaKueri {
		teks := strings.ToUpper(query(nama))
		require.NotContains(t, teks, "COMMIT", "kueri %q tidak boleh commit sendiri", nama)
		require.NotContains(t, teks, "BEGIN", "kueri %q tidak boleh blok PL/SQL", nama)
	}
}

// TestKedua KueriBacaBerkolomSama menjaga satu fungsi pemindai tetap sah.
func TestKeduaKueriBacaBerkolomSama(t *testing.T) {
	kolom := func(nama string) string {
		teks := strings.ToUpper(query(nama))
		potong := strings.SplitN(teks, "FROM GENERAL.", 2)
		require.Len(t, potong, 2, "kueri %q tidak punya klausa FROM yang dikenali", nama)
		return strings.Join(strings.Fields(potong[0]), " ")
	}

	require.Equal(t, kolom("dokumen_per_klaim"), kolom("dokumen_menurut_imageid"),
		"kedua kueri baca dipindai fungsi yang SAMA, jadi kolomnya harus sama persis")
}
