package sqlstore

import (
	"fmt"
	"regexp"
	"slices"
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
		"FROM DUAL":   "hilangkan klausa FROM",
		"ADD_MONTHS(": "pakai penambahan INTERVAL",
		"SELECT *":    "sebutkan nama kolom",

		// VARCHAR2 hanya ada di Oracle, dan ia mudah lolos: tempatnya bukan di daftar
		// kolom melainkan di dalam klausa COLUMNS milik JSON_TABLE — bagian kueri yang
		// jarang dibaca ulang. PostgreSQL tidak mengenal tipe itu sama sekali, sehingga
		// kueri yang memuatnya gagal seketika di sana. VARCHAR diterima keduanya.
		"VARCHAR2": "pakai VARCHAR",

		// ====================================================================
		// Dua padanan `09-DATABASE-STRATEGY.md` §4 yang TIDAK DAPAT DIJALANKAN
		// ====================================================================
		//
		// Diuji langsung ke Oracle 19c lewat `EXPLAIN PLAN` pada 2026-10-09 —
		// bukan disimpulkan dari dokumen:
		//
		//   POSITION('/' IN 'a/b')  -> ORA-00907: missing right parenthesis
		//   INSTR('a/b','/')        -> diterima
		//   STRING_AGG(x, ',')      -> ORA-06553: PLS-306
		//   LISTAGG(x, ',') ...     -> diterima
		//
		// §4 menyuruh menempuh arah yang justru tidak berjalan di basis data yang
		// dipakai hari ini. Keduanya diajukan sebagai usulan koreksi Steering;
		// sampai itu diputuskan, yang berlaku di sini adalah bukti.
		//
		// POSITION dilarang karena tidak jalan di Oracle. Penggantinya INSTR, dan
		// pemakaiannya dikunci TestInstrHanyaDiKueriYangDisepakati.
		"POSITION(": "tidak jalan di Oracle 19c (ORA-00907) — pakai INSTR",

		// LISTAGG dan STRING_AGG SAMA-SAMA dilarang, dan itu disengaja: tidak ada
		// bentuk penggabungan baris yang berjalan di keduanya. Penggabungannya
		// dikerjakan di Go — lihat `dominanfactor.go`.
		"LISTAGG(":    "gabungkan di Go, bukan di SQL",
		"STRING_AGG(": "tidak jalan di Oracle 19c (ORA-06553) — gabungkan di Go",
	}

	for name, text := range query {
		upper := strings.ToUpper(text)
		for pola, saran := range terlarang {
			require.NotContainsf(t, upper, strings.ToUpper(pola),
				"kueri %q memakai %s — %s", name, pola, saran)
		}
	}
}

// bindMengikutiKemunculan mengunci satu sifat driver yang tidak terlihat dari kodenya.
//
// # Sifatnya
//
// go-ora mengikat `:n` menurut **urutan kemunculan**, bukan menurut nomornya. Diuji
// langsung ke Oracle 19c pada 2026-10-09 (`cmd/cekddl`, fungsi ujiBind):
//
//	SELECT :3, :1, :2        3 argumen a,b,c  ->  hasil  a b c
//	SELECT :1, :2, :1        2 argumen a,b    ->  hasil  a b a
//	SELECT :1, :2, :3, :3    3 argumen a,b,c  ->  hasil  a b c c
//	SELECT :1, :2, :3, :3    4 argumen a..d   ->  hasil  a b c d
//
// Baris pertama yang menentukan: `:3` menerima argumen PERTAMA. Nomornya diabaikan.
//
// # Kenapa ini diuji, bukan sekadar dikomentari
//
// Kueri yang nomornya tidak berurut tetap SAH, tetap lulus EXPLAIN PLAN, dan tetap
// berjalan — ia hanya mengikat nilai ke tempat yang salah. Dua kueri modul ini pernah
// begitu, dan keduanya tidak menyalak:
//
//	report_close_klaim_nonmbu   ORA-01008 — ketahuan hanya karena jumlahnya kebetulan kurang
//	report_komite_nonmbu        TIDAK ADA GALAT — tanggal masuk ke kolom status, hasilnya
//	                            nol baris selamanya
//
// Yang kedua adalah bentuk kegagalan terburuk yang mungkin ada di modul laporan: berkas
// kosong, tanpa pesan, dan pengguna tidak punya cara tahu sebabnya.
func TestNomorBindMengikutiUrutanKemunculan(t *testing.T) {
	bind := regexp.MustCompile(`:\d+`)

	for name, text := range query {
		// Baris komentar dibuang — blok "Bind:" di kepala kueri menyebut `:1` dan
		// kawan-kawannya sebagai dokumentasi, bukan sebagai placeholder.
		var sql []string
		for _, l := range strings.Split(text, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(l), "--") {
				sql = append(sql, l)
			}
		}

		var urutPertama []string
		for _, m := range bind.FindAllString(strings.Join(sql, "\n"), -1) {
			if !slices.Contains(urutPertama, m) {
				urutPertama = append(urutPertama, m)
			}
		}

		for i, m := range urutPertama {
			require.Equalf(t, fmt.Sprintf(":%d", i+1), m,
				"kueri %q: placeholder ke-%d yang muncul bernomor %s — penomoran harus "+
					"mengikuti urutan kemunculan, karena driver mengabaikan nomornya",
				name, i+1, m)
		}
	}
}

