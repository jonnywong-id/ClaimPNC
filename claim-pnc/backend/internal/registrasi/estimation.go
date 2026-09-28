package registrasi

import (
	"context"
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
//
// Untuk lini Fire, pilihannya item properti polis (`PropertyItemList` di
// POOLDATA.T_PROPERTYLIST: ItemType dan PropertyItemGroup). Untuk lini lain sumbernya
// tidak ada di export; daftarnya kosong dan layar memakai isian bebas.
type ItemOption struct {
	Name  string
	Group string
	TSI   Money
}

// ItemOptionSource adalah seam ke pilihan item objek dari polis.
type ItemOptionSource interface {
	ItemOptions(ctx context.Context, policy Policy, objectID string) ([]ItemOption, error)
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
