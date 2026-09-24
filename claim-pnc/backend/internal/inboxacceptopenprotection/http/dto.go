// Package inboxacceptopenprotectionhttp adalah lapisan transport modul Inbox Accept Open
// Protection.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
package inboxacceptopenprotectionhttp

import (
	"time"

	"claim-pnc/internal/inboxacceptopenprotection"
)

// protectionDTO adalah satu baris pada antrean akseptasi.
//
// Nama field berbahasa Indonesia karena ia KONTRAK yang dibaca frontend — salah satu dari
// lima pengecualian `D-80`.
type protectionDTO struct {
	NomorProteksi string `json:"nomor_proteksi"` // kolom "No Proteksi"
	NomorPolis    string `json:"nomor_polis"`    // kolom "No Polis"
	NomorKlaim    string `json:"nomor_klaim"`    // kolom "No Klaim"
	// TipeProteksi adalah KODE tipe, PROTECTION_TYPE_ID. Ia yang menentukan antrean.
	TipeProteksi string `json:"tipe_proteksi"`

	// NamaTipeProteksi adalah nama dari POOLDATA.M_CLAIM_PROTECTION_TYPE.
	//
	// KOSONG bila kodenya tidak terdaftar di master. Layar menampilkan kodenya apa adanya
	// dalam keadaan itu: petugas yang menyetujui pembukaan proteksi berhak tahu bahwa tipe
	// yang dihadapinya tidak dikenal sistem.
	NamaTipeProteksi string `json:"nama_tipe_proteksi"`
	TanggalProteksi  string `json:"tanggal_proteksi"` // kolom "Tanggal Proteksi Dibuat"
	Keterangan       string `json:"keterangan"`       // kolom "Keterangan"
	UserCreate       string `json:"user_create"`      // kolom "User Create"

	// Antrean menyatakan baris ini milik antrean mana. Diturunkan di server dari tipe
	// proteksi, bukan dihitung ulang di peramban.
	Antrean string `json:"antrean"`
}

// detailDTO adalah bentuk satu proteksi pada form akseptasi.
//
// Ia memuat data polis yang tidak ada di daftar — `Section/AcceptProtectionSection` memuat
// "Nama Tertanggung", "Start Date Time", dan "End Date Time".
type detailDTO struct {
	protectionDTO

	NamaTertanggung string `json:"nama_tertanggung"`
	PolisMulai      string `json:"polis_mulai"`
	PolisAkhir      string `json:"polis_akhir"`

	// StatusAkseptasi dikirim sebagai kode yang sama dengan yang disimpan: "" belum,
	// "1" disetujui, "2" ditolak.
	StatusAkseptasi string `json:"status_akseptasi"`

	TanggalAkseptasi string `json:"tanggal_akseptasi"`
	DiaksepOleh      string `json:"diaksep_oleh"`

	// MenungguKeputusan diturunkan di server supaya layar tidak perlu menafsirkan kode
	// status sendiri. Tombol setuju dan tolak hanya muncul ketika ia bernilai true.
	MenungguKeputusan bool `json:"menunggu_keputusan"`
}

// listResponse adalah badan respons daftar.
type listResponse struct {
	Proteksi []protectionDTO `json:"proteksi"`
	Total    int             `json:"total"`
	Antrean  string          `json:"antrean"`
}

// decideRequest adalah badan permintaan akseptasi.
//
// Ia memuat SATU field. Pelaku dan waktunya diterbitkan server — menerimanya dari klien akan
// membuat siapa pun dapat mengakseptasi atas nama orang lain, pada waktu yang ia tentukan
// sendiri.
type decideRequest struct {
	// Keputusan bernilai "setuju" atau "tolak". Nilai lain ditolak; tidak ada bawaan.
	Keputusan string `json:"keputusan"`
}

// ── Penerjemahan ─────────────────────────────────────────────────────────────────

const tanggalFormat = "2006-01-02"

func toProtectionDTO(p inboxacceptopenprotection.Protection, location *time.Location) protectionDTO {
	return protectionDTO{
		NomorProteksi:    p.Number,
		NomorPolis:       p.PolicyNumber,
		NomorKlaim:       p.ClaimNumber,
		TipeProteksi:     p.Type,
		NamaTipeProteksi: p.TypeName,
		TanggalProteksi:  formatDate(p.InputDate, location),
		Keterangan:       p.Note,
		UserCreate:       p.CreatedBy,
		Antrean:          string(inboxacceptopenprotection.QueueOf(p.Type)),
	}
}

func toDetailDTO(p inboxacceptopenprotection.Protection, location *time.Location) detailDTO {
	return detailDTO{
		protectionDTO:     toProtectionDTO(p, location),
		NamaTertanggung:   p.InsuredName,
		PolisMulai:        formatDatePtr(p.PolicyStart, location),
		PolisAkhir:        formatDatePtr(p.PolicyEnd, location),
		StatusAkseptasi:   p.AcceptStatus,
		TanggalAkseptasi:  formatDatePtr(p.AcceptedAt, location),
		DiaksepOleh:       p.AcceptedBy,
		MenungguKeputusan: p.Pending(),
	}
}

func formatDate(t time.Time, location *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if location == nil {
		location = time.UTC
	}
	return t.In(location).Format(tanggalFormat)
}

func formatDatePtr(t *time.Time, location *time.Location) string {
	if t == nil {
		return ""
	}
	return formatDate(*t, location)
}
