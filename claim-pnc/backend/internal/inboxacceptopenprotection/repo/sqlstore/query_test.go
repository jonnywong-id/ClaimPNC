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
		"POLICY_NO", "CLAIM_NO", "ID_CLAIM", "PROTECTION_TYPE_ID", "CREATE_DATE", "CREATED_BY",
		"NOTES", "OLD_DATA", "NEW_DATA", "OBJECT_NAME", "BRANCH_NAME", "STATUS_ACTIVE",
	} {
		require.NotContains(t, bagianSet, kolom,
			"keputusan akseptasi tidak boleh menulis %s (P-1)", kolom)
	}

	for _, kolom := range []string{"APPROVAL_STATUS", "RESOLVED_BY", "RESOLVED_DATETIME"} {
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
		// Dua literal teks yang diizinkan, dan keduanya BUKAN nilai dari pemanggil:
		// '1' penanda STATUS_ACTIVE, dan '%' pembungkus pola LIKE.
		//
		// `'0'` sempat ditambahkan 2026-09-27 sebagai karakter pengisi `LPAD`, lalu
		// DICABUT kembali pada hari yang sama ketika urutannya berpindah ke
		// `CAST(... AS NUMERIC)`. Kelonggaran yang tidak lagi dipakai harus dicabut:
		// yang tertinggal akan membiarkan literal berikutnya lolos tanpa pertimbangan.
		for _, literal := range []string{"'1'", "'%'"} {
			teks = strings.ReplaceAll(teks, literal, "")
		}
		require.NotContains(t, teks, "'",
			"kueri %q memuat literal teks selain yang diizinkan; nilai harus lewat bind", nama)
	}
}

// TestKueriBacaMemuatDetailPerubahan menjaga panel "Detail Perubahan" tetap punya datanya.
//
// Keempat kolom ini pernah HILANG dari kueri, dan hilangnya tidak menimbulkan galat apa pun —
// form tetap terbuka, tombolnya tetap berfungsi, hanya panelnya yang tidak pernah muncul.
// Petugas lalu menyetujui perubahan DOL dan Cause of Loss tanpa melihat apa yang diubah.
//
// Uji ini yang membuat penghapusannya kelak menjadi kegagalan, bukan kemunduran senyap.
func TestKueriBacaMemuatDetailPerubahan(t *testing.T) {
	kolom := []string{"OLD_DATA", "NEW_DATA", "OBJECT_NAME", "BRANCH_NAME"}

	for _, nama := range []string{"acceptance_list", "acceptance_get"} {
		teks := strings.ToUpper(query(nama))
		for _, k := range kolom {
			require.Contains(t, teks, k,
				"kueri %q harus membaca %s untuk panel Detail Perubahan", nama, k)
		}
	}
}

// kolomTerpilih membaca nama kolom pada klausa SELECT teratas sebuah kueri.
//
// # Pemisahan komanya menghormati tanda kurung, dan itu WAJIB
//
// Versi pertama memisah pada setiap koma. Ia pecah begitu subkueri masuk, sebab
// `LPAD(TRIM(g.PRODKE), 10, '0')` sendiri memuat dua koma — hasilnya "kolom" bernama
// `PRODKE`, `10`, dan `ONLY`.
//
// Yang dipakai sekarang menghitung kedalaman kurung dan hanya memisah pada koma tingkat
// TERATAS, sehingga satu subkueri tetap satu kolom.
func kolomTerpilih(t *testing.T, nama string) []string {
	t.Helper()

	teks := query(nama)
	besar := strings.ToUpper(teks)
	mulai := strings.Index(besar, "SELECT ")
	akhir := strings.Index(besar, "\n  FROM ")
	require.Greater(t, akhir, mulai, "kueri %q tidak berbentuk SELECT … FROM", nama)

	var hasil []string
	for _, bagian := range pisahKomaTeratas(teks[mulai+len("SELECT ") : akhir]) {
		bagian = strings.TrimSpace(bagian)
		if bagian == "" {
			continue
		}
		hasil = append(hasil, namaKolomDari(bagian))
	}
	return hasil
}

// pisahKomaTeratas memisah daftar kolom hanya pada koma di luar tanda kurung.
func pisahKomaTeratas(daftar string) []string {
	var hasil []string
	kedalaman, mulai := 0, 0
	for i, r := range daftar {
		switch r {
		case '(':
			kedalaman++
		case ')':
			kedalaman--
		case ',':
			if kedalaman == 0 {
				hasil = append(hasil, daftar[mulai:i])
				mulai = i + 1
			}
		}
	}
	return append(hasil, daftar[mulai:])
}

