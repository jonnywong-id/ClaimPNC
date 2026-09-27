// Package laporanhasilai adalah inti modul Laporan Hasil AI.
//
// # Apa yang dilaporkan layar ini
//
// Hasil penilaian **AI** atas sebuah klaim, disandingkan dengan **keputusan komite** yang
// menyusul. Satu baris = satu penilaian AI pada sebuah objek pertanggungan, beserta
// jawaban komite yang memutuskannya.
//
// Gunanya membandingkan keduanya: seberapa sering komite sependapat dengan AI, dan pada
// kasus seperti apa keduanya berbeda. Itulah sebabnya layar ini menggambar DUA grid —
// satu ringkasan pencacah di atas, satu rincian baris di bawah.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/Har_LaporanHasilAI-Harness.xml      layar "Laporan Hasil AI" (MENU_ID 82)
//	Section/SecLaporanHasilAI-Section.xml       isi layar — 2 isian, 2 tombol, 2 grid
//	Activity/SearchDataLaporanAI-Act.xml        pengisi kedua grid SEKALIGUS pembuat CSV
//	RDB List/CountAIDiterima_SQL-SQL.xml        kuerinya
//	Database/INSERTDATAAIKLAIMPNC.prc           daftar kolom T_CLAIM_DATA_RESULTS_AI
//	Database/m_menu_aplikasi_pnc.csv:77         butir menunya
//
// # KUERINYA TERTINGGAL DARI LAYARNYA — dan itu direplikasi, bukan diperbaiki
//
// `CountAIDiterima_SQL` ber-`pyMemo = "work in progress"`, dan itu bukan sekadar catatan
// pengembang: kuerinya **hanya mengisi 5 dari 10 kolom** yang digambar grid.
//
//	kolom grid           properti yang dibaca      diisi kueri?
//	No Klaim             .ClaimID                  ya
//	Object Name          .ObjectName               TIDAK
//	Komite Status        .KomiteAccepted           ya
//	Tanggal Komite       .TanggalComitee           ya
//	AI Status            .ResultAI                 ya
//	Tanggal AI           .TanggalAI                ya
//	Note AI Terima       .Notes                    TIDAK
//	Note AI Tolak        .NoteAkseptasi            TIDAK
//	Coverage Final       .COVERAGE_AI_FINAL        TIDAK
//	Kategori Kronologi   .KATEGORI_KRONOLOGI       TIDAK
//
// Dua di antaranya nyaris terisi: kueri memilih `NOTETERIMA AS "NoteAITerima"` dan
// `NOTETOLAK AS "NoteAITolak"`, sementara grid membaca `.Notes` dan `.NoteAkseptasi` —
// nama yang berbeda, sehingga nilainya tidak pernah sampai.
//
// Work Owner memutuskan pada 2026-09-26: **replikasi apa adanya**. Kelima kolom itu ada
// di layar dan di berkas CSV, dan isinya kosong — persis seperti Pega hari ini.
//
// Keputusan itu mudah dibalik bila kelak berubah, dan tempatnya sudah diketahui: tambahkan
// kelima kolomnya pada `repo/sqlstore/laporanhasilai.sql` dan isikan ke Row di `scanRow`.
// Kelimanya ADA di `POOLDATA.T_CLAIM_DATA_RESULTS_AI` — `RDB List/GetKomitePAditerima-SQL.xml`
// membacanya dari tabel yang sama, sehingga yang dibutuhkan hanyalah keputusan, bukan
// artefak baru.
//
// # Penamaan ulang yang WAJIB dilakukan di sini (D-19)
//
// Layar lama terikat properti klipboard kelas `ASM-FW-GCNMFW-Data-Adjustment` yang dipakai
// ulang dari layar lain, sehingga namanya tidak ada hubungannya dengan isinya:
//
//	grid ringkasan, kolom Keputusan   .BatasUmur       bukan batas umur
//	grid ringkasan, kolom Total       .NoteAITerima    bukan catatan
//	grid ringkasan, kolom Diterima    .BatasLapor      bukan batas lapor
//	grid ringkasan, kolom Ditolak     .NoteKomite      bukan catatan komite
//	isian Tgl Input Dari              .AnalystTransferDate
//	isian Tgl Input Sampai            .DateOfLoss      bukan tanggal kejadian
//
// Tidak satu pun dibawa. Yang dipakai di sini adalah padanan Inggris yang benar (`D-80`).
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package laporanhasilai

