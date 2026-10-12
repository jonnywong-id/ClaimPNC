package registrasi

import (
	"context"
	"strings"
	"time"
)

// Berkas ini memodelkan modal **"Kirim ke RCL/PUCL"** — local action `KomentarRCLPUCL`
// (`Flow Action/KomentarRCLPUCL-FA.xml`, judul "KomentarRCL/PUCL") di atas
// `Section/SectionPUCL-sect.xml`, yang dibuka dari tombol sel ke-6 `ClaimSurvey_sect`.
//
// # Empat artefak pendukungnya TIDAK ADA di export (`R-16`)
//
//	isiDefaultPUCLRCL      mengisi nilai bawaan saat pilihan RCL/PUCL berubah
//	InputKeteranganIsi     tombol "Pilih" pada grid alasan — mengisi Keterangan Isi
//	BrowseReasonReject_RD  sumber grid alasan
//	BrowsePerihalRCLPUCL_RD sumber pilihan Perihal
//
// Yang dapat dibaca hanyalah NAMA keempatnya, properti yang mereka sentuh, dan tabel
// yang mereka baca. Perilakunya di sini karena itu diturunkan dari tabelnya langsung —
// dibaca dari katalog Oracle 2026-10-04 — bukan dari rule-nya. Setiap penyimpulan
// seperti itu ditandai di tempatnya.
//
// # Penampungnya SUDAH ADA, dan itu bukan dugaan
//
// `POOLDATA.TC_PNC_PUCL` (34 kolom) sudah dibaca modul `inboxrclpucl` sebagai sumber
// ketiga tab layar Inbox RCL/PUCL. Modal ini menulis ke tabel yang sama, sehingga klaim
// yang dikirim dari sini MUNCUL di tab "Cetak Surat" layar itu — bukan hilang ke tabel
// yang tidak dibaca siapa pun.

// TIGA jalur penanganan — `.ClaimData.PUCLStatus.RCL_PUCL`, radio "Pilih RCL / PUCL".
//
// Ketiganya TERBUKTI dari export, bukan ditebak dari urutan. `Activity/PUCLPost-Act.xml`
// punya tiga langkah yang masing-masing menetapkan kategori surat, dan prekondisi tiap
// langkah menyebut kodenya:
//
//	RCL_PUCL==2  ->  kategori "PUCL"          (:887, prekondisi :1092)
//	RCL_PUCL==1  ->  kategori "RCL"           (:1241, prekondisi :1471)
//	RCL_PUCL==3  ->  kategori "Notification"  (:1635, prekondisi :1866)
//
// Dua di antaranya dikuatkan dari arah lain: sebuah ekspresi `@if` berbunyi
// `RCL_PUCL_1 = '1' then 'RCL'` dan `= '2' then 'PUCL'`, dan nilai `2` itu pula yang
// diuji `When/IsPUCL-When.xml`. Ketiganya juga sudah dimodelkan modul `inboxrclpucl`
// sebagai `TrackCodeRCL`, `TrackCodePUCL`, dan `TrackCodeNotification`.
//
// Satu nilai lain hidup di data produksi dan BUKAN pilihan modal ini: `0` — belum
// dipilih, 32 baris.
const (
	PUCLTrackRCL          = 1
	PUCLTrackPUCL         = 2
	PUCLTrackNotification = 3
)

// PUCLTrackName menerjemahkan kode jalur menjadi nama yang dipakai layar lama.
// Kode di luar kedua nilai itu mengembalikan teks kosong — pemanggil yang memutuskan
// apa artinya, persis seperti `inboxrclpucl.TrackOf`.
func PUCLTrackName(track int) string {
	switch track {
	case PUCLTrackRCL:
		return "RCL"
	case PUCLTrackPUCL:
		return "PUCL"
	case PUCLTrackNotification:
		return "Notification"
	}
	return ""
}

