package registrasi

import (
	"math/big"
	"strings"
	"time"
)

// # Uang dan persentase disimpan sebagai bilangan bulat
//
// `ADR-0016` menuntut nilai uang presisi penuh dan pembulatan hanya saat ditampilkan.
// Tipe pecahan biner (float64) tidak dapat memenuhi itu: 0.1 tidak punya wakil yang
// tepat, dan selisihnya menumpuk di sepanjang rantai Estimasi → Usulan → Akseptasi →
// Dibayar.
//
// Karena itu uang disimpan dalam SEN (1/100 rupiah) dan persentase dalam PER SEPULUH
// RIBU (1/10000 persen) — keduanya bilangan bulat, keduanya eksak. Empat desimal pada
// persentase dipilih karena itu persis presisi yang `ADR-0016` pakai untuk memvalidasi
// total spreading.

// Money adalah nilai rupiah dalam sen. Rp 1.000 = 100_000.
type Money int64

// Rupiah membentuk Uang dari jumlah rupiah bulat.
func Rupiah(n int64) Money { return Money(n * 100) }

// Percent adalah persentase dalam satuan 1/10000 persen. 100% = 1_000_000.
type Percent int64

// PercentFull adalah 100%.
const PercentFull Percent = 1_000_000

// ExchangeRate adalah nilai tukar dalam satuan 1/10000. Kurs 16.250,5 = 162_505_000.
type ExchangeRate int64

// ExchangeRateOne adalah kurs 1,0000 — dipakai rupiah terhadap dirinya sendiri.
const ExchangeRateOne ExchangeRate = 10_000

// Convert mengubah nilai valuta asing menjadi rupiah memakai kurs.
//
// Perkalian dilakukan dengan bilangan bulat presisi sembarang lalu baru dibagi, supaya
// hasilnya tidak bergantung pada urutan operasi dan tidak melimpah pada nilai klaim
// besar — Rp 1 triliun dalam sen dikalikan kurs empat desimal melewati batas int64.
func (u Money) Convert(k ExchangeRate) Money {
	result := new(big.Int).Mul(big.NewInt(int64(u)), big.NewInt(int64(k)))
	result.Quo(result, big.NewInt(int64(ExchangeRateOne)))
	return Money(result.Int64())
}

// LineOfBusiness adalah kode Group Panel polis — penentu percabangan utama alur Register.
//
// Kodenya dibaca langsung dari rule When yang dipakai flow:
// `isPA_PNC` (`002`), `IsTravel` (`005`), `IsNonMBU` (`003`, `004`, `006`, `009`).
// Kode `007` dan `008` muncul di export tetapi tidak disebut satu pun rule itu.
type LineOfBusiness string

const (
	LinePersonalAccident LineOfBusiness = "002"
	LineMiscellaneous    LineOfBusiness = "003"
	LineMarineCargo      LineOfBusiness = "004"
	LineTravel           LineOfBusiness = "005"
	LineFire             LineOfBusiness = "006"
)

// ProcessStatus adalah posisi klaim dalam alur kerja — `StatusWork` di sistem lama
// (`ADR-0018`).
type ProcessStatus string

const (
	ProcessRunning  ProcessStatus = "BERJALAN"
	ProcessDone     ProcessStatus = "SELESAI"
	ProcessRejected ProcessStatus = "DITOLAK"
)

// ClaimStatus adalah status bisnis klaim — `StatusClaim` di sistem lama, 33 kode
// `1134`–`1166` yang berlabel di master (`ADR-0018`).
//
// Yang disebut namanya di sini hanya kode yang benar-benar dipakai modul ini. Sisanya
// tetap sah sebagai nilai; artinya dibaca dari master, bukan dari kode.
type ClaimStatus string

const (
	// StatusRegistered ditetapkan saat klaim selesai didaftarkan.
	StatusRegistered ClaimStatus = "1147"

	// StatusReturned ditetapkan saat petugas menekan Back.
	//
	// Bukti: `Activity/InputRegister_act-Act.xml` langkah 6 — `Property-Set` dengan
	// prasyarat `.pyNote=="Back"` mengisi `.ClaimData.StatusClaim` dengan `"1146"`.
	// Ini menutup sebagian `R-06`, yang mencatat arti kode `1142`–`1151` tidak
	// diketahui: arti `1146` kini terbukti.
	StatusReturned ClaimStatus = "1146"
)

