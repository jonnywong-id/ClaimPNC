package usecase

import (
	"context"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Query adalah satu permintaan daftar dari layar.
//
// Ia sengaja terpisah dari inboxlaporanklaim.Filter: yang boleh dipilih pengguna hanya
// sebagian, dan sisanya diturunkan dari identitas pemanggil. Menyatukannya berarti layar
// dapat mengirim nilai untuk isian yang bukan haknya — misalnya cabang milik orang lain.
type Query struct {
	Category     inboxlaporanklaim.Category
	RegionCode   string
	BusinessLine inboxlaporanklaim.BusinessLine
	Keyword      string
	Pagination   inboxlaporanklaim.Pagination
}

// ListResult adalah jawaban lengkap satu permintaan daftar.
//
// Halaman dan pencacah dikembalikan bersama karena layar menggambar keduanya sekaligus.
// Meminta keduanya lewat dua permintaan HTTP akan membuat lencana di atas tab dan isi
// tabel di bawahnya berasal dari dua saat yang berbeda — dan selisih itu terlihat persis
// ketika berkas sedang ramai masuk.
type ListResult struct {
	Page    inboxlaporanklaim.Page
	Summary inboxlaporanklaim.Summary
}

// List mengembalikan satu halaman berkas laporan beserta pencacah seluruh tab.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxlaporanklaim.Caller,
	query Query,
) (ListResult, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return ListResult{}, err
	}

	filter, err := s.buildFilter(caller, query)
	if err != nil {
		return ListResult{}, err
	}

	page, err := repo.List(ctx, filter, query.Pagination.Clean())
	if err != nil {
		return ListResult{}, err
	}

	// Pencacah memakai penyaring yang SAMA, kecuali kategorinya: ia justru menghitung
	// setiap kategori sekaligus. Menghitungnya tanpa penyaring akan membuat lencana
	// menyebut angka yang tidak berhubungan dengan isi tabel di bawahnya — misalnya
	// jumlah seluruh kanwil padahal yang tampil satu kanwil saja.
	summary, err := repo.Summarize(ctx, filter)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{Page: page, Summary: summary}, nil
}

// Regions mengembalikan isi dropdown "Pilih Kanwil".
func (s *Service) Regions(ctx context.Context, portalAlias string) ([]inboxlaporanklaim.Region, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListRegions(ctx)
}

// Get mengembalikan satu berkas laporan.
func (s *Service) Get(
	ctx context.Context,
	portalAlias, id string,
) (inboxlaporanklaim.ClaimReport, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}
	return repo.Get(ctx, id)
}

