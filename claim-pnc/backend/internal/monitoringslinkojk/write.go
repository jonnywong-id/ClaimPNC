package monitoringslinkojk

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// Tiga aksi tulis layar Monitoring SLINK OJK.
//
// ============================================================================
// KETIGANYA MENULIS TABEL YANG SELAMA MASA PARALEL MASIH DIISI PEGA
// ============================================================================
//
// `POOLDATA.T_CLAIM_SLIK_OJK` diisi sistem lama pada jalur AKSEPTASI
// (`InsertAdjustmentList`, `InsertAdjustmentListKredit`), dan `P-1` menetapkan satu tabel
// hanya boleh ditulis satu sistem selama masa paralel.
//
// Work Owner memutuskan pada 2026-09-26 bahwa ketiga tombol ini **tetap dibangun**,
// sesudah konsekuensinya disampaikan: dua sistem menulis satu tabel yang sama, dan bila
// keduanya menyusun baris untuk klaim yang sama, tabel laporan regulator memuat baris
// ganda. Yang mencegahnya hanyalah disiplin pemakaian, bukan mekanisme apa pun.
//
// Satu pengaman yang DAPAT dibangun sudah dipasang, dan ia berasal dari Pega sendiri:
// `GetCountTClaimSlikOJK` memeriksa apakah klaimnya sudah ada, dan hasilnya menentukan
// `operasidata` — `'U'` bila sudah ada, `'C'` bila belum. Lihat DataOperation.
//
// ============================================================================
// DARI MANA PEMETAAN KOLOMNYA DIBACA
// ============================================================================
//
// Ketiga tombol ini TIDAK terhubung aktivitas apa pun di `Sec_SegmentD01_1` — hanya
// empat tombol lain yang punya `<pyActivity>`. Logikanya karena itu dibaca dari tempat
// yang benar-benar melakukannya:
//
//	RDB List/InsertDataSlikOJKF06-SQL.xml      INSERT 28 kolom beserta asal nilainya
//	RDB List/GetCountTClaimSlikOJK-SQL.xml     pencacah yang menentukan 'U' versus 'C'
//	Activity/PNCUploadAutoClaimSlikOJK-Act.xml jalur unggah CSV
//	Activity/InsertAdjustmentListKredit-Act.xml jalur akseptasi
//	RDB List/QuerySLINKIndividu-SQL.xml        nomor urut + INSERT pengiriman
//	RDB List/UpdateTransactionClaimSlinkIndividu-SQL.xml  penyimpanan id transaksi

// ReportEntry adalah satu baris laporan SLIK yang akan disusun ke
// `POOLDATA.T_CLAIM_SLIK_OJK`.
//
// # Kenapa seluruh nilainya bertipe TEKS
//
// Karena tipe kolomnya belum diketahui (`R-08`), dan sistem lama pun memperlakukannya
// sebagai teks: `InsertDataSlikOJKF06` menyisipkan properti klipboard apa adanya. Mengubah
// sebagiannya menjadi angka di sini berarti menebak — dan tebakan yang salah pada kolom
// laporan regulator tidak menghasilkan galat, hanya nilai yang berbeda.
type ReportEntry struct {
	// ClaimID dan ContractNo mengidentifikasi barisnya. Keduanya WAJIB.
	ClaimID    string
	ContractNo string

	FacilityAccountNo string // norekfasilitas
	DebtorCIF         string // nocifdebitur
	FacilityTypeCode  string // kodejenisfasilitas
	FundSource        string // sumberdana

	// PolicyStart dan PolicyEnd mengisi `tanggalmulai` dan `tanggalakhir`.
	//
	// Di Pega keduanya ditampung properti bernama `District` dan `DistrictID` — alias
	// yang tidak ada hubungannya dengan isinya, dan tidak dibawa ke sini.
	PolicyStart string
	PolicyEnd   string

	InterestRate          string // sukubunga
	CurrencyCode          string // kodevaluta
	OriginalCurrencyValue string // nilaimatauangasal

	// Obligation mengisi `nominal`. Di Pega ia ditampung properti `TSIObject`.
	Obligation string

	CollectibilityCode string // kodekolektabilitas — ejaan kolomnya memang begitu
	DefaultDate        string // tanggalmacet
	DefaultReasonCode  string // kodesebabmacet

	// Arrears mengisi `tunggakan`, dan ia SUDAH DIKURANGI Recovery.
	//
	// Pengurangannya dilakukan Netting, bukan diserahkan pemanggil — lihat di sana.
	Arrears string

	ArrearsDays   string // jumlahharitunggakan
	ConditionDate string // tanggalkondisi
	ConditionCode string // kodekondisi
	BranchCode    string // kodekantorcabang
	Remark        string // keterangan

	// ReportMonth mengisi `bulanlapor`, berbentuk `YYYYMM`.
	//
	// Di Pega ia ditampung properti `City` dan diisi
	// `@DateTime.FormatDateTime(@CurrentDateTime(),"yyyyMM","","")` pada jalur unggah.
	ReportMonth string

	// DataOperation mengisi `operasidata` — `'C'` baru, `'U'` sudah pernah ada.
	//
	// Ia TIDAK diisi pemanggil melainkan ditetapkan lapisan aplikasi dari hasil pencacah,
	// persis seperti `PNCUploadAutoClaimSlikOJK` melakukannya. Lihat DataOperationFor.
	DataOperation string

	Recovery    string // recoveryclaim
	IDCardNo    string // noktp
	CompanyNPWP string // npwpperusahaan
	PolicyNo    string // nopolis
	ClientID    string // clientid
}

