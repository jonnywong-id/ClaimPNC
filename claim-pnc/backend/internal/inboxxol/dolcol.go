package inboxxol

import (
	"strings"
	"time"
)

// Aksi "INSERT DOL DAN COL" pada tab 1 Inbox XOL.
//
// # Dari mana aturannya
//
// Seluruhnya dari rule sistem lama, bukan dari tampilan:
//
//	Section/InboxClaimXOL-Section.xml:4621    tombol pembukanya
//	Section/InboxClaimXOL-Section.xml:10863   tombol Simpan → activity di bawah
//	Activity/InsertDateAndCauseLossXOL-Act.xml  pemeriksaan dan pemetaan nilainya
//	RDB List/DeleteDataInXOLSummarybasedondol-SQL.xml  sisipan akhirnya
//
// # Satu baris per GROUP BUSINESS, bukan satu baris per simpan
//
// Activity-nya memecah `InsertCauseOfDol.CoverageNote` — daftar kode group business
// perjanjian yang dipilih — menjadi page list, lalu mengulang RDB-Save untuk setiap
// anggotanya dengan `TempGetDOLCOL.District` diisi kode itu
// (`InsertDateAndCauseLossXOL-Act.xml`, langkah 7.1 dan 7.2).
//
// Perjanjian yang belum punya satu pun group business karena itu TIDAK menghasilkan baris
// apa pun. Di sistem lama hal itu lewat tanpa pesan — tombolnya terlihat berhasil dan
// gridnya tetap kosong. Di sini ia ditolak beserta sebabnya; lihat DolColRequest.Validate.

// DolColCurrencyID adalah kode mata uang yang dituliskan "INSERT DOL DAN COL".
//
// Sistem lama menuliskannya sebagai literal `"10026"` pada `TempGetDOLCOL.Currency`, bukan
// membacanya dari perjanjian maupun dari isian pengguna. Ia disalin apa adanya supaya
// baris yang ditulis aplikasi baru tidak berbeda dari baris yang ditulis Pega (`P-5`).
//
// Ia BELUM SEHARUSNYA berupa konstanta — `D-15` menetapkan tempatnya adalah master Mata
// Uang & Kurs pada `F-4`, dan master itu belum ada (`TKT-F4-004`). Dicatat di sini supaya
// ketika master tiba, yang berubah hanya satu tempat.
const DolColCurrencyID = "10026"

// DolColRequest adalah isian modal "INSERT DOL DAN COL".
type DolColRequest struct {
	// MasterID adalah perjanjian XOL yang dipilih lewat tombol "Pilih" di grid modal.
	// Ia menentukan group business MANA yang menerima baris baru.
	MasterID string

	// LossDate adalah Tanggal Kejadian dalam bentuk `DD/MM/YYYY` — bentuk yang sama
	// dengan kolom `DOL` dan dengan `tanggal_kejadian` di seluruh kontrak modul ini.
	LossDate string

	// CauseOfLoss adalah DESKRIPSI penyebab kerugian, bukan ID-nya: kolom `CAUSEOFLOSS`
	// menyimpan deskripsinya, dan `claim_summary` menyaringnya terhadap
	// `V_D_CAUSE_OF_LOSS.DESCRIPTION`.
	CauseOfLoss string
}

// Clean membuang spasi di tepi setiap isian.
func (r DolColRequest) Clean() DolColRequest {
	return DolColRequest{
		MasterID:    strings.TrimSpace(r.MasterID),
		LossDate:    strings.TrimSpace(r.LossDate),
		CauseOfLoss: strings.TrimSpace(r.CauseOfLoss),
	}
}

// lossDateLayout adalah bentuk kolom `DOL` di `POOLDATA.XOL_TABLE_ALL_KLAIM`.
//
// Kolomnya TEKS, bukan tanggal — `claim_summary` membacanya dengan
// `TO_DATE(k.DOL, 'dd/mm/yyyy')`. Menulisnya dalam bentuk lain berarti baris yang
// tersimpan tidak pernah terbaca grid mana pun, dan kegagalannya tidak menghasilkan satu
// pun pesan galat.
const lossDateLayout = "02/01/2006"

