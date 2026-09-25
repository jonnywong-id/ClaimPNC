// Package usecase mengorkestrasi modul Inbox Salvage.
//
// Empat operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List      mengambil isi satu daftar
//	Counts    mengambil tabel ringkas "Status Salvage / Jumlah"
//	Create    menyimpan satu pengajuan salvage beserta detail itemnya
//
// Ekspor TIDAK menjadi operasi kelima: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// # Yang MENULIS, dan kenapa itu berbeda dari modul inbox lain
//
// Create menulis ke `POOLDATA.PNC_SALVAGE` dan `POOLDATA.DETAIL_PNC_SALVAGE`. Keduanya
// dimiliki modul ini selama masa paralel — tidak ada layar Pega lain yang menulisinya —
// sehingga `P-1` terpenuhi.
//
// Empat langkah yang dijalankan `Activity/SetStsSalvagePNC_act-Act.xml` dan TIDAK dijalankan
// di sini, seluruhnya menembak sesuatu di luar basis data ini: unggah berkas ke penyimpanan
// eksternal, kirim ke balai lelang SimasBid, kirim email, dan sisipkan salinan JSON klaim.
// Ketiadaannya dinyatakan ke pengguna lewat PlannedDifferences, bukan disamarkan.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxsalvage"
)

// Service melayani modul Inbox Salvage.
type Service struct {
	repoSelector inboxsalvage.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxsalvage.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Salvage.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxsalvage/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi daftar.
type Metadata struct {
	// Tabs adalah ketiga belas daftar beserta kolomnya.
	Tabs []inboxsalvage.Tab

	// DefaultTab adalah daftar yang terbuka pertama kali.
	DefaultTab string

	// StatusOptions adalah isi daftar pilihan "Status Salvage" pada form Tambah.
	StatusOptions []inboxsalvage.StatusOption

	// UploadColumns adalah judul kolom berkas "Upload Detail Salvage".
	//
	// Ia dikirim ke layar supaya keterangan di dekat tombol unggah menyebut kolom yang
	// BENAR-BENAR dibaca — bukan daftar yang ditulis ulang di layar dan kelak berbeda dari
	// yang dibaca pengurai.
	UploadColumns []string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	columns := make([]string, 0,
		len(inboxsalvage.RequiredUploadColumn)+len(inboxsalvage.OptionalUploadColumn))
	columns = append(columns, inboxsalvage.RequiredUploadColumn...)
	columns = append(columns, inboxsalvage.OptionalUploadColumn...)

	return Metadata{
		Tabs:               inboxsalvage.Tabs(),
		DefaultTab:         inboxsalvage.DefaultTab,
		StatusOptions:      inboxsalvage.StatusOptions(),
		UploadColumns:      columns,
		PlannedDifferences: inboxsalvage.PlannedDifferences,
	}
}

// Listed adalah isi satu daftar beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxsalvage.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim:
	// daftar yang diminta kosong menjadi daftar bawaan, dan layar harus tahu daftar mana
	// yang sebenarnya dijawab.
	Query inboxsalvage.Query
}

// List mengambil isi satu daftar.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
	input inboxsalvage.QueryInput,
	page inboxsalvage.Pagination,
) (Listed, error) {
	query, err := inboxsalvage.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi daftar %s: %w", query.Tab.Code, err)
	}

	// SETIAP pembukaan dicatat, bukan hanya yang mencurigakan.
	//
	// Seluruh baris layar ini memuat nomor klaim DAN nilai uang — nilai pengajuan PIC,
	// nilai request balai lelang, nilai penawaran. Selama pemeriksaan peran belum ada
	// (`TKT-F3-004`), jejak inilah satu-satunya hal yang menyatakan siapa yang membukanya,
	// dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru untuk
	// keadaan seperti ini.
	//
	// Kata kunci pencarian TIDAK ikut dicatat: ia dapat memuat nomor klaim, dan nomor
	// klaim adalah data nasabah (`D-69`). Yang dicatat adalah APAKAH pengguna mencari.
	if s.logger != nil {
		s.logger.Info(
			"daftar salvage dibuka",
			slog.String("modul", "inbox-salvage"),
			slog.String("daftar", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Bool("mencari", query.Search != ""),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}

// Counts mengambil tabel ringkas "Status Salvage / Jumlah".
//
// Ia terpisah dari List, dan itu disengaja: tabel ringkasnya TIDAK berubah saat pengguna
// berpindah daftar, sehingga layar dapat menyimpannya lebih lama daripada isi tabel. Satu
// pemanggilan yang mengembalikan keduanya sekaligus akan memaksa keempat belas hitungannya
// dijalankan ulang setiap kali pengguna membuka tab lain.
func (s *Service) Counts(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
) ([]inboxsalvage.StatusCount, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return nil, inboxsalvage.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	counts, err := repo.Counts(ctx, cleanCaller)
	if err != nil {
		return nil, fmt.Errorf("mengambil tabel ringkas salvage: %w", err)
	}
	return counts, nil
}

// Created adalah hasil penyimpanan satu pengajuan.
type Created struct {
	// SalvageID adalah ID yang terbit.
	//
	// Di sistem lama nilai ini keluar lewat parameter `ErrMsg OUT` yang SEKALIGUS membawa
	// pesan galat (`Database/INSERT_SALVAGE.prc:26`) — kontrak yang `D-68` nyatakan tidak
	// dibawa. Di sini ID ada di sini, dan kegagalan ada di galat.
	SalvageID string

	// ItemCount adalah jumlah baris Detail Item Salvage yang ikut tersimpan.
	ItemCount int
}

// Create menyimpan satu pengajuan salvage beserta detail itemnya.
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
	input inboxsalvage.FormInput,
) (Created, error) {
	form, err := inboxsalvage.NewForm(input, caller)
	if err != nil {
		return Created{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Created{}, err
	}

	salvageID, err := repo.Create(ctx, form)
	if err != nil {
		return Created{}, fmt.Errorf("menyimpan pengajuan salvage: %w", err)
	}

	// Penyimpanan dicatat SELALU, dan lebih rinci daripada pembacaan.
	//
	// Ia menulis nilai uang ke tabel yang dibaca laporan, dan `D-59` menetapkan tidak ada
	// pemisahan tugas formal — orang yang sama dapat membuat dan menyetujui. Jejak inilah
	// satu-satunya kontrol pengimbangnya.
	//
	// Nomor klaim TIDAK ikut dicatat: ia data nasabah (`D-69`). Yang dicatat adalah ID
	// salvage, yang menunjuk barisnya tanpa menyebut klaim siapa.
	if s.logger != nil {
		s.logger.Info(
			"pengajuan salvage disimpan",
			slog.String("modul", "inbox-salvage"),
			slog.String("id_salvage", salvageID),
			slog.String("mode", string(form.Mode)),
			slog.String("pemanggil", form.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Int("detail_item", len(form.Items)),
		)
	}

	return Created{SalvageID: salvageID, ItemCount: len(form.Items)}, nil
}
