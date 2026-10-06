package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

// Setiap laporan yang katalog nyatakan DAPAT DIJALANKAN wajib punya rencana kueri.
// Tanpa uji ini, laporan yang lupa didaftarkan baru ketahuan saat pengguna menekan
// tombolnya — dan yang ia terima adalah galat, bukan berkas.
func TestSetiapLaporanSiapPunyaRencanaKueri(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		if !r.Availability.Ready {
			continue
		}
		_, ada := plans[r.Code]
		require.Truef(t, ada, "%s (%s) tidak punya rencana kueri", r.Title, r.Code)
	}
}

// Sebaliknya: laporan yang TERHALANG tidak boleh punya rencana kueri. Mendaftarkannya
// berarti menyediakan jalan yang reportklaim.Lookup justru ada untuk menutupnya.
func TestLaporanTerhalangTidakPunyaRencanaKueri(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		if r.Availability.Ready {
			continue
		}
		_, ada := plans[r.Code]
		require.Falsef(t, ada, "%s (%s) terhalang tetapi punya rencana kueri", r.Title, r.Code)
	}
}

// kombinasiPenyaring adalah kelima pilihan lini bisnis — setiap laporan harus dapat
// memilih kuerinya untuk semuanya.
var kombinasiPenyaring = []reportklaim.Filter{
	{BusinessLine: reportklaim.BusinessLineAll},
	{BusinessLine: reportklaim.BusinessLinePA},
	{BusinessLine: reportklaim.BusinessLineTravel},
	{BusinessLine: reportklaim.BusinessLineNonMBU},
	{BusinessLine: reportklaim.BusinessLineBonding},
}

// kueriBelumDipindahkan adalah daftar kueri yang MEMANG belum ada, beserta sebabnya.
//
// # Kenapa daftarnya KOSONG, dan tetap berdiri
//
// Karena ia mengunci keadaan dari DUA arah. Ketiga kueri Non-MBU yang dulu terdaftar di
// sini sudah dipindahkan, dan daftar ini kosong sejak 2026-09-25. Ia tidak dihapus: bila
// kelak ada rencana yang menunjuk kueri yang tidak ada, uji di bawah gagal menyebut nama
// laporan dan lini bisnisnya — bukan diam sampai seseorang menekan tombolnya.
//
// Menambahkan nama ke sini adalah pernyataan sadar bahwa sebuah laporan memang belum
// selesai, dan pernyataan itu ikut terbaca di mode periksa lewat NotPorted().
var kueriBelumDipindahkan = map[string]string{}

// Setiap nama kueri yang dapat dipilih rencana harus ADA di berkas .sql — kecuali yang
// terdaftar di kueriBelumDipindahkan.
func TestSeluruhKueriYangDipilihRencanaAdaKecualiYangTerdaftar(t *testing.T) {
	hilang := map[string]bool{}

	for code, p := range plans {
		for _, f := range kombinasiPenyaring {
			name := p.query(f)
			if hasQuery(name) {
				continue
			}
			_, diketahui := kueriBelumDipindahkan[name]
			require.Truef(t, diketahui,
				"laporan %s lini %q menunjuk kueri %q yang tidak ada dan tidak terdaftar sebagai belum dipindahkan",
				code, f.BusinessLine, name)
			hilang[name] = true
		}
	}

	// Yang terdaftar tetapi ternyata SUDAH ada harus dibuang dari daftar, supaya daftar
	// ini tidak menjadi catatan usang yang menyembunyikan kemajuan.
	for name := range kueriBelumDipindahkan {
		require.Falsef(t, hasQuery(name),
			"kueri %q sudah ada; buang namanya dari kueriBelumDipindahkan", name)
	}
	require.Len(t, hilang, len(kueriBelumDipindahkan))
}