import (
	"context"
	"strings"
	"time"
)

// Nilai kolom RESULTAI yang dicacah grid ringkasan.
//
// Ketiganya dibaca dari precondition langkah pencacah pada
// `Activity/SearchDataLaporanAI-Act.xml`, yang berbunyi `.ResultAI=="DITERIMA"`,
// `.ResultAI=="DITOLAK"`, dan `.ResultAI==""`.
//
// Perbandingannya di sana PERSIS, bukan mengandung. Yang dilakukan di sini hanya
// memangkas spasi lebih dulu — kolom bertipe CHAR berlebar tetap memadatkan nilainya
// dengan spasi tanpa memberi tanda apa pun, dan tanpa pemangkasan itu baris yang sah akan
// terhitung sebagai "belum dinilai".
const (
	AIAccepted = "DITERIMA"
	AIRejected = "DITOLAK"
)

// Nilai kolom STATUSAPPROVE pada POOLDATA.T_CLAIM_KOMITE_LIST.
//
// Dibaca dari `CASE` pada `RDB List/CountAIDiterima_SQL-SQL.xml` — `1` diterima, `2`
// ditolak, `0` menunggu — dan dikuatkan precondition langkah pencacah yang membandingkannya
// sebagai TEKS: `.KOMITESTATUS=="1"`.
//
// # Kenapa teks, bukan angka
//
// Tipe kolomnya belum diketahui; DDL-nya belum ada (`R-08`). Kueri lama membandingkannya
// dengan angka (`B.STATUSAPPROVE = 1`) sementara rule when membandingkannya dengan teks.
// Modul `komite` sudah menempuh jalan yang sama dan membacanya sebagai teks
// (`komite/repo/sqlstore/inbox.go` fungsi `legacyOutcome`); mengikutinya membuat kedua
// modul menafsirkan satu kolom dengan satu cara.
const (
	CommitteeApprovedCode = "1"
	CommitteeRejectedCode = "2"
	CommitteePendingCode  = "0"
)

// Label keputusan yang dibaca pengguna.
//
// Nilainya SAMA PERSIS dengan `CASE` kueri lama, termasuk huruf besarnya. Ia teks yang
// dilihat pengguna, sehingga tetap berbahasa Indonesia (`D-80`).
const (
	LabelAccepted = "DITERIMA"
	LabelRejected = "DITOLAK"
	LabelPending  = "MENUNGGU"
)

// Subject menamai pihak yang dicacah grid ringkasan.
//
// Keduanya berasal dari `TempTotal.pxResults(<APPEND>).BatasUmur := "Komite"` dan
// `:= "AI"` pada `Activity/SearchDataLaporanAI-Act.xml` — teks yang benar-benar tergambar
// di kolom "Keputusan".
const (
	SubjectAI        = "AI"
	SubjectCommittee = "Komite"
)