// namaKolomDari mengambil nama kolom sumber dari satu item SELECT.
//
// Untuk subkueri, yang diambil adalah kolom yang DIPILIH di dalamnya — `THEINSURED` dari
// `(SELECT g.THEINSURED FROM …)`. Itu yang bermakna bagi penjaga: yang perlu dijaga adalah
// nilai apa yang dibaca, bukan bentuk ekspresinya.
func namaKolomDari(item string) string {
	rapi := strings.TrimSpace(item)

	if strings.HasPrefix(rapi, "(") {
		if i := strings.Index(strings.ToUpper(rapi), "SELECT "); i >= 0 {
			medan := strings.Fields(rapi[i+len("SELECT "):])
			if len(medan) > 0 {
				rapi = medan[0]
			}
		}
	}
	if titik := strings.LastIndex(rapi, "."); titik >= 0 {
		rapi = rapi[titik+1:]
	}
	return strings.ToUpper(strings.TrimSpace(rapi))
}

// TestDaftarAdalahAWALAN dari kolom detail menjaga satu fungsi pemindaian tetap melayani
// keduanya.
//
// `scanProtection` dipakai List maupun Get, dan sejak 2026-09-27 keduanya TIDAK lagi
// berkolom sama: detail menambahkan tiga kolom polis yang tidak ditampilkan daftar.
//
// Yang menggantikan aturan "harus sama" adalah aturan yang lebih tepat: **kolom daftar
// adalah awalan persis dari kolom detail**. Itulah yang membuat satu pemindai tetap sah —
// ia membaca enam belas kolom pertama pada keduanya, lalu tiga tambahan hanya pada detail.
//
// Bila urutannya bergeser, Scan tidak selalu gagal: kolom bertipe sama akan tertukar
// diam-diam, dan yang terlihat hanyalah nilai yang salah tempat.
func TestDaftarAdalahAwalanDariKolomDetail(t *testing.T) {
	daftar := kolomTerpilih(t, "acceptance_list")
	detail := kolomTerpilih(t, "acceptance_get")

	require.Greater(t, len(detail), len(daftar),
		"detail harus punya kolom LEBIH banyak daripada daftar")
	require.Equal(t, daftar, detail[:len(daftar)],
		"kolom daftar wajib menjadi AWALAN persis dari kolom detail — satu pemindai "+
			"membaca keduanya")
}

// TestDetailMenambahTepatTigaKolomPolis menyatakan ketiganya secara eksplisit.
//
// Awalan yang cocok saja tidak cukup: tanpa uji ini, menghapus salah satu subkueri polis
// tetap lolos selama sisanya masih awalan yang sama — dan yang hilang adalah nilai yang
// memang tidak pernah punya penjaga sebelumnya.
func TestDetailMenambahTepatTigaKolomPolis(t *testing.T) {
	daftar := kolomTerpilih(t, "acceptance_list")
	detail := kolomTerpilih(t, "acceptance_get")

	require.Equal(t, []string{"THEINSURED", "STARTDATE", "ENDDATE"}, detail[len(daftar):],
		"detail wajib menambahkan tepat ketiga kolom polis, dalam urutan itu")
}

// TestPolisDiambilDariPerpanjanganTERBARU menjaga aturan yang Work Owner tetapkan:
// *"gunakan order by sebagaimana mestinya agar data yang diambil selalu prodke paling baru
// (angka terbesar)"*.
//
// Urutannya karena itu NUMERIK, bukan teks. `PRODKE` bertipe VARCHAR2, tetapi yang
// menentukan perpanjangan terbaru adalah nilainya sebagai bilangan.
//
// `to_number(prodke)` tidak dipakai karena bentuk berargumen satu tidak sah di PostgreSQL —
// di sana `to_number` menuntut format mask, dan `D-20` menetapkan satu set SQL untuk kedua
// basis data. `CAST(... AS NUMERIC)` adalah bentuk ANSI-nya, dan Oracle menerimanya (diuji).
func TestPolisDiambilDariPerpanjanganTerbaru(t *testing.T) {
	teks := strings.ToUpper(query("acceptance_get"))

	require.Equal(t, 3, strings.Count(teks, "POOLDATA.T_GENERAL"),
		"ketiga nilai polis diambil dari T_GENERAL")
	require.Equal(t, 3, strings.Count(teks, "CAST(TRIM(G.PRODKE) AS NUMERIC) DESC"),
		"ketiganya WAJIB memakai urutan yang sama — urutan berbeda akan menggabungkan "+
			"nama tertanggung satu perpanjangan dengan tanggal perpanjangan lain")

	require.NotContains(t, teks, "TO_NUMBER(",
		"to_number PostgreSQL menuntut format mask; padanannya CAST(... AS NUMERIC)")

	// Urutan TEKS — polos maupun ber-LPAD — tidak boleh kembali. Yang polos salah pada
	// 13.181 baris berawalan nol; yang ber-LPAD benar hari ini tetapi salah DIAM-DIAM
	// begitu PRODKE melewati lebar yang dipatok.
	require.NotContains(t, teks, "LPAD(",
		"urutan wajib numerik, bukan trik penyamaan lebar teks")
}

// namaKueriKlaim adalah kueri yang menyentuh DATA KLAIM, bukan tabel proteksi.
//
// Dipisahkan dari namaKueri karena sebagian uji di atas memang khusus tabel proteksi —
// `STATUS_ACTIVE`, misalnya, tidak ada di `T_CLAIM_PNC` maupun `T_CLAIM_OBJECTCOVERAGE`.
// Yang BERLAKU bagi keduanya tetap ditegakkan di bawah.
var namaKueriKlaim = []string{
	"claim_apply_loss_date",
	"cause_of_loss_describe",
	"claim_apply_cause_of_loss",
}

