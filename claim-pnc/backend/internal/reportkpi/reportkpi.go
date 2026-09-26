// Package reportkpi adalah inti modul Report KPI PNC.
//
// # Nama modul ini
//
// Butir menu `MENU_ID 84` pada `Database/m_menu_aplikasi_pnc.csv` berbunyi
// **"Report KPI PNC"** dan menunjuk `MENU_PROGRAM` `ReportKPIHarness`. Harness itu sendiri
// berjudul **"Report KPI"** (`pyLabel`). `D-81` menetapkan nama modul mengikuti nama yang
// dipakai Work Owner, sehingga folder ini bernama `reportkpi` — akhiran "PNC" dibuang
// karena SELURUH aplikasi ini adalah Claim PNC, persis seperti `masterstatus` membuang
// "Klaim" dari "Master Status Klaim".
//
// # Artefak Pega yang dibaca
//
//	Harness/ReportKPIHarness-Harness.xml          judul layar, satu section
//	Section/ReportKPI_Section-Section.xml         SELURUH bentuk layar (1,7 MB) — tiga tab
//	Activity/PNCReportKPIAdjuster_act-Act.xml     tab KPI Adjuster: penyaring + hitungan
//	Activity/GetFilterKPI-Act.xml                 bekal isian dropdown
//	RDB List/GetSummaryKPIAdjuster-SQL.xml        grid Summary, satu tipe
//	RDB List/GetSummaryKPIAdjusterALL-SQL.xml     grid Summary, tipe ALL (UNION dua tipe)
//	RDB List/GetSummaryKPIAdjusterKuartal-SQL.xml varian per tahun — TIDAK dibangun, lihat §
//	Database/INSERT_KPIADJUSTER.prc               DDL efektif POOLDATA.DETAIL_KPI_ADJUSTER
//
// # TIGA TAB, dan hanya SATU yang dibangun di tahap ini
//
// Section-nya memuat tiga tab, terbaca dari `pyTitle` layout group-nya:
//
//	KPI PIC Teknik    grid "Data KPI PIC Teknik"   — BELUM dibangun
//	KPI Adjuster      grid "Summary KPI Adjuster"  — DIBANGUN
//	                  grid "Detail KPI Adjuster"   — DIBANGUN
//	KPI Admin         grid "Data KPI"              — BELUM dibangun
//
// Keputusan Work Owner 2026-09-24: KPI Adjuster lebih dulu. Alasannya bukan selera
// melainkan penghalang — kedua tab lain bertumpu pada hal yang belum ada:
//
//	DATAMINING.GET_WORKING_HOURS@ASMD  dipakai 18 kali di 7 berkas; ia DB link (`R-03`)
//	POOLDATA.M_KPI_PNC                 tangga penilaian; ISINYA tidak ada di export
//	operator yang di-hardcode          6 nama menentukan siapa yang ikut dihitung (`D-15`)
//
// Tab KPI Adjuster tidak menyentuh satu pun dari ketiganya: sumbernya SATU tabel datar
// yang nilainya sudah jadi.
//
// # Apa yang sebenarnya diukur tab ini
//
// Kinerja **adjuster eksternal** — pihak ketiga yang menilai kerugian di lapangan
// (`CONTEXT.md`: Surveyor / Loss Adjuster). Sembilan komponen dinilai per kasus survei,
// lalu dirata-ratakan per adjuster. Lihat component.go.
//
// # SATU HAL YANG HARUS DISADARI SEBELUM MEMBACA SISA BERKAS INI
//
// Di Pega, tab ini MENGHITUNG ULANG lalu MENYIMPAN. `PNCReportKPIAdjuster_act` mendaftar
// kasus survei (`GetDataCaseSurvey`, `GetDataCaseSurveyALL`), membuka tiap objek kerjanya,
// memanggil empat sub-activity penilai, lalu menulis hasilnya lewat
// `CallProcedureInsertKPISurvey` → `Database/INSERT_KPIADJUSTER.prc` — sebuah upsert ke
// `POOLDATA.DETAIL_KPI_ADJUSTER`. Grid Summary barulah merata-ratakan tabel itu.
//
// Modul ini **hanya MEMBACA tabel itu**, dan itu keputusan sadar dengan dua dasar:
//
//   - `P-1` — selama masa paralel tepat satu sistem yang menulis sebuah tabel, dan
//     penulisnya hari ini Pega. Menghitung ulang dari sini berarti dua sistem menulis
//     baris yang sama.
//   - Penilaiannya tidak dapat direproduksi setia. Empat sub-activity penilainya berjumlah
//     ~900 KB langkah Java, dan ambang nilainya hidup di `POOLDATA.M_KPI_PNC` yang isinya
//     tidak ada di export (`R-16`) — keadaan yang sama persis dengan `GCNM_FEE_SCALE` pada
//     `B-5`.
//
// Akibat yang diterima: baris yang **belum pernah dihitung Pega belum muncul di sini**.
// Itu dinyatakan ke pengguna lewat PlannedDifferences, bukan dibiarkan ditemukan sendiri.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package reportkpi

