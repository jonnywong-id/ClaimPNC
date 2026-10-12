// Package usecase mengorkestrasi modul Inbox RCL.
//
// Empat operasi:
//
//	Metadata  judul kolom, selisih terencana, dan keterbatasan yang berlaku
//	List      satu halaman antrean milik pemanggil (`M_LOGIN_PNC.LOGIN_ID`)
//	Detail    isi layar kerja `RCLDokter` satu klaim di antrean pemanggil
//	Decide    keputusan dokter RCL — padanan `SendToPUCL` (decide.go); satu-satunya yang
//	          menulis, dan yang ditulisnya tabel aplikasi, bukan tabel Pega (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxrcl"
)

// Service melayani modul Inbox RCL.
type Service struct {
	repoSelector inboxrcl.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxrcl.RepoSelector
}

// NewService membentuk layanan modul Inbox RCL.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxrcl/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Column adalah satu judul kolom pada layar.
//
// Judul datang dari server karena ia HASIL PEMBACAAN `Harness/RCL_Harness-Harness.xml`
// (rule `pyCaption …`) — satu tempat pencatatan, bukan dua.
type Column struct {
	Key   string
	Title string
	Note  string
}

// columns adalah kelima kolom layar, dalam urutan tampilnya.
//
// Urutannya mengikuti kelima sel berkepala pada `Section/InboxRCLDokter_Section-Section.xml`:
// `.pyID` (tautan) · `.Policy.PolicyNo` · `.Policy.QQName` · `.ClaimData.TanggalAnalystSendRCL`
// · `.ClaimData.PUCLStatus.KomentarAnalisator`.
var columns = []Column{
	{Key: "nomor_case", Title: "Nomor Case"},
	{Key: "nomor_polis", Title: "No Polis"},
	{Key: "nama_tertanggung", Title: "Nama Tertanggung"},
	{
		Key:   "tanggal_masuk_inbox",
		Title: "Tanggal Masuk Inbox",
		Note:  "Waktu analis mengirim klaim ke dokter RCL.",
	},
	{Key: "deskripsi_analyst", Title: "Deskripsi Analyst"},
}

// Columns menyerahkan salinan daftar kolom.
func Columns() []Column {
	result := make([]Column, len(columns))
	copy(result, columns)
	return result
}

// PlannedDifferences adalah perbedaan yang DISENGAJA terhadap layar Pega (`D-54`).
func PlannedDifferences() []string {
	return []string{
		"Halaman dipotong basis data, bukan setelah seluruh baris ditarik. Layar lama " +
			"menarik semuanya, memotongnya di 500 baris, lalu menomori halamannya di " +
			"memori; antrean yang melampaui 500 karena itu tidak pernah terlihat utuh.",
		"Kotak cari Nomor Case dan No Polis adalah TAMBAHAN. Layar lama tidak punya " +
			"penyaring apa pun, dan tanpa pencarian sisi server satu klaim menjadi sulit " +
			"ditemukan begitu antreannya dipaginasi.",
	}
}

// Limitations adalah keterbatasan yang berlaku hari ini dan akan hilang dengan sendirinya.
func Limitations() []string {
	return []string{
		"Antrean disaring dengan login Anda di POOLDATA.M_LOGIN_PNC (harus aktif). Di layar " +
			"lama ia identitas lama dari T_ACCESS_GROUP_PNC; tabel itu tidak dipakai lagi.",
		"Daftar dan layar kerja dibaca dari POOLDATA.TC_PNC_PUCL — tabel status RCL/PUCL " +
			"yang juga dibaca Inbox RCL/PUCL — bukan tabel Pega.",
		"Pemeriksaan kewenangan menu belum ada (`TKT-F3-005`). Yang menjaga layar ini " +
			"sekarang hanyalah sesi, portal aktif, dan penyaring identitas.",
	}
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	Columns            []Column
	PlannedDifferences []string
	Limitations        []string
	PageSize           int
}

