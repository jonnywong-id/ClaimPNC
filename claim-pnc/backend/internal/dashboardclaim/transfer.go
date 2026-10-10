package dashboardclaim

import (
	"context"
	"strings"
	"time"
)

// TransferScope membedakan kedua tombol Transfer pada layar lama.
//
//	baris    tombol "Transfer" pada satu baris Inbox Outstanding / Inbox Tampungan PIC
//	massal   "Transfer All Case By UserID" — seluruh pekerjaan satu operator sekaligus
//	saring   "Select All" — seluruh klaim yang cocok penyaring layar, lintas halaman
//
// Keduanya dicatat di tabel yang SAMA karena akibatnya sama: penugasan berpindah. Yang
// membedakan hanya cakupannya.
type TransferScope string

const (
	TransferRow  TransferScope = "baris"
	TransferBulk TransferScope = "massal"

	// TransferFilter memindahkan SELURUH hasil penyaring, bukan baris pada halaman yang
	// terbuka. Ia ada karena "Select All" berarti seluruhnya: pada data hari ini 1.639 klaim,
	// dan mengirimkannya sebagai 1.639 permintaan per baris membuat kegagalan di tengah
	// meninggalkan sebagian berpindah — keadaan yang tidak dapat dibedakan dari keberhasilan.
	TransferFilter TransferScope = "saring"
)

// ParseTransferScope membaca lingkup dari badan permintaan.
func ParseTransferScope(raw string) (TransferScope, bool) {
	switch TransferScope(strings.ToLower(strings.TrimSpace(raw))) {
	case TransferRow:
		return TransferRow, true
	case TransferBulk:
		return TransferBulk, true
	case TransferFilter:
		return TransferFilter, true
	default:
		return "", false
	}
}

// TransferStatus adalah keadaan sebuah permintaan.
//
// Aplikasi ini hanya pernah menulis `menunggu`. Dua nilai lainnya ditulis pelaksana —
// mekanisme di sisi Pega yang membaca antrean ini.
type TransferStatus string

const (
	TransferPending  TransferStatus = "menunggu"
	TransferExecuted TransferStatus = "dijalankan"
	TransferCanceled TransferStatus = "dibatalkan"
)

// UserType adalah "Type User" pada form Transfer All Case By UserID.
//
// Ketiga nilainya diambil langsung dari percabangan
// `Activity/GCNMTransferDataKlaim_act-Act.xml`, yang membandingkan `TempIns.CaseID` dengan
// tepat tiga teks. Nilainya ditulis PERSIS seperti di sana — termasuk spasi pada
// "PIC Teknik" — karena pelaksananya membandingkan teks, bukan kode.
type UserType string

const (
	UserTypeNone         UserType = ""
	UserTypeAdmin        UserType = "Admin"
	UserTypeTechnicalPIC UserType = "PIC Teknik"
	UserTypeOther        UserType = "Other"
)

var userTypes = []UserType{UserTypeNone, UserTypeAdmin, UserTypeTechnicalPIC, UserTypeOther}

// UserTypes mengembalikan isi dropdown "Pilih Type User".
func UserTypes() []UserType {
	result := make([]UserType, len(userTypes))
	copy(result, userTypes)
	return result
}

// Label adalah teks yang dibaca pengguna. Sama dengan nilainya, kecuali yang kosong.
func (u UserType) Label() string {
	if u == UserTypeNone {
		return "Pilih Type User"
	}
	return string(u)
}

// Known menjawab apakah nilainya salah satu dari ketiganya. Kosong DIKENAL: layar lama pun
// mengizinkan form dikirim tanpa memilih Type User.
func (u UserType) Known() bool {
	for _, known := range userTypes {
		if known == u {
			return true
		}
	}
	return false
}

// ParseUserType membaca pilihan dropdown.
//
// Pencocokannya PERSIS huruf besar-kecilnya, bukan case-insensitive: pelaksananya
// membandingkan teks apa adanya, sehingga "ADMIN" tidak akan cocok dengan cabang mana pun di
// sana. Menerimanya di sini berarti mencatat permintaan yang akan diam saat dijalankan.
func ParseUserType(raw string) (UserType, bool) {
	value := UserType(strings.TrimSpace(raw))
	if !value.Known() {
		return "", false
	}
	return value, true
}

// Nama field yang dapat dilanggar pada form Transfer.
const (
	FieldScope       = "lingkup"
	FieldClaimID     = "klaim_id"
	FieldFromUser    = "user_id_lama"
	FieldToUser      = "user_id_baru"
	FieldUserType    = "tipe_pengguna"
	FieldReasonTrans = "alasan"
)