import (
	"context"
	"strings"
	"time"
)

// SourceTable adalah satu-satunya tabel yang dibaca modul ini.
//
// Namanya ditulis di sini, bukan hanya di berkas .sql, supaya `-periksa` dapat
// menyebutnya saat melaporkan kesiapan tanpa mengimpor lapisan SQL.
const SourceTable = "POOLDATA.DETAIL_KPI_ADJUSTER"

// Caller adalah identitas pemanggil.
//
// Tidak satu pun kueri modul ini menyaring menurut nilai ini — laporan KPI adalah
// pandangan penyelia atas SELURUH adjuster, dan Pega pun tidak menyaringnya per pengguna.
//
// Ia dibawa untuk JEJAK. Barisnya memuat penilaian kinerja orang yang dapat dinamai, dan
// selama pemeriksaan peran belum ada (`TKT-F3-004`), catatan siapa yang membukanya adalah
// satu-satunya kontrol yang tersisa (`D-59`).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama. Bukan NIK.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Score adalah satu nilai komponen KPI.
//
// # Kenapa ia BUKAN float64 telanjang
//
// Karena kolom sumbernya menyimpan TEKS. `INSERT_KPIADJUSTER.prc` mendeklarasikan
// kesembilan komponennya `in varchar2`, dan kueri Summary membungkus setiap satunya dengan
// `to_number(...)` sebelum merata-ratakan. Teks yang tidak dapat dibaca sebagai angka
// karena itu adalah keadaan yang NYATA, bukan teoretis — dan ia harus terbedakan dari
// nilai nol.
//
// Nol dan kosong adalah dua hal yang berbeda bagi pembaca laporan kinerja: yang pertama
// berarti adjuster tidak mendapat poin, yang kedua berarti komponennya belum dinilai.
// Menyamakan keduanya menurunkan rata-rata tanpa seorang pun tahu.
type Score struct {
	// Value adalah nilainya. Hanya berarti bila Present bernilai true.
	Value float64

	// Present menyatakan nilainya ada dan terbaca sebagai angka.
	Present bool
}

// NewScore membentuk nilai yang ada.
func NewScore(value float64) Score { return Score{Value: value, Present: true} }

// EmptyScore membentuk nilai yang tidak ada.
func EmptyScore() Score { return Score{} }

// AdjusterSummary adalah satu baris grid **Summary KPI Adjuster** — satu adjuster.
//
// Kesembilan nilainya adalah RATA-RATA seluruh kasus adjuster itu yang cocok dengan
// penyaring, dibulatkan dua desimal. Pembulatannya dikerjakan basis data
// (`round(avg(to_number(...)),2)`) dan dipertahankan apa adanya (`P-5`).
type AdjusterSummary struct {
	// Adjuster adalah nama adjuster <- kolom `ADJUSTER`.
	//
	// Ia NAMA, bukan kode. Alias Pega-nya "UserTeknisGroup" dan itu menyesatkan — kolomnya
	// tidak ada hubungannya dengan PIC Teknik maupun dengan group mana pun (`D-19`).
	Adjuster string

	// ReportType menyebut tipe laporan baris ini berasal.
	//
	// Hanya terisi pada tipe `ALL`, yang menggabungkan dua tipe dengan `UNION ALL` sehingga
	// satu adjuster dapat muncul DUA KALI — sekali untuk OUTSTANDING, sekali untuk FINAL.
	// Tanpa kolom ini, kedua barisnya terlihat sebagai baris ganda yang tidak dapat
	// dijelaskan.
	ReportType ReportType

	// Scores adalah kesembilan nilai rata-rata, dikunci dengan kode komponen.
	// Kuncinya adalah Component.Code — lihat component.go.
	Scores map[string]Score
}

