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

	// TipeProteksi adalah KODE tipe, `PROTECTION_TYPE_ID`.
	//
	// Ia tetap dikirim meski namanya sudah ada, karena kodenya yang dipakai form saat
	// menyunting dan yang menentukan antrean akseptasi.
	TipeProteksi string `json:"tipe_proteksi"`

	// NamaTipeProteksi adalah nama dari `POOLDATA.M_CLAIM_PROTECTION_TYPE`.
	//
	// KOSONG bila kodenya tidak terdaftar di master. Layar menampilkan kodenya apa adanya
	// dalam keadaan itu — BUKAN tanda hubung, dan bukan tebakan.
	//
	// Sebelum master diterima (2026-09-24), seluruh layar memang menampilkan kode: label
	// '1', '3', '4', '5', '6', dan '9' tidak ada di satu pun berkas export (`R-16`).
	// Keputusan menampilkan kode apa adanya saat itu ternyata menyelamatkan tipe '9', yang
	// dipakai 25 baris produksi dan nol kemunculan di export.
	NamaTipeProteksi string `json:"nama_tipe_proteksi"`

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

	// ReferensiKlaim adalah **ClaimID**, kolom `ID_CLAIM`.
	//
	// Hanya DIBACA — form tidak lagi mengirimkannya, karena server menurunkannya dari
	// nomor klaim (Work Owner, 2026-09-24: keduanya berisi nilai yang sama).
	//
	// Tetap dikirim ke klien karena pada baris WARISAN nilainya BERBEDA: Pega menyimpan
	// kunci teknisnya di sana, `ASM-FW-GCNMFW-WORK PNC-xxxx`. Menghilangkannya akan membuat
	// baris warisan tampak seolah ClaimID-nya sama dengan nomor klaimnya.
	ReferensiKlaim string `json:"referensi_klaim"`

	// DetailPerubahan terisi hanya untuk tipe '7' dan '8'.
	DetailPerubahan detailPerubahanDTO `json:"detail_perubahan"`
}

// detailPerubahanDTO adalah isi panel "Detail Perubahan" pada form.
// Seluruh field di sini HANYA DIBACA. Yang dikirim klien saat menyimpan ada di
// detailPerubahanRequest, dan isinya hanya dua.
type detailPerubahanDTO struct {
	// DOLSebelum adalah **Current Date Of Loss** — dari KLAIM, bukan dari isian.
	DOLSebelum string `json:"dol_sebelum"`

	// DOLBaru adalah **Next Date Of Loss** — dipilih pengguna.
	DOLBaru string `json:"dol_baru"`

	// PenyebabKerugian adalah **Cause Of Loss Dipilih** — dari KLAIM.
	PenyebabKerugian string `json:"penyebab_kerugian"`

	// PenyebabKerugianMaster adalah **Next Cause Of Loss** — dipilih pengguna.
	//
	// Namanya menyesatkan dan dipertahankan apa adanya supaya kontraknya tidak berubah dua
	// kali dalam satu hari. Yang menjadi acuan adalah label Pega di atas.
	PenyebabKerugianMaster string `json:"penyebab_kerugian_master"`

	// NamaObjek dan NamaCabang — dari KLAIM.
	NamaObjek  string `json:"nama_objek"`
	NamaCabang string `json:"nama_cabang"`
}

// detailPerubahanRequest adalah bagian panel yang BENAR-BENAR dikirim klien.
//
// # Kenapa bentuknya berbeda dari yang dibaca
//
// Empat dari enam field pada panel berasal dari KLAIM, bukan dari isian — lihat
// `Activity/OpenProtection-Act.xml`. Menerimanya dari klien berarti mempercayai pengirim
// untuk menyatakan keadaan klaim yang bukan miliknya.
//
// Bentuk yang berbeda antara baca dan tulis di sini bukan ketidakrapian melainkan
// pernyataan: yang dapat dikirim hanyalah yang memang milik pengirim.
type detailPerubahanRequest struct {
	// DOLBaru — "Next Date Of Loss". Wajib untuk tipe '7'.
	DOLBaru string `json:"dol_baru"`

	// PenyebabKerugianBaru — "Next Cause Of Loss". Wajib untuk tipe '8'.
	PenyebabKerugianBaru string `json:"penyebab_kerugian_baru"`
}

// listResponse adalah badan respons daftar.
type listResponse struct {
	Proteksi []protectionDTO `json:"proteksi"`
	Total    int             `json:"total"`
}

// protectionTypeDTO adalah satu pilihan tipe proteksi pada form.
//
// Nama fieldnya `kode` dan `nama`, bukan `id` dan `label`: keduanya menyebut apa yang
// dilihat pengguna, dan `kode` itulah yang muncul di layar ketika namanya tidak ada.
type protectionTypeDTO struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// protectionTypeListResponse membungkus daftar tipe.
//
// Dibungkus objek, bukan dikirim sebagai array telanjang. Array di akar tanggapan tidak
// punya tempat untuk menambahkan keterangan kelak — dan setiap penambahan setelahnya
// menjadi perubahan yang merusak klien.
type protectionTypeListResponse struct {
	Tipe []protectionTypeDTO `json:"tipe"`
}