// ClaimFlag adalah penanda biner — `ClaimStatus` di sistem lama (`ADR-0018`).
//
// `T-5` mencatat properti ini sebenarnya milik ruleset GISFW, bukan Claim PNC. Ia
// dibawa karena pemilik bisnis menyatakan keempat status memang berbeda (`D-18`);
// maknanya masih perlu dikonfirmasi.
type ClaimFlag string

const (
	FlagUnset ClaimFlag = "0"
	FlagSet   ClaimFlag = "1"
)

// ProgressPositionStatus adalah posisi pada rangkaian tahapan progres — `StatusPosisi` di
// sistem lama (`ADR-0018`).
type ProgressPositionStatus string

const (
	PositionInProgress ProgressPositionStatus = "On Progress"
	PositionDone       ProgressPositionStatus = "Done"
)

// Policy adalah bagian snapshot polis yang dibutuhkan registrasi.
//
// Ia BUKAN salinan penuh polis: kepemilikan data polis ada pada GISFW (`ADR-0006`), dan
// modul ini hanya membaca. Field yang ada di sini hanya yang benar-benar dipakai
// validasi Input Register.
type Policy struct {
	Number string
	Line   LineOfBusiness

	// BusinessType adalah `Quotation.BusinessType`. Ia dipakai membedakan Bonding dan
	// MBU, yang tidak dapat diturunkan dari Lini.
	BusinessType string

	CoverageStart         time.Time
	CoverageEnd           time.Time
	Declaration           bool
	Currency              string
	CreditGuarantee       bool
	InsuredName           string
	BranchCode            string
	HasSpreadingAvailable bool
}

// InsuredItem adalah satu objek pertanggungan yang tertimpa kejadian.
//
// Rincian objek per lini bisnis adalah lingkup `B-3`; yang ada di sini hanya yang
// dipakai validasi registrasi.
type InsuredItem struct {
	ID       string
	Name     string
	Location string
	Coverage []Coverage
}

// Coverage adalah satu jaminan yang dipakai pada sebuah objek.
type Coverage struct {
	ID          string
	CauseOfLoss string
	TSI         Money
	Spreading   []Spreading
}

// CauseOfLossPA adalah kode penyebab kerugian yang menjadi bagian kunci duplikasi
// khusus lini Personal Accident.
//
// Bukti: `Activity/InputRegister_act-Act.xml` langkah 33.2.1 — pemeriksaan duplikat
// kedua hanya dijalankan untuk baris coverage yang `CauseOfLossID=="12002"`.
const CauseOfLossPA = "12002"

// Spreading adalah satu baris pembagian risiko ke reasuransi.
//
// Pembagiannya sendiri milik `B-4`; registrasi hanya memeriksa keberadaannya dan
// totalnya.
type Spreading struct {
	TreatyKind string
	Name       string
	Share      Percent

	// Removed menandai baris yang ditandai terhapus di layar. `ADR-0012` melarang
	// penghapusan fisik; baris tetap ada dan tidak ikut dihitung.
	Removed bool

	// FacOfferItem terisi untuk treaty Fac Out. Kosongnya adalah galat kelengkapan.
	FacOfferItem string
}

// TreatyFacOut adalah kode treaty yang menuntut rincian Fac Offer terisi.
//
// Bukti: langkah 37.3.5.1 dan 37.3.5.2 pada `InputRegister_act`.
const TreatyFacOut = "10015"

// TreatyExGratia adalah kode treaty yang menggantikan seluruh baris spreading pada klaim
// ex gratia.
//
// Bukti: langkah 37.3.5.10 pada `InputRegister_act` — `Property-Set` dengan prasyarat
// `pyWorkPage.ClaimData.ExGratia=="1"` menimpa `.TreatyType` dengan `"10007"`. Penimpaan
// itu berlaku untuk SELURUH baris, bukan sebagian.
const TreatyExGratia = "10007"

// Reporter adalah orang yang melaporkan kejadian.
type Reporter struct {
	Name    string
	Phone   string
	Email   string
	Address string

	// Relation adalah kode hubungan pelapor dengan tertanggung. Kode 7 berarti
	// "lain-lain" dan menuntut OtherRelation terisi (langkah 28).
	Relation      int
	OtherRelation string
}

// RelationOther adalah kode hubungan pelapor yang menuntut keterangan tambahan.
const RelationOther = 7