// Mekanisme penolakan kueri yang belum dipindahkan TETAP diuji meski daftarnya kosong.
//
// Yang diuji adalah PERILAKUNYA, bukan keadaan hari ini: kueri yang tidak ada harus
// ditolak dengan sebab yang menyebut laporan dan lini bisnisnya, bukan menghasilkan
// berkas kosong yang terbaca sebagai "tidak ada data pada periode itu".
//
// Tanpa uji ini, jaring pengamannya ikut hilang bersama daftar yang kosong — dan laporan
// berikutnya yang kuerinya belum ditulis akan gagal dengan cara yang jauh lebih buruk.
func TestKueriBelumDipindahkanDitolakDenganSebabnya(t *testing.T) {
	report, ok := reportklaim.Find(reportklaim.CodeCloseKlaim)
	require.True(t, ok)

	// Rencana tiruan yang menunjuk kueri yang memang tidak ada.
	asli := plans[report.Code]
	plans[report.Code] = plan{
		query: satu("report_yang_belum_ada"),
		args:  asli.args,
	}
	t.Cleanup(func() { plans[report.Code] = asli })

	repo := NewRepo(nil, nil)
	err := repo.Stream(
		t.Context(),
		report,
		reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU},
		func(reportklaim.Row) error { return nil },
	)
	require.ErrorIs(t, err, ErrQueryNotPorted)
	require.Contains(t, err.Error(), "close-klaim")
	require.Contains(t, err.Error(), "346", "lini bisnisnya ikut disebut")
}

// Ketiga kueri Non-MBU yang dulu belum dipindahkan kini ADA. Uji ini menahan
// kemundurannya: menghapus salah satunya membuat panel yang bersangkutan diam-diam
// kehilangan susunan rincinya.
func TestKetigaKueriNonMBUSudahAda(t *testing.T) {
	for _, code := range []reportklaim.Code{
		reportklaim.CodeCloseKlaim,
		reportklaim.CodeTemporaryCloseKlaim,
		reportklaim.CodeKomite,
	} {
		name := plans[code].query(reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU})
		require.Truef(t, hasQuery(name), "kueri %q untuk laporan %s tidak ada", name, code)
	}
}

// Lini SELAIN Non-MBU pada laporan yang sama TIDAK boleh ikut tertolak — kuerinya ada.
func TestLiniLainPadaLaporanYangSamaTidakIkutTertolak(t *testing.T) {
	report, ok := reportklaim.Find(reportklaim.CodeCloseKlaim)
	require.True(t, ok)

	require.True(t, hasQuery(plans[report.Code].query(
		reportklaim.Filter{BusinessLine: reportklaim.BusinessLinePA})))
}

// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3 dan `09-DATABASE-STRATEGY.md` §4.
//
// Ia yang benar-benar menjaga `D-20` tetap berlaku setelah bulan ketiga — ketika tekanan
// jadwal membuat orang menempuh jalan pintas.
func TestTidakAdaPolaSQLTerlarang(t *testing.T) {
	terlarang := map[string]string{
		"NVL(":        "pakai COALESCE",
		"SYSDATE":     "pakai CURRENT_TIMESTAMP",
		"DECODE(":     "pakai CASE WHEN ... END",
		"ROWNUM":      "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":      "pakai POSITION",
		"LISTAGG(":    "pakai STRING_AGG",
		"FROM DUAL":   "hilangkan klausa FROM",
		"ADD_MONTHS(": "pakai penambahan INTERVAL",
		"SELECT *":    "sebutkan nama kolom",

		// VARCHAR2 hanya ada di Oracle, dan ia mudah lolos: tempatnya bukan di daftar
		// kolom melainkan di dalam klausa COLUMNS milik JSON_TABLE — bagian kueri yang
		// jarang dibaca ulang. PostgreSQL tidak mengenal tipe itu sama sekali, sehingga
		// kueri yang memuatnya gagal seketika di sana. VARCHAR diterima keduanya.
		"VARCHAR2": "pakai VARCHAR",
	}

	for name, text := range query {
		upper := strings.ToUpper(text)
		for pola, saran := range terlarang {
			require.NotContainsf(t, upper, strings.ToUpper(pola),
				"kueri %q memakai %s — %s", name, pola, saran)
		}
	}
}