// Metadata menyerahkan keterangan layar. Ia tidak menyentuh basis data.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Columns:            Columns(),
		PlannedDifferences: PlannedDifferences(),
		Limitations:        Limitations(),
		PageSize:           inboxrcl.DefaultLimit,
	}
}

// Listed adalah satu halaman antrean beserta penyaring yang benar-benar dipakai.
type Listed struct {
	Filter inboxrcl.Filter
	Page   inboxrcl.Page

	// IdentityFound menyatakan login pemanggil ditemukan dan aktif di `M_LOGIN_PNC`.
	//
	// Tidak digambar layar (keputusan Work Owner 2026-09-27: layar sama dengan Pega, tanpa
	// peringatan). Tetap dikirim supaya antrean kosong dapat ditelusuri dari jawaban API:
	// "tidak ada pekerjaan" lawan "login tidak dikenal".
	IdentityFound bool
}

// List mengambil satu halaman antrean milik pemanggil.
//
// Login diperiksa lebih dulu, lalu portal dipilih (`M_LOGIN_PNC` tersimpan per entitas), lalu
// login dicari di `M_LOGIN_PNC`. Bila tidak ada atau tidak aktif, antrean TIDAK dicari sama
// sekali.
//
// # Kenapa antrean tidak dicari dengan operator kosong
//
// Di Pega `Param.assign` lalu bernilai string kosong, dan kedua penyaringnya (A dan D) tidak
// punya opsi "abaikan bila kosong". Oracle memperlakukan string kosong sebagai NULL, sehingga
// hasilnya nol baris. Tidak menjalankan kuerinya memberi hasil yang SAMA tanpa bergantung pada
// dialek: di PostgreSQL string kosong bukan NULL, dan kueri yang sama akan mencocokkan setiap
// klaim yang dokter RCL-nya kosong.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxrcl.Caller,
	filter inboxrcl.Filter,
) (Listed, error) {
	if caller.Login == "" {
		return Listed{}, inboxrcl.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	clean := filter.Normalize()

	operator, err := repo.OperatorFor(ctx, caller.Login)
	if err != nil {
		return Listed{}, fmt.Errorf("mencari login pemanggil di M_LOGIN_PNC: %w", err)
	}
	if operator == "" {
		return Listed{
			Filter: clean,
			Page:   inboxrcl.Page{Tasks: []inboxrcl.RCLTask{}},
		}, nil
	}

	page, err := repo.List(ctx, operator, clean)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil antrean RCL Dokter: %w", err)
	}

	return Listed{Filter: clean, Page: page, IdentityFound: true}, nil
}

// Detail mengambil isi layar kerja `RCLDokter` satu klaim — dibuka saat Nomor Case diklik.
//
// Kewenangannya sama dengan antrean: klaim hanya terbuka bila tampil di antrean pemanggil.
// Login yang tidak dikenal atau tidak aktif menghasilkan ErrClaimNotFound, bukan galat sesi:
// sesinya sah, hanya tidak ada antrean RCL miliknya.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxrcl.Caller,
	claimNumber string,
) (inboxrcl.RCLDetail, error) {
	if caller.Login == "" {
		return inboxrcl.RCLDetail{}, inboxrcl.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxrcl.RCLDetail{}, err
	}

	operator, err := repo.OperatorFor(ctx, caller.Login)
	if err != nil {
		return inboxrcl.RCLDetail{}, fmt.Errorf("mencari login pemanggil di M_LOGIN_PNC: %w", err)
	}
	if operator == "" {
		return inboxrcl.RCLDetail{}, inboxrcl.ErrClaimNotFound
	}

	detail, err := repo.Detail(ctx, operator, claimNumber)
	if err != nil && !errors.Is(err, inboxrcl.ErrClaimNotFound) {
		return inboxrcl.RCLDetail{}, fmt.Errorf("mengambil layar kerja RCL Dokter: %w", err)
	}
	return detail, err
}