// Nilai `operasidata` sebagaimana ditulis sistem lama.
const (
	// OperationCreate — klaimnya belum pernah masuk laporan.
	OperationCreate = "C"

	// OperationUpdate — klaimnya sudah pernah masuk laporan.
	OperationUpdate = "U"
)

// DataOperationFor menurunkan `operasidata` dari jumlah baris yang sudah ada.
//
// Disalin dari `PNCUploadAutoClaimSlikOJK`:
//
//	TempCount.pxResults(1).CityID > 0  -> "U"
//	TempCount.pxResults(1).CityID = 0  -> "C"
//
// Pencacahnya `GetCountTClaimSlikOJK`, yang menghitung baris ber-`claimid` sama.
func DataOperationFor(existing int) string {
	if existing > 0 {
		return OperationUpdate
	}
	return OperationCreate
}

// ReportMonthOf membentuk `bulanlapor` dari sebuah waktu.
//
// `YYYYMM`, sama dengan `@DateTime.FormatDateTime(…,"yyyyMM","","")`. Waktunya diserahkan
// pemanggil, bukan dibaca dari jam sistem di sini — itulah yang membuatnya dapat diuji
// tanpa bergantung pada mesin penjalan (`F-5`).
func ReportMonthOf(now time.Time) string {
	return now.Format("200601")
}

// Netting mengurangi tunggakan dengan nilai recovery.
//
// Disalin dari `PNCUploadAutoClaimSlikOJK`, yang mengisi tunggakan dengan
// `.TUNGGAKAN - .RecoveryClaim`.
//
// # Kenapa hasilnya TEKS, dan kenapa nilai yang tidak terbaca dikembalikan APA ADANYA
//
// Kolomnya bertipe belum diketahui (`R-08`) dan nilainya datang sebagai teks. Bila salah
// satunya tidak dapat dibaca sebagai angka, nilai tunggakan dikembalikan **apa adanya**
// alih-alih menjadi nol: nol adalah angka yang tampak sah pada laporan regulator, dan
// tunggakan yang diam-diam menjadi nol tidak akan ditanyakan siapa pun.
func Netting(arrears, recovery string) string {
	left, okLeft := parseAmount(arrears)
	right, okRight := parseAmount(recovery)
	if !okLeft || !okRight {
		return strings.TrimSpace(arrears)
	}
	return formatAmount(left - right)
}

func parseAmount(raw string) (float64, bool) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return 0, true
	}
	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// formatAmount menulis angka tanpa notasi ilmiah, sama seperti pembaca kolomnya.