// Row adalah satu baris grid rincian.
//
// # Satu baris = satu penilaian AI, BUKAN satu klaim
//
// Kueri lama menggabungkan `T_CLAIM_DATA_RESULTS_AI` dengan `T_CLAIM_KOMITE_LIST` secara
// langsung, dan tabel penilaian AI memuat satu baris per objek pertanggungan. Sebuah klaim
// beranggota tiga objek karena itu menghasilkan tiga baris.
//
// Itu bukan penggandaan yang terlewat: kolom "Object Name", "Coverage Final", dan
// "Kategori Kronologi" hanya masuk akal bila barisnya memang per objek. Modul `komite`
// justru MENGAGREGASI tabel yang sama supaya inbox-nya tidak berganda — dua layar, dua
// kebutuhan, dan keduanya benar untuk dirinya sendiri.
type Row struct {
	// ID adalah kunci baris untuk tabel di layar. Ia TIDAK digambar sebagai kolom.
	//
	// Ia disusun dari empat kolom kunci alami, bukan satu: tidak ada satu kolom pun yang
	// diketahui unik per baris hasil gabungan, dan DDL yang dapat membuktikannya belum ada
	// (`R-08`). Susunannya sama dengan urutan pengurutan, sehingga kunci yang sama selalu
	// menunjuk baris yang sama di halaman mana pun.
	ID string

	// ClaimNumber adalah kolom "No Klaim" — `T_CLAIM_KOMITE_LIST.NO_KLAIM`.
	//
	// # Ia SENGAJA kosong pada jenjang komite kedua ke atas
	//
	// Kueri lama menulis `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`. Nomor
	// klaim hanya tampil pada baris jenjang pertama; baris jenjang berikutnya bernomor
	// kosong, sehingga satu klaim terbaca sebagai satu kelompok.
	//
	// Work Owner memutuskan pada 2026-09-26 untuk MENIRUNYA. Akibatnya diterima secara
	// sadar: berkas CSV hasil ekspor memuat sel kosong pada baris lanjutan, sehingga ia
	// tidak dapat disaring maupun di-pivot menurut nomor klaim tanpa diisi lebih dulu.
	ClaimNumber string

	// ObjectName adalah kolom "Object Name" — `T_CLAIM_DATA_RESULTS_AI.OBJECTNAME`.
	//
	// SELALU KOSONG. Kuerinya tidak memilih kolom ini. Lihat bagian "KUERINYA TERTINGGAL
	// DARI LAYARNYA" pada doc paket.
	ObjectName string

	// CommitteeStatus adalah kolom "Komite Status" — label yang dibaca pengguna.
	//
	// Diturunkan dari CommitteeStatusCode lewat CommitteeLabel, bukan dibaca dari basis
	// data: kueri lama pun menurunkannya dengan `CASE`, dan menurunkannya di satu tempat
	// membuat label dan pencacah ringkasan tidak dapat berbeda.
	CommitteeStatus string

	// CommitteeStatusCode adalah nilai mentah `STATUSAPPROVE`.
	//
	// Ia dibawa sampai ke layar, tetapi TIDAK digambar sebagai kolom. Alasannya satu:
	// tanpa nilai mentahnya, "MENUNGGU" dan "kode yang tidak dikenal" terlihat sama di
	// layar sementara keduanya berarti hal yang berbeda saat ditelusuri.
	CommitteeStatusCode string

	// CommitteeDate adalah kolom "Tanggal Komite" — `T_CLAIM_KOMITE_LIST.TANGGALKOMITE`.
	//
	// Nol berarti kolomnya NULL. Itu tidak mungkin terjadi pada baris yang lolos penyaring
	// — `TANGGALKOMITE IS NOT NULL` adalah bagian dari syaratnya — tetapi tipenya tetap
	// membolehkannya, dan layar menampilkan tanda pisah alih-alih tanggal nol.
	CommitteeDate time.Time

	// AIStatus adalah kolom "AI Status" — `T_CLAIM_DATA_RESULTS_AI.RESULTAI`.
	//
	// Isinya DITERIMA, DITOLAK, atau kosong. Kosong berarti AI belum menilai baris itu,
	// dan barisnya TETAP muncul — yang dilaporkan layar ini justru termasuk kasus yang
	// komitenya memutuskan tanpa penilaian AI.
	AIStatus string

	// AIDate adalah kolom "Tanggal AI" — `T_CLAIM_DATA_RESULTS_AI.TGLAI`.
	AIDate time.Time

	// AcceptNote adalah kolom "Note AI Terima" — `NOTETERIMA`.
	//
	// SELALU KOSONG; lihat ObjectName.
	AcceptNote string

	// RejectNote adalah kolom "Note AI Tolak" — `NOTETOLAK`.
	//
	// SELALU KOSONG; lihat ObjectName.
	RejectNote string

	// CoverageFinal adalah kolom "Coverage Final" — `COVERAGE_AI_FINAL`.
	//
	// SELALU KOSONG; lihat ObjectName.
	CoverageFinal string

	// ChronologyCategory adalah kolom "Kategori Kronologi" — `KATEGORI_KRONOLOGI`.
	//
	// SELALU KOSONG; lihat ObjectName.
	ChronologyCategory string
}

