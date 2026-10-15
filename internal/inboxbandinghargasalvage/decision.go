package inboxbandinghargasalvage

import (
	"strings"
	"time"
)

// Berkas ini memuat panel rincian pada grid "History Cheker" — daftar KEPUTUSAN banding
// harga untuk satu klaim.
//
// # Apa yang digantikan
//
//	Flow Action/DetailHistoryRequestSalvage-FA.xml  pembungkusnya; dikunci nomor klaim
//	Section/DetailHistReqSalvage-Section.xml        ketujuh kolomnya
//	Activity/ShowDtlHistoryReqSalvage_Act-Act.xml   pemasoknya (param `noklaim`)
//	RDB List/DetailHistReqSalvage_SQL-SQL.xml       kuerinya
//
// # Kenapa ia daftar, bukan satu baris
//
// Karena satu klaim dapat punya beberapa barang yang dibanding harganya, dan tiap barang
// punya keputusannya sendiri. Grid History menampilkan PENGAJUAN SALVAGE; panel ini
// menampilkan setiap keputusan di bawah klaim yang sama.

// Decision adalah satu keputusan banding harga atas satu barang.
type Decision struct {
	// ApprovedAt adalah Tgl Approve — `TGLAPPROVE`.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan. Di panel ini ia selalu terisi, karena kuerinya
	// menyaring `STATUSAPPROVE IS NOT NULL` — tetapi bentuk datanya tidak menjamin itu.
	ApprovedAt *time.Time

	// DetailObject adalah Detail Object — `IDDETAILSALVAGE`.
	DetailObject string

	// ItemName adalah Nama Barang — `NAMABARANG`.
	ItemName string

	// ItemPrice adalah Harga Barang — `HARGABARANG`, harga yang diajukan PIC.
	//
	// Teks, bukan pecahan: nilai uang disimpan presisi penuh dan pembulatan hanya terjadi
	// saat ditampilkan (`I-12`).
	ItemPrice string

	// RequestPrice adalah Harga Request — `HARGAREQUEST`, harga tandingan balai lelang.
	RequestPrice string

	// Status adalah kode `STATUSAPPROVE` APA ADANYA, bukan kalimatnya.
	//
	// Penerjemahannya ada di DecisionLabel, dan itu disengaja — lihat catatan di sana.
	Status string

	// CommitteeName adalah Nama Checker — `NAMAKOMITE`.
	CommitteeName string
}

// DecisionQuery adalah permintaan isi panel rincian yang sudah tervalidasi.
type DecisionQuery struct {
	// ClaimNo adalah nomor klaim yang dibuka, sudah dipangkas dan dihurufbesarkan.
	ClaimNo string

	// Reviewer menyatakan antrean SIAPA yang dibaca.
	//
	// Kuerinya menyaring `NAMAKOMITE` pula, persis seperti kedua grid. Tanpa itu, seorang
	// komite dapat membaca keputusan komite lain hanya dengan menebak nomor klaim — dan
	// nomor klaim bukan rahasia.
	Reviewer Reviewer
}

// NewDecisionQuery membentuk permintaan panel rincian yang sah.
func NewDecisionQuery(claimNo string, caller Caller) (DecisionQuery, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return DecisionQuery{}, ErrCallerUnknown
	}

	clean := strings.ToUpper(strings.TrimSpace(claimNo))
	if clean == "" {
		return DecisionQuery{}, NewValidationError([]Violation{{
			Field:   FieldClaimNo,
			Message: "Nomor klaim wajib diisi.",
		}})
	}

	return DecisionQuery{ClaimNo: clean, Reviewer: ReviewerFor(cleanCaller)}, nil
}

// Kode `STATUSAPPROVE` beserta artinya.
//
// Dibaca dari `RDB List/DetailHistReqSalvage_SQL-SQL.xml`, yang menerjemahkannya di dalam
// `CASE WHEN`, dan DIKUATKAN dari arah yang berbeda oleh
// `Activity/ApprovalCheckerSalvage-Act.xml` langkah 11:
//
//	SalvagePrice = @if(statusapprove == "1", hargarequest, hargasalvage)
//
// yakni `1` menerima harga tandingan balai lelang dan `0` mempertahankan harga semula.
//
// # Kenapa arti kode ini layak ditulis dua kali
//
// Karena kode yang SAMA berarti hal yang BERBEDA di tabel lain. Pada
// `T_CLAIM_KOMITE_LIST`, `0` berarti MENUNGGU — bukan ditolak
// (`RDB List/CountAIDiterima_SQL`). Seseorang yang menyimpulkannya dari sana akan membaca
// penolakan sebagai "belum diputus".
const (
	DecisionApproved = "1"
	DecisionRejected = "0"
)