func formatAmount(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// Clean merapikan seluruh isian dan mengembalikan salinannya.
func (e ReportEntry) Clean() ReportEntry {
	trim := strings.TrimSpace
	e.ClaimID = trim(e.ClaimID)
	e.ContractNo = trim(e.ContractNo)
	e.FacilityAccountNo = trim(e.FacilityAccountNo)
	e.DebtorCIF = trim(e.DebtorCIF)
	e.FacilityTypeCode = trim(e.FacilityTypeCode)
	e.FundSource = trim(e.FundSource)
	e.PolicyStart = trim(e.PolicyStart)
	e.PolicyEnd = trim(e.PolicyEnd)
	e.InterestRate = trim(e.InterestRate)
	e.CurrencyCode = trim(e.CurrencyCode)
	e.OriginalCurrencyValue = trim(e.OriginalCurrencyValue)
	e.Obligation = trim(e.Obligation)
	e.CollectibilityCode = trim(e.CollectibilityCode)
	e.DefaultDate = trim(e.DefaultDate)
	e.DefaultReasonCode = trim(e.DefaultReasonCode)
	e.Arrears = trim(e.Arrears)
	e.ArrearsDays = trim(e.ArrearsDays)
	e.ConditionDate = trim(e.ConditionDate)
	e.ConditionCode = trim(e.ConditionCode)
	e.BranchCode = trim(e.BranchCode)
	e.Remark = trim(e.Remark)
	e.ReportMonth = trim(e.ReportMonth)
	e.DataOperation = trim(e.DataOperation)
	e.Recovery = trim(e.Recovery)
	e.IDCardNo = trim(e.IDCardNo)
	e.CompanyNPWP = trim(e.CompanyNPWP)
	e.PolicyNo = trim(e.PolicyNo)
	e.ClientID = trim(e.ClientID)
	return e
}

// Validate memeriksa isian yang tanpanya baris laporan tidak dapat ditelusuri.
//
// Hanya DUA yang wajib, dan keduanya wajib karena alasan yang sama: `claimid` adalah
// kunci yang dipakai pencacah untuk menentukan `'C'` versus `'U'`, dan `contractno`
// yang membedakan dua fasilitas kredit pada satu klaim. Tanpa keduanya, baris yang
// tersusun tidak dapat dicocokkan kembali ke klaimnya.
//
// Sisanya TIDAK divalidasi, dan itu disengaja: sistem lama pun menyisipkan apa adanya,
// dan menolak baris karena satu kolom kosong berarti menahan laporan yang di Pega lolos.
func (e ReportEntry) Validate() error {
	violations := make([]Violation, 0, 2)
	if e.ClaimID == "" {
		violations = append(violations, Violation{
			Field:   FieldClaimID,
			Message: "No Klaim wajib diisi.",
		})
	}
	if e.ContractNo == "" {
		violations = append(violations, Violation{
			Field:   FieldContractNo,
			Message: "Contract No wajib diisi.",
		})
	}
	return NewValidationError(violations)
}

// WriteOutcome adalah hasil satu aksi tulis.
type WriteOutcome struct {
	// Created dan Updated menghitung baris menurut `operasidata` yang ditetapkan.
	//
	// Keduanya dipisah karena artinya berbeda bagi pelapor: `Created` adalah klaim yang
	// baru masuk laporan, `Updated` adalah klaim yang SUDAH pernah dilaporkan dan kini
	// dilaporkan lagi — dan yang kedua itulah yang patut diperiksa sebelum dikirim ke OJK.
	Created int
	Updated int

	// Skipped adalah baris yang ditolak beserta sebabnya.
	//
	// Satu baris cacat TIDAK menggagalkan seluruh unggahan: berkas berisi seribu baris
	// yang ditolak seluruhnya karena satu baris kosong memaksa pelapor menebak baris mana.
	Skipped []SkippedRow
}

// Total adalah banyaknya baris yang benar-benar tersusun.
func (o WriteOutcome) Total() int { return o.Created + o.Updated }

// SkippedRow adalah satu baris yang tidak dapat disusun.
type SkippedRow struct {
	// Line adalah nomor baris pada berkas unggahan, dihitung dari 1 dan TIDAK menghitung
	// baris kepala. Nol untuk baris yang tidak berasal dari berkas.
	Line int

	ClaimID string
	Reason  string
}

// Submission adalah satu pengiriman data debitur ke SLIK — tombol "SLIK OJK".
//
// Disalin dari `Activity/InsertDataSlinkOJKIndividu-Act.xml`, yang menempuh empat langkah:
//
//  1. nomor urut   `select nvl(max(id),0)+1 from t_claim_slink_individu`
//  2. catat        `insert into t_claim_slink_individu (NO_KLAIM, CONTRACT_NO, ID)`
//  3. kirim        Connect-REST `Rest_SendDataClientBasedDebitur`
//  4. simpan hasil `update … set clientid = ?, id_transaction = ? where id = ? and no_klaim = ?`
//
// Langkah 3 TIDAK dapat dibangun: berkas Connect-REST itu **nol kemunculan** di direktori
// `Connect REST/` (`R-16`). Ia karena itu berada di balik seam Sender, dan tanpa
// konfigurasi seam itu menolak dengan sebab yang terbaca — bukan diam-diam berhasil.
type Submission struct {
	// ID adalah nomor urut pengiriman, `nvl(max(id),0)+1`.
	ID int64

	ClaimID    string
	ContractNo string
}

// SubmissionResult adalah jawaban sistem SLIK atas satu pengiriman.
type SubmissionResult struct {
	// ClientID dan TransactionID disimpan kembali ke baris pengirimannya.
	//
	// Keduanya tidak punya arti bagi modul ini selain sebagai bukti kirim — dan itulah
	// satu-satunya jejak bahwa sebuah klaim sudah dilaporkan.
	ClientID      string
	TransactionID string
}

// Sender mengirim data debitur ke sistem SLIK.
//
// Seam, bukan pemanggilan langsung, dengan alasan yang lebih tegas daripada biasanya:
// kontraknya BELUM ADA. Adapter pertamanya karena itu dibangun terhadap kontrak yang
// belum diterima, sama seperti seam Identity pada `F-3` (`04-FUTURE-ARCHITECTURE.md` §3.5).
type Sender interface {
	Send(ctx context.Context, submission Submission) (SubmissionResult, error)
}