// Validate memeriksa seluruh isian sekaligus.
//
// SELURUH pelanggaran dikembalikan bersamaan, meniru sistem lama yang menampilkan semua
// pesannya sekaligus (`P-5`) — bukan satu per satu yang memaksa pengguna menekan Simpan
// tiga kali untuk mengetahui tiga hal yang salah.
//
// Ketiga pemeriksaan wajib ada di sistem lama dan dipertahankan di sini
// (`InsertDateAndCauseLossXOL-Act.xml`, tiga Property-Set pertama). Yang DITAMBAHKAN
// hanyalah keterbacaan tanggalnya: Pega hanya memeriksa kosong atau tidak, karena
// isiannya kalender yang tidak dapat diketik. Di sini isiannya dapat diketik, sehingga
// "31/02/2026" dapat sampai ke server — dan bila lolos, ia tersimpan sebagai baris yang
// `TO_DATE` tolak selamanya.
func (r DolColRequest) Validate() error {
	clean := r.Clean()

	var violations []Violation
	violations = appendMissing(violations, FieldMasterID, clean.MasterID,
		"Perjanjian XOL wajib dipilih lebih dulu.")
	violations = appendMissing(violations, FieldLossDate, clean.LossDate,
		"Date Of Loss wajib diisi.")
	violations = appendMissing(violations, FieldCauseOfLoss, clean.CauseOfLoss,
		"Cause Of Loss wajib dipilih.")

	if clean.LossDate != "" {
		if _, err := time.Parse(lossDateLayout, clean.LossDate); err != nil {
			violations = append(violations, Violation{
				Field:   FieldLossDate,
				Message: "Date Of Loss tidak sah. Isi dengan bentuk DD/MM/YYYY.",
			})
		}
	}

	return NewValidationError(violations)
}

// appendMissing menambahkan pelanggaran bila isian wajib kosong.
//
// Kembarannya ada di usecase sebagai appendRequired; yang ini berada di domain supaya
// Validate tidak perlu mengimpor usecase — arah ketergantungan yang justru terbalik.
func appendMissing(violations []Violation, field, value, message string) []Violation {
	if strings.TrimSpace(value) == "" {
		return append(violations, Violation{Field: field, Message: message})
	}
	return violations
}

// DolColInsert adalah SATU baris yang disisipkan ke `POOLDATA.XOL_TABLE_ALL_KLAIM`.
//
// Nama fieldnya mengikuti nama kolomnya, bukan nama properti klipboard Pega. Properti itu
// dialiaskan secara menyesatkan — `TempGetDOLCOL.NoKTP` memuat Tanggal Kejadian dan
// `TempGetDOLCOL.District` memuat kode group business — persis utang teknis §4.2 yang
// `D-19` perintahkan tidak dibawa.
type DolColInsert struct {
	// BusinessGroupID → GROUPBUSINESS. Satu baris per group business perjanjian.
	BusinessGroupID string

	// LossDate → DOL, dalam bentuk `DD/MM/YYYY`.
	LossDate string

	// CurrencyID → CURRENCY. Selalu DolColCurrencyID.
	CurrencyID string

	// LBUID → LBU_ID. Sistem lama menuliskannya KOSONG (`TempGetDOLCOL.AlasanKlaim` di-set
	// `""`), dan itu disalin apa adanya.
	LBUID string

	// OutstandingValue → OSVALUE. Selalu nol saat disisipkan; nilainya diisi kemudian oleh
	// proses lain. Baris ini mendaftarkan kombinasi DOL × COL, bukan nilainya.
	OutstandingValue float64

	// AcceptedValue → AKSEPVALUE. Selalu nol, alasan yang sama.
	AcceptedValue float64

	// CauseOfLoss → CAUSEOFLOSS, berisi DESKRIPSI-nya.
	CauseOfLoss string

	// SalvageValue → SALVAGEVALUE. Selalu nol, alasan yang sama.
	SalvageValue float64
}

// NewDolColRows merakit baris yang akan disisipkan untuk satu perjanjian.
//
// Perjanjian tanpa group business menghasilkan senarai KOSONG; pemanggilnya yang menolak,
// bukan fungsi ini — ia tidak tahu field mana yang harus ditandai di layar.
func NewDolColRows(request DolColRequest, businessGroupIDs []string) []DolColInsert {
	clean := request.Clean()
	rows := make([]DolColInsert, 0, len(businessGroupIDs))
	for _, id := range businessGroupIDs {
		group := strings.TrimSpace(id)
		if group == "" {
			continue
		}
		rows = append(rows, DolColInsert{
			BusinessGroupID:  group,
			LossDate:         clean.LossDate,
			CurrencyID:       DolColCurrencyID,
			LBUID:            "",
			OutstandingValue: 0,
			AcceptedValue:    0,
			CauseOfLoss:      clean.CauseOfLoss,
			SalvageValue:     0,
		})
	}
	return rows
}
