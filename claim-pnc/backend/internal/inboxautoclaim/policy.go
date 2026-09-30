package inboxautoclaim

import (
	"context"
	"math/big"
	"strings"
)

// Pesan hasil pemeriksaan polis dan premi saat unggah.
//
// Tiga yang pertama DISALIN HARFIAH dari activity unggahan per bisnis
// (InboxAutoClaim/InsertKlaimToTable_*). Yang keempat baru — Pega tidak punya jalur untuk
// layanan premi yang mati — dan kalimatnya disetujui Work Owner 2026-09-29.
const (
	// MessageLossOutsidePolicyTravel: tanggal kejadian di luar periode polis, tab Travel
	// (InsertKlaimToTable_Travel :5739). Teksnya BERBEDA dari MessageLossOutsidePolicy
	// milik ANEKA, dan perbedaan itu dipertahankan — kolom TMP_MESSAGE lama memuat
	// keduanya.
	MessageLossOutsidePolicyTravel = "DOL tidak dalam range polis"

	// MessagePolicyCancelled: polis sudah dibatalkan, tab Kredit
	// (InsertKlaimToTable_Kredit :9301), bila StatusBusiness "3" dan FlagEdmBatal "1".
	MessagePolicyCancelled = "Polis sudah dibatalkan"

	// MessageAlreadyClaimed: nomor kontrak yang sama sudah pernah diunggah dan belum
	// gagal, tab Kredit (InsertKlaimToTable_Kredit :6857, lewat `CekObjekNotDouble`).
	MessageAlreadyClaimed = "Sudah Klaim"

	// MessagePremiumCheckFailed: layanan cek premi tidak dapat dihubungi, belum
	// terdaftar untuk portal ini, atau jawabannya tidak dapat dibaca. Keputusan Work Owner
	// 2026-09-29: barisnya DITANDAI GAGAL — tetap tersimpan, terlihat di grid, dan ikut
	// Export Gagal — bukan menolak seluruh unggahan dan bukan meloloskannya.
	MessagePremiumCheckFailed = "Cek premi gagal: layanan tidak dapat dihubungi"
)

// PetBusinessCode adalah kode produk hewan peliharaan. Hanya untuk produk ini tab ANEKA
// memeriksa tanggal kejadian terhadap periode polis (InsertKlaimToTable_Other :5005).
const PetBusinessCode = "10166"

// OpenProtectionPremiumType adalah tipe Open Protection yang membebaskan status "premi
// belum lunas" di tab Travel dan ANEKA (FindOpenProtecionAsKredit: `.TypeProtection==3`).
const OpenProtectionPremiumType = "3"

// PolicyDetail adalah data polis yang dibutuhkan pemeriksaan unggahan.
//
// Sumbernya POOLDATA.T_GENERAL — BUKAN `JSON_POLIS.DATA_JSONBLOB` yang dibaca Pega, karena
// kolom itu sudah tidak dipakai (Work Owner 2026-09-29). Katalog Oracle membuktikan
// T_GENERAL memuat seluruh field yang dibaca activity lama dari snapshot JSON.
type PolicyDetail struct {
	// StartDate dan EndDate berbentuk dd/mm/yyyy — tanggal saja, tanpa jam, supaya
	// pembandingannya per HARI dan tidak menafsirkan zona waktu (F-5).
	StartDate string
	EndDate   string

	BusinessCode     string
	StatusBusiness   string
	FlagEdmBatal     string
	Currency         string
	SourceOfBusiness string
	GroupPanel       string
}

// Cancelled menyatakan polis sudah dibatalkan (InsertKlaimToTable_Kredit :9281).
func (p PolicyDetail) Cancelled() bool {
	return strings.TrimSpace(p.StatusBusiness) == "3" && strings.TrimSpace(p.FlagEdmBatal) == "1"
}

// Covers menyatakan tanggal kejadian (dd/mm/yyyy) berada di dalam periode polis,
// termasuk hari pertama dan terakhirnya.
//
// Pembandingannya per HARI. Travel di Pega secara eksplisit membebaskan hari yang sama
// dengan awal/akhir periode (:5673); ANEKA tidak, tetapi karena STARTDATE/ENDDATE dibaca
// sebagai tanggal saja, hari pertama dan terakhir diterima di kedua tab. Periode yang
// tidak terbaca dianggap MENCAKUP — lebih aman meloloskan daripada menggagalkan atas dasar
// yang tidak dapat dipastikan.
func (p PolicyDetail) Covers(dateOfLoss string) bool {
	loss, ok := sortableDate(dateOfLoss)
	start, okStart := sortableDate(p.StartDate)
	end, okEnd := sortableDate(p.EndDate)
	if !ok || !okStart || !okEnd {
		return true
	}
	return start <= loss && loss <= end
}

// PremiumQuery adalah masukan satu pemeriksaan premi.
//
// Mengikuti parameter `CekPremiAutoKlaim` (nopolis, prodke); `idpega` belum ada saat
// unggah, sehingga tidak dikirim.
type PremiumQuery struct {
	PolicyNo   string
	ProductSeq string
}

// PremiumAnswer adalah jawaban layanan premi yang dipakai.
type PremiumAnswer struct {
	// AgingAmount dibaca apa adanya dari jawaban (`TempPNC2.PaymentData.AgingAmount`).
	AgingAmount string
}

// Unpaid menyatakan premi belum lunas, dengan kondisi Pega yang sama persis:
// `AgingAmount=="" || @toDecimal(AgingAmount) > 1` (InsertKlaimToTable_Kredit :8382).
//
// Angkanya dibandingkan sebagai bilangan rasional, bukan float — nilai uang tidak boleh
// kehilangan presisi (I-12). Teks yang bukan angka dianggap BELUM LUNAS: keadaannya tidak
// dapat dipastikan, dan meloloskannya berarti memproses klaim yang preminya mungkin belum
// dibayar.
func (a PremiumAnswer) Unpaid() bool {
	value := strings.TrimSpace(a.AgingAmount)
	if value == "" {
		return true
	}
	amount, ok := new(big.Rat).SetString(value)
	if !ok {
		return true
	}
	return amount.Cmp(big.NewRat(1, 1)) > 0
}

// PremiumChecker adalah seam ke layanan cek premi (`CekPremiAutoKlaim`).
//
// Dua pengisi: layanan REST Pega yang alamatnya dibaca dari POOLDATA.GCNM_CONNECT_REST
// per portal, dan Fake untuk pengujian serta mode memori.
//
// Galat apa pun — jaringan, alamat belum terdaftar, jawaban tidak terbaca — berarti
// MessagePremiumCheckFailed; pemanggil tidak perlu membedakannya.
type PremiumChecker interface {
	CheckPremium(ctx context.Context, portalAlias string, query PremiumQuery) (PremiumAnswer, error)

	// PremiumPaidBySource membaca total premi terbayar satu pasangan bisnis + sumber
	// bisnis untuk tab Cek Premi (`GetPremiumPaid_SPK`, TYPESERVICE `PREMI-API`).
	PremiumPaidBySource(ctx context.Context, portalAlias string, query PremiumCheckQuery) (string, error)
}