// Panjang kolom `POOLDATA.TC_PNC_PUCL`, dibaca dari katalog 2026-10-04. Dipakai
// memvalidasi di domain alih-alih membiarkan Oracle menolaknya dengan ORA-12899, yang
// tidak dapat ditunjukkan ke pengguna sebagai pesan per isian.
const (
	MaxPUCLSubject     = 4000 // PERIHAL
	MaxPUCLNote        = 4000 // KETERANGAN1 · KETERANGAN2 · KETERANGAN3
	MaxPUCLAnalystNote = 4000 // KOMENTAR_ANALISATOR

	// MaxPUCLDoctorName adalah batas KARAKTER "Nama Dokter" — dan ia ditentukan oleh kolom
	// yang paling sempit dari DUA tabel, bukan satu.
	//
	//	T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1   VARCHAR2(128 CHAR)
	//	TC_PNC_PUCL.NAMA_DOKTER_RCL         VARCHAR2(255 BYTE)   (Work Owner, 2026-10-07)
	//
	// Nilai yang sama ditulis ke keduanya, sehingga yang berlaku adalah yang lebih dulu
	// penuh. Untuk teks ASCII itu **128 karakter**.
	//
	// Angkanya sempat 512 — lebih longgar daripada KEDUA kolomnya. Isian itu dulu teks
	// bebas, jadi nama panjang yang diketik analis lolos validasi lalu ditolak Oracle dengan
	// ORA-12899 — galat yang tidak dapat ditunjukkan sebagai pesan per isian. Dropdown dua
	// baris membuatnya tidak pernah terpicu, bukan membuatnya benar.
	MaxPUCLDoctorName = 128

	// MaxPUCLDoctorNameBytes adalah batas BYTE kolom `TC_PNC_PUCL.NAMA_DOKTER_RCL`.
	//
	// Diperiksa TERPISAH dari batas karakter karena satuannya memang berbeda: 128 karakter
	// non-ASCII dapat menjadi 384 byte pada AL32UTF8 dan menembus kolom ini walau batas
	// karakternya terpenuhi. Memeriksa satu saja meninggalkan celah yang hanya muncul pada
	// nama tertentu — kelas cacat yang paling sulit ditelusuri.
	MaxPUCLDoctorNameBytes = 255
)

// Nilai yang menempatkan baris baru di tab **"Cetak Surat"** layar Inbox RCL/PUCL.
//
// Ketiganya adalah penyaring tab itu, dibaca dari
// `internal/inboxrclpucl/repo/sqlstore/inboxrclpucl.sql` kueri `list_cetak_surat`:
//
//	STATUS_WORK             <> 'Resolved-Completed'   -> WorkStatusNew
//	TGL_CETAK_DOKUMEN_PUCL  IS NULL                   -> tidak diisi
//	STATUS_CASE             =  '0'                    -> PUCLStatusCaseOpen
//
// `STATUS_WORK` memakai `WorkStatusNew` — kosakata PEGA, bukan kosakata proses aplikasi
// ini (`ProcessRunning`). Kolomnya sepadan dengan `T_CLAIMLIST_ADMIN.PYSTATUSWORK`, dan
// kedua baris yang sudah ada di tabel ini pun berbunyi `New`.
//
// `TGL_CETAK_DOKUMEN_PUCL` sengaja TIDAK diisi: ia diisi tombol "Download Dokumen" pada
// layar itu, yang sekaligus memindahkan baris ke tab berikutnya.
const PUCLStatusCaseOpen = "0"

