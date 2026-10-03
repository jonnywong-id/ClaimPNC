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

// IsNonMBU adalah When `IsNonMBU`: Group Panel 003, 004, 006, atau 009.
func (l LineOfBusiness) IsNonMBU() bool {
	switch l {
	case LineMiscellaneous, LineMarineCargo, LineFire, "009":
		return true
	}
	return false
}

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

// ClaimStatusNames adalah nama sebagian kode Status Klaim (master V_STS_CLAIM), dipakai
// penyimpanan memori. Penyimpanan SQL membaca masternya langsung.
var ClaimStatusNames = map[ClaimStatus]string{
	"1142": "Rejected Claim",
	"1146": "View Polis",
	"1147": "Register",
}

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

	CoverageStart time.Time
	CoverageEnd   time.Time
	Declaration   bool

	// Kind adalah TypeOfPolicy apa adanya. Nilai "2" adalah Open Policy, yang pada
	// lini kargo mengubah asal spreading (lihat BuildInsuredItems).
	Kind            string
	Currency        string
	CreditGuarantee bool
	InsuredName     string

	// QQName dan DeliveryAddress (DeliveryAddressList(1).ASMAddress) dipakai membentuk
	// penerima klaim bawaan — lihat receiver.go.
	QQName                string
	DeliveryAddress       string
	BranchCode            string
	HasSpreadingAvailable bool

	// Medan di bawah ini diisi Pega ke POOLDATA.T_CLAIM_PNC saat klaim dibuat, dari
	// `PolicyData` yang dibekukan bersama klaimnya — `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc`
	// baris 317–373. Semuanya disimpan apa adanya; tidak satu pun ikut menentukan aturan
	// registrasi.

	BusinessCode         string // Quotation.BusinessCode   -> BUSINESSCODE
	BusinessName         string // Quotation.BusinessName   -> BUSINESSNAME
	BranchName           string // Quotation.BranchName     -> BRANCHNAME
	SourceOfBusiness     string // Quotation.SourceOfBusiness -> SOBNAMEID
	SourceOfBusinessName string // Quotation.SobName        -> SOBNAME
	ProdKe               string // ProdKe                   -> PRODKE
	PolicyLeader         string // PolicyLeader             -> POLISLEADER
	TypeOfCoins          string // TypeOfCoins              -> TYPEOFCOINS

	// Coinsurance diturunkan dari CoinsList — lihat DeriveCoinsurance.
	Coinsurance Coinsurance
}

// Coinsurance adalah posisi Asuransi Sinar Mas pada koasuransi polis ini.
type Coinsurance struct {
	Name     string  // COINSNAME     — nama perusahaan leader
	Role     string  // LEADER_MEMBER — LEADER, MEMBER, atau FAC IN
	ShareASM Percent // SHAREASM      — bagian Sinar Mas

	// HasShare membedakan share yang memang tidak diketahui dari share nol. Pega
	// menyimpan NULL untuk yang pertama, dan keduanya tidak boleh tertukar.
	HasShare bool
}

// CoinsuranceRow adalah satu baris `CoinsList` pada dokumen polis.
type CoinsuranceRow struct {
	Leader       string // "true" bila baris ini leader
	CoinsName    string
	PercentShare Percent
	HasShare     bool
}

// OwnCompany adalah penanda perusahaan sendiri pada CoinsList.
//
// Pega mencarinya dengan like '%ASURANSI SINAR MAS%', bukan kesamaan — nama pada
// dokumen polis dapat berawalan atau berakhiran lain.
const OwnCompany = "ASURANSI SINAR MAS"