// CommitteeLabel menerjemahkan STATUSAPPROVE menjadi label yang dibaca pengguna.
//
// # Cabang `else` DIPERTAHANKAN kosong, dan itu berbeda dari modul komite
//
// Kueri layar ini menutup `CASE`-nya dengan `ELSE ''` — kode yang tidak dikenal
// menghasilkan sel kosong. Di modul `komite`, rule lamanya justru menutup dengan
// `ELSE 'DITOLAK'`, sehingga kasus yang belum diputuskan terbaca sudah ditolak; modul itu
// menolak menirunya karena akibatnya menyesatkan pada sebuah daftar pekerjaan.
//
// Di sini tidak ada yang perlu ditolak: `ELSE ''` sudah jujur — ia tidak mengaku tahu.
// Yang ditiru adalah kuerinya sendiri, bukan pola dari layar lain.
func CommitteeLabel(code string) string {
	switch strings.TrimSpace(code) {
	case CommitteeApprovedCode:
		return LabelAccepted
	case CommitteeRejectedCode:
		return LabelRejected
	case CommitteePendingCode:
		return LabelPending
	default:
		return ""
	}
}

// Tally adalah satu baris grid ringkasan.
type Tally struct {
	// Subject adalah isi kolom "Keputusan" — SubjectAI atau SubjectCommittee.
	Subject string

	// Accepted adalah isi kolom "Diterima".
	Accepted int

	// Rejected adalah isi kolom "Ditolak".
	Rejected int

	// Pending adalah isi kolom "Menunggu".
	//
	// # Kolom ini TIDAK ADA di layar lama, dan itu penambahan yang disengaja
	//
	// Work Owner memutuskan pada 2026-09-26 untuk menambahkannya. Alasannya ada pada doc
	// Total di bawah: tanpa kolom ini, selisih antara Total dan jumlah baris grid tidak
	// dapat dijelaskan siapa pun yang melihat layarnya.
	//
	// Ia tidak mengubah satu pun angka yang sudah ada — Total tetap dihitung dengan cara
	// yang sama, dan Menunggu hanya menampakkan sisa yang selama ini tidak tergambar.
	Pending int
}

// Total adalah isi kolom "Total".
//
// # Ia BUKAN jumlah baris, dan itu ditiru apa adanya
//
// `Activity/SearchDataLaporanAI-Act.xml` mengisinya `Local.terima + Local.tolak` — hanya
// yang diterima dan yang ditolak. Baris ber-AI Status kosong dan ber-Komite Status
// MENUNGGU tidak ikut dihitung.
//
// Akibatnya Total dapat lebih kecil daripada jumlah baris grid rincian, dan itu BUKAN
// salah hitung. Kolom Menunggu ada supaya selisihnya terbaca.
func (t Tally) Total() int { return t.Accepted + t.Rejected }

// Rows adalah jumlah SELURUH baris yang dicacah — termasuk yang menunggu.
//
// Ia tidak digambar sebagai kolom. Gunanya satu: pembuktian di pengujian bahwa pencacah
// ringkasan dan pencacah paginasi membaca himpunan baris yang sama.
func (t Tally) Rows() int { return t.Accepted + t.Rejected + t.Pending }

// Summary adalah isi grid ringkasan — dua baris, pada urutan yang tergambar.
//
// Urutannya Komite lebih dulu, lalu AI, mengikuti urutan `<APPEND>` pada activity lamanya.
// Ia tampak sepele, tetapi mengubahnya berarti layar baru berbeda dari layar yang sudah
// dihafal penggunanya (`D-13`).
type Summary struct {
	Committee Tally
	AI        Tally
}

// Tallies mengembalikan kedua baris pada urutan yang tergambar di layar.
func (s Summary) Tallies() []Tally {
	return []Tally{s.Committee, s.AI}
}