func TestKueriKlaimTermuat(t *testing.T) {
	for _, nama := range namaKueriKlaim {
		require.NotEmpty(t, strings.TrimSpace(query(nama)), "kueri %q kosong atau tidak ada", nama)
	}
}

// TestKueriKlaimMengikutiDisiplinSQLPortabel menjaga `D-20` pada kueri sisi klaim.
//
// Ketiganya sebelumnya TIDAK diuji sama sekali — `claim_apply_loss_date` ada sejak modul ini
// menerapkan DOL, dan tidak pernah masuk daftar mana pun. Konstruksi khas Oracle di sana
// akan lolos sampai cutover PostgreSQL.
func TestKueriKlaimMengikutiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []string{
		"NVL(", "SYSDATE", "ROWNUM", "DECODE(", "TO_CHAR(", "TO_NUMBER(",
		"TRUNC(", "FROM DUAL", "SELECT *", "INSTR(", "LISTAGG(",
	}

	for _, nama := range namaKueriKlaim {
		teks := strings.ToUpper(query(nama))
		for _, pola := range terlarang {
			require.NotContains(t, teks, pola,
				"kueri %q memakai %s yang dilarang 09-DATABASE-STRATEGY.md §4", nama, pola)
		}
	}
}

// TestKueriKlaimMemakaiParameterBinding menjaga penutup celah `{ASIS:…}` warisan.
func TestKueriKlaimMemakaiParameterBinding(t *testing.T) {
	for _, nama := range namaKueriKlaim {
		require.NotContains(t, query(nama), "'",
			"kueri %q memuat literal teks; nilai harus lewat bind", nama)
	}
}

// TestPenerapanPenyebabKerugianMenulisKeduaKolom menjaga ketetapan Work Owner 2026-10-05:
// *"ingat ganti cause of loss itu ganti causeoflossid juga"*.
//
// Menulis salah satunya saja menghasilkan baris yang namanya berkata satu hal dan kodenya
// berkata hal lain — dan laporan yang mengelompokkan menurut kode akan menghitungnya ke
// golongan lama sementara layar menampilkan yang baru.
func TestPenerapanPenyebabKerugianMenulisKeduaKolom(t *testing.T) {
	teks := strings.ToUpper(query("claim_apply_cause_of_loss"))
	bagianSet := strings.SplitN(teks, "WHERE", 2)[0]

	require.Contains(t, bagianSet, "CAUSEOFLOSSID")
	require.Contains(t, bagianSet, "CAUSEOFLOSS ")
}

// TestPenerapanPenyebabKerugianMengunciTigaKolom menjaga sasarannya SATU baris.
//
// Pada klaim `PNC-1452`, `JackHugh / Resiko A` muncul tiga kali dengan Penyebab Kerugian
// berbeda. Kehilangan salah satu penyaring berarti mengubah baris yang tidak diminta — dan
// pada data nyata, dua dari tiga kali yang salah.
func TestPenerapanPenyebabKerugianMengunciTigaKolom(t *testing.T) {
	teks := strings.ToUpper(query("claim_apply_cause_of_loss"))
	bagianWhere := strings.SplitN(teks, "WHERE", 2)[1]

	for _, kolom := range []string{"CLAIMID", "OBJECTID", "OBJECTCOVERAGEID"} {
		require.Contains(t, bagianWhere, kolom,
			"penerapan penyebab kerugian harus menyaring %s", kolom)
	}
}

// TestPenerapanPenyebabKerugianMenghormatiSoftDelete menjaga `D-66`.
//
// Coverage yang sudah dibuang dari klaim tidak boleh berubah karena persetujuan yang
// menunjuknya. Penyaring yang sama dipakai saat menawarkan pilihannya, sehingga yang dapat
// dipilih dan yang dapat diubah adalah himpunan yang sama.
func TestPenerapanPenyebabKerugianMenghormatiSoftDelete(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("claim_apply_cause_of_loss")),
		"DIHAPUS_PADA IS NULL")
}

// TestDeskripsiPenyebabDibacaDariTabelInduk menjaga ketetapan Work Owner 2026-10-05:
// *"menggunakan D_CAUSE_OF_LOSS jangan view"*.
//
// Dropdown yang menawarkan pilihan dan penerapan yang menuliskannya WAJIB membaca sumber
// yang sama — kalau tidak, sebuah kode dapat tampil di dropdown lalu ditolak saat diterapkan.
func TestDeskripsiPenyebabDibacaDariTabelInduk(t *testing.T) {
	teks := strings.ToUpper(query("cause_of_loss_describe"))
	require.Contains(t, teks, "POOLDATA.D_CAUSE_OF_LOSS")
	require.NotContains(t, teks, "V_D_CAUSE_OF_LOSS")
}