// AdjusterDetail adalah satu baris grid **Detail KPI Adjuster** — satu kasus survei.
type AdjusterDetail struct {
	// Adjuster adalah nama adjuster <- `ADJUSTER`.
	Adjuster string

	// CaseID adalah kunci kasus survei <- `CASEID`.
	//
	// Isinya `pzInsKey` objek kerja kelas `ASM-FW-GCNMFW-Work-SurveyClaim`, yaitu berbentuk
	// `ASM-FW-GCNMFW-WORK ...` — kunci teknis Pega yang bocor ke data bisnis (utang teknis
	// §4.1). Ia ditampilkan apa adanya: baris ini milik Pega, dan menyunting nomornya di
	// layar akan membuat pengguna tidak dapat mencocokkannya dengan sistem lama.
	CaseID string

	// ReportType <- `TIPE`. OUTSTANDING atau FINAL.
	ReportType ReportType

	// ScoredOn <- `TANGGAL`, tanggal penilaian, bentuk `YYYY-MM-DD`.
	//
	// Ia pula yang disaring rentang periode. Kosong berarti kolomnya NULL di basis data.
	ScoredOn string

	// Scores adalah kesembilan nilai kasus ini, dikunci dengan kode komponen.
	Scores map[string]Score
}

// DetailPage adalah satu halaman grid Detail beserta jumlah seluruhnya.
type DetailPage struct {
	Rows  []AdjusterDetail
	Total int
}

// Pagination adalah permintaan satu halaman.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxPageSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam.
//
// PegaMaxRecords disimpan sebagai catatan, BUKAN untuk ditegakkan (`ADR-0011`).
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
	PegaMaxRecords  = 500
)

// Normalize membetulkan paginasi ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari
// parameter query yang mudah salah ketik, dan menolak seluruh permintaan karena
// `halaman=0` membuat layar gagal tanpa alasan yang terbaca pengguna.
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