// Claim adalah aggregate yang dicatat pada tahap Input Register.
//
// Ia sengaja tidak memuat seluruh pohon data klaim — pohon itu menyentuh 12 tabel dan
// menjadi lingkup `B-3`, `B-4`, dan `B-5`. Yang ada di sini adalah apa yang dilihat dan
// diisi petugas pada layar Input Register, ditambah empat status yang `ADR-0018`
// tetapkan.
type Claim struct {
	ID     string
	Number string
	Portal string

	Policy Policy

	DateOfLoss   time.Time
	ReportDate   time.Time
	DateReceived time.Time

	Location   string
	Chronology string
	Reporter   Reporter

	EstimateValue Money
	Currency      string

	// SLIKNumber wajib untuk lini SPK / Asuransi Kredit. Kosongnya ditolak.
	SLIKNumber string

	// ExGratia menandai klaim yang dibayar di luar ketentuan polis. Ia mengubah jenis
	// treaty spreading menjadi ORS (langkah 37.3.5.10).
	ExGratia bool

	// TechnicalPIC adalah PIC teknik yang menerima klaim setelah registrasi.
	TechnicalPIC string

	// RCVID menautkan klaim ke pencatatan penerimaan dokumen (`B-14`). Kosong berarti
	// klaim tidak berasal dari Receive Document.
	RCVID string

	InsuredItem []InsuredItem

	// PUCLStatus menyimpan `ClaimData.PUCLStatus.RCL_PUCL`. Nilai 2 mengarahkan klaim
	// ke tahap RCL/PUCL (rule `IsPUCL`).
	PUCLStatus int

	// ComplianceTransfer menyimpan `ClaimData.isComplianceTransfer`. Terisi `"1"`
	// mengarahkan klaim ke tahap Compliance (rule `IsCompliance`).
	ComplianceTransfer bool

	// RequestReturn menggantikan `.pyNote == "Back"`.
	//
	// Di sistem lama, tombol Back bekerja dengan menuliskan teks "Back" ke field
	// catatan, lalu rule `IsBackStage` membandingkannya. Perbandingan teks bebas
	// sebagai penanda alur adalah cacat yang tidak perlu dibawa: satu perbedaan
	// kapitalisasi mengubah ke mana klaim pergi. Yang dibawa adalah PERILAKUNYA,
	// bukan cara menyimpannya.
	RequestReturn bool

	ProcessStatus          ProcessStatus
	ClaimStatus            ClaimStatus
	ClaimFlag              ClaimFlag
	ProgressPositionStatus ProgressPositionStatus

	CurrentStage string

	CreatedBy string
	CreatedAt time.Time
	UpdatedBy string
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Deleted menyatakan klaim sudah ditandai terhapus (`ADR-0012`).
//
// Claim terhapus tetap ada di tabel dan tetap menempati kunci alaminya, tetapi tidak
// dihitung sebagai duplikat dan tidak muncul di daftar tugas.
func (k Claim) Deleted() bool { return k.DeletedAt != nil }

// AllCoverages mengembalikan seluruh coverage dari seluruh objek.
func (k Claim) AllCoverages() []Coverage {
	var result []Coverage
	for _, o := range k.InsuredItem {
		result = append(result, o.Coverage...)
	}
	return result
}

// TotalSpreading menjumlahkan share seluruh baris spreading yang tidak ditandai
// terhapus, pada seluruh coverage seluruh objek.
func (k Claim) TotalSpreading() Percent {
	var total Percent
	for _, c := range k.AllCoverages() {
		for _, s := range c.Spreading {
			if s.Removed {
				continue
			}
			total += s.Share
		}
	}
	return total
}

// SpreadingCount menghitung banyaknya baris spreading yang tidak ditandai terhapus.
func (k Claim) SpreadingCount() int {
	n := 0
	for _, c := range k.AllCoverages() {
		for _, s := range c.Spreading {
			if !s.Removed {
				n++
			}
		}
	}
	return n
}

// HighestTSI mengembalikan TSI tertinggi di antara seluruh coverage.
//
// Nilai estimasi dibandingkan terhadapnya: estimasi yang melebihi TSI berarti klaim
// menuntut lebih dari yang dipertanggungkan.
func (k Claim) HighestTSI() Money {
	var max Money
	for _, c := range k.AllCoverages() {
		if c.TSI > max {
			max = c.TSI
		}
	}
	return max
}

func equalFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