// kueriDenganDBLink adalah satu-satunya kueri yang MASIH memuat DB Link.
//
// Work Owner memutuskan 2026-09-24: yang berupa SUB-QUERY tetap memakai DB Link; selain
// itu memakai koneksi langsung. Laporan TAT punya satu sub-query seperti itu — kolom
// "PolicyRange" yang membaca `collection.mst_det_sales@ASMD`.
const kueriDenganDBLink = "report_tat"

// Dua pola Oracle hanya boleh muncul DI DALAM sub-query DB Link yang dipertahankan.
//
// Keduanya dilarang di tempat lain (`09-DATABASE-STRATEGY.md` §4): TO_CHAR karena
// pemformatan dikerjakan di Go, MONTHS_BETWEEN karena ia tidak ada di PostgreSQL. Yang
// dipertahankan di sini adalah sub-query aslinya APA ADANYA — termasuk cara ia menghitung
// dan memformat — karena itulah arti keputusan "tetap memakai DB Link".
//
// Uji ini menjaga dua hal sekaligus: keduanya tidak merambat ke kueri lain, DAN kueri
// yang dikecualikan benar-benar memuat DB Link — bukan lolos karena namanya kebetulan
// masuk daftar kecuali.
// kueriDenganJamTampil adalah kueri yang keluarannya memuat JAM, bukan hanya tanggal.
//
// # Kenapa ia pengecualian TERSENDIRI, bukan menumpang kueriDenganDBLink
//
// Karena sebabnya berbeda sama sekali. Yang di atas dipertahankan karena sub-query DB
// Link dibawa apa adanya; yang di sini karena pemformat bersama `text()` hanya
// mengeluarkan TANGGAL, sedangkan kolom laporan ini menampilkan jam.
//
// `report_mitra` adalah laporan SLA: "Tgl Awal" dan "Tgl Akhir" adalah jam masuk dan jam
// keluar penugasan, dan aging-nya dihitung dalam detik. Laporan itu kehilangan artinya
// bila jamnya dibuang.
//
// # Kenapa ini tidak melanggar maksud `D-20`
//
// Larangannya menyasar dua akibat: ketidakportabelan, dan tanggal yang berubah menjadi
// teks sehingga pengurutan serta index-nya rusak (`09-DATABASE-STRATEGY.md` §3.2).
// Keduanya tidak berlaku di sini — `to_char` dengan model format yang sama ada di Oracle
// maupun PostgreSQL, dan kedua kolom ini hanya DIKELUARKAN, tidak pernah dipakai
// menyaring maupun mengurutkan.
var kueriDenganJamTampil = map[string]bool{"report_mitra": true}

func TestPolaOracleHanyaPadaSubQueryDBLinkYangDipertahankan(t *testing.T) {
	hanyaDiDBLink := []string{"TO_CHAR(", "MONTHS_BETWEEN("}

	for name, text := range query {
		if name == kueriDenganDBLink {
			continue
		}
		upper := strings.ToUpper(text)
		for _, pola := range hanyaDiDBLink {
			if pola == "TO_CHAR(" && kueriDenganJamTampil[name] {
				// Dikecualikan, TETAPI hanya untuk pemformatan yang benar-benar memuat
				// jam. Tanpa pemeriksaan ini, pengecualiannya lama-lama menjadi izin
				// memformat tanggal biasa — dan justru itu yang dilarang.
				require.Containsf(t, upper, "HH24:MI:SS",
					"kueri %q dikecualikan karena menampilkan jam, tetapi tidak satu pun "+
						"TO_CHAR-nya memuat jam", name)
				continue
			}
			require.NotContainsf(t, upper, pola,
				"kueri %q memakai %s di luar sub-query DB Link", name, pola)
		}
	}

	// Pengecualian yang tidak lagi terpakai harus dibuang, bukan dibiarkan menganggur.
	for name := range kueriDenganJamTampil {
		require.Containsf(t, strings.ToUpper(query[name]), "TO_CHAR(",
			"kueri %q tidak lagi memakai TO_CHAR; buang dari kueriDenganJamTampil", name)
	}

	dikecualikan := strings.ToUpper(query[kueriDenganDBLink])
	require.Contains(t, dikecualikan, "@ASMD",
		"kueri yang dikecualikan wajib benar-benar memuat DB Link")
	for _, pola := range hanyaDiDBLink {
		require.Containsf(t, dikecualikan, pola,
			"pengecualian %s tidak lagi terpakai; buang dari daftar", pola)
	}
}