// PUCLLetter adalah satu baris `POOLDATA.TC_PNC_PUCL` — surat RCL/PUCL yang baru
// diajukan analis.
//
// Kolom bernama rangkap (POLICY_NO, QQ_NAME, BUSINESS_NAME, …) adalah SNAPSHOT klaim,
// bukan kunci asing. Tabelnya memang rata: layar Inbox RCL/PUCL membacanya tanpa join
// satu pun, dan itulah alasan kolom-kolom itu ada.
type PUCLLetter struct {
	// ClaimID adalah `CLAIMID` — nomor klaim apa adanya (`PNCN.YY.xxxx`), bukan
	// `pzInsKey` berawalan kelas Pega (`D-22`, `D-71`).
	ClaimID string

	// Track adalah `RCL_PUCL`: PUCLTrackRCL, PUCLTrackPUCL, atau PUCLTrackNotification.
	Track int

	// AnalystNote adalah "Catatan untuk RCL/PUCL" — `KOMENTAR_ANALISATOR`,
	// `.ClaimData.PUCLStatus.KomentarAnalisator`. Section mewajibkannya.
	AnalystNote string

	// Subject adalah "Perihal" — `PERIHAL`, `.ClaimData.PUCLStatus.Perihal`.
	//
	// Di Pega ia TEKS, bukan kode: `InputPerihalRCLPUCL_act` menyalin
	// `PERIHAL_NAME` master ke properti klaim. Yang tersimpan karena itu dapat
	// menyimpang dari master — dan memang menyimpang: kedua baris yang ada di
	// produksi berbunyi "Kelengkapan Data Dokumen…" sedangkan masternya
	// "Kelengkapan Data dan Dokumen…". Modal ini tetap menawarkan masternya sebagai
	// pilihan, tetapi menyimpan teksnya.
	Subject string

	// Tiga keterangan surat — `KETERANGAN1`, `KETERANGAN2`, `KETERANGAN3`.
	// Hanya yang kedua ("Keterangan Isi") diwajibkan section.
	OpeningNote string
	BodyNote    string
	ClosingNote string

	// DoctorName adalah "Nama Dokter" — `.ClaimData.NamaDokterRCL`.
	//
	// Isiannya tampil bila `RCL_PUCL != 2 && IsPA` — yakni jalur RCL ATAU Notification
	// pada lini Personal Accident, bukan jalur RCL saja. `TC_PNC_PUCL` TIDAK punya
	// kolom untuknya; penampungnya `T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1`, yang dibaca
	// layar Inbox RCL.
	//
	// # Kolom ini BUKAN sekadar keterangan — ia penyaring
	//
	// Inbox RCL menyaring `UPPER(TRIM(NAMADOKTERRCL_1))` terhadap identitas pemanggil,
	// jadi yang tersimpan di sini menentukan SIAPA yang melihat klaimnya. Nama yang
	// tidak cocok dengan identitas siapa pun membuat klaim berjalur RCL tidak muncul di
	// inbox mana pun — hilang tanpa satu pun galat.
	//
	// Karena itu, pada jalur RCL yang isiannya kosong — dan ia memang kosong di seluruh
	// lini selain PA, karena layar tidak menampilkannya — nilainya diisi PEMILIK tugas
	// RCLDokter. Dengan begitu penyaring "dokter" dan penyaring "pemilik tugas"
	// (`PXASSIGNEDOPERATORID`) menunjuk orang yang sama. Lihat usecase.SendToRCLPUCL.
	// Nama yang dipilih di sini BELUM memindahkan klaimnya ke akun dokter itu — ditahan
	// Work Owner (2026-10-06) sampai pemetaannya ke `M_LOGIN_PNC.LOGIN_ID` terverifikasi.
	// Yang memindahkan klaim adalah `ASSIGNED_OPERATOR_ID`, bukan kolom ini; lihat
	// `usecase.SendToRCLPUCL`.
	DoctorName string

	// Tiga penunjuk adjustment yang menjadi pokok surat ini — `ID_OBJECT`,
	// `ID_COVERAGE`, `ID_ADJUSTMENT`.
	//
	// # Untuk apa Pega memakainya
	//
	// `PUCLPost` menerimanya sebagai `idObj`, `idCov`, `idAdj` (terlihat pada konfigurasi
	// tombol "Kirim Ke Analyst"), lalu memakainya MENCARI baris yang tepat:
	//
	//	.ObjectID   == param.idObj
	//	.CoverageID == param.idCov
	//
	// dan pada baris yang cocok ia menstempel `AdjustmentList(<LAST>).PUCLStatus.PUCLApprove`
	// serta `.KomentarPUCL`. Keduanya properti clipboard — Pega TIDAK menyimpannya ke
	// `T_CLAIM_ADJUSTMENT`, dan tabel itu memang tidak punya kolomnya (katalog 2026-10-05).
	// Karena itu aplikasi ini pun tidak menulis ke sana (keputusan Work Owner 2026-10-05).
	//
	// # Asalnya, dan SATU ASIMETRI DI PEGA SENDIRI
	//
	// `Activity/ValidationAdjustment-act.xml` — yang di aplikasi ini berpadanan dengan
	// penyiapan adjustment (tombol "Tambah") — menulis:
	//
	//	PUCLStatus.IDObject    :=  .ObjectID          <- ID objek
	//	PUCLStatus.IDCoverage  :=  .pxListSubscript   <- INDEKS jaminan, bukan ID-nya
	//
	// Perhatikan: yang DITULIS sebagai IDCoverage adalah indeks, sementara yang
	// DIBANDINGKAN `PUCLPost` adalah `.CoverageID`. Keduanya hanya cocok bila ID jaminan
	// kebetulan sama dengan nomor urutnya — dan pada data Pega yang ada memang begitu
	// (keduanya `1`), tetapi tidak pada klaim aplikasi ini (`COVERAGEID = 10003`).
	//
	// Yang diikuti di sini adalah PENULISNYA, karena hanya penulis itu yang terbaca di
	// export; asimetrinya dibawa apa adanya (`P-5`) dan dicatat supaya tidak tampak
	// sebagai kekeliruan pembacaan.
	//
	// `IDAdjustment` TIDAK PERNAH di-set di satu rule pun dalam export — ia hanya dibaca
	// kedua section RCL/PUCL. Sejalan dengan itu, langkah yang menstempel pun memakai
	// `AdjustmentList(<LAST>)`, bukan `idAdj`. Di sini ia diisi nomor urut adjustment
	// terakhir, yaitu baris yang sama dengan yang `<LAST>` tunjuk.
	//
	// # Ketiganya diisi SAAT KIRIM, bukan ditinggalkan kosong
	//
	// Ketiga nilai memakai kunci yang SAMA PERSIS dengan `T_CLAIM_ADJUSTMENT`
	// (`settlement.sql`): `OBJECTID` ID objek, `OBJECTCOVERAGEID` nomor urut jaminan di
	// dalam objek, `ADJUSTMENTID` nomor urut baris. Jadi ketiganya memang dapat dipakai
	// menemukan barisnya kembali, bukan sekadar catatan.
	//
	// Klaim yang belum punya satu pun adjustment — tombol ini ada sejak tahap Choose
	// Surveyor, jauh sebelum adjustment ditambahkan — tetap membawa objek dan jaminan
	// TERAKHIRNYA, sehingga surat selalu menunjuk pokok yang dibicarakan. Yang dibiarkan
	// kosong hanya `AdjustmentIndex`: menulis `1` di sana akan menunjuk baris yang tidak
	// ada, dan itu lebih buruk daripada kosong.
	ObjectID        string
	CoverageIndex   int
	AdjustmentIndex int

	// Snapshot klaim — kolom rata yang dibaca layar Inbox RCL/PUCL tanpa join.
	PolicyNumber string
	InsuredName  string
	BusinessName string
	BranchName   string
	SourceName   string
	GroupPanel   string
	TechnicalPIC string
	ClaimStatus  string

	// ClaimAdmin adalah admin klaim — `ClaimData.UserAdmin`, tersimpan di
	// `T_CLAIM_PNC.ADMINKLAIM` dan dikenal di sini sebagai `Claim.CreatedBy`.
	//
	// Dipakai HANYA oleh jalur PUCL; lihat AssignedOperator.
	ClaimAdmin string

	// Operator adalah `OPERATOR_ID` — analis yang menekan Kirim.
	//
	// `ASSIGNED_OPERATOR_ID` TIDAK lagi diisi nilai yang sama; lihat AssignedOperator.
	Operator string

	// DateOfLoss mengisi `DATE_OF_LOSS`; SentAt mengisi `TGL_CREATE_PUCL` (kunci
	// bersama CLAIMID) sekaligus `TGL_KIRIM_PUCL` ("Tanggal Masuk Inbox" di layar).
	DateOfLoss time.Time
	SentAt     time.Time
}