// Create membuat berkas laporan baru dan mengembalikan baris tersimpannya.
//
// # Kenapa tidak ada satu pun isian dari pengguna
//
// Karena tombol "Buat Baru" di sistem lama memang tidak meminta apa pun.
// `Activity/CreateNewCaseRCV-Act.xml` hanya mengisi lima nilai, dan kelimanya diturunkan
// dari petugas yang menekannya:
//
//	Sender       := OperatorID.pyUserName
//	TelpPengirim := OperatorID.pyTelephone
//	DateForAging := @CurrentDateTime()
//	KodeCabang   := hasil GetIDCabang atas OperatorID
//	pyLabel      := AccessGroup.pyAccessGroup
//
// Berkasnya lahir KOSONG lalu dibuka supaya isinya dilengkapi di layar berikutnya. Isian
// itu — nama pelapor, nomor polis, kronologi, dan seterusnya — dikumpulkan layar
// `ViewReceiveDocument` yang menu dan modulnya terpisah (`B-14`), bukan layar ini.
//
// Menambahkan form di sini akan terasa lebih berguna, dan justru karena itu perlu
// ditolak: ia mengubah perilaku yang `P-5` tetapkan dipertahankan lebih dulu, dan
// perubahannya tidak ada di daftar 13 perbaikan eksplisit `D-49`.
//
// # Ke mana barisnya ditulis
//
// Ke POOLDATA.CPNC_LAPORAN_KLAIM, tabel milik aplikasi ini — BUKAN ke
// DATAPEGA.PC_ASM_FW_GCNMFW_WORK. Ditetapkan Work Owner 2026-09-19.
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	caller inboxlaporanklaim.Caller,
) (inboxlaporanklaim.ClaimReport, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}

	clean := caller.Clean()
	if clean.Login == "" {
		return inboxlaporanklaim.ClaimReport{}, inboxlaporanklaim.ErrCallerUnknown
	}

	// Cabang yang kosong TIDAK menghalangi pembuatan berkas. Alasannya ada di
	// inboxlaporanklaim/errors.go pada catatan "Kenapa TIDAK ada ErrBranchUnknown":
	// sistem lama pun tidak memeriksanya, dan menambah pemeriksaan di sini adalah aturan
	// baru yang tidak ada di 13 butir perbaikan `D-49`.

	now := s.clock.Now()

	// Nomornya TIDAK diisi di sini: ia diterbitkan repo di dalam operasi yang sama
	// dengan penyisipannya. Lihat catatan pada seam Repo.Insert.
	return repo.Insert(ctx, inboxlaporanklaim.ClaimReport{
		BranchCode: clean.BranchCode,

		// Nama petugas pembuat, bukan nama pelapor — berkas yang baru dibuat memang
		// belum tahu siapa pelapornya. Itu yang `CreateNewCaseRCV` lakukan, dan itu pula
		// yang ditiru di sini.
		ReporterName: clean.Name,

		CreatedBy: clean.Login,
		CreatedAt: now,
		AgingAt:   now,

		Transferred: false,
		ClaimNumber: "",
		Position:    inboxlaporanklaim.DerivePosition(false, false),
		Origin:      inboxlaporanklaim.OriginNew,
	})
}

// buildFilter menyusun penyaring akhir dari pilihan pengguna dan identitas pemanggil.
//
// # Batas data, dan satu andaian yang perlu dibaca
//
// `SetListRCV_Act` menyusun DUA potongan penyaring cabang yang keduanya ditempelkan ke
// kueri yang sama:
//
//	tempQuery.NoKTP  := "... branch where ID='<cabang petugas>'"        cabang sendiri
//	tempQuery.Remark := "... branch where basterritory=<kanwil pilihan>" satu kanwil
//
// Rule When yang memutuskan mana yang dipakai TIDAK ADA di export (bagian dari 137 When
// rule yang hilang, `R-16`), sehingga urutannya tidak dapat dibaca dari sumber. Kalau
// keduanya selalu berlaku bersamaan, dropdown Kanwil tidak pernah dapat menampilkan
// apa pun di luar cabang petugas sendiri — dan dropdown yang tidak dapat mengubah apa pun
// tidak akan dibuat.
//
// Yang diberlakukan di sini karena itu: **cabang petugas adalah batas bawaan, dan memilih
// kanwil menggantikannya.** Ini ANDAIAN, ditandai sebagai andaian, dan ditujukan ke Work
// Owner untuk dikonfirmasi — bukan diam-diam dianggap fakta.
func (s *Service) buildFilter(
	caller inboxlaporanklaim.Caller,
	query Query,
) (inboxlaporanklaim.Filter, error) {
	category, known := inboxlaporanklaim.FindCategory(string(query.Category))
	if !known {
		return inboxlaporanklaim.Filter{}, inboxlaporanklaim.ErrUnknownCategory
	}

	line, known := inboxlaporanklaim.FindBusinessLine(string(query.BusinessLine))
	if !known {
		return inboxlaporanklaim.Filter{}, inboxlaporanklaim.ErrUnknownBusinessLine
	}

	clean := caller.Clean()
	filter := inboxlaporanklaim.Filter{
		Category:     category,
		RegionCode:   query.RegionCode,
		BranchCode:   clean.BranchCode,
		BusinessLine: line,
		Keyword:      query.Keyword,
		Operator:     clean.Login,
	}

	// Kanwil yang dipilih menggantikan batas cabang. Lihat catatan di atas.
	if filter.Clean().RegionCode != "" {
		filter.BranchCode = ""
	}

	return filter.Clean(), nil
}
