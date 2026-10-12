// Package dashboardclaim memodelkan layar Dashboard Claim.
//
// Layar ini adalah PANDANGAN MANAJERIAL: empat angka yang menghitung pekerjaan seluruh
// organisasi pada satu entitas, masing-masing dapat ditelusuri menjadi daftar barisnya.
//
// Rujukan sistem lama:
//
//	Harness/DashboardClaim_Harness-Harness.xml       layar, judul "Dashboard Claim"
//	Activity/ActLoadDashboarClaim-Act.xml            pemuat awal (salah ketik aslinya)
//	Activity/SetDashboardClaim-Act.xml               dispatcher empat tile
//	Section/DashboardClaim_Section-Section.xml       keempat kartu penghitung
//	Section/DashboardClaim_Section2-Section.xml      grid telusur bertipe klaim
//	Section/DashboardClaim_Section1-Section.xml      grid telusur bertipe survei
//
// # Empat tile, dan dari mana angkanya
//
// `SetDashboardClaim` bercabang pada `param.tipe`/`TempView.CityID`:
//
//	tipe 0  OUTSTANDING        RDB List/GcnmBrowseCase_SQL-SQL.xml
//	tipe 1  CLOSE CLAIM        RDB List/GcnmBrowseReopenCase_SQL-SQL.xml
//	tipe 2  LOSS ADJUSTER      RDB List/BrowseLossAdjuster-SQL.xml      SURVEYORTYPE_1='2'
//	tipe 3  INTERNAL SURVEYOR  RDB List/BrowseInternalSurveyor-SQL.xml  SURVEYORTYPE_1='1'
//
// # Kenapa modul ini TIDAK memakai ulang inboxoutstanding
//
// Namanya mirip dan isinya tidak sama — persoalan yang sama yang sudah dicatat
// `inboxcloseclaim` terhadap modul yang sama.
//
// `inboxoutstanding` adalah **"My Inbox"**: kuerinya menyaring
// `b.pxassignedoperatorid IN (…)` sehingga hanya menampilkan pekerjaan MILIK PEMANGGIL.
// Tile OUTSTANDING di layar ini tidak punya saringan itu sama sekali — ia menghitung
// pekerjaan SELURUH organisasi pada entitas itu, dibatasi hanya oleh lini bisnis dan
// `branchname != 'ASNET'`.
//
// Memakainya ulang akan membuat dashboard manajerial menampilkan angka pekerjaan satu
// orang. Kegagalan itu TIDAK menghasilkan galat apa pun — hanya angka yang salah, pada
// layar yang justru dibaca untuk mengambil keputusan.
//
// # Yang JUSTRU dipakai ulang: CLOSE CLAIM
//
// Populasi tile CLOSE CLAIM sama persis dengan `inboxcloseclaim`:
//
//	PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'
//	PYSTATUSWORK IN ('Resolved-Completed', 'Resolved-Rejected')
//	BRANCHNAME <> 'ASNET'
//	+ penyaring lini bisnis yang identik
//
// Karena itu angkanya TIDAK dihitung ulang di sini. Ia dibaca lewat seam ClosedClaimReader
// yang dipenuhi adapter atas modul itu — satu kueri, satu tempat. Menyalin kuerinya berarti
// dua salinan aturan yang dapat menyimpang tanpa ada yang menandainya.
package dashboardclaim

import (
	"context"
	"strings"
	"time"
)

// Tile adalah salah satu dari empat kartu penghitung di layar.
//
// Nilainya dikirim layar sebagai bagian jalur (`/dashboard-claim/{tile}`), sehingga ia
// KONTRAK — bukan sekadar penamaan internal.
type Tile string

const (
	TileOutstanding      Tile = "outstanding"
	TileCloseClaim       Tile = "close-claim"
	TileLossAdjuster     Tile = "loss-adjuster"
	TileInternalSurveyor Tile = "internal-surveyor"
)

// tiles adalah keempatnya dalam urutan tampilnya di layar.
//
// Urutannya mengikuti `param.tipe` 0…3 pada `SetDashboardClaim`, bukan abjad: itulah urutan
// yang sudah dikenal pengguna (`D-13`).
var tiles = []Tile{TileOutstanding, TileCloseClaim, TileLossAdjuster, TileInternalSurveyor}