// AssignedOperator adalah nilai `ASSIGNED_OPERATOR_ID` — **bukan** analis yang menekan
// Kirim, dan **bukan** nama antrean `RCLPUCL`.
//
// # Siapa pemiliknya bergantung pada JALUR
//
//	PUCL (2)          admin klaim  — klaim langsung masuk antrean RCL/PUCL
//	RCL (1) · Notif (3)  PIC Teknik — klaim singgah di Inbox RCL lebih dulu
//
// Pembagian itu ditetapkan Work Owner pada 2026-10-07 ("saat pilih PUCL maka langsung
// `ASSIGNED_OPERATOR_ID` UserAdmin"), dan ia sejalan dengan tujuan masing-masing jalur:
// hanya PUCL yang tidak singgah di dokter (lihat InRCLPUCLQueue), sehingga hanya ia yang
// tidak membutuhkan pemilik yang dapat dicocokkan penyaring Inbox RCL.
//
// Keputusan Setuju di Inbox RCL memindahkan kolom ini ke admin klaim juga — dengan kata
// lain, begitu klaim benar-benar berada di antrean RCL/PUCL, pemiliknya adalah admin,
// dari jalur mana pun ia tiba (`inboxrcl/repo/sqlstore/decision.go`).
//
// # Kenapa kolom ini tidak boleh berisi analis
//
// Inbox RCL (layar dokter RCL) menyaring TEPAT dengan kolom ini dan tidak dengan yang
// lain:
//
//	UPPER(TRIM(p.ASSIGNED_OPERATOR_ID)) = UPPER(<LOGIN_ID pemanggil>)
//
// Selama kolom itu berisi analis, klaim berjalur RCL mendarat di Inbox RCL **analis
// sendiri** — orang yang baru saja melepasnya — bukan di inbox petugas yang harus
// menanganinya. Cacat itu pernah dicatat sebagai "ditahan" di `usecase.SendToRCLPUCL`
// menunggu pemetaan nama dokter ke `M_LOGIN_PNC.LOGIN_ID`; Work Owner menjawabnya pada
// 2026-10-07 dengan menetapkan **user teknis** sebagai pemiliknya, sehingga pemetaan itu
// tidak lagi dibutuhkan: PIC Teknik memang sudah login yang sah di aplikasi ini.
//
// # Kenapa analis tetap menjadi cadangan, bukan NULL
//
// Tombol "Kirim ke RCL/PUCL" sudah ada sejak tahap Choose Surveyor, dan klaim yang belum
// pernah melewati tahap berouter PIC Teknik bisa saja belum punya `TechnicalPIC`.
// Mengosongkan kolom pada kasus itu membuat klaimnya tidak muncul di Inbox RCL siapa pun
// — hilang tanpa satu pun galat, kelas cacat yang paling mahal di modul ini. Analis yang
// mengirimnya adalah satu-satunya identitas yang PASTI ada pada saat itu, sehingga ia
// yang dipakai: klaimnya tetap terlihat oleh seseorang yang mengenalnya.
// Urutan cadangannya pun berbeda menurut jalur, dan keduanya berakhir di analis — satu
// nilai yang PASTI ada saat tombolnya ditekan. Tidak ada cabang yang boleh mengembalikan
// teks kosong.
func (l PUCLLetter) AssignedOperator() string {
	admin := strings.TrimSpace(l.ClaimAdmin)
	pic := strings.TrimSpace(l.TechnicalPIC)

	// Jalur PUCL: admin klaim, lalu PIC Teknik, lalu analis.
	//
	// PIC Teknik menjadi cadangan pertama — bukan langsung analis — karena klaim lama
	// kerap punya PIC tetapi belum punya `ADMINKLAIM`, dan PIC masih orang yang mengenal
	// klaimnya. Rantai yang sama dipakai keputusan Setuju di Inbox RCL.
	if l.InRCLPUCLQueue() {
		if admin != "" {
			return admin
		}
		if pic != "" {
			return pic
		}
		return strings.TrimSpace(l.Operator)
	}

	// Jalur RCL dan Notification: PIC Teknik, lalu analis. Admin TIDAK dipakai di sini —
	// klaimnya singgah di Inbox RCL, dan yang harus menemukannya di sana adalah petugas
	// teknis, bukan admin yang mendaftarkannya.
	if pic != "" {
		return pic
	}
	return strings.TrimSpace(l.Operator)
}

