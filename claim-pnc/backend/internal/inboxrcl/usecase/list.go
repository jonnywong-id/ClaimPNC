// Package usecase mengorkestrasi modul Inbox RCL.
//
// Dua operasi, dan keduanya hanya MEMBACA:
//
//	Metadata  judul kolom, selisih terencana, dan keterbatasan yang berlaku
//	List      satu halaman antrean milik identitas LAMA pemanggil
//
// Tidak ada operasi yang menulis. Menyelesaikan tugas RCL Dokter berarti menjalankan Flow
// Action `SendToRCLDokter`, yang memindahkan penugasan — milik Pega selama masa paralel
// (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/platform/tabletext"
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
var columns = tabletext.Rows[Column](`
	Key                 | Title               | Note
	nomor_case          | Nomor Case          |
	nomor_polis         | No Polis            |
	nama_tertanggung    | Nama Tertanggung    |
	tanggal_masuk_inbox | Tanggal Masuk Inbox | Waktu analis mengirim klaim ke dokter RCL.
	deskripsi_analyst   | Deskripsi Analyst   |
`)

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
		"Antrean yang kosong karena identitas lama Anda tidak ditemukan dinyatakan sebagai " +
			"keadaan tersendiri. Di layar lama keduanya tampil sama — grid kosong tanpa " +
			"keterangan.",
	}
}

// Limitations adalah keterbatasan yang berlaku hari ini dan akan hilang dengan sendirinya.
func Limitations() []string {
	return []string{
		"Antrean disaring dengan identitas LAMA Anda (`POOLDATA.T_ACCESS_GROUP_PNC`), dan " +
			"hanya identitas lama yang tercatat pada grup Administrators, PNCKomite, atau " +
			"CaseManager. Itu perilaku layar lama apa adanya.",
		"Data dibaca dari tabel klaim POOLDATA.T_CLAIMLIST_ADMIN, bukan tabel Pega. Kolom " +
			"Tanggal Masuk Inbox, Nama Dokter RCL, dan Deskripsi Analyst sudah ada di tabel " +
			"itu, tetapi BELUM DIISI proses pengisinya — sampai itu terjadi, antrean ini " +
			"kosong bagi semua orang.",
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

	// LegacyIdentityFound menyatakan identitas lama pemanggil ditemukan.
	//
	// # Kenapa ia dibawa, bukan cukup dengan halaman kosong
	//
	// Karena dua keadaan yang sama sekali berbeda menghasilkan halaman kosong yang sama:
	//
	//	identitas ditemukan, antreannya kosong  -> memang tidak ada pekerjaan
	//	identitas TIDAK ditemukan                -> belum diketahui pekerjaan siapa
	//
	// Pega menampilkan keduanya sebagai grid kosong. Hasil yang dilihat pengguna tetap sama
	// (`P-5`) — tetap tidak ada baris — tetapi layar kini dapat menyebut sebabnya.
	LegacyIdentityFound bool
}

// List mengambil satu halaman antrean milik identitas lama pemanggil.
//
// # Urutannya disengaja
//
// Login diperiksa lebih dulu — tanpanya identitas lama tidak dapat dicari. Lalu portal
// dipilih, karena identitas lama tersimpan di basis data entitas. Lalu identitas lama
// dicari, dan bila tidak ada, antrean TIDAK dicari sama sekali.
//
// # Kenapa antrean tidak dicari dengan identitas kosong
//
// Di Pega `Param.assign` lalu bernilai string kosong, dan kedua penyaringnya (A dan D) tidak
// punya opsi "abaikan bila kosong" — `InboxRCLDokter_RD` tidak menyalakannya. Oracle
// memperlakukan string kosong sebagai NULL, sehingga perbandingan sama-dengan tidak
// mencocokkan baris apa pun dan hasilnya nol baris. Tidak menjalankan kuerinya memberi
// hasil yang SAMA tanpa bergantung pada dialek: di PostgreSQL string kosong bukan NULL, dan
// kueri yang sama akan mencocokkan setiap klaim yang dokter RCL-nya kosong.
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

	legacy, err := repo.LegacyOperatorFor(ctx, caller.Login)
	if err != nil {
		return Listed{}, fmt.Errorf("mencari identitas lama pemanggil: %w", err)
	}
	if legacy == "" {
		return Listed{
			Filter: clean,
			Page:   inboxrcl.Page{Tasks: []inboxrcl.RCLTask{}},
		}, nil
	}

	page, err := repo.List(ctx, legacy, clean)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil antrean RCL Dokter: %w", err)
	}

	return Listed{Filter: clean, Page: page, LegacyIdentityFound: true}, nil
}
