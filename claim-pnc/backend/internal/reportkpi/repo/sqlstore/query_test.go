package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Uji di berkas ini memeriksa TEKS KUERI, bukan hasilnya terhadap basis data.
//
// Itu yang dapat diperiksa tanpa Oracle, dan justru kelas cacat yang paling mahal di modul
// ini: satu alias yang tertukar menaruh nilai komponen di bawah judul komponen lain, dan
// tidak ada satu pun galat yang menandakannya. Uji terhadap basis data nyata adalah
// pekerjaan gerbang 1 (`S-8`), yang masih menunggu Pega staging (`ADR-0027`).

func TestSetiapKueriYangDipakaiKodeAdaDiBerkasSQL(t *testing.T) {
	// Disebut lengkap, bukan disimpulkan dari isi berkas: daftar yang dihitung sendiri
	// oleh uji akan ikut hilang bila kuerinya terhapus.
	for _, name := range []string{
		"summary", "detail", "adjusters", "check_source", "check_distinct_types",
	} {
		require.NotPanicsf(t, func() { _ = query(name) },
			"kueri %q tidak ditemukan di berkas .sql", name)
		require.NotEmptyf(t, strings.TrimSpace(query(name)),
			"kueri %q kosong", name)
	}
}

// Urutan alias nilai WAJIB sama dengan urutan komponen domain.
//
// Ini uji terpenting di berkas ini. Pemindai di reportkpi.go memasangkan hasil pindaian
// ke kode komponen MENURUT URUTAN, sehingga satu pergeseran menaruh nilai "TANGGAPAN
// KOMUNIKASI" di bawah judul "UPDATE PROGRESS" — angka yang salah, tanpa satu pun galat.
func TestUrutanAliasNilaiSamaDenganUrutanKomponen(t *testing.T) {
	codes := reportkpi.ComponentCodes()
	require.Len(t, scoreColumns, len(codes),
		"jumlah alias nilai berbeda dari jumlah komponen")

	// Pasangannya diperiksa lewat kolom basis datanya, bukan lewat nama alias — alias
	// dipilih bebas, sedangkan nama kolom adalah fakta.
	expected := []string{
		"SURVEYLAP", "IMMEDIATEADVICE", "PRELIMINARYADVICE", "INTERIM",
		"PROGRESS", "KOMUNIKASI", "PROPOSE", "FINALREPORT", "NILAI",
	}
	for i, code := range codes {
		component, found := reportkpi.FindComponent(code)
		require.Truef(t, found, "komponen %q tidak ditemukan", code)
		require.Equalf(t, expected[i], component.Column,
			"komponen ke-%d (%s) menunjuk kolom yang tidak diharapkan", i+1, code)
	}
}

// Alias di berkas .sql harus muncul DALAM URUTAN yang sama dengan scoreColumns.
//
// Diperiksa dengan mencari posisi tiap alias di dalam teks kueri: urutan kemunculan itulah
// urutan kolom hasil, dan itulah yang dipasangkan pemindai.
func TestAliasNilaiMunculBerurutanDiKeduaKueri(t *testing.T) {
	for _, name := range []string{"summary", "detail"} {
		text := query(name)

		previous := -1
		for _, alias := range scoreColumns {
			at := strings.Index(text, " AS "+alias)
			require.GreaterOrEqualf(t, at, 0,
				"kueri %q tidak memuat alias %q", name, alias)
			require.Greaterf(t, at, previous,
				"alias %q pada kueri %q tidak berurutan", alias, name)
			previous = at
		}
	}
}

func TestKueriSummaryMengembalikanAliasYangDiharapkan(t *testing.T) {
	text := query("summary")
	for _, alias := range summaryColumns {
		require.Containsf(t, text, " AS "+alias,
			"kueri summary tidak memuat alias %q", alias)
	}
	require.NotContains(t, text, "TOTAL_ROWS",
		"summary tidak dipaginasi, sehingga tidak boleh menghitung total baris")
}

func TestKueriDetailMengembalikanAliasYangDiharapkan(t *testing.T) {
	text := query("detail")
	for _, alias := range detailColumns {
		require.Containsf(t, text, " AS "+alias,
			"kueri detail tidak memuat alias %q", alias)
	}
}

