package usecase

import (
	"context"
	"fmt"

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

	// BranchScope menyebut cabang yang membatasi daftar ini.
	//
	// Kosong hanya punya SATU arti: pengguna memilih kanwil, dan pilihan itu
	// menggantikan batas cabangnya. Ia tidak pernah berarti "batasnya hilang" — daftar
	// yang cabangnya tidak dapat ditentukan tidak dikembalikan sama sekali, melainkan
	// ditolak dengan ErrBranchUnknown.
	BranchScope string
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

	branch, err := s.requireBranch(ctx, caller)
	if err != nil {
		return ListResult{}, err
	}

	filter, err := s.buildFilter(caller, branch, query)
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

	return ListResult{
		Page:        page,
		Summary:     summary,
		BranchScope: filter.BranchCode,
	}, nil
}

// requireBranch menerjemahkan login petugas menjadi kode cabang klaimnya, dan MENOLAK
// permintaan bila cabang itu tidak dapat ditentukan.
//
// # Kenapa menolak, bukan menampilkan seluruh cabang
//
// Work Owner menetapkan 2026-09-22: petugas yang cabangnya tidak terbaca tidak boleh
// melihat seluruh cabang. Cabang karena itu bukan kenyamanan penyaring melainkan **batas
// data** — dan batas data yang tidak dapat ditentukan berarti permintaannya tidak dapat
// dilayani, bukan berarti batasnya gugur.
//
// # Kenapa menolak, bukan mengembalikan daftar kosong
//
// Sistem lama menghasilkan daftar kosong dalam keadaan ini, tetapi kekosongan itu **tidak
// terbedakan dari "tidak ada pekerjaan hari ini"**. Ketidakterbedaan itulah yang membuat
// cacat penyaring cabang bertahan tanpa seorang pun melaporkannya. Penolakan yang
// menyebutkan sebabnya menutup akses yang sama, dan sekaligus mengatakan apa yang harus
// dibetulkan.
//
// # Kenapa penjagaannya di sini, bukan di setiap pemanggil
//
// Supaya tidak ada jalan masuk yang terlewat. List dan Create memanggilnya, dan Ekspor
// menempuh List — satu tempat, tiga permukaan.
func (s *Service) requireBranch(
	ctx context.Context,
	caller inboxlaporanklaim.Caller,
) (string, error) {
	login := caller.Clean().Login
	if login == "" {
		return "", inboxlaporanklaim.ErrCallerUnknown
	}
	if s.branchResolver == nil {
		// Rakitan yang tidak memasang penerjemah sama sekali. Ini cacat pemrograman di
		// cmd, bukan keadaan data — dan ia harus terlihat, bukan diam-diam membuka batas.
		return "", inboxlaporanklaim.ErrBranchUnreadable
	}

	code, resolved, err := s.branchResolver.Resolve(ctx, login)
	switch {
	case err != nil:
		// Sebab aslinya dibungkus, bukan dibuang: yang dikirim ke peramban hanya pesan
		// umum, tetapi yang masuk log harus menyebut DB link atau tabel mana yang gagal.
		return "", fmt.Errorf("%w: %v", inboxlaporanklaim.ErrBranchUnreadable, err)
	case !resolved:
		return "", inboxlaporanklaim.ErrBranchUnknown
	}
	return code, nil
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

	// Cabang diselesaikan dengan cara yang SAMA seperti saat menyaring daftar — dan
	// ditolak dengan syarat yang sama pula.
	//
	// # Kenapa pembuatan ikut ditolak, padahal Pega membolehkannya
	//
	// `CreateNewCaseRCV` memang tidak memeriksa hasil `GetIDCabang`, dan versi
	// sebelumnya meniru itu dengan benar. Yang berubah adalah akibatnya: sejak cabang
	// menjadi batas data yang mengikat (Work Owner, 2026-09-22), berkas yang lahir tanpa
	// cabang **tidak akan pernah terlihat siapa pun** — pembuatnya sendiri tidak dapat
	// membuka daftarnya, dan petugas cabang mana pun tersaring darinya.
	//
	// Membolehkannya berarti menerbitkan baris yang dijamin tidak dapat dikerjakan. Itu
	// lebih buruk daripada menolak, dan penolakannya menyebutkan sebab yang sama dengan
	// yang menutup daftarnya.
	//
	// Ini PERLUASAN atas keputusan Work Owner yang berbunyi tentang "melihat", bukan
	// "membuat". Ditulis terbuka di sini supaya dapat dikoreksi, bukan disisipkan diam-
	// diam.
	branch, err := s.requireBranch(ctx, caller)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}

	now := s.clock.Now()

	// Nomornya TIDAK diisi di sini: ia diterbitkan repo di dalam operasi yang sama
	// dengan penyisipannya. Lihat catatan pada seam Repo.Insert.
	return repo.Insert(ctx, inboxlaporanklaim.ClaimReport{
		BranchCode: branch,

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

// Save menyimpan isian form Input Receive Document ke atas berkas yang sudah ada.
//
// # Urutannya, dan kenapa begitu
//
//  1. Berkasnya DIBACA lebih dulu. Tanpa itu, penyimpanan terhadap berkas yang tidak ada
//     akan terbaca sebagai "nol baris diubah" — dan nol baris punya dua sebab yang
//     berbeda jauh: berkasnya tidak ada, atau berkasnya milik Pega.
//  2. Asalnya diperiksa. Berkas warisan ditolak di sini, sebelum satu pun pemeriksaan
//     isian dijalankan: menolak setelah memeriksa isian akan memberi pengguna daftar
//     galat isian pada form yang sebenarnya memang tidak dapat disimpan sama sekali.
//  3. Isiannya dibersihkan lalu diperiksa — SELURUH pelanggaran sekaligus.
//  4. Baru ditulis, dengan jejak siapa dan kapan.
func (s *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	caller inboxlaporanklaim.Caller,
	detail inboxlaporanklaim.Detail,
) (inboxlaporanklaim.ClaimReport, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}

	clean := caller.Clean()
	if clean.Login == "" {
		return inboxlaporanklaim.ClaimReport{}, inboxlaporanklaim.ErrCallerUnknown
	}

	existing, err := repo.Get(ctx, id)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}
	if existing.Origin == inboxlaporanklaim.OriginLegacy {
		return inboxlaporanklaim.ClaimReport{}, inboxlaporanklaim.ErrReadOnlyOrigin
	}

	cleanDetail := detail.Clean()
	if err := cleanDetail.Check(); err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}

	saved := cleanDetail.Apply(existing)
	saved.UpdatedBy = clean.Login
	saved.UpdatedAt = s.clock.Now()

	if err := repo.Update(ctx, saved); err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}
	return saved, nil
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
	branchCode string,
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
		BranchCode:   branchCode,
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