// DeriveCoinsurance menurunkan posisi koasuransi persis seperti
// `PEGA_CONVERT_JSONKLAIM_PNC.prc` baris 315–373, termasuk perilakunya yang tampak janggal
// (`P-5`).
//
// # NULL, bukan teks kosong
//
// Di Oracle, CoinsName yang kosong adalah NULL, dan pada NULL baik LIKE maupun NOT LIKE
// tidak benar. Baris leader tanpa nama karena itu TIDAK menjadi MEMBER: ia jatuh ke cabang
// ketiga — LEADER dengan share 100 — sedangkan nama leader-nya ikut menjadi kosong.
// Fungsi ini meniru tiga keadaan itu (memuat, tidak memuat, tidak ada), bukan dua.
//
// # Yang terbawa antarbaris
//
// Nama leader dan share tidak diatur ulang di awal setiap baris, sehingga baris
// sesudahnya dapat menimpa atau mewarisi nilai baris sebelumnya — sama dengan variabel
// PL/SQL yang dipakai berulang.
//
// Polis fakultatif masuk (TypeOfCoins = "F") menimpa seluruhnya: perannya FAC IN, nama
// perusahaannya ceding company, dan sharenya dari penawaran fakultatif.
func DeriveCoinsurance(typeOfCoins string, rows []CoinsuranceRow, cedingName string, facShare Percent, facHasShare bool) Coinsurance {
	c := Coinsurance{Name: OwnCompany}
	role := ""

	for _, r := range rows {
		named := r.CoinsName != ""
		own := named && strings.Contains(r.CoinsName, OwnCompany)
		notOwn := named && !own

		if r.Leader == "true" {
			c.Name = r.CoinsName
		}
		if own {
			c.ShareASM, c.HasShare = r.PercentShare, r.HasShare
		}
		switch {
		case r.Leader == "true" && own:
			role = "LEADER"
		case r.Leader == "true" && notOwn:
			role = "MEMBER"
		case !named:
			role = "LEADER"
			c.ShareASM, c.HasShare = PercentFull, true
		}
	}

	// Daftar yang tidak ada, kosong, atau tidak menghasilkan peran sama sekali jatuh ke
	// bawaan yang sama: Sinar Mas sebagai leader penuh (baris 356–367).
	if role == "" {
		role = "LEADER"
		c.Name = OwnCompany
		c.ShareASM, c.HasShare = PercentFull, true
	}
	c.Role = role

	if typeOfCoins == "F" {
		c.Role = "FAC IN"
		c.Name = cedingName
		c.ShareASM, c.HasShare = facShare, facHasShare
	}
	return c
}

// Area adalah wilayah kejadian — `ClaimData.Country` sampai `ClaimData.PostalCode`.
//
// Setiap tingkat disimpan sebagai pasangan kode dan nama, persis seperti Pega: kode
// dipakai menyaring tingkat di bawahnya, nama yang ditampilkan dan dicetak. Keduanya
// disimpan supaya klaim tetap terbaca meski master wilayahnya kelak berubah.
//
// Kota sampai Kode Pos hanya berlaku bila negaranya Indonesia — section Pega
// menyembunyikannya dengan kondisi `.ClaimData.Country = 'INDONESIA'`.
type Area struct {
	Country, CountryID   string
	Province, ProvinceID string
	City, CityID         string
	District, DistrictID string // Kabupaten di layar; isinya kecamatan (KEC. …)
	RW, RWID             string // Kelurahan di layar
	PostalCode           string
}

// CountryIndonesia adalah nama negara yang membuka isian Kota sampai Kode Pos.
const CountryIndonesia = "INDONESIA"

// Nilai Prinsip Mengenal Nasabah. NORMAL adalah bawaan layar (`pyDefaultValue` 1).
const (
	CustomerPrincipleNormal     = "1"
	CustomerPrincipleSuspicious = "2"
)

// InsuredItem adalah satu objek pertanggungan yang tertimpa kejadian.
//
// Rincian objek per lini bisnis adalah lingkup `B-3`; yang ada di sini hanya yang
// dipakai validasi registrasi.
type InsuredItem struct {
	ID       string
	Name     string
	Location string
	Coverage []Coverage

	// Job dan DateOfBirth adalah Pekerjaan dan Tanggal Lahir peserta PA — kolom grid objek
	// `ShowObjectAdj` (IsPA). Hanya dibaca, dari POOLDATA.T_PERSONLIST polis (ASMJOBNAME,
	// ASMDATEOFBIRTH), sama dengan `GetListObjectPATravel`; T_CLAIM_OBJECTLIST tidak mengisinya.
	// DateOfBirth teks apa adanya: yyyymmdd atau dd/mm/yyyy.
	Job         string
	DateOfBirth string
}

