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
//
// Keduanya dicatat di tabel yang SAMA karena akibatnya sama: penugasan berpindah. Yang
// membedakan hanya cakupannya.
type TransferScope string

const (
	TransferRow  TransferScope = "baris"
	TransferBulk TransferScope = "massal"
)

// ParseTransferScope membaca lingkup dari badan permintaan.
func ParseTransferScope(raw string) (TransferScope, bool) {
	switch TransferScope(strings.ToLower(strings.TrimSpace(raw))) {
	case TransferRow:
		return TransferRow, true
	case TransferBulk:
		return TransferBulk, true
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
	// Nilainya TIDAK divalidasi: daftar pilihannya adalah Rule-Obj-FieldValue yang hilang
	// dari export (`R-16`). Menebak isinya berarti menolak nilai sah yang tidak kita kenal.
	UserType string

	Reason string
	Status TransferStatus

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
	UserType     string
	Reason       string
}

// Batas panjang isian, mengikuti lebar kolomnya di migrasi `0014`.
//
// Diperiksa di DOMAIN, bukan hanya di basis data: isian yang terlalu panjang ditolak dengan
// pesan yang menyebutkan batasnya, bukan dengan ORA-12899 yang tidak berarti apa-apa bagi
// pengguna.
const (
	MaxOperatorLength = 64
	MaxUserTypeLength = 50
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

	if len(strings.TrimSpace(c.UserType)) > MaxUserTypeLength {
		violations = append(violations, Violation{
			Field:   FieldUserType,
			Message: "Type User terlalu panjang.",
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

// TransferRepo adalah seam ke penyimpanan permintaan transfer.
//
// TERPISAH dari Repo, dan pemisahannya mengikuti kepemilikan tabel: Repo membaca tabel milik
// Pega, TransferRepo menulis tabel milik aplikasi ini sendiri (`P-1`). Menyatukan keduanya
// akan menyembunyikan batas itu di balik satu antarmuka.
type TransferRepo interface {
	// Record mencatat satu permintaan.
	Record(ctx context.Context, request TransferRequest) error

	// PendingFor memetakan ClaimID ke permintaan yang masih menunggu atasnya.
	//
	// Ia ada supaya layar dapat MENYATAKANNYA. Tanpa itu, pengguna yang sudah menekan
	// Transfer tidak melihat perubahan apa pun — barisnya tetap di sana, karena penugasannya
	// memang belum berpindah — lalu menekannya lagi.
	PendingFor(ctx context.Context, claimIDs []string) (map[string][]TransferRequest, error)
}

// TransferRepoSelector memilih TransferRepo milik satu portal entitas.
//
// Permintaan atas klaim milik satu badan hukum yang tercatat di basis data badan hukum lain
// adalah kebocoran yang persis `R-20` larang — dan pada tabel ini akibatnya melampaui
// tampilan, karena yang tercatat adalah perintah yang akan dijalankan.
type TransferRepoSelector func(portalAlias string) (TransferRepo, error)