// InRCLPUCLQueue menyatakan apakah surat ini LANGSUNG masuk antrean Inbox RCL/PUCL.
//
// # Hanya jalur PUCL — RCL dan Notification singgah di dokter lebih dulu
//
// Klaim berjalur RCL harus singgah di **Inbox RCL** lebih dulu (Work Owner 2026-10-05),
// bukan muncul serentak di kedua layar. **Notification berlaku sama sejak 2026-10-07**:
// layar kerja dokter digambar untuk `RCL_PUCL` 1 DAN 3, dan isian "Nama Dokter" tampil
// bila `RCL_PUCL != 2` — keduanya memisahkan PUCL sendirian. Alasan lengkapnya beserta
// ketiga artefaknya ada di `usecase.TicketSendToRCLPUCL`.
//
// Jadi hanya jalur **PUCL** yang tidak punya singgahan, dan hanya ia yang langsung masuk
// antrean RCL/PUCL. Pembagian itu sejalan dengan tahap tujuannya, yang sudah lebih dulu
// dibedakan TicketSendToRCLPUCL.
//
// # Bagaimana "tidak masuk antrean" itu dinyatakan
//
// Baris suratnya TETAP ditulis — Perihal dan ketiga keterangan yang diketik analis tidak
// boleh hilang hanya karena jalurnya berbeda. Yang TIDAK diisi adalah `STATUS_CASE`,
// penyaring tab "Cetak Surat". Ketiga tab layar Inbox RCL/PUCL karena itu melewatkannya:
//
//	tab "Cetak Surat"          STATUS_CASE = '0'                   NULL, tidak lolos
//	tab "Kelengkapan Dokumen"  TGL_CETAK_DOKUMEN_PUCL IS NOT NULL  tidak diisi, tidak lolos
//	tab "Klaim MSIG"           idem
//
// Jadi barisnya ada dan tetap terbaca lewat nomor klaim, tetapi tidak muncul di satu tab
// pun. Tidak ada nilai baru yang dikarang, dan tidak ada kueri modul `inboxrclpucl` yang
// perlu disunting — yang berubah hanya apa yang modul ini TULIS.
//
// Ketika kelak dokter RCL meneruskan klaimnya ke RCL/PUCL, yang dibutuhkan hanyalah
// mengisi `STATUS_CASE` pada baris yang sama; suratnya sudah ada di sana.
func (l PUCLLetter) InRCLPUCLQueue() bool { return l.Track == PUCLTrackPUCL }