// Filter adalah penyaring layar — dua isian tanggal, dan tidak ada yang lain.
//
// # Labelnya "Tgl Input", isinya TANGGAL KOMITE
//
// Kedua isian di layar lama berlabel "Tgl Input Dari" dan "Tgl Input Sampai". Klausa yang
// disusun activity dari keduanya berbunyi:
//
//	AND trunc(TANGGALKOMITE) >= to_date(awal) and trunc(TANGGALKOMITE) <= to_date(akhir)
//
// Yang disaring adalah `T_CLAIM_KOMITE_LIST.TANGGALKOMITE` — kolom yang juga digambar
// sebagai "Tanggal Komite". Labelnya menyesatkan sejak di Pega.
//
// Work Owner memutuskan pada 2026-09-26: **label lama dipakai apa adanya**, tanpa
// keterangan tambahan (`D-13` — teks mengikuti layar Pega). Kebenarannya dicatat di sini
// dan di dokumen, bukan di layar.
type Filter struct {
	// From adalah batas bawah, INKLUSIF.
	From time.Time

	// To adalah batas atas yang DIKETIK PENGGUNA, dan ia INKLUSIF.
	//
	// Yang dikirim ke basis data bukan nilai ini melainkan ToExclusive — lihat alasannya
	// di sana.
	To time.Time
}

// Clean menormalkan kedua tanggal menjadi tanggal kalender polos.
//
// Jam dibuang, dan zona waktunya TIDAK dikonversi. Keduanya disengaja:
//
//   - Jam dibuang karena `trunc(...)` pada kueri lama pun membuangnya, dan isian layarnya
//     memang hanya menerima tanggal.
//   - Zona tidak dikonversi karena kolom pembandingnya `DATE` Oracle — jam dinding tanpa
//     zona. Mengonversinya ke UTC akan menggeser tanggalnya sehari pada sebagian nilai,
//     dan pergeseran itu tepat `R-12`. Tidak ada penambahan tujuh jam di mana pun
//     (`08-TECHNICAL-STRATEGY.md` §4.4).
func (f Filter) Clean() Filter {
	return Filter{From: dateOnly(f.From), To: dateOnly(f.To)}
}

// ToExclusive adalah batas atas yang dikirim ke basis data — EKSKLUSIF.
//
// # Kenapa bukan `<=` seperti kueri lama
//
// Kueri lama menulis `trunc(TANGGALKOMITE) <= to_date(akhir)`. `TRUNC` pada kolom
// mematikan index-nya dan tidak portabel ke PostgreSQL (`09-DATABASE-STRATEGY.md` §4).
//
// Penggantinya `>= awal AND < akhir + 1 hari`, yang MEMILIH BARIS YANG SAMA PERSIS dan
// tetap dapat memakai index. Pola yang sama sudah dipakai `inboxrclpucl` dan
// `inboxoutstanding`; yang ditambahkan modul ini hanyalah pemakaian, bukan cara baru.
func (f Filter) ToExclusive() time.Time {
	return dateOnly(f.To).AddDate(0, 0, 1)
}

// Validate mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// # Kenapa KEDUA tanggal wajib
//
// Layar lama tidak dapat berjalan tanpanya, dan itu bukan tafsiran: activity-nya menyusun
// klausa `to_date('<isian>','dd/mm/yyyy')` dari potongan teks isian tanggal. Isian yang
// kosong menghasilkan `to_date('//','dd/mm/yyyy')`, dan Oracle menolaknya. Jejaknya bahkan
// tertinggal di `pyMemo` activity itu sendiri:
//
//	"TempDatalaporanAI.NoteAITerima bikin error pdc ORA-00907: missing right parenthesis"
//
// Yang berubah di sini hanyalah BENTUK penolakannya: pesan yang menunjuk isian mana yang
// belum diisi, bukan galat basis data. Kesetaraan perilaku tidak menuntut kesetaraan cara
// gagal ketika cara gagal yang lama adalah galat mentah.
func (f Filter) Validate() error {
	var violations []Violation

	clean := f.Clean()

	if clean.From.IsZero() {
		violations = append(violations, Violation{
			Field:   FieldDateFrom,
			Message: "Tgl Input Dari belum diisi.",
		})
	}
	if clean.To.IsZero() {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: "Tgl Input Sampai belum diisi.",
		})
	}

	// Rentang terbalik hanya diperiksa bila keduanya ada. Memeriksanya saat salah satu
	// kosong akan menambahkan pesan ketiga yang membingungkan: pengguna diminta
	// membetulkan urutan tanggal yang salah satunya belum ada.
	if len(violations) == 0 && clean.To.Before(clean.From) {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: "Tgl Input Sampai tidak boleh lebih awal daripada Tgl Input Dari.",
		})
	}

	if len(violations) == 0 {
		return nil
	}
	return NewValidationError(violations)
}