// TransferRequest adalah satu permintaan pemindahan penugasan yang tercatat.
type TransferRequest struct {
	ID    string
	Scope TransferScope

	// ClaimID dan ClaimNumber terisi pada lingkup `baris`.
	ClaimID     string
	ClaimNumber string

	// FromOperator terisi pada lingkup `massal` — "User ID Lama".
	FromOperator string

	// ToOperator WAJIB pada kedua lingkup — "User ID Baru".
	ToOperator string

	// UserType adalah "Type User" pada layar lama. Boleh kosong.
	//
	// KOREKSI (2026-10-06): sebelumnya dinyatakan "tidak divalidasi karena daftar
	// pilihannya hilang dari export". Daftarnya TIDAK hilang — ia tidak berbentuk
	// Rule-Obj-FieldValue melainkan tertanam sebagai perbandingan di dalam
	// `Activity/GCNMTransferDataKlaim_act-Act.xml`, yang bercabang pada tepat tiga nilai:
	//
	//	TempIns.CaseID == "Admin"
	//	TempIns.CaseID == "PIC Teknik"
	//	TempIns.CaseID == "Other"
	//
	// Nilai di luar ketiganya tidak akan cocok dengan satu cabang pun, sehingga menerimanya
	// berarti mencatat permintaan yang pelaksananya tidak tahu harus berbuat apa.
	UserType UserType

	Reason string
	Status TransferStatus

	// MovedCount adalah BERAPA klaim yang benar-benar berpindah.
	//
	// Ia ada karena jalur massal tidak dapat menjawabnya di muka: "seluruh pekerjaan si A"
	// bisa berarti satu klaim atau empat puluh, dan layar perlu menyatakannya. Tanpa angka
	// ini, pengguna menekan Transfer lalu tidak tahu apakah ada yang berpindah sama sekali.
	MovedCount int

	RequestedBy     string
	RequestedByName string
	RequestedAt     time.Time
}

// TransferCommand adalah permintaan transfer dari layar, sebelum tercatat.
type TransferCommand struct {
	Scope TransferScope

	ClaimID     string
	ClaimNumber string

	FromOperator string
	ToOperator   string
	UserType     UserType
	Reason       string

	// Filter terisi pada lingkup `saring` saja.
	//
	// Ia dibaca dari parameter kueri permintaan — parameter yang SAMA dengan yang dipakai
	// layar saat memuat daftarnya. Menyusunnya ulang di server akan membuat kedua himpunan
	// menyimpang diam-diam begitu salah satunya berubah.
	Filter Filter
}

// Batas panjang isian, mengikuti lebar kolomnya di migrasi `0014`.
//
// Diperiksa di DOMAIN, bukan hanya di basis data: isian yang terlalu panjang ditolak dengan
// pesan yang menyebutkan batasnya, bukan dengan ORA-12899 yang tidak berarti apa-apa bagi
// pengguna.
const (
	MaxOperatorLength = 64
	MaxReasonLength   = 1500
	MaxClaimIDLength  = 64
)

// Validate memeriksa permintaan dan mengumpulkan SELURUH pelanggarannya.
//
// Tidak berhenti pada yang pertama: form ini punya empat isian, dan mengembalikan satu pesan
// per percobaan akan menyiksa pengguna (`12-CROSSCUTTING` §1.2 butir 1).
func (c TransferCommand) Validate() error {
	var violations []Violation

	switch c.Scope {
	case TransferRow:
		if strings.TrimSpace(c.ClaimID) == "" {
			violations = append(violations, Violation{
				Field:   FieldClaimID,
				Message: "Klaim yang dipindahkan wajib disebutkan.",
			})
		}
	case TransferBulk:
		if strings.TrimSpace(c.FromOperator) == "" {
			violations = append(violations, Violation{
				Field:   FieldFromUser,
				Message: "User ID Lama wajib diisi pada transfer massal.",
			})
		}
	default:
		violations = append(violations, Violation{
			Field:   FieldScope,
			Message: "Lingkup transfer tidak dikenal.",
		})
	}

	to := strings.TrimSpace(c.ToOperator)
	switch {
	case to == "":
		violations = append(violations, Violation{
			Field:   FieldToUser,
			Message: "User ID Baru wajib diisi.",
		})
	case len(to) > MaxOperatorLength:
		violations = append(violations, Violation{
			Field:   FieldToUser,
			Message: "User ID Baru terlalu panjang.",
		})
	}

	// Transfer ke operator yang SAMA ditolak.
	//
	// Ia tidak memindahkan apa pun, tetapi tetap mencatat baris permintaan dan menambah
	// antrean pelaksana — pekerjaan yang hasilnya nol bagi semua pihak.
	if c.Scope == TransferBulk &&
		strings.EqualFold(strings.TrimSpace(c.FromOperator), to) &&
		to != "" {
		violations = append(violations, Violation{
			Field:   FieldToUser,
			Message: "User ID Baru sama dengan User ID Lama — tidak ada yang berpindah.",
		})
	}

	// Panjangnya tidak lagi diperiksa — nilainya kini terbatas pada tiga yang dikenal, dan
	// ketiganya pendek. Yang diperiksa adalah apakah ia SALAH SATU dari ketiganya.
	if !c.UserType.Known() {
		violations = append(violations, Violation{
			Field:   FieldUserType,
			Message: "Type User tidak dikenal.",
		})
	}
	if len(strings.TrimSpace(c.Reason)) > MaxReasonLength {
		violations = append(violations, Violation{
			Field:   FieldReasonTrans,
			Message: "Alasan terlalu panjang.",
		})
	}

	return NewValidationError(violations)
}

