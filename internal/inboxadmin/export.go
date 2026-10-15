package inboxadmin

import (
	"context"
	"errors"
	"time"
)

// Tiga tombol ekspor pada layar lama, seluruhnya menghasilkan CSV dan seluruhnya
// MEMBACA saja:
//
//	Tombol                    Activity Pega                   Sumber
//	------------------------  ------------------------------  ----------------------------
//	Export LOD                DownloadAllKlaimCabangLOD       baris tab Branch Claim
//	Export Hasil Auto Claim   PNCExportAutoClaimPNC_Act       RDB GetHasilAutoClaim
//	Export Klaim Gagal        PNCExportAutoClaimGagal_Act     RDB GetHasilAutoClaimGagal
//
// Ketiganya memakai `pxConvertResultsToCSV`, sehingga judul kolom dan urutannya diambil
// dari parameter `CSVPropHeaders` masing-masing — termasuk dua judul yang hanya berupa
// nama properti Pega (`NOPOLIS`, `CLIENTID`, ...) pada Export Klaim Gagal, persis seperti
// berkas yang selama ini diterima pengguna (`D-13`).

// AutoClaimResult adalah satu baris POOLDATA.TMP_BATCH_CLAIM_KREDIT yang berhasil
// diproses menjadi klaim.
//
// Nama field mengikuti ISI kolomnya, bukan alias Pega: kueri lama mengaliaskan
// `NILAIKLAIM AS "BRANCHCODE"` dan `ACCEPTNO AS "EDMNO"` supaya cocok dengan properti
// klipboard yang sudah ada (`D-19`).
type AutoClaimResult struct {
	Initial          string // AGENID
	PolicyNumber     string // NOPOLIS
	ClaimID          string // IDPEGA
	AcceptanceNumber string // ACCEPTNO
	ClaimValue       string // NILAIKLAIM
	ObjectNumber     string // NOASURANSI
	Message          string // TMP_MESSAGE
}

// AutoClaimFailure adalah satu baris batch Auto Claim yang GAGAL diproses.
type AutoClaimFailure struct {
	PolicyNumber    string // NOPOLIS
	InsuranceNumber string // NOASURANSI
	ClaimValue      string // NILAIKLAIM
	ClaimType       string // TYPEKLAIM
	Currency        string // nama mata uang dari POOLDATA.CURRENCY, atau kodenya bila tidak ada
	AgentID         string // AGENID
	Message         string // TMP_MESSAGE
}

// AutoClaimRepo membaca hasil batch Auto Claim.
//
// Ia antarmuka TERSENDIRI, tidak ditambahkan ke Repo: data Auto Claim bukan antrean
// kerja, dan menambahkannya ke Repo akan memaksa setiap tiruan Repo di pengujian ikut
// menulis dua metode yang tidak berhubungan dengan isinya. Repo yang mampu membacanya
// cukup memenuhi antarmuka ini juga.
type AutoClaimRepo interface {
	// AutoClaimResults mengembalikan baris yang berhasil, diproses dalam rentang
	// [from, to), dan dimasukkan login yang disebut.
	AutoClaimResults(ctx context.Context, login string, from, to time.Time) ([]AutoClaimResult, error)

	// AutoClaimFailures mengembalikan SELURUH baris yang gagal, tanpa batas tanggal dan
	// tanpa saringan pengguna — persis seperti kueri lama.
	AutoClaimFailures(ctx context.Context) ([]AutoClaimFailure, error)
}

// ErrExportUnavailable berarti penyimpanan portal ini tidak dapat membaca data ekspor.
var ErrExportUnavailable = errors.New("inboxadmin: ekspor ini belum tersedia pada penyimpanan yang dipakai")

// ProcessingDay mengembalikan rentang satu hari kalender WIB yang memuat `now`.
//
// Kueri lama menyaring `trunc(tglproses) = trunc(sysdate)` — "hari ini" menurut jam
// server basis data. Di sini harinya ditentukan jam aplikasi dalam WIB
// (`04-FUTURE-ARCHITECTURE.md` §3.2). Batasnya dikirim sebagai tanggal-jam tanpa zona,
// karena kolom TGLPROSES (TIMESTAMP tanpa zona) menyimpan jam dinding WIB.
func ProcessingDay(now time.Time) (from, to time.Time) {
	wib := now.In(jakarta)
	from = time.Date(wib.Year(), wib.Month(), wib.Day(), 0, 0, 0, 0, time.UTC)
	return from, from.AddDate(0, 0, 1)
}

var jakarta = loadJakarta()

func loadJakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}