// DB Link hanya boleh muncul pada SUB-QUERY, dan hanya di satu kueri.
//
// Keputusan Work Owner 2026-09-24: yang berupa sub-query tetap memakai DB Link; selain
// itu memakai koneksi langsung. Uji ini menjaga agar DB Link tidak merambat kembali ke
// kueri lain lewat salin-tempel.
func TestDBLinkHanyaPadaKueriYangDiputuskanMempertahankannya(t *testing.T) {
	var memakai []string
	for name, text := range query {
		if strings.Contains(strings.ToUpper(text), "@ASMD") {
			memakai = append(memakai, name)
		}
	}
	require.Equal(t, []string{kueriDenganDBLink}, memakai)
}

// Kueri yang dijalankan pada koneksi KEDUA tidak boleh memuat akhiran DB Link — justru
// koneksi itu yang menggantikannya.
func TestKueriKoneksiKeduaTanpaDBLink(t *testing.T) {
	kalender := query["report_holiday_calendar"]
	require.NotEmpty(t, kalender)
	require.NotContains(t, strings.ToUpper(kalender), "@ASMD")
	require.Contains(t, strings.ToLower(kalender), "general.hrd_lbr")
}

// Pemanggilan fungsi tersimpan dilarang (`D-02`): logikanya naik ke Go.
func TestTidakAdaPemanggilanFungsiTersimpan(t *testing.T) {
	terlarang := []string{
		"GET_INTERPOLASIPNC",
		"GET_POSISI_PROGRESS",
		"GETCURRENCYSTANDARD",
		"GETSELISIHJAM",
		"GET_WORKING_HOURS",
	}
	for name, text := range query {
		upper := strings.ToUpper(text)
		for _, fn := range terlarang {
			require.NotContainsf(t, upper, fn,
				"kueri %q memanggil fungsi tersimpan %s; logikanya ditulis ulang di Go (D-02)", name, fn)
		}
	}
}

// Setiap kueri harus punya nama yang dipakai rencana mana pun — kueri yatim menandakan
// rencana yang lupa diperbarui, atau kueri yang tertinggal setelah laporannya berubah.
func TestTidakAdaKueriYatim(t *testing.T) {
	dipakai := map[string]bool{
		// Kelimanya dipanggil langsung, bukan lewat rencana laporan: satu mengisi
		// dropdown, dan empat sisanya adalah MASTER yang dibaca sekali per laporan lalu
		// dipakai menghitung kolom turunan (kalender libur, tangga fee adjuster, nama
		// tahapan progres, faktor dominan).
		"report_business_options": true,
		"report_holiday_calendar": true,
		"report_fee_scale":        true,
		"report_progress_names":   true,
		"report_dominant_factors": true,

		// Dipakai `mitraKeep` sebagai PENYARING BARIS, bukan sebagai kueri laporan.
		//
		// Ia tidak dapat ditemukan dengan menyusuri `plans` karena penyaringnya sebuah
		// closure — yang terlihat di sana hanyalah `keep: mitraKeep`, bukan nama kueri
		// yang dibacanya.
		"report_mitra_logins": true,
	}
	for _, p := range plans {
		for _, f := range kombinasiPenyaring {
			dipakai[p.query(f)] = true
		}
	}

	for name := range query {
		require.Truef(t, dipakai[name], "kueri %q tidak dipakai rencana mana pun", name)
	}
}
