package registrasi

import (
	"context"
	"strings"
	"time"
)

// # Input Estimasi
//
// Tahap `Assignment7` (Non-MBU) dan `Assignment10` (Travel) pada `Flow/Register_Flow.xml`,
// flow action `InputEstimasi`. Section yang direndernya, `InputEstimasiAdmin`, TIDAK ADA di
// export (`R-16`). Bentuk isiannya karena itu diturunkan dari tiga sumber lain yang ada:
//
//   - Section `Estimasi` (kelas `Data-ObjectItem`): kolom baris estimasi — Estimasi Ke,
//     Tanggal Estimasi, Tipe Estimasi, Mata Uang, Nilai Estimasi, Nilai Kurs (IDR).
//   - `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc` baris 1012–1046: jalurnya
//     `ObjectList → ObjectCoverageList → ObjectItemList → EstimationList`, disimpan ke
//     POOLDATA.T_CLAIM_ESTIMASI.
//   - `Database/CREATE_TABLE_2.sql`: TC_PNC_OBJECTITEM, tabel item objek yang dirancang
//     Work Owner.
//
// Rincian khusus Travel (keterlambatan, bagasi, pembatalan) BELUM dibawa.

// Kode Tipe Estimasi, dibaca dari `Activity/SummaryEstimasiKlaimPerpolisIDR-Act.xml`:
// tipe 2 dijumlahkan sebagai "Estimasi adjuster", selebihnya sebagai estimasi klaim.
// Rule Property yang memuat labelnya tidak ada di export; tipe 3 dan 4 (41 dan 1 baris
// di data Pega) tidak terbaca artinya dan tidak ditawarkan.
const (
	EstimateClaim    = "1"
	EstimateAdjuster = "2"
)

// ObjectItem adalah satu item pada coverage — satu baris TC_PNC_OBJECTITEM.
type ObjectItem struct {
	// Name dan Description adalah OBJECTITEMNAME dan DESKRIPSIOBJECT.
	Name        string
	Description string

	// Group adalah PROPERTYITEMGROUP — kelompok item properti polis Fire.
	Group string

	Estimation []Estimation
}

// ItemOption adalah satu pilihan "Objek" pada daftar item sebuah coverage.
type ItemOption struct {
	// ID adalah ObjectItemID pilihan — COVERAGETRAVEL.ID untuk Travel; kosong untuk lini lain.
	ID    string
	Name  string
	Group string
	TSI   Money
}

// # Objek item estimasi per lini — mengikuti `Section/ObjectItemList_sect.xml`
//
// Seluruh lini memakai DROPDOWN (pxAutoComplete), dengan dua sumber yang dipisah
// `IsTravel`:
//
//	Travel      dropdown `tempTravel.pxResults`: manfaat plan dari POOLDATA.COVERAGETRAVEL,
//	            plan = kode coverage Travel (`SearchCoverageTravel_RD`, Param.plan =
//	            .CoverageOldID). Terukur 2026-10-09: ObjectItemID klaim Pega = COVERAGETRAVEL.ID.
//	!IsTravel   dropdown `Showobjectitem`. Rule-nya tidak ada di export (`R-16`); untuk Fire
//	            isinya item properti polis (T_PROPERTYITEMLIST). Aneka, Marine Cargo, dan PA
//	            tidak punya daftar item — item klaim Pega di lini itu hampir seluruhnya
//	            "Others" (5.353 dari 5.445 item Aneka), sehingga pilihannya hanya itu.
//
// Item yang baru dibuat diberi nama bawaan seperti pembentukan objek Pega
// (DefaultItemName).

// ItemFromPropertyList menyatakan pilihan Objek dibaca dari item properti polis (Fire).
func ItemFromPropertyList(p Policy) bool { return SourceOf(p) == SourceProperty }

// ItemFromTravelPlan menyatakan pilihan Objek dibaca dari manfaat plan Travel.
func ItemFromTravelPlan(p Policy) bool {
	return p.Line == LineTravel || strings.EqualFold(strings.TrimSpace(p.BusinessType), "Travel")
}

// ItemOthers adalah satu-satunya pilihan Objek bagi lini tanpa daftar item, sekaligus nama
// item bawaannya — `GetObjectFromTable` dan kembarannya menulis "Others".
const ItemOthers = "Others"

// DefaultItemName adalah nama item yang dibuat otomatis untuk jaminan tanpa item:
// `GetObjectFromTable_Travel` menulis "OTHERS", `GetObjectFromTable` dan kembarannya
// "Others". Fire tidak diberi nama bawaan — itemnya dipilih dari item properti polis.
func DefaultItemName(p Policy) string {
	switch {
	case ItemFromPropertyList(p):
		return ""
	case ItemFromTravelPlan(p):
		return "OTHERS"
	default:
		return ItemOthers
	}
}

// ItemChoices adalah isi dropdown Objek item estimasi untuk satu jaminan.
type ItemChoices struct {
	Option []ItemOption
	// Default adalah nama item bawaan bagi item baru — lihat DefaultItemName.
	Default string
}

