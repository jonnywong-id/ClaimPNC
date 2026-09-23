// Package inputreqprotectionhttp adalah lapisan transport modul Input Req Protection.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4). Memakai
// tipe domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien, dan
// sebaliknya — penggantian nama field di domain menjadi perubahan yang merusak antarmuka.
package inputreqprotectionhttp

import (
	"time"

	"claim-pnc/internal/inputreqprotection"
)

// protectionDTO adalah satu baris pada layar.
//
// # Nama field berbahasa Indonesia
//
// Ia KONTRAK yang dibaca frontend, dan termasuk lima pengecualian `D-80` bersama komentar,
// nama kolom basis data, teks layar, dan variabel lingkungan. Yang berbahasa Inggris
// hanyalah nama tipe dan field Go-nya.
type protectionDTO struct {
	// NomorProteksi adalah kolom **"No Proteksi"**.
	//
	// Dua bentuk dapat muncul berdampingan: `OPC-XXX` warisan Pega dan `OPCN.YY.xxxx`
	// terbitan aplikasi ini. Layar menampilkan keduanya apa adanya.
	NomorProteksi string `json:"nomor_proteksi"`

	NomorPolis string `json:"nomor_polis"` // kolom "No Polis"
	NomorKlaim string `json:"nomor_klaim"` // kolom "No Klaim"

	// TipeProteksi dikirim sebagai KODE, bukan label.
	//
	// Labelnya tidak diketahui untuk sebagian besar nilai — daftarnya tinggal di rule
	// Property yang tidak ikut diekspor (`R-16`). Mengirim label yang ditebak akan
	// diterima pengguna begitu saja; mengirim kode membuatnya segera ditanyakan.
	TipeProteksi string `json:"tipe_proteksi"`

	// TanggalProteksi adalah kolom **"Tanggal Proteksi Dibuat"**.
	//
	// Dikirim sebagai teks ISO-8601 tanggal saja (YYYY-MM-DD), bukan timestamp. Kolom yang
	// ditampilkan adalah TANGGAL, dan mengirim timestamp UTC memaksa setiap tempat di
	// frontend memutuskan sendiri cara mengubahnya ke WIB — cara paling mudah membuat
	// tanggal bergeser satu hari tanpa ada yang menyadarinya (`R-12`).
	TanggalProteksi string `json:"tanggal_proteksi"`

	Keterangan string `json:"keterangan"`  // kolom "Keterangan"
	UserCreate string `json:"user_create"` // kolom "User Create"

	// DapatDisunting menggantikan `pyDisabledWhen` pada layar lama.
	//
	// Dihitung di SERVER, bukan di peramban. Aturannya — terkunci begitu nomor klaim
	// terisi — adalah aturan bisnis, dan aturan bisnis yang hidup di peramban tidak dapat
	// diuji bersama aturan lainnya. Frontend hanya mematikan tautannya.
	DapatDisunting bool `json:"dapat_disunting"`

	// Premi menyatakan baris ini masuk antrean akseptasi PREMI (`TipeProteksi == "2"`).
	//
	// Tidak ditampilkan sebagai kolom; ia dipakai layar untuk menerangkan ke mana baris ini
	// akan pergi setelah tertaut klaim.
	Premi bool `json:"premi"`
}

// detailDTO adalah bentuk satu proteksi beserta isian formnya.
//
// Dipisahkan dari protectionDTO karena keduanya menjawab pertanyaan yang berbeda: yang satu
// "apa isi daftar", yang satu "apa isi form". Menyatukannya akan mengirim detail perubahan
// pada setiap baris daftar — muatan yang tidak dipakai, pada layar yang barisnya banyak.
type detailDTO struct {
	protectionDTO

	// ReferensiKlaim adalah klaim yang benar-benar DITEMUKAN, bukan yang diketik.
	ReferensiKlaim string `json:"referensi_klaim"`

	// DetailPerubahan terisi hanya untuk tipe '7' dan '8'.
	DetailPerubahan detailPerubahanDTO `json:"detail_perubahan"`
}

// detailPerubahanDTO adalah isi panel "Detail Perubahan" pada form.
type detailPerubahanDTO struct {
	// DOLSebelum dan DOLBaru dipakai tipe '7'. Kosong berarti tidak diisi.
	DOLSebelum string `json:"dol_sebelum"`
	DOLBaru    string `json:"dol_baru"`

	// PenyebabKerugian dan PenyebabKerugianMaster dipakai tipe '8'.
	PenyebabKerugian       string `json:"penyebab_kerugian"`
	PenyebabKerugianMaster string `json:"penyebab_kerugian_master"`

	NamaObjek  string `json:"nama_objek"`
	NamaCabang string `json:"nama_cabang"`
}