// Tiles mengembalikan keempat tile dalam urutan tampilnya.
func Tiles() []Tile {
	result := make([]Tile, len(tiles))
	copy(result, tiles)
	return result
}

// Label adalah teks yang dibaca pengguna pada kartu.
func (t Tile) Label() string {
	switch t {
	case TileOutstanding:
		return "Outstanding"
	case TileCloseClaim:
		return "Close Claim"
	case TileLossAdjuster:
		return "Loss Adjuster"
	case TileInternalSurveyor:
		return "Internal Surveyor"
	default:
		return string(t)
	}
}

// Shape menyatakan bentuk baris telusur tile ini.
//
// Dua tile pertama menelusur KLAIM, dua terakhir menelusur SURVEI — dan keduanya punya
// kolom yang berbeda, bukan sekadar isi yang berbeda. Layar perlu mengetahuinya untuk
// memilih susunan kolom, dan menebaknya dari nama tile akan membuat penambahan tile kelak
// diam-diam salah gambar.
func (t Tile) Shape() RowShape {
	switch t {
	case TileLossAdjuster, TileInternalSurveyor:
		return ShapeSurvey
	default:
		return ShapeClaim
	}
}

// RowShape membedakan kedua bentuk baris telusur.
type RowShape string

const (
	ShapeClaim  RowShape = "klaim"
	ShapeSurvey RowShape = "survei"
)

// ParseTile membaca tile dari jalur permintaan.
//
// Isian yang tidak dikenali menghasilkan galat, BUKAN tile bawaan: jalur yang salah ketik
// harus terlihat sebagai 404, bukan diam-diam menampilkan angka tile lain.
func ParseTile(raw string) (Tile, bool) {
	value := Tile(strings.ToLower(strings.TrimSpace(raw)))
	for _, known := range tiles {
		if known == value {
			return value, true
		}
	}
	return "", false
}

// SurveyorType membedakan kedua jenis survei di `POOLDATA.T_CLAIMLIST_ADMIN`.
//
// Nilainya persis seperti yang tersimpan di kolom `SURVEYORTYPE_1`, dan itu bukan pilihan
// gaya: ia dikirim sebagai nilai bind ke kueri, sehingga mengubahnya di sini mengubah baris
// yang terbaca.
type SurveyorType string

const (
	// SurveyorInternal adalah surveyor internal ASM — `SURVEYORTYPE_1 = '1'`.
	SurveyorInternal SurveyorType = "1"

	// SurveyorAdjuster adalah loss adjuster eksternal — `SURVEYORTYPE_1 = '2'`.
	SurveyorAdjuster SurveyorType = "2"
)

// SurveyorTypeFor memetakan tile survei ke jenis surveyornya.
//
// Tile yang bukan survei mengembalikan false, sehingga pemanggil tidak dapat keliru
// menjalankan kueri survei untuk tile klaim.
func SurveyorTypeFor(t Tile) (SurveyorType, bool) {
	switch t {
	case TileInternalSurveyor:
		return SurveyorInternal, true
	case TileLossAdjuster:
		return SurveyorAdjuster, true
	default:
		return "", false
	}
}

// Counts adalah keempat angka pada kartu.
//
// # Kedua angka survei TIDAK menghitung satuan yang sama
//
// Ini bukan kelalaian penulisan melainkan perilaku sistem lama yang direplikasi apa adanya
// (`P-5`), dan selisihnya nyata:
//
//	Get_CountLostAdjusterClaim    SUM(CASE WHEN (…) > 0 THEN 1 ELSE 0 END)
//	Get_CountInternalSurveyor     SUM(CASE WHEN (…) > 0 THEN (… COUNT yang sama …) ELSE 0 END)
//
// Yang pertama menghitung JUMLAH KLAIM yang punya sekurang-kurangnya satu survei adjuster.
// Yang kedua menghitung JUMLAH SURVEI-nya. Satu klaim dengan tiga survei internal menambah
// 3 pada InternalSurveyor tetapi hanya menambah 1 pada LossAdjuster bila kasusnya sejajar.
//
// Akibatnya kedua angka tidak dapat dijumlahkan maupun dibandingkan satu sama lain.
// Diperbaiki atau tidak adalah keputusan Work Owner, bukan keputusan yang diambil sambil
// menulis kode — sampai itu diputuskan, keduanya dipertahankan dan dinyatakan di sini.
type Counts struct {
	Outstanding      int
	CloseClaim       int
	LossAdjuster     int
	InternalSurveyor int
}