// ItemOptionSource adalah seam ke pilihan item objek dari polis: item properti polis Fire
// dan manfaat plan Travel. Lini lain tidak membaca apa pun.
type ItemOptionSource interface {
	ItemOptions(ctx context.Context, policy Policy, objectID string) ([]ItemOption, error)
	TravelBenefits(ctx context.Context, plan string) ([]ItemOption, error)
}

// Estimation adalah satu baris estimasi — satu baris T_CLAIM_ESTIMASI.
type Estimation struct {
	Type     string    // ESTIMATIONTYPE
	Currency string    // KURSID — kode mata uang, mis. 10026
	Date     time.Time // ESTIMATIONDATE
	Value    Money     // ESTIMATIONVALUE, dalam mata uang estimasi

	// Rate dan Converted dihitung layanan, tidak pernah dikirim layar: Rate adalah kurs
	// pada tanggal kejadian (`ADR-0015`), Converted nilai rupiahnya.
	Rate      ExchangeRate
	Converted Money

	// FaceSheet adalah PRINTFACECLAIM = 1: estimasi sudah dibuatkan Claim Face Sheet dan
	// tidak dapat diubah lagi. FaceSheetDate adalah CFSDATE.
	FaceSheet     bool
	FaceSheetDate time.Time
}

// msgNoSpreadingForEstimate adalah pesan `ValidateInputEstimate_act` langkah 2–4: polis
// tanpa spreading tidak boleh diberi estimasi.
const msgNoSpreadingForEstimate = "Tidak Dapat Tambah Estimasi Untuk Polis Tanpa Spreading"

// msgSendNeedsFaceSheet tidak ada padanannya di Pega: di sana tombolnya hanya mati. Teks ini
// tambahan, sehingga berbahasa Inggris (`D-80`).
const msgSendNeedsFaceSheet = "Download the Claim Face Sheet before sending the claim to PIC Teknik."

// SumConverted menjumlahkan nilai rupiah estimasi klaim (bukan adjuster) sebuah item —
// SUMESTIMATION.
func (o ObjectItem) SumConverted() Money {
	var total Money
	for _, e := range o.Estimation {
		if e.Type != EstimateAdjuster {
			total += e.Converted
		}
	}
	return total
}

// EstimationCount menghitung seluruh baris estimasi klaim.
func (k Claim) EstimationCount() int {
	n := 0
	for _, c := range k.AllCoverages() {
		for _, item := range c.Item {
			n += len(item.Estimation)
		}
	}
	return n
}

// HasFaceSheet adalah When `isCFS`: `.ClaimData.IsCFS_PNC = "1"` ATAU
// `ObjectList(1).ObjectCoverageList(1).IsCFS = "1"`.
//
// `DownloadClaimFaceSheet_act` mengisi IsCFS_PNC dari IsCFS jaminan yang dicetak, hanya
// bila IsCFS_PNC masih kosong — sehingga ia benar sejak Claim Face Sheet PERTAMA pada
// jaminan mana pun. Padanannya di sini: ada satu estimasi yang sudah dikunci CFS.
func (k Claim) HasFaceSheet() bool {
	for _, c := range k.AllCoverages() {
		for _, item := range c.Item {
			for _, e := range item.Estimation {
				if e.FaceSheet {
					return true
				}
			}
		}
	}
	return false
}

// ValidateEstimate adalah gerbang tahap Input Estimasi — tombol Kirim PIC Teknik.
//
// Aturan pertama dari `ValidateInputEstimate_act`: estimasi tidak dapat diberikan pada
// polis tanpa spreading. Aturan keduanya (estimasi berikutnya menunggu Claim Face Sheet)
// ditegakkan saat estimasi dipasang — lihat CheckNewEstimates.
//
// Tombol Kirim PIC Teknik pada `InputEstimasiAdmin_SECT` ber-`pyDisabledWhen !isCFS`.
// Di Pega itu hanya penonaktifan tombol; di sini ditegakkan juga di server (`D-59`).
//
// Nilai estimasi nol atau negatif ditolak per baris.
func ValidateEstimate(k Claim) error {
	var v collector
	if k.EstimationCount() > 0 && k.SpreadingCount() == 0 {
		v.add(ViolationNoSpreading, "estimasi", msgNoSpreadingForEstimate)
	}
	if k.EstimationCount() == 0 {
		v.add(ViolationEstimateMissing, "estimasi", "Nilai estimasi belum diisi.")
	} else if !k.HasFaceSheet() {
		v.add(ViolationSendNeedsFaceSheet, "estimasi", msgSendNeedsFaceSheet)
	}
	for _, c := range k.AllCoverages() {
		for _, item := range c.Item {
			for _, e := range item.Estimation {
				if e.Value <= 0 {
					v.add(ViolationEstimateMissing, "estimasi", "Nilai estimasi harus lebih dari nol.")
				}
			}
		}
	}
	if len(v.violations) == 0 {
		return nil
	}
	return &ValidationError{Violation: v.violations}
}

// CurrencyOption adalah satu pilihan Mata Uang — POOLDATA.CURRENCY.
type CurrencyOption struct {
	ID   string
	Name string
}

// CurrencyDirectory adalah seam ke master mata uang.
type CurrencyDirectory interface {
	Currencies(ctx context.Context) ([]CurrencyOption, error)
}