// DecisionLabel menerjemahkan kode keputusan menjadi teks yang dibaca pengguna.
//
// # Kenapa penerjemahannya di sini, bukan di dalam SQL
//
// Kueri lama menerjemahkannya di dalam `CASE WHEN`. Pemetaan yang hidup di dalam SQL tidak
// dapat diuji tanpa basis data, sementara pemetaan yang salah menampilkan keputusan yang
// keliru di layar tanpa satu pun galat. Modul Inbox Salvage menempuh jalan yang sama untuk
// `STSTRANSFER`.
//
// Ketiga teksnya disalin harfiah dari kuerinya — termasuk "PROSES" yang berhuruf kapital
// seluruhnya, karena itulah yang dibaca pengguna hari ini (`D-13`).
func DecisionLabel(code string) string {
	switch normalizeDecisionCode(code) {
	case DecisionApproved:
		return "Setuju"
	case DecisionRejected:
		return "Tidak setuju"
	default:
		// Termasuk kode kosong. Kueri menyaring `STATUSAPPROVE IS NOT NULL`, sehingga
		// kosong seharusnya tidak muncul — tetapi "PROSES" memang cabang `ELSE` kueri
		// aslinya, dan kode asing lebih baik terbaca sebagai "belum diputus" daripada
		// disembunyikan.
		return "PROSES"
	}
}

// normalizeDecisionCode merapikan kode yang datang dari kolom bertipe angka.
//
// `STATUSAPPROVE` dibandingkan sebagai ANGKA di kueri aslinya (`STATUSAPPROVE = 1`), sehingga
// kolomnya kemungkinan besar NUMBER. Driver memetakan NUMBER ke pecahan, dan pemindaian ke
// teks karena itu dapat menghasilkan `"1"` pada satu driver dan `"1.0"` pada driver lain.
//
// Tanpa perapian ini, selisih sekecil itu membuat SETIAP baris terbaca "PROSES" — keputusan
// yang sudah diambil tampil seolah belum. Ia tidak menghasilkan galat apa pun.
func normalizeDecisionCode(code string) string {
	clean := strings.TrimSpace(code)
	if dot := strings.IndexByte(clean, '.'); dot >= 0 {
		if strings.Trim(clean[dot+1:], "0") == "" {
			clean = clean[:dot]
		}
	}
	return clean
}

// Nama field JSON pada satu baris panel rincian.
const (
	FieldApprovedAt    = "tanggal_approve"
	FieldDecision      = "jawaban_checker"
	FieldCheckerName   = "nama_checker"
	FieldDecisionPrice = "harga_request_keputusan"
)

// decisionColumns adalah ketujuh kolom panel rincian, berurutan seperti sel gridnya pada
// `Section/DetailHistReqSalvage-Section.xml` (offset 58.757–83.635).
//
// Pasangan judul dan properti, satu per satu — dan seperti biasa tidak satu pun aliasnya
// menyatakan isinya:
//
//	Tgl Approve       .DateOfLoss   <- TGLAPPROVE
//	Detail Object     .Notes        <- IDDETAILSALVAGE
//	Nama Barang       .CityID       <- NAMABARANG
//	Harga Barang      .ContractNo   <- HARGABARANG
//	Harga Request     .BranchID     <- HARGAREQUEST
//	Jawaban Checker   .AgentID      <- CASE atas STATUSAPPROVE
//	Nama Checker      .ClaimID      <- NAMAKOMITE
//
// `.DateOfLoss` di sini berarti TANGGAL PUTUSAN, bukan tanggal kejadian — alias yang sama
// berarti tanggal kejadian di hampir setiap layar lain.
func decisionColumns() []Column {
	return []Column{
		{Key: FieldApprovedAt, Title: "Tgl Approve"},
		{Key: FieldDetailObject, Title: "Detail Object"},
		{Key: FieldItemName, Title: "Nama Barang"},
		{Key: FieldItemPrice, Title: "Harga Barang", Numeric: true},
		{Key: FieldDecisionPrice, Title: "Harga Request", Numeric: true},
		{Key: FieldDecision, Title: "Jawaban Checker"},
		{Key: FieldCheckerName, Title: "Nama Checker"},
	}
}

// DecisionColumns menyerahkan salinan daftar kolom panel rincian.
//
// Salinan, bukan senarai aslinya, dengan alasan yang sama seperti Tabs().
func DecisionColumns() []Column {
	source := decisionColumns()
	result := make([]Column, len(source))
	copy(result, source)
	return result
}