// saveRequest adalah badan permintaan simpan, dipakai membuat maupun menyunting.
//
// Ia TIDAK memuat nomor proteksi, pembuat, maupun waktu pembuatan. Ketiganya diterbitkan
// server: nomor dari pencacah, pembuat dari sesi, waktu dari jam aplikasi. Menerimanya dari
// klien akan membuat siapa pun dapat menentukan nomor proteksinya sendiri dan menyimpan
// atas nama orang lain.
// # `referensi_klaim` TIDAK diterima di sini, dan itu disengaja
//
// Work Owner menegaskan 2026-09-24 bahwa ClaimNo dan ClaimID berisi nilai yang sama.
// `ID_CLAIM` karena itu diturunkan server dari `nomor_klaim`.
//
// Menerimanya lalu MENGABAIKANNYA diam-diam akan lebih buruk daripada menolaknya: klien
// yang mengirimnya akan mengira nilainya tersimpan. Field yang tidak dikenal pada badan
// JSON diabaikan `encoding/json` tanpa galat — itu perilaku yang diterima, dan bentuk
// tanggapannya yang memberi tahu nilai sebenarnya yang tersimpan.
type saveRequest struct {
	// NomorPolis TIDAK diterima di sini, dan itu disengaja.
	//
	// `Activity/OpenProtection-Act.xml` mengisi `pyWorkPage.PolicyNo` dari klaim yang
	// ditemukan — di sistem lama pun ia tidak pernah diketik. Server menurunkannya.
	NomorKlaim   string `json:"nomor_klaim"`
	TipeProteksi string `json:"tipe_proteksi"`
	Keterangan   string `json:"keterangan"`

	DetailPerubahan detailPerubahanRequest `json:"detail_perubahan"`
}

// ── Penerjemahan ─────────────────────────────────────────────────────────────────

// tanggalFormat adalah bentuk tanggal pada kontrak: ISO-8601 tanggal saja.
const tanggalFormat = "2006-01-02"

// toProtectionDTO menerjemahkan satu proteksi menjadi baris layar.
func toProtectionDTO(p inputreqprotection.Protection, location *time.Location) protectionDTO {
	return protectionDTO{
		NomorProteksi:    p.Number,
		NomorPolis:       p.PolicyNumber,
		NomorKlaim:       p.ClaimNumber,
		TipeProteksi:     p.Type,
		NamaTipeProteksi: p.TypeName,
		TanggalProteksi:  formatDate(p.InputDate, location),
		Keterangan:       p.Note,
		UserCreate:       p.CreatedBy,
		DapatDisunting:   p.Editable(),
		Premi:            inputreqprotection.IsPremium(p.Type),
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
		ClaimNumber: req.NomorKlaim,
		Type:        req.TipeProteksi,
		Note:        req.Keterangan,
		Change: inputreqprotection.ChangeRequest{
			LossDateAfter:    parseDate(req.DetailPerubahan.DOLBaru, location),
			CauseOfLossAfter: req.DetailPerubahan.PenyebabKerugianBaru,
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

// claimDTO adalah hasil pencarian klaim, untuk mengisi field TURUNAN pada form.
//
// Seluruh isinya hanya-baca di layar. Ia dikirim supaya pengguna MELIHAT apa yang akan
// tersimpan sebelum menekan Simpan — bukan supaya layar mengirimkannya kembali.
type claimDTO struct {
	NomorKlaim string `json:"nomor_klaim"`

	// NomorPolis dan NamaTertanggung mengisi kepala form.
	NomorPolis      string `json:"nomor_polis"`
	NamaTertanggung string `json:"nama_tertanggung"`

	// DateOfLoss adalah **Current Date Of Loss** pada panel tipe '7'.
	//
	// KOSONG bila klaimnya tidak punya DOL tercatat. Itu keadaan yang sah: hanya 1.740 dari
	// 2.166 baris `T_CLAIM_PNC` memilikinya, dan menolak klaim seperti itu akan menolak
	// klaim yang jelas-jelas ada.
	DateOfLoss string `json:"dol"`

	// PenyebabKerugian adalah **Cause Of Loss Dipilih** pada panel tipe '8'.
	PenyebabKerugian string `json:"penyebab_kerugian"`

	NamaObjek  string `json:"nama_objek"`
	NamaCabang string `json:"nama_cabang"`
}

// toClaimDTO menerjemahkan hasil pencarian menjadi bentuk layar.
func toClaimDTO(c inputreqprotection.Claim, location *time.Location) claimDTO {
	return claimDTO{
		NomorKlaim:       c.Number,
		NomorPolis:       c.PolicyNumber,
		NamaTertanggung:  c.InsuredName,
		DateOfLoss:       formatDatePtr(c.LossDate, location),
		PenyebabKerugian: c.CauseOfLoss,
		NamaObjek:        c.ObjectName,
		NamaCabang:       c.BranchName,
	}
}