// EntersRCLInbox menyatakan apakah surat ini menempatkan klaim di antrean **Inbox RCL**.
//
// Penyaring layar itu dua kolom `T_CLAIMLIST_ADMIN` yang ditambahkan migrasi 0012 —
// `TANGGALANALYSTSENDRCL_1 IS NOT NULL` dan `NAMADOKTERRCL_1` yang dicocokkan dengan
// identitas pemanggil. Keduanya ditulis pada jalur **RCL dan Notification**, dan TIDAK
// pada jalur PUCL: menulisnya di sana akan menaruh klaim yang sudah berada di antrean
// RCL/PUCL ke antrean dokter RCL pula.
//
// Notification ikut sejak 2026-10-07 — lihat `usecase.TicketSendToRCLPUCL` untuk ketiga
// artefak yang memisahkan PUCL sendirian.
func (l PUCLLetter) EntersRCLInbox() bool { return l.Track != PUCLTrackPUCL }

// PUCLSubjectOption adalah satu baris `POOLDATA.M_PERIHAL_RCLPUCL` (12 baris).
//
// `STS_STATUS` memisahkan masternya menjadi dua kelompok yang isinya berbeda jenis:
// `1` untuk "Tolakan klaim…" dan `2` untuk "Kelengkapan…"/"Pemberitahuan Penundaan…".
// Pembagian itu sejalan dengan arti jalurnya — menolak versus memproses ulang —
// sehingga modal ini memakainya untuk MENYARING pilihan menurut jalur yang dipilih.
//
// **Ini penyimpulan, bukan bukti.** Rule yang menyaringnya (`BrowsePerihalRCLPUCL_RD`)
// tidak ada di export, jadi yang dibandingkan adalah isi masternya dengan arti
// jalurnya. Bila ternyata keliru, yang perlu diubah hanya penyaring di satu kueri.
type PUCLSubjectOption struct {
	ID int

	// Name adalah `PERIHAL_NAME` — teks inilah yang tersimpan pada klaim.
	Name string

	// Track adalah jalur tempat pilihan ini berlaku, diturunkan dari `STS_STATUS`.
	Track int
}