// Offset adalah jumlah baris yang dilewati untuk mencapai halaman ini.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// Repo adalah seam penyimpanan modul ini.
//
// Ia dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang
// mengisinya (`08-TECHNICAL-STRATEGY.md` §2 aturan 2).
type Repo interface {
	// Summary mengembalikan grid Summary: satu baris per adjuster, nilai dirata-ratakan.
	//
	// Ia TIDAK dipaginasi, dan itu mengikuti layar lama: jumlah adjuster eksternal
	// terhitung puluhan, bukan puluhan ribu, dan grid Summary memang dibaca sekaligus.
	Summary(ctx context.Context, query Query) ([]AdjusterSummary, error)

	// Detail mengembalikan SATU HALAMAN grid Detail beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam supaya pengisi SQL dapat memotongnya di basis
	// data; pengisi memori memotongnya sendiri untuk hasil yang sama.
	Detail(ctx context.Context, query Query, page Pagination) (DetailPage, error)

	// Adjusters mengembalikan nama adjuster yang MEMANG punya baris KPI, terurut.
	//
	// # Kenapa daftarnya dari tabel KPI, bukan dari master surveyor
	//
	// Layar lama mengisinya dari `V_D_SURVEYORS` lewat RDB list `BrowseAdjsuterExternal`,
	// dan rule itu **tidak ada di export** (`R-16`) — bentuk kuerinya tidak diketahui.
	//
	// Yang diketahui pasti adalah cara nilainya dipakai: `and adjuster='<pilihan>'`
	// terhadap kolom `ADJUSTER` tabel ini. Mengambil daftarnya dari kolom yang SAMA dengan
	// yang disaring menutup satu kelas kegagalan seluruhnya — tidak akan pernah ada
	// pilihan dropdown yang menghasilkan grid kosong karena namanya dieja berbeda di dua
	// tabel.
	//
	// Selisihnya dinyatakan di PlannedDifferences: adjuster yang terdaftar di master
	// tetapi belum punya satu pun penilaian TIDAK muncul di dropdown.
	Adjusters(ctx context.Context, query Query) ([]string, error)

	// AdminTotals mengembalikan angka MENTAH kartu skor tab KPI Admin.
	//
	// Ia mengembalikan angka, bukan kartu yang sudah tersusun: urutan metrik, labelnya,
	// dan kesimpulan tercapai atau tidak adalah keputusan domain, bukan keputusan
	// penyimpanan. Lihat BuildScorecard.
	AdminTotals(ctx context.Context, query AdminQuery) (AdminTotals, error)

	// AdminDetail mengembalikan SATU HALAMAN grid rincian tab KPI Admin.
	AdminDetail(ctx context.Context, query AdminQuery, page Pagination) (AdminDetailPage, error)

	// Bands mengembalikan pita nilai satu komponen dari `POOLDATA.M_KPI_PNC`, terurut `ID`.
	//
	// Urutannya mengikat: pita bertetangga BERTINDIH di titik batas, dan yang pertama cocok
	// yang dipakai. Lihat BandFor.
	//
	// `note` menyaring kolom `NOTE` dan hanya berarti pada `SLA KLAIM`, yang punya pita
	// berbeda untuk LEADER dan MEMBER. Kosong berarti tidak disaring.
	Bands(ctx context.Context, job, note string) ([]Band, error)

	// ThresholdDays mengembalikan ambang hari komponen dari kolom `DAY` tabel yang sama.
	//
	// Meniru `GetDaySurveyAdjuster`: `tipe='PIC' AND job=? AND day IS NOT NULL`, ditambah
	// `note=?` bila diisi. Tidak ada berarti komponennya memang tidak punya ambang hari —
	// Progress tidak punya, karena ia tidak mengukur lama.
	ThresholdDays(ctx context.Context, job, note string) (Score, error)

	// PICs mengembalikan petugas satu lini bisnis beserta penanda Leader-nya.
	//
	// Meniru `GetDataPIC`: `select operator_id, sts_leader from pooldata.mst_user_teknik
	// where type_business=?`.
	PICs(ctx context.Context, line BusinessLine) ([]PICProfile, error)

	// ProgressCounts mengembalikan cacah pembaruan progres per PIC pada satu periode.
	//
	// Satu kueri untuk SELURUH PIC sekaligus — bukan satu kueri per orang seperti sistem
	// lama. Hasilnya sama karena kuerinya memang `group by a.pic`; yang berubah hanya
	// berapa kali ia dijalankan.
	ProgressCounts(ctx context.Context, query PICQuery) ([]ProgressCount, error)

	// AnalysisSpans mengembalikan pasangan tanggal penilaian Analisa Klaim per PIC.
	//
	// Repositori mengembalikan TANGGALNYA, bukan kesimpulan tepat atau tidak: penentuan
	// itu menempuh kalender hari kerja, dan kalender adalah aturan bisnis yang hidup di Go
	// (`D-50`) — bukan di dalam SQL.
	AnalysisSpans(ctx context.Context, query PICQuery) ([]DateSpan, error)

	// AcceptanceSpans mengembalikan pasangan tanggal penilaian Akseptasi Klaim per PIC.
	AcceptanceSpans(ctx context.Context, query PICQuery) ([]AcceptanceSpan, error)

	// ClosureSpans mengembalikan pasangan tanggal penilaian SLA Klaim per PIC.
	ClosureSpans(ctx context.Context, query PICQuery) ([]ClosureSpan, error)

	// Holidays mengembalikan tanggal libur pada satu rentang, DI LUAR akhir pekan.
	//
	// Akhir pekan sudah dikeluarkan di sisi kueri persis seperti `CheckHoliday_SQL`, supaya
	// libur yang jatuh pada Sabtu atau Minggu tidak terpotong dua kali.
	Holidays(ctx context.Context, from, to time.Time) ([]time.Time, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Setiap badan hukum punya basis datanya sendiri (`ADR-0030`), sehingga penyimpanan
// dipilih per permintaan — bukan satu penyimpanan untuk seluruh aplikasi.
type RepoSelector func(alias string) (Repo, error)
