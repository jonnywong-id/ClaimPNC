// Package inboxacceptopenprotectionhttp adalah lapisan transport modul Inbox Accept Open
// Protection.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
package inboxacceptopenprotectionhttp

import (
	"strings"
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

	// DetailPerubahan berisi panel "Detail Perubahan", dan bernilai null bagi tipe yang
	// tidak memunculkannya.
	//
	// Pointer, bukan struct kosong: layar membedakan "tipe ini tidak punya panel" dari
	// "punya, tetapi isinya kosong". Yang kedua terjadi pada baris warisan Pega.
	DetailPerubahan *detailPerubahanDTO `json:"detail_perubahan"`
}

// detailPerubahanDTO adalah isi panel "Detail Perubahan" pada form akseptasi.
//
// Field yang tidak berlaku bagi tipe yang sedang dibuka dikirim KOSONG, bukan dihilangkan —
// bentuk respons yang tetap membuat layar tidak perlu menebak field mana yang ada.
type detailPerubahanDTO struct {
	// Judul adalah judul panel, diturunkan di server dari tipe proteksi supaya layar tidak
	// menyimpan pemetaan kode yang kedua.
	Judul string `json:"judul"`

	// DolSebelum dan DolSesudah dipakai tipe '7'. Label layarnya "Current Date Of Loss"
	// dan "Next Date Of Loss".
	DolSebelum string `json:"dol_sebelum"`
	DolSesudah string `json:"dol_sesudah"`

	// PenyebabSebelum dan PenyebabSesudah dipakai tipe '8'. Label layarnya "Cause Of Loss
	// Dipilih" dan "Next Cause Of Loss".
	PenyebabSebelum string `json:"penyebab_sebelum"`
	PenyebabSesudah string `json:"penyebab_sesudah"`

	NamaObjek  string `json:"nama_objek"`  // "Object Name"
	NamaCabang string `json:"nama_cabang"` // "Branch Name"

	// Kosong menyatakan tidak ada satu pun isi yang terisi — panelnya tetap ditampilkan,
	// dengan keterangan bahwa rinciannya tidak tersedia. Baris warisan Pega tidak punya
	// kolom asal untuk kedua kolom ini.
	Kosong bool `json:"kosong"`
}

// queuesResponse adalah badan respons daftar antrean yang boleh dibuka pemanggil.
//
// Dibungkus objek, bukan larik telanjang: larik di akar respons menutup kemungkinan
// menambahkan field lain kelak tanpa merusak klien (`10-API-STRATEGY.md` §3).
type queuesResponse struct {
	Antrean []string `json:"antrean"`
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
		DetailPerubahan:   toDetailPerubahanDTO(p, location),
	}
}

// toDetailPerubahanDTO menyusun panel "Detail Perubahan", atau nil bila tipenya tidak
// memunculkannya.
//
// Syaratnya meniru `pyContainerVisibleWhen: .TypeProtection==8 || .TypeProtection==7` apa
// adanya — keputusan tampil ada di server, bukan di peramban, supaya kode tipe tidak perlu
// ditafsirkan di dua tempat.
func toDetailPerubahanDTO(
	p inboxacceptopenprotection.Protection,
	location *time.Location,
) *detailPerubahanDTO {
	if !inboxacceptopenprotection.ShowsChangeDetail(p.Type) {
		return nil
	}

	d := p.Change

	// Tanggal dikirim dalam bentuk yang sama dengan tanggal lain pada respons ini, bukan
	// apa adanya dari kolom — layar memformat seluruh tanggal dengan satu cara.
	return &detailPerubahanDTO{
		Judul:           judulPanelPerubahan(p.Type),
		DolSebelum:      formatDatePtr(d.LossDateBefore, location),
		DolSesudah:      formatDatePtr(d.LossDateAfter, location),
		PenyebabSebelum: d.CauseOfLossBefore,
		PenyebabSesudah: d.CauseOfLossAfter,
		NamaObjek:       d.ObjectName,
		NamaCabang:      d.BranchName,
		Kosong:          d.Empty(),
	}
}

// judulPanelPerubahan mengikuti `pyTitle` kedua panel di
// `Section/AcceptProtectionSection-Section.xml` apa adanya (`D-13`).
func judulPanelPerubahan(protectionType string) string {
	if strings.TrimSpace(protectionType) == inboxacceptopenprotection.TypeChangeCauseOfLoss {
		return "Detail Perubahan Cause Of Loss"
	}
	return "Detail Perubahan DOL"
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