// ClaimRow adalah satu baris telusur bertipe klaim — tile Outstanding dan Close Claim.
//
// Nama fieldnya mengikuti `CONTEXT.md`, BUKAN alias kolom sistem lama. Alias lamanya
// menyesatkan sampai tidak dapat dibaca tanpa menebak, dan itu persis utang teknis yang
// `D-19` hapus. Yang digantikan, dari `GcnmBrowseCase_SQL`:
//
//	a.pyid         AS "City"      -> ClaimNumber
//	a.pzinskey     AS "CaseID"    -> ClaimID
//	policyno       AS "Currency"  -> PolicyNumber
//	qqname         AS "CityID"    -> InsuredName
//	businessname   AS "District"  -> BusinessName
//	sobname        AS "DistrictID"-> BusinessSource
//	branchname     AS "Country"   -> BranchName
//	dateofloss_1   AS "CountryID" -> LossDate
//	userteknis_1   AS "CauseOfLoss" -> TechnicalPIC
//	pxCreateOpName AS "ClaimID"   -> AdminPNC
//
// Perhatikan baris terakhir: alias lama `"ClaimID"` dipakai untuk NAMA OPERATOR, sementara
// `"CaseID"` dipakai untuk kunci klaim. Membaca kueri itu berarti menebak.
//
// Kolomnya sengaja sama dengan `inboxcloseclaim.ClosedClaim` pada bagian yang ditampilkan,
// karena `Section/DashboardClaim_Section2-Section.xml` memang menampilkan keenam kolom yang
// sama: No Klaim, No Polis, Nama Tertanggung, Nama Bisnis, Sumber Bisnis, Nama Cabang.
type ClaimRow struct {
	// ClaimID adalah kunci teknis Pega (`PZINSKEY`), dipakai membuka layar detail.
	//
	// Ia TIDAK ditampilkan sebagai teks: ia memuat nama kelas internal Pega yang bocor ke
	// data bisnis (utang teknis §4.1), dan layar baru tidak menampilkannya kepada pengguna.
	ClaimID string

	ClaimNumber    string // "No Klaim"         <- PYID
	PolicyNumber   string // "No Polis"         <- POLICYNO
	InsuredName    string // "Nama Tertanggung" <- QQNAME
	BusinessName   string // "Nama Bisnis"      <- BUSINESSNAME
	BusinessSource string // "Sumber Bisnis"    <- SOBNAME
	BranchName     string // "Nama Cabang"      <- BRANCHNAME
	TechnicalPIC   string // "PIC Teknik"       <- USERTEKNIS_1
	AdminPNC       string // "Admin PNC"        <- PXCREATEOPNAME

	// ClaimStatusCode adalah `STATUSCLAIM_1` — salah satu dari 33 kode `1134`–`1166`.
	//
	// Artinya TIDAK disimpulkan di sini. Tiga arti kode yang pernah disimpulkan dari
	// pemakaiannya seluruhnya terbukti salah saat master diterima (`R-06`), sehingga
	// pelabelannya adalah urusan master status klaim — bukan urusan modul ini.
	ClaimStatusCode string

	// ClaimStatusLabel adalah artinya, dibaca dari `POOLDATA.V_STS_CLAIM.LSC_NOTE`.
	//
	// Inilah yang digambar kolom "Claim status" pada layar lama — "Register", "Waiting
	// Survey", dan seterusnya. Kodenya sendiri tidak pernah ditampilkan kepada pengguna.
	ClaimStatusLabel string

	// `Position` dan `Progress` DICABUT 2026-10-07 — grid Pega tidak menggambarnya.
	//
	// Keduanya dibangun atas pembacaan `InboxOutstandingClaim_Section` yang keliru. Grid yang
	// sebenarnya berakhir di "Claim status", lalu langsung tombol Transfer; Work Owner
	// membuktikannya dengan menggulir ke kanan.
	//
	// Bila kelak keduanya memang diminta, yang perlu dibangun ulang adalah
	// `GET_POSISI_PROGRESS_PNC(pyID,'POSISI')` dan `(pyID,'sts_prg2')` — satu kueri untuk
	// seluruh halaman, bukan dua panggilan per baris seperti sistem lama. Alasan lengkapnya
	// di `keputusan-implementasi.md` §207.

	// ReportDate adalah `RECEIVEDDATE_1` — kolom "Report Date" pada layar lama.
	//
	// Ia BUKAN tanggal pendaftaran dan bukan tanggal kejadian: ia tanggal laporan diterima,
	// dan ketiganya digambar sebagai kolom yang berbeda. Dapat kosong pada data warisan.
	ReportDate *time.Time

	// ProcessStatus adalah `PYSTATUSWORK` — status alur kerja, bukan status bisnis.
	//
	// Keduanya konsep berbeda yang di sistem lama bernama mirip dan sering tertukar
	// (`D-18`); di sini namanya dibuat tidak mungkin tertukar.
	ProcessStatus string

	// LossDate adalah Tanggal Kejadian (DOL). Dapat kosong pada data warisan.
	LossDate *time.Time

	// RegisteredAt adalah `PXCREATEDATETIME` — dipakai sebagai urutan baku daftar.
	RegisteredAt time.Time
}