// Coverage adalah satu jaminan yang dipakai pada sebuah objek.
type Coverage struct {
	ID string

	// Name adalah CoverageNote polis — nama jaminan, disimpan ke COVERAGENAME.
	Name        string
	CauseOfLoss string
	TSI         Money
	Spreading   []Spreading

	// Item adalah daftar item objek beserta estimasinya (tahap Input Estimasi).
	Item []ObjectItem

	// Settlement adalah AdjustmentList jaminan ini (tahap InputSurveyor) — lihat settlement.go.
	Settlement []SettlementLine

	// AnalystTransferred adalah ISANALISTRANSFER = 1 — `.IsAnalisTransfer` jaminan, diisi tombol
	// "Transfer ke Analyst" (`setTicketToAnalyst` step 5, bersama ISKOMITETRANSFER dan
	// USERBUSINESSPA). Jaminan yang sudah ditandai tidak menampilkan tombol itu lagi. Penyimpanan
	// hanya MENGISI penanda ini, tidak pernah mengosongkannya.
	AnalystTransferred bool
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

	// Area adalah wilayah kejadian bertingkat di bawah isian Lokasi pada layar Input
	// Register (`Section/ViewInputRegisterDetail-Section.xml`).
	Area Area

	// CustomerPrinciple adalah Prinsip Mengenal Nasabah — `ClaimData.CustomerPrinciple`.
	// Nilainya "1" NORMAL (bawaan layar) atau "2" SUSPICIOUS, dan ia ikut menentukan komite:
	// `Activity/SetEmailKomite-Act.xml` memeriksa `CustomerPrinciple == "2"`.
	CustomerPrinciple string

	// SuspiciousComment hanya tampil — dan hanya bermakna — bila CustomerPrinciple "2".
	SuspiciousComment string

	// Isian tab Input Register (`Section/InputRegisterDetail2_sect.xml`).
	//
	// EmailLOD — `ClaimData.EmailLOD` (EMAIL_LOD), hanya tampil untuk PA (Group Panel 002).
	// RemarkRecommendation — `ClaimData.RemarkRecommendation` (REMARKRECOMENDATION) dan
	// SubjectEmail — `ClaimData.SubjectEmail` (SUBJECTEMAIL), keduanya hanya selain PA.
	// SalvageStatus — `ClaimData.StatusSalvage` (STSSALVAGE), kode 1–5.
	EmailLOD             string
	RemarkRecommendation string
	SubjectEmail         string
	SalvageStatus        string

	EstimateValue Money
	Currency      string

	// SLIKNumber wajib untuk lini SPK / Asuransi Kredit. Kosongnya ditolak.
	SLIKNumber string

	// ExGratia menandai klaim yang dibayar di luar ketentuan polis. Ia mengubah jenis
	// treaty spreading menjadi ORS (langkah 37.3.5.10).
	ExGratia bool

	// TechnicalPIC adalah PIC teknik yang menerima klaim setelah registrasi.
	TechnicalPIC string

	// LargeLossNoticed menandai Notice of Large Losses sudah pernah terbit untuk klaim
	// ini — `ClaimData.FlagNOLL` di sistem lama.
	//
	// Ia menentukan SUBJEK pemberitahuan berikutnya, bukan apakah ia dikirim:
	// `Activity/SendEmailLargeLoss_act.xml` langkah 7 memakai subjek biasa selama flag
	// masih kosong, langkah 8 memakai "(REVISE)" setelah ia bernilai "1", dan langkah 11
	// mengisinya tepat setelah surel dikirim.
	//
	// Di Pega ia hidup di dalam BLOB work object dan TIDAK punya kolom sendiri —
	// terverifikasi 2026-09-24: tidak ada satu pun kolom bernama `%NOLL%` di POOLDATA
	// maupun DATAPEGA. Di sini ia diberi kolom `FLAG_NOLL` (migrasi `0010`), karena tanpa
	// tempat menyimpannya setiap pemberitahuan akan selamanya terbaca sebagai yang pertama.
	LargeLossNoticed bool

	// RCVID menautkan klaim ke pencatatan penerimaan dokumen (`B-14`). Kosong berarti
	// klaim tidak berasal dari Receive Document.
	RCVID string

	InsuredItem []InsuredItem

	// Receiver adalah ClaimData.ReceiverClaim — penerima klaim (T_CLAIM_RECEIVER).
	Receiver []Receiver

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

	ProcessStatus ProcessStatus
	ClaimStatus   ClaimStatus

	// AnalystTransferredAt adalah ANALYST_TRANSFERDATE — `ClaimData.AnalystTransferDate`, diisi
	// tombol "Transfer ke Analyst" (`setTicketToAnalyst` step 9) hanya bila masih kosong. Selain
	// tanggal, ia menggantikan penanda `IsAnalisTransfer = "1"` yang di Pega hidup di BLOB tanpa
	// kolom: klaim yang sudah pernah ditransfer ke Analyst tidak menampilkan tombol itu lagi.
	AnalystTransferredAt time.Time

	// ClaimStatusName adalah nama ClaimStatus dari master V_STS_CLAIM — hanya dibaca,
	// diisi penyimpanan.
	ClaimStatusName        string
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

// TotalSpreading menjumlahkan share baris spreading coverage ini yang tidak ditandai
// terhapus.
//
// Totalnya PER COVERAGE, bukan per klaim: `InputRegister_act` langkah 37.3.1 mereset
// `local.totalspreading := 0` di dalam loop ObjectCoverageList, menambahkannya di 37.3.5.9,
// dan memeriksanya di 37.3.6 — masih di dalam loop yang sama. Invarian `I-1` menyatakan
// hal yang sama. Klaim berobjek/berjaminan lebih dari satu karena itu berjumlah 100% di
// SETIAP jaminan, bukan 100% secara keseluruhan.
func (c Coverage) TotalSpreading() Percent {
	var total Percent
	for _, s := range c.Spreading {
		if s.Removed {
			continue
		}
		total += s.Share
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