// INSTR adalah pengecualian dialek yang disadari — bukan izin umum.
//
// Ia dipakai HANYA di tempat sumbernya memang memakainya, dan daftarnya dikunci di sini
// supaya ia tidak menyebar diam-diam ke kueri lain ketika seseorang kelak membutuhkan
// "cari posisi karakter" dan menyalin dari tetangganya.
//
// Padanan PostgreSQL-nya `strpos`/`position`, dan itulah yang ditukar saat pindah basis
// data — satu tempat, bukan tersebar.
func TestInstrHanyaDiKueriYangDisepakati(t *testing.T) {
	disepakati := map[string]bool{
		// Sumbernya `RDB List/GetDataAIKlaim-SQL.xml` memang memakai INSTR.
		"report_ai_klaim": true,
	}

	for name, text := range query {
		if strings.Contains(strings.ToUpper(text), "INSTR(") {
			require.Truef(t, disepakati[name],
				"kueri %q memakai INSTR tanpa disepakati — gabungkan ke daftar ini bila memang perlu", name)
		}
	}
}

// kueriDenganDBLink adalah kueri yang MASIH memuat DB Link, beserta sebabnya.
//
//	report_tat                      Work Owner 2026-09-24 — sub-query dibawa apa adanya.
//	                                Kolom "PolicyRange" membaca
//	                                `collection.mst_det_sales@ASMD`.
//
//	report_holiday_calendar_dblink  Work Owner 2026-10-09 — cadangan SEMENTARA selama
//	report_mitra_logins_dblink      `ANEKA_<PORTAL_ALIAS>_*` belum terisi. Keduanya
//	                                berhenti terpakai sendiri begitu koneksi kedua ada.
var kueriDenganDBLink = map[string]bool{
	"report_tat":                     true,
	"report_holiday_calendar_dblink": true,
	"report_mitra_logins_dblink":     true,
}

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
		if kueriDenganDBLink[name] {
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

	// Setiap kueri yang dikecualikan wajib benar-benar memuat DB Link — bukan lolos
	// karena namanya kebetulan masuk daftar.
	for name := range kueriDenganDBLink {
		require.Containsf(t, strings.ToUpper(query[name]), "@ASMD",
			"kueri %q dikecualikan tetapi tidak memuat DB Link; buang dari daftar", name)
	}

	// Dan kedua pola Oracle yang dikecualikan harus masih terpakai di SALAH SATU-nya.
	// Yang memakainya hanya `report_tat`; kedua kueri cadangan tidak, dan memang tidak
	// seharusnya — ia salinan kueri portabel, hanya dengan akhiran DB Link.
	for _, pola := range hanyaDiDBLink {
		require.Containsf(t, strings.ToUpper(query["report_tat"]), pola,
			"pengecualian %s tidak lagi terpakai; buang dari daftar", pola)
	}
}

// DB Link hanya boleh muncul pada kueri yang memang diputuskan mempertahankannya.
//
// Tiga kueri, DUA sebab yang berbeda — dan perbedaannya ditulis di sini karena yang satu
// permanen sementara yang dua sementara:
//
//	report_tat                      sub-query dibawa apa adanya (Work Owner 2026-09-24)
//	report_*_dblink                 cadangan selama ANEKA_* kosong  (Work Owner 2026-10-09)
//
// Uji ini menjaga agar DB Link tidak merambat ke kueri lain lewat salin-tempel — godaan
// yang justru menguat sejak ada tiga contohnya di berkas yang sama.
func TestDBLinkHanyaPadaKueriYangDiputuskanMempertahankannya(t *testing.T) {
	memakai := map[string]bool{}
	for name, text := range query {
		if strings.Contains(strings.ToUpper(text), "@ASMD") {
			memakai[name] = true
		}
	}
	require.Equal(t, kueriDenganDBLink, memakai)
}

// Setiap kueri cadangan wajib punya pasangan tanpa DB Link, dan isinya wajib SAMA —
// berbeda hanya pada akhiran DB Link-nya.
//
// Tanpa uji ini, perbaikan pada salah satunya tidak akan ikut ke pasangannya, dan
// laporan akan berperilaku berbeda tergantung apakah `ANEKA_*` kebetulan terisi. Selisih
// seperti itu hampir mustahil ditelusuri: keduanya "jalan", hasilnya saja yang berbeda.
func TestKueriCadanganSamaDenganAslinya(t *testing.T) {
	const link = "@ASMD.SINARMAS.CO.ID"

	n := 0
	for name := range query {
		asli, cadangan := strings.CutSuffix(name, "_dblink")
		if !cadangan {
			continue
		}
		n++

		require.Truef(t, hasQuery(asli), "kueri cadangan %q tanpa pasangan %q", name, asli)
		require.Equal(t, badanKueri(query[asli]),
			strings.ReplaceAll(badanKueri(query[name]), link, ""),
			"isi %q berbeda dari %q di luar akhiran DB Link-nya", name, asli)
	}
	require.Equal(t, len(kueriCadanganDBLink), n, "jumlah kueri cadangan berubah")
}

// kueriCadanganDBLink adalah kueri cadangan yang dipakai selama `ANEKA_*` kosong.
var kueriCadanganDBLink = map[string]bool{
	"report_holiday_calendar_dblink": true,
	"report_mitra_logins_dblink":     true,
}

// badanKueri membuang baris komentar supaya yang dibandingkan SQL-nya saja.
func badanKueri(text string) string {
	var isi []string
	for _, l := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(l); t != "" && !strings.HasPrefix(t, "--") {
			isi = append(isi, t)
		}
	}
	return strings.Join(isi, "\n")
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

		// Kedua cadangan DB Link dipilih Repo.koneksiKedua saat ANEKA_* kosong, dan
		// pemilihnya pun tidak terlihat dari plans.
		"report_holiday_calendar_dblink": true,
		"report_mitra_logins_dblink":     true,
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