// SurveyRow adalah satu baris telusur bertipe survei — tile Loss Adjuster dan Internal
// Surveyor.
//
// Alias yang digantikan, dari `BrowseLossAdjuster` dan `BrowseInternalSurveyor`:
//
//	a.PYID                 AS "CaseID"        -> SurveyNumber
//	a.PZINSKEY             AS "ClaimNo"       -> SurveyID
//	a.POLICYNO             AS "CityID"        -> PolicyNumber
//	a.QQNAME               AS "Country"       -> InsuredName
//	a.REFNO_1              AS "Province"      -> ReferenceNumber
//	a.SURVEYORNAME_1       AS "CountryID"     -> SurveyorName
//	a.USERTEKNIS_1         AS "District"      -> TechnicalPIC
//	a.RescheduleLocation_1 AS "DistrictID"    -> SurveyLocation
//	a.AdjusterStatus_1     AS "ProvinceID"    -> SurveyStatus
//	a.PYSTATUSWORK         AS "NamaSurveyor"  -> ProcessStatus
//	a.AdjusterPIC_1        AS "City"          -> AdjusterPIC
//
// Dua di antaranya layak diperhatikan: `PYSTATUSWORK` dialiaskan menjadi `"NamaSurveyor"`,
// dan `POLICYNO` menjadi `"CityID"`. Nama alias tidak mencerminkan isinya sama sekali.
type SurveyRow struct {
	// SurveyID adalah kunci teknis survei (`PZINSKEY`), tidak ditampilkan sebagai teks.
	SurveyID string

	SurveyNumber    string // "No Survey"        <- PYID
	ClaimNumber     string // "No Klaim" induk   <- lihat catatan di bawah
	PolicyNumber    string // "No Polis"         <- POLICYNO
	InsuredName     string // "Nama Tertanggung" <- QQNAME
	ReferenceNumber string // "No Referensi"     <- REFNO_1
	SurveyorName    string // "Surveyor"         <- SURVEYORNAME_1
	TechnicalPIC    string // "PIC Teknik"       <- USERTEKNIS_1
	SurveyLocation  string // "Lokasi Survei"    <- RESCHEDULELOCATION_1
	SurveyStatus    string // "Status Survei"    <- ADJUSTERSTATUS_1
	ProcessStatus   string // "Status Proses"    <- PYSTATUSWORK
	AdjusterPIC     string // "PIC Adjuster"     <- ADJUSTERPIC_1

	// ScheduledAt adalah `RESCHEDULEDATE_1` — hanya terisi pada tile Internal Surveyor.
	//
	// Kueri loss adjuster memang tidak mengambilnya; membiarkannya kosong di sana lebih
	// jujur daripada mengisinya dengan tanggal lain yang kebetulan ada.
	ScheduledAt *time.Time

	// AssignedAt adalah `PXCREATEDATETIME` survei — urutan baku daftar.
	AssignedAt time.Time
}