// dateOnly membuang jam dan menyisakan tanggal kalendernya.
//
// Ia memakai komponen tanggalnya apa adanya dan menyusun ulang di UTC. Bukan konversi
// zona: yang dihasilkan adalah tanggal kalender yang sama dengan yang diketik pengguna,
// dan driver mengirimkannya sebagai `DATE` tanpa zona — persis yang dibandingkan kolomnya.
func dateOnly(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// Batas paginasi.
//
// Mengikuti `10-API-STRATEGY.md` §4: permintaan yang melebihi MaxPageSize DIBETULKAN ke
// batas itu, bukan dipenuhi.
//
// # Layar lama TIDAK memaginasi di server, dan itu bukan alasan untuk tidak melakukannya
//
// Grid lamanya memuat seluruh hasil ke klipboard lalu memaginasinya di peramban — persis
// pola "3.189 grid terikat page list klipboard" yang `15-NFR-PERFORMANCE-SCALABILITY.md`
// §3.2 sebut sebagai masalah nyata. Paginasi di server adalah PERUBAHAN PERILAKU yang
// disengaja dan sudah diputuskan di tingkat Steering, bukan penyimpangan modul ini.
//
// Yang penting: ringkasan di atas grid TIDAK ikut dipaginasi. Ia dihitung atas seluruh
// baris yang cocok, lewat kueri agregat tersendiri. Ringkasan yang hanya mencacah halaman
// yang sedang terlihat adalah angka yang berubah saat pengguna menekan "berikutnya", dan
// tidak ada satu pun di layar yang akan menjelaskan kenapa.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// Pagination adalah jendela halaman yang diminta.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah banyaknya baris per halaman.
	Size int
}

// Normalize membetulkan paginasi ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: nomor halaman datang dari tautan
// paginasi, dan tautan yang basi bukan kesalahan yang dapat diperbaiki pengguna dengan
// mengetik.
func (p Pagination) Normalize() Pagination {
	clean := p
	if clean.Page < 1 {
		clean.Page = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah banyaknya baris yang dilewati untuk mencapai halaman ini.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// Page adalah satu halaman grid rincian.
type Page struct {
	// Rows adalah baris pada halaman ini, paling banyak Pagination.Size.
	Rows []Row

	// Total adalah cacah SELURUH baris yang cocok dengan penyaringnya.
	Total int
}

// Result adalah seluruh isi layar setelah tombol "Cari Data" ditekan.
//
// Keduanya dikembalikan bersama karena layar lama pun mengisinya dalam SATU kali jalan —
// `SearchDataLaporanAI` mengisi `DatasearchLaporan` dan `TempTotal` berurutan. Memisahkan
// keduanya menjadi dua permintaan membuka kemungkinan ringkasan dan rinciannya dibaca dari
// keadaan basis data yang berbeda.
type Result struct {
	Page    Page
	Summary Summary
}

// Repo adalah seam ke penyimpanan Laporan Hasil AI SATU portal.
type Repo interface {
	// List mengembalikan satu halaman rincian beserta cacah seluruh baris yang cocok.
	//
	// Filter sudah harus melewati Clean dan Validate.
	List(ctx context.Context, filter Filter, page Pagination) (Page, error)

	// Summarize mencacah seluruh baris yang cocok, dipecah menurut keputusan AI dan
	// keputusan komite.
	//
	// Ia TERPISAH dari List dan menerima penyaring yang sama persis. Itu yang membuat
	// angka ringkasan dan isi grid selalu berbicara tentang himpunan baris yang sama.
	Summarize(ctx context.Context, filter Filter) (Summary, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan penilaian AI
// atas klaim satu badan hukum kepada pengguna badan hukum lain tanpa satu pun pesan galat
// (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