// PUCLRejectReason adalah satu baris `POOLDATA.M_REASON_REJECT_REPRO` (2.116 baris).
//
// Tabelnya menyimpan `REASON_ID` sebagai kolom dan SELURUH sisanya sebagai satu CLOB
// JSON (`JSONDATA`), berkunci `REASON_ID`, `REASON_NAME`, `REASON_DESC`, dan tujuh
// kunci lain yang tidak dipakai grid ini.
type PUCLRejectReason struct {
	ID string

	// Name adalah `REASON_NAME` — kolom "Reason Name" pada grid.
	Name string

	// Description adalah `REASON_DESC` — kolom "Reason Description", dan ISI inilah
	// yang tombol "Pilih" salin ke Keterangan Isi.
	Description string
}

// PUCLLetterStore menyimpan surat RCL/PUCL.
//
// # Kepemilikan tabel (`P-1`)
//
// `POOLDATA.TC_PNC_PUCL` dimiliki Pega selama masa paralel. Penulisan dari sini
// dibatasi pada klaim `PNCN.*` — klaim yang memang dibuat aplikasi ini — persis seperti
// pembatasan yang sudah berlaku pada `T_CLAIMLIST_ADMIN` (lihat `inboxentry.sql`).
// Baris Pega tidak pernah tersentuh.
type PUCLLetterStore interface {
	SaveLetter(ctx context.Context, letter PUCLLetter) error
}

// RCLDoctorOption adalah satu pilihan dropdown "Nama Dokter".
//
// # Sumbernya kini TERBACA, bukan lagi diturunkan
//
// Sebelumnya daftar ini dibaca dari `POOLDATA.T_ACCESS_GROUP_PNC` — hasil penalaran,
// karena rule sumbernya tidak ada di export (`R-16`): nilainya disimpan ke
// `T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1`, dan penyaring D Inbox RCL mencocokkan kolom itu
// dengan identitas lama pemanggil, jadi daftarnya "pasti" himpunan identitas lama.
//
// Penalaran itu masuk akal dan tetap salah. Property `NamaDokterRCL` diserahkan Work
// Owner pada 2026-10-06, dan dropdown-nya ternyata **daftar tetap pada property-nya
// sendiri** — `pyTableOption = PromptList` dengan `pyPromptTableList` berisi TEPAT DUA
// baris, bukan kueri ke tabel mana pun:
//
//	pyStandardValue        pyLocalizedValue           Rule-Obj-FieldValue
//	---------------------  -------------------------  ------------------------
//	WAHYUKRISTANTI         WAHYUKRISTANTI             pyCaption WAHYUKRISTANTI
//	MARGARETHAROSAGUNAWAN  MARGARETHA ROSA GUNAWAN    pyCaption MARGARETHA ROSA GUNAWAN
//
// Kedua `Rule-Obj-FieldValue` itu ikut terdaftar di `pxRuleReferences` property-nya —
// bukti kedua bahwa labelnya memang dua itu saja, bukan cuplikan dari daftar yang lebih
// panjang.
//
// Inilah sebab layarnya kosong: kuerinya mencari 17 identitas yang tidak pernah menjadi
// isi dropdown ini, dan di basis data yang dipakai layar tidak satu pun terambil.
//
// # Yang TIDAK berubah
//
// Penyaring D Inbox RCL tetap mencocokkan `NAMADOKTERRCL_1` dengan identitas lama
// pemanggil. Bentuk `pyStandardValue` di atas — huruf besar, tanpa spasi — memang bentuk
// `OPERATOR_ID`, sehingga keduanya tetap sebangun. Yang gugur hanya anggapan bahwa
// daftarnya seluas himpunan itu: Pega membatasinya pada dua orang.
type RCLDoctorOption struct {
	// ID adalah nilai yang DISIMPAN ke `NAMADOKTERRCL_1` — `pyStandardValue`.
	ID string

	// Label adalah yang DITAMPILKAN — `pyLocalizedValue`.
	//
	// Dua field, bukan satu seperti sebelumnya: pada baris kedua keduanya BERBEDA
	// (`MARGARETHAROSAGUNAWAN` versus `MARGARETHA ROSA GUNAWAN`). Menyimpan label
	// berarti menyimpan nilai yang tidak pernah cocok dengan penyaring inbox;
	// menampilkan nilai simpan berarti menampilkan nama tanpa spasi.
	Label string
}