// Summary dan Detail menjawab pertanyaan yang sama dengan bentuk berbeda.
//
// Satu penyaring yang tertinggal di salah satunya membuat rata-rata dan rinciannya tidak
// dapat dicocokkan pengguna — dan itu tidak menghasilkan satu pun galat.
func TestSummaryDanDetailMemakaiPenyaringYangSama(t *testing.T) {
	for _, name := range filteredQueries {
		text := query(name)

		require.Containsf(t, text, "k.TIPE = :1",
			"kueri %q tidak menyaring tipe report", name)
		require.Containsf(t, text, "k.ADJUSTER = :2",
			"kueri %q tidak menyaring adjuster", name)
		require.Containsf(t, text, "k.TANGGAL >= TO_DATE(:3, 'YYYY-MM-DD')",
			"kueri %q tidak menyaring batas bawah periode", name)
		require.Containsf(t, text, "k.TANGGAL < TO_DATE(:4, 'YYYY-MM-DD') + INTERVAL '1' DAY",
			"kueri %q tidak menyaring batas atas periode secara setengah terbuka", name)
	}
}

// Penyaring tipe dan adjuster WAJIB berbentuk `(:n IS NULL OR …)`.
//
// Itulah yang membuat tipe ALL dan "seluruh adjuster" dilayani kueri yang SAMA. Tanpa
// bentuk itu, NULL akan menghasilkan nol baris — bukan seluruh baris — dan tab ALL akan
// tampak kosong tanpa sebab.
func TestPenyaringOpsionalMemakaiBentukIsNullOr(t *testing.T) {
	for _, name := range filteredQueries {
		text := query(name)
		require.Containsf(t, text, "(:1 IS NULL OR k.TIPE = :1)",
			"kueri %q tidak memperlakukan tipe NULL sebagai seluruh tipe", name)
		require.Containsf(t, text, "(:2 IS NULL OR k.ADJUSTER = :2)",
			"kueri %q tidak memperlakukan adjuster NULL sebagai seluruh adjuster", name)
	}
}

// Batas atas periode TIDAK BOLEH ditulis `<=`.
//
// `TANGGAL` bertipe DATE di Oracle dan membawa jam. `<= TO_DATE(sampai)` membuang seluruh
// baris yang jam-nya bukan tengah malam pada hari terakhir — cacat yang hanya terlihat
// bila kebetulan ada baris di hari itu.
func TestBatasAtasPeriodeSetengahTerbuka(t *testing.T) {
	for _, name := range []string{"summary", "detail", "adjusters"} {
		text := query(name)
		require.NotContainsf(t, text, "k.TANGGAL <= ",
			"kueri %q memakai batas atas tertutup terhadap kolom bertipe DATE", name)
		require.Containsf(t, text, "+ INTERVAL '1' DAY",
			"kueri %q tidak memuat pergeseran satu hari pada batas atas", name)
	}
}

// Tidak satu pun nilai boleh dirangkai ke dalam teks SQL.
//
// Di sistem lama KETIGA penyaring modul ini dirangkai begitu, dan dua di antaranya dari
// isian yang DIKETIK pengguna (`08-TECHNICAL-STRATEGY.md` §4.3).
func TestTidakAdaPerangkaianNilaiKeDalamSQL(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %q masih memuat pola {ASIS:}", name)
		require.NotContainsf(t, text, "||'", "kueri %q merangkai nilai ke dalam SQL", name)
		require.NotContainsf(t, text, "|| '", "kueri %q merangkai nilai ke dalam SQL", name)
		require.NotContainsf(t, text, "%s", "kueri %q memuat penanda format Go", name)
	}
}

// `SELECT *` dilarang: kolom baru di basis data tidak boleh diam-diam mengubah perilaku
// aplikasi (`08-TECHNICAL-STRATEGY.md` §4.3).
func TestTidakAdaSelectBintang(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "SELECT *",
			"kueri %q memakai SELECT *", name)
	}
}