// PICMove adalah satu pemindahan PIC Teknik.
//
// ClaimNumber ikut dibawa meski klaimnya sudah ditunjuk ClaimID, karena keduanya menunjuk
// kunci yang BERBEDA: `PZINSKEY` untuk tabel kerja, `NOKLAIM` untuk tabel ringkasan
// dashboard. Menurunkan yang satu dari yang lain menuntut pembacaan tambahan di dalam
// transaksi, dan layar sudah memegang keduanya.
type PICMove struct {
	ClaimID     string
	ClaimNumber string
	ToOperator  string
}

// PICMoveResult membawa keadaan SEBELUM pemindahan.
//
// FromOperator dibaca di dalam transaksi yang sama, bukan dikirim layar: layar memegang
// nilai yang dibacanya saat halaman dimuat, dan nilai itu dapat sudah berubah. Pencacah
// beban yang diturunkan harus milik orang yang benar-benar memegang klaim itu sesaat sebelum
// dipindahkan.
type PICMoveResult struct {
	FromOperator string
}

// AssignmentWriter memindahkan PIC Teknik sebuah klaim.
//
// # Kenapa seam TERSENDIRI, bukan method pada TransferRepo
//
// Karena keduanya menulis tabel milik pihak yang berbeda, dan batas itu wajib terlihat.
// `TransferRepo` menulis tabel milik aplikasi ini; `AssignmentWriter` menulis
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` — tabel milik Pega, yang dibaca 116 rule di sana.
//
// Menyatukannya ke satu antarmuka akan membuat satu-satunya tulisan aplikasi ini ke skema
// Pega tersembunyi di antara method lain yang tidak berbahaya.
type AssignmentWriter interface {
	// MovePIC memindahkan PIC Teknik satu klaim beserta pencacah bebannya, dalam SATU
	// transaksi.
	//
	// Keempat tulisannya tidak boleh terpisah: PIC yang berpindah tanpa pencacah yang ikut
	// bergeser membuat pemilihan petugas otomatis (`R-04`) memakai angka yang salah, dan
	// kesalahan itu tidak tampak sebagai galat — hanya sebagai pembagian beban yang timpang.
	MovePIC(ctx context.Context, move PICMove) (PICMoveResult, error)

	// MoveAllForOperator memindahkan SELURUH klaim berjalan milik satu petugas.
	//
	// Mengembalikan jumlah klaim yang berpindah. Nol bukan galat: petugas yang tidak
	// memegang satu klaim pun adalah keadaan yang sah, dan layar menyatakannya.
	//
	// Satu pernyataan, bukan perulangan: jumlah klaim seorang petugas tidak diketahui di
	// muka, dan memutarnya satu per satu membuat kegagalan di tengah meninggalkan sebagian
	// berpindah dan sebagian tidak.
	MoveAllForOperator(ctx context.Context, from, to string) (int, error)

	// MoveAllMatching memindahkan seluruh klaim yang cocok dengan penyaring layar.
	//
	// Yang dikirim PENYARINGNYA, bukan daftar nomor klaim. Dua sebabnya: daftar 1.639 nomor
	// tidak muat pada badan permintaan yang dibatasi, dan ia sudah basi antara saat layar
	// membacanya dan saat server menjalankannya. Penyaring selalu menunjuk himpunan yang
	// sama dengan yang dilihat pengguna.
	MoveAllMatching(ctx context.Context, filter Filter, to string) (int, error)
}

// AssignmentWriterSelector memilih penulis milik satu portal entitas.
//
// Alasannya sama dengan TransferRepoSelector, dan di sini lebih keras: yang ditulis adalah
// klaim milik satu badan hukum, dan menulisnya ke basis data badan hukum lain bukan sekadar
// menampilkan yang keliru (`R-20`) melainkan MENGUBAH yang keliru.
type AssignmentWriterSelector func(portalAlias string) (AssignmentWriter, error)