// listResponse adalah badan respons daftar.
type listResponse struct {
	Proteksi []protectionDTO `json:"proteksi"`
	Total    int             `json:"total"`
}

// saveRequest adalah badan permintaan simpan, dipakai membuat maupun menyunting.
//
// Ia TIDAK memuat nomor proteksi, pembuat, maupun waktu pembuatan. Ketiganya diterbitkan
// server: nomor dari pencacah, pembuat dari sesi, waktu dari jam aplikasi. Menerimanya dari
// klien akan membuat siapa pun dapat menentukan nomor proteksinya sendiri dan menyimpan
// atas nama orang lain.
type saveRequest struct {
	NomorPolis     string `json:"nomor_polis"`
	NomorKlaim     string `json:"nomor_klaim"`
	ReferensiKlaim string `json:"referensi_klaim"`
	TipeProteksi   string `json:"tipe_proteksi"`
	Keterangan     string `json:"keterangan"`

	DetailPerubahan detailPerubahanDTO `json:"detail_perubahan"`
}

// ── Penerjemahan ─────────────────────────────────────────────────────────────────

// tanggalFormat adalah bentuk tanggal pada kontrak: ISO-8601 tanggal saja.
const tanggalFormat = "2006-01-02"

// toProtectionDTO menerjemahkan satu proteksi menjadi baris layar.
func toProtectionDTO(p inputreqprotection.Protection, location *time.Location) protectionDTO {
	return protectionDTO{
		NomorProteksi:   p.Number,
		NomorPolis:      p.PolicyNumber,
		NomorKlaim:      p.ClaimNumber,
		TipeProteksi:    p.Type,
		TanggalProteksi: formatDate(p.InputDate, location),
		Keterangan:      p.Note,
		UserCreate:      p.CreatedBy,
		DapatDisunting:  p.Editable(),
		Premi:           inputreqprotection.IsPremium(p.Type),
	}
}

// toDetailDTO menerjemahkan satu proteksi beserta isian formnya.
func toDetailDTO(p inputreqprotection.Protection, location *time.Location) detailDTO {
	return detailDTO{
		protectionDTO:  toProtectionDTO(p, location),
		ReferensiKlaim: p.ClaimReference,
		DetailPerubahan: detailPerubahanDTO{
			DOLSebelum:             formatDatePtr(p.ChangeDetail.LossDateBefore, location),
			DOLBaru:                formatDatePtr(p.ChangeDetail.LossDateAfter, location),
			PenyebabKerugian:       p.ChangeDetail.CauseOfLossID,
			PenyebabKerugianMaster: p.ChangeDetail.CauseOfLossMasterID,
			NamaObjek:              p.ChangeDetail.ObjectName,
			NamaCabang:             p.ChangeDetail.BranchName,
		},
	}
}

// toDraft menerjemahkan badan permintaan menjadi isian domain.
//
// Tanggal yang tidak dapat dibaca menjadi nil, BUKAN galat penguraian. Alasannya: field
// tanggal yang wajib akan ditolak validasi domain beserta seluruh pelanggaran lain
// sekaligus, dengan pesan yang menyebut kolomnya. Menolaknya di sini akan mengembalikan satu
// galat bentuk yang menyembunyikan pelanggaran lain yang juga ada.
func toDraft(req saveRequest, location *time.Location) inputreqprotection.Draft {
	return inputreqprotection.Draft{
		PolicyNumber:   req.NomorPolis,
		ClaimNumber:    req.NomorKlaim,
		ClaimReference: req.ReferensiKlaim,
		Type:           req.TipeProteksi,
		Note:           req.Keterangan,
		ChangeDetail: inputreqprotection.ChangeDetail{
			LossDateBefore:      parseDate(req.DetailPerubahan.DOLSebelum, location),
			LossDateAfter:       parseDate(req.DetailPerubahan.DOLBaru, location),
			CauseOfLossID:       req.DetailPerubahan.PenyebabKerugian,
			CauseOfLossMasterID: req.DetailPerubahan.PenyebabKerugianMaster,
			ObjectName:          req.DetailPerubahan.NamaObjek,
			BranchName:          req.DetailPerubahan.NamaCabang,
		},
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

// parseDate membaca tanggal WIB dari kontrak. Teks yang tidak dapat dibaca menjadi nil.
func parseDate(s string, location *time.Location) *time.Time {
	if s == "" {
		return nil
	}
	if location == nil {
		location = time.UTC
	}

	parsed, err := time.ParseInLocation(tanggalFormat, s, location)
	if err != nil {
		return nil
	}
	return &parsed
}