// AgeInDays adalah umur klaim dalam hari — kolom "Lama Waktu Klaim" pada layar lama.
//
// Dihitung terhadap TANGGAL WIB, bukan terhadap selisih stempel waktu UTC. Keduanya berbeda
// pada klaim yang didaftarkan menjelang tengah malam: selisih stempel waktu membulatkan ke
// bawah dan menghasilkan umur yang kurang satu hari (`F-5`).
//
// Jam diambil dari pemanggil, bukan dari time.Now() di sini, supaya hasilnya dapat diuji
// secara pasti — pola `Set7Hours` sistem lama justru lahir dari kebalikannya.
func (c ClaimRow) AgeInDays(now time.Time, location *time.Location) int {
	return daysBetween(c.RegisteredAt, now, location)
}

// AgeInDays adalah umur penugasan survei dalam hari — kolom "Aging" pada layar lama.
func (s SurveyRow) AgeInDays(now time.Time, location *time.Location) int {
	return daysBetween(s.AssignedAt, now, location)
}

// daysBetween menghitung selisih TANGGAL kalender, bukan selisih jam dibagi 24.
//
// Mengembalikan 0 bila titik awalnya kosong, dan tidak pernah mengembalikan angka negatif:
// tanggal pendaftaran yang berada di masa depan adalah data yang keliru, dan menampilkannya
// sebagai umur negatif hanya memindahkan kebingungannya ke layar.
func daysBetween(from, now time.Time, location *time.Location) int {
	if from.IsZero() {
		return 0
	}
	if location == nil {
		location = time.UTC
	}

	start := from.In(location)
	end := now.In(location)

	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, location)

	days := int(endDay.Sub(startDay).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// HoldingRow adalah satu baris tab **Inbox Tampungan PIC**.
//
// Ia klaim yang sudah terdaftar tetapi BELUM punya PIC Teknik — tugasnya masih diparkir di
// akun penampung `ServicePNC`.
//
// Bentuknya terpisah dari ClaimRow, bukan memakai ulang, karena kolomnya memang lebih
// sedikit: tab ini tidak menggambar PIC Teknik (menurut definisi belum ada), tidak
// menggambar status, dan tidak menggambar umur. Memakai ClaimRow akan membawa enam field
// yang selalu kosong, dan field yang selalu kosong tidak dapat dibedakan dari data hilang.
//
// Alias menyesatkan yang digantikan, dari `BrowseCaseNotAssigned`:
//
//	A.POLICYNO       AS "City"       -> PolicyNumber
//	A.QQNAME         AS "CityID"     -> InsuredName
//	A.BUSINESSNAME   AS "Country"    -> BusinessName
//	A.SOBNAME        AS "CountryID"  -> BusinessSource
//	A.BRANCHNAME     AS "District"   -> BranchName
//	A.PXCREATEOPNAME AS "DistrictID" -> AdminPNC
//	PXCREATEDATETIME AS "Province"   -> RegisteredAt
//
// Ketujuhnya memakai nama geografis untuk data yang sama sekali bukan geografis.
type HoldingRow struct {
	// ClaimID adalah kunci teknis Pega, tidak ditampilkan sebagai teks (`D-22`).
	ClaimID string

	ClaimNumber    string // "No Klaim"         <- PYID
	PolicyNumber   string // "No Polis"         <- POLICYNO
	InsuredName    string // "Nama Tertanggung" <- QQNAME
	BusinessName   string // "Nama Bisnis"      <- BUSINESSNAME
	BusinessSource string // "Sumber Bisnis"    <- SOBNAME
	BranchName     string // "Nama Cabang"      <- BRANCHNAME
	AdminPNC       string // "Admin PNC"        <- PXCREATEOPNAME

	RegisteredAt time.Time // "Tanggal Pendaftaran"
}

// HoldingPage adalah satu halaman penampungan beserta jumlah seluruh baris yang cocok.
type HoldingPage struct {
	Rows  []HoldingRow
	Total int
}

// ClaimPage adalah satu halaman baris klaim beserta jumlah seluruh baris yang cocok.
type ClaimPage struct {
	Rows  []ClaimRow
	Total int
}

// SurveyPage adalah satu halaman baris survei beserta jumlah seluruh baris yang cocok.
type SurveyPage struct {
	Rows  []SurveyRow
	Total int
}

// Repo adalah seam ke penyimpanan untuk tile yang dihitung modul ini sendiri.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya.
// Diisi `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// Ia HANYA MEMBACA, dan ketiadaan method tulis disengaja: seluruh tabel yang dibacanya
// masih ditulis Pega selama masa paralel, dan `P-1` menetapkan satu tabel hanya ditulis satu
// sistem. Layar ini memang tidak punya satu pun aksi tulis.
//
// Tile CLOSE CLAIM tidak ada di sini — ia dibaca lewat ClosedClaimReader supaya kuerinya
// tidak disalin. Lihat catatan paket.
type Repo interface {
	// CountOutstanding menghitung klaim berjalan seluruh organisasi pada entitas ini.
	CountOutstanding(ctx context.Context, f Filter) (int, error)

	// CountSurvey menghitung survei menurut jenis surveyornya.
	//
	// Kedua jenis TIDAK menghitung satuan yang sama; lihat catatan pada Counts.
	CountSurvey(ctx context.Context, kind SurveyorType, f Filter) (int, error)

	// ListOutstanding membaca satu halaman klaim berjalan.
	ListOutstanding(ctx context.Context, f Filter) (ClaimPage, error)

	// ListSurvey membaca satu halaman survei menurut jenis surveyornya.
	ListSurvey(ctx context.Context, kind SurveyorType, f Filter) (SurveyPage, error)

	// ListHolding membaca satu halaman tab Inbox Tampungan PIC.
	//
	// Ia TIDAK menerima penyaring lini bisnis — kueri lamanya tidak punya penandanya, dan
	// layar lamanya tidak menggambar dropdown Bisnis pada tab ini. Hanya kotak cari No Klaim
	// dan paginasi yang berlaku, dan keduanya dibawa Filter.
	ListHolding(ctx context.Context, f Filter) (HoldingPage, error)

	// ClaimDetailReader menyatukan pembacaan rincian satu klaim ke dalam Repo yang sama.
	//
	// Disertakan di SINI, bukan sebagai selector tersendiri, karena rincian klaim berada di
	// basis data entitas yang sama dengan daftarnya. Selector kedua akan membuka kemungkinan
	// keduanya menunjuk portal yang berbeda — dan itu kebocoran data antar badan hukum
	// (`R-20`), bukan sekadar ketidakrapian.
	ClaimDetailReader
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas, bukan satu database bersama dengan
// penanda entitas. Klaim milik Asuransi Sinar Mas dan klaim milik Simas Insurtech karena itu
// tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan angka satu badan
// hukum di layar badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Pada layar INI akibatnya khas: yang bocor bukan satu baris melainkan RINGKASAN — angka
// yang justru dibaca untuk mengambil keputusan.
type RepoSelector func(portalAlias string) (Repo, error)

// ClosedClaimReader membaca tile CLOSE CLAIM dari modul yang sudah memilikinya.
//
// Ia sengaja berbentuk seam, bukan pemanggilan langsung ke `inboxcloseclaim`: modul domain
// tidak boleh saling impor, dan yang menghubungkan keduanya adalah adapter di
// `repo/closeclaim` yang dirakit di cmd.
//
// Yang dipakai ulang adalah KUERINYA, bukan sekadar kodenya. Populasi tile ini dan populasi
// layar Inbox Close Claim wajib sama; kalau keduanya disalin terpisah, keduanya akan
// menyimpang pada perubahan berikutnya dan tidak ada yang menandainya.
type ClosedClaimReader interface {
	// Count menghitung klaim tutup yang cocok dengan penyaring.
	Count(ctx context.Context, portalAlias string, f Filter) (int, error)

	// List membaca satu halaman klaim tutup.
	List(ctx context.Context, portalAlias string, f Filter) (ClaimPage, error)
}

// TechnicalPICRow adalah satu baris daftar **PIC Teknik** yang dapat menerima pemindahan.
//
// # Kenapa daftar, bukan isian bebas
//
// `Section/PNCTransferManagement_sec-Section.xml` menggambar **grid**, bukan formulir: ia
// memuat daftar user teknis dengan tombol **"Assign"** pada setiap baris. Yang dipilih
// pengguna adalah barisnya, dan tombol itu mengirim `UserID` baris tersebut ke
// `PNC_ReassignPNCTeknik`.
//
// Isian bebas akan menerima operator yang tidak ada, tidak aktif, atau tidak melayani lini
// bisnis klaim itu — tiga hal yang justru disaring oleh daftarnya.
type TechnicalPICRow struct {
	// OperatorID adalah nilai yang dikirim sebagai `UserID` saat baris ini dipilih.
	OperatorID string

	Name      string // MCL_NAME
	Email     string // EMAIL
	TeamGroup string // TEAM_GROUP

	// Workload adalah COUNTER_QUOTA — pencacah beban yang dipakai `BrowsePICRandomTeam-SQL`
	// untuk memilih petugas dengan beban paling sedikit (`R-04`).
	//
	// Ditampilkan supaya pemilihan manual dapat mempertimbangkan hal yang sama dengan
	// pemilihan otomatis, bukan menebak.
	Workload int

	// TOTAL_JOB TIDAK ada di sini, dan itu disengaja (2026-10-07).
	//
	// Kolomnya tidak ada pada tabel yang dibaca daftar ini, hanya pada view
	// V_MST_USER_TEKNIS. Field-nya sempat ada dan SELALU bernilai nol di Oracle, sementara
	// adapter memori mengisinya dengan angka contoh — sehingga layar menampilkan angka yang
	// masuk akal saat dicoba tanpa Oracle lalu nol di produksi, tanpa satu pun galat.
	//
	// Field yang hanya benar di satu adapter lebih buruk daripada field yang tidak ada.
}

// TechnicalPICPage adalah satu halaman daftar PIC Teknik.
type TechnicalPICPage struct {
	Rows  []TechnicalPICRow
	Total int
}

// TechnicalPICFilter menyaring daftar PIC Teknik.
//
// BusinessType adalah `TYPE_BUSINESS` dan **wajib** — `BrowseVMstUserTeknis_RD` menyaring
// `TYPE_BUSINESS = Param.type_business AND STS_AKTIF = 1`, dan tanpa lini bisnisnya daftar
// akan memuat petugas yang tidak melayani klaim yang sedang dipindahkan.
type TechnicalPICFilter struct {
	BusinessType string
	Search       string
	Limit        int
	Offset       int
}

// Normalize merapikan penyaring dan menerapkan batas halaman.
func (f TechnicalPICFilter) Normalize() TechnicalPICFilter {
	f.BusinessType = strings.ToUpper(strings.TrimSpace(f.BusinessType))
	f.Search = strings.TrimSpace(f.Search)

	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f
}

// TechnicalPICReader membaca daftar PIC Teknik.
//
// Seam tersendiri, bukan method pada Repo: Repo melayani tile dashboard atas tabel klaim,
// sedangkan ini membaca **master** `POOLDATA.MST_USER_TEKNIK`. Menyatukannya membuat satu
// interface memikul dua sumber data yang tidak berhubungan.
type TechnicalPICReader interface {
	ListTechnicalPIC(ctx context.Context, filter TechnicalPICFilter) (TechnicalPICPage, error)
}

// TechnicalPICReaderSelector memilih pembaca sesuai portal aktif (`ADR-0030`).
type TechnicalPICReaderSelector func(portalAlias string) (TechnicalPICReader, error)

// ClaimPosition adalah dua kolom posisi sebuah klaim, sudah digabung.
//
// Ia tipe tersendiri — bukan dua string lepas — supaya pembacanya tidak dapat memasangkan
// posisi sebuah klaim dengan progres klaim yang lain tanpa sengaja.
type ClaimPosition struct {
	Position string
	Progress string
}