// Modul ini MEMBACA saja. Kata kunci yang menulis tidak boleh muncul sama sekali —
// tabelnya masih dimiliki Pega selama masa paralel (`P-1`).
func TestTidakAdaPernyataanYangMenulis(t *testing.T) {
	forbidden := regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|MERGE|TRUNCATE)\b`)
	for name, text := range queries {
		require.Falsef(t, forbidden.MatchString(text),
			"kueri %q memuat pernyataan yang menulis", name)
	}
}

// Seluruh kueri tab ADJUSTER membaca tabel yang SATU itu.
//
// Dibatasi pada tab Adjuster dengan sengaja: tab KPI Admin memang membaca tabel yang
// berbeda — ia menghitung dari tabel klaim, bukan membaca nilai yang sudah jadi. Menguji
// keduanya dengan syarat yang sama akan memaksa salah satunya dilonggarkan, dan yang
// longgar tidak menjaga apa pun.
func TestSeluruhKueriAdjusterMembacaTabelSumberYangSama(t *testing.T) {
	for _, name := range []string{
		"summary", "detail", "adjusters", "check_source", "check_distinct_types",
	} {
		require.Containsf(t, strings.ToUpper(query(name)),
			strings.ToUpper(reportkpi.SourceTable),
			"kueri %q tidak membaca %s", name, reportkpi.SourceTable)
	}
}

// adminQueries adalah keempat kueri tab KPI Admin.
var adminQueries = []string{
	"admin_scorecard_nonmbu", "admin_detail_nonmbu",
	"admin_scorecard_pa", "admin_detail_pa",
}

func TestSetiapKueriAdminAdaDiBerkasSQL(t *testing.T) {
	for _, name := range adminQueries {
		require.NotPanicsf(t, func() { _ = query(name) },
			"kueri %q tidak ditemukan di berkas .sql", name)
	}
}

// Keenam Operator ID yang menentukan siapa yang dihitung WAJIB ada apa adanya.
//
// Work Owner, 2026-09-24: "seperti aplikasi PEGA saja". Uji ini menjaga ketetapan itu tetap
// disengaja — siapa pun yang membuangnya kelak, atau mengganti salah satunya, akan melihat
// ujinya gagal dan membaca alasannya.
//
// Ketika masternya kelak dibuat (`D-15`), uji inilah yang pertama harus diubah — dan
// perubahannya akan terlihat di review, bukan lolos diam-diam.
func TestOperatorYangDiHardcodeTetapApaAdanya(t *testing.T) {
	nonMBU := []string{"SOPHIANOVITAEVELYN_1", "SOPHIANOVITAEVELYN", "RUTHCLARA"}
	pa := []string{"IRMANOPITAPURBA_1", "IRMANOPITAPURBA", "YUNIARPAMORSUARI"}

	for _, name := range []string{"admin_scorecard_nonmbu", "admin_detail_nonmbu"} {
		for _, operator := range nonMBU {
			require.Containsf(t, query(name), operator,
				"kueri %q kehilangan operator %q", name, operator)
		}
	}
	for _, name := range []string{"admin_scorecard_pa", "admin_detail_pa"} {
		for _, operator := range pa {
			require.Containsf(t, query(name), operator,
				"kueri %q kehilangan operator %q", name, operator)
		}
	}
}

// Kartu skor NON-MBU menyaring Group Panel 009; grid rinciannya TIDAK.
//
// Ini keanehan sistem lama yang direplikasi, dan uji ini ada supaya ia tidak "diperbaiki"
// diam-diam. Menyamakan keduanya akan mengubah angka pada kartu skor — ke arah yang belum
// pernah diminta siapa pun.
func TestPenyaringGroupPanelKartuSkorDanRincianMemangBerbeda(t *testing.T) {
	require.Contains(t, query("admin_scorecard_nonmbu"), "'003', '004', '006', '009'",
		"kartu skor NON-MBU seharusnya menghitung Group Panel 009")
	require.Contains(t, query("admin_detail_nonmbu"), "'003', '004', '006')",
		"grid rincian NON-MBU seharusnya TIDAK menghitung Group Panel 009")
	require.NotContains(t, query("admin_detail_nonmbu"), "'009'",
		"grid rincian NON-MBU seharusnya TIDAK menghitung Group Panel 009")
}

// Rentang 2023 yang tertanam pada pembagi PA WAJIB tetap ada.
//
// Ia tampak seperti sisa uji coba, dan justru karena itu mudah "dibersihkan" oleh
// pembaca berikutnya. Membuangnya mengubah angka TOTAL KLAIM BAYAR — dan keputusan itu
// milik Work Owner, bukan milik siapa pun yang kebetulan membaca kuerinya.
func TestRentang2023YangTertanamTetapAda(t *testing.T) {
	text := query("admin_scorecard_pa")
	require.Contains(t, text, "TO_DATE('2023-01-01', 'YYYY-MM-DD')")
	require.Contains(t, text, "TO_DATE('2023-11-10', 'YYYY-MM-DD')")
}

// Ambang "melewati SLA" BERBEDA antar kelompok, dan itu bukan salah ketik.
//
//	NON-MBU  tat_regis > 1   satu hari kerja
//	PA       tat > 0         nol hari kerja
//
// Menyamakannya akan mengubah angka pada salah satu kartu skor.
func TestAmbangSLABerbedaAntarKelompok(t *testing.T) {
	require.Contains(t, query("admin_scorecard_nonmbu"), "tat_regis > 1")
	require.Contains(t, query("admin_scorecard_pa"), "tat_regis > 0")
	require.Contains(t, query("admin_scorecard_pa"), "tat_bayar > 0")
}

// Pembagian pada kartu skor dijaga NULLIF.
//
// Tanpa itu, periode tanpa satu pun klaim menghasilkan `ORA-01476` yang sampai ke pengguna
// sebagai kegagalan mentah — persis yang terjadi di Pega hari ini.
func TestPembagianKartuSkorDijagaNullif(t *testing.T) {
	for _, name := range []string{"admin_scorecard_nonmbu", "admin_scorecard_pa"} {
		require.Containsf(t, query(name), "NULLIF(",
			"kueri %q membagi tanpa penjaga nol", name)
	}
}

// Paginasi hanya pada kueri rincian, dan bentuknya yang portabel.
//
// `ROWNUM` khas Oracle dan `D-20` menggantinya dengan `OFFSET … FETCH NEXT`, yang didukung
// Oracle 12c+ maupun PostgreSQL.
func TestPaginasiMemakaiBentukPortabel(t *testing.T) {
	require.Contains(t, query("detail"), "OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY")
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "ROWNUM",
			"kueri %q memakai ROWNUM", name)
	}
}

// `TRUNC` pada kolom mematikan index-nya, dan `D-20` mendaftarkannya sebagai bentuk khas
// Oracle yang diganti.
func TestTidakAdaTruncPadaKolomTanggal(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "TRUNC(K.TANGGAL)",
			"kueri %q memakai TRUNC pada kolom tanggal", name)
	}
}

// Pemformatan tanggal dikerjakan Go, bukan SQL (`08-TECHNICAL-STRATEGY.md` §4.3).
//
// Tanggal yang dikembalikan sebagai teks membuat pengurutan menjadi pengurutan teks —
// persis cacat yang `D-20` hapus dengan membuang 411 pemakaian `TO_CHAR`.
func TestTanggalTidakDiformatDiSQL(t *testing.T) {
	require.NotContains(t, strings.ToUpper(query("detail")), "TO_CHAR(K.TANGGAL")
	require.Contains(t, query("detail"), "k.TANGGAL                     AS SCORED_ON")
}

// Pembulatan ringkasan dua desimal, sama dengan `round(avg(...),2)` di kueri lama (`P-5`).
func TestRingkasanMembulatkanDuaDesimal(t *testing.T) {
	text := query("summary")
	for _, alias := range scoreColumns {
		require.Containsf(t, text, alias,
			"kueri summary tidak memuat alias %q", alias)
	}
	require.Equal(t, len(scoreColumns), strings.Count(text, "ROUND(AVG(TO_NUMBER("),
		"setiap komponen ringkasan wajib dibulatkan dua desimal")
	require.Equal(t, len(scoreColumns), strings.Count(text, ")), 2)"),
		"setiap komponen ringkasan wajib dibulatkan dua desimal")
}

// `DEFAULT NULL ON CONVERSION ERROR` sengaja TIDAK dipakai — lihat alasannya di
// reportkpi.sql. Uji ini menjaga keputusan itu tetap disengaja: siapa pun yang
// menambahkannya kelak akan melihat uji ini gagal dan membaca alasannya.
func TestKonversiAngkaTidakDiperlunak(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "ON CONVERSION ERROR",
			"kueri %q memperlunak konversi angka; itu MENGGESER rata-rata (P-5)", name)
	}
}