// rclDoctorPromptList adalah `pyPromptTableList` property `NamaDokterRCL`, dalam URUTAN
// ASLINYA (`REPEATINGINDEX` 1 lalu 2) — bukan diurutkan abjad.
//
// Urutan dipertahankan karena itulah urutan yang dilihat petugas hari ini; menatanya
// ulang memindahkan pilihan yang sudah dihafal. Alasan yang sama dipakai dropdown Bisnis
// `casestudyclaim`.
//
// # Kedua nama ini hardcode, dan itu disadari
//
// `D-15` menetapkan nilai semacam ini menjadi master data (`F-4`), yang belum ada —
// kedua nama ini bagian dari 24 Operator ID hardcode yang dicatat di sana. Sampai `F-4`
// tiba, daftarnya tinggal di sini, tempat asalnya di Pega dapat dibaca berdampingan.
// Menuliskan nama Operator ID lengkap diizinkan `D-69`.
//
// # Keduanya DIKONFIRMASI masih berlaku (Work Owner, 2026-10-06)
//
// Ditanyakan justru karena daftarnya berumur: property-nya dibuat 2018 dan terakhir
// disunting 2020, sementara orang berpindah tugas. Jawabannya "masih" — sehingga
// daftar ini direplikasi sebagai perilaku yang BENAR, bukan sekadar perilaku lama yang
// ditiru menunggu koreksi (`P-5`).
//
// Yang BELUM dijawab: siapa yang berwenang menambah atau menghapus dokter setelah `F-4`
// ada. Selama belum, penambahan dokter menempuh perubahan kode dan rilis — konsekuensi
// yang perlu disebut saat `F-4` dirancang, bukan ditemukan saat ada dokter baru masuk.
var rclDoctorPromptList = []RCLDoctorOption{
	{ID: "WAHYUKRISTANTI", Label: "WAHYUKRISTANTI"},
	{ID: "MARGARETHAROSAGUNAWAN", Label: "MARGARETHA ROSA GUNAWAN"},
}

// RCLDoctorOptions mengembalikan isi dropdown "Nama Dokter" dalam urutan layar.
//
// Salinan, bukan irisan aslinya: pemanggil yang mengurutkan hasilnya di tempat tidak
// boleh ikut menata ulang daftar yang dipakai bersama.
func RCLDoctorOptions() []RCLDoctorOption {
	out := make([]RCLDoctorOption, len(rclDoctorPromptList))
	copy(out, rclDoctorPromptList)
	return out
}

// PUCLOptionSource melayani kedua daftar pilihan modal yang datang dari basis data.
//
// "Nama Dokter" TIDAK di sini: ia daftar tetap pada property-nya sendiri
// (`RCLDoctorOptions`), sehingga tidak punya adapter dan tidak dapat gagal.
type PUCLOptionSource interface {
	// SubjectOptions mengembalikan pilihan Perihal untuk satu jalur.
	SubjectOptions(ctx context.Context, track int) ([]PUCLSubjectOption, error)

	// RejectReasons mencari alasan penolakan. keyword kosong berarti tanpa pencarian.
	//
	// limit WAJIB: tabelnya 2.116 baris, dan grid Pega-nya berpaginasi
	// (`pyGridPaginator`). Mengirim seluruhnya ke layar akan mengulang persis cacat
	// yang `NFR-12` larang.
	RejectReasons(ctx context.Context, keyword string, limit int) ([]PUCLRejectReason, error)
}
