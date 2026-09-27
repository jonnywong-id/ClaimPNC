// Package usecase mengorkestrasi perkara master tipe sparepart.
//
// Ia yang mengetahui urutan langkah; aturan isian ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/mastertipesparepart"
)

// Service adalah pintu masuk seluruh perkara master tipe sparepart.
type Service struct {
	repoSelector mastertipesparepart.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastertipesparepart.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// TANPA Clock, sama seperti Master Kategori Sparepart dan berbeda dari Master Sparepart:
// `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak punya satu pun kolom waktu, sehingga tidak ada
// yang perlu distempel. Menerimanya "untuk jaga-jaga" berarti menerima bahan yang tidak
// pernah dipakai, dan itu menyesatkan pembaca berikutnya.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastertipesparepart/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field, dan seperti Master Kategori Sparepart ia **TIDAK TERSIMPAN**:
// `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak punya kolom pencatat pelaku sama sekali.
//
// Ia tetap diterima karena dipakai LOG — peristiwa yang mengubah tabel acuan yang menyuapi
// Master Sparepart layak terbaca di log beserta pelakunya, terlebih karena basis datanya
// sendiri tidak menyimpannya.
type Actor struct {
	Login string
}

// OptionSet adalah daftar pilihan yang dibutuhkan form.
//
// Satu daftar saja — Kategori. Ia tetap dibungkus struct alih-alih dikembalikan sebagai
// slice telanjang supaya penambahan daftar kedua kelak tidak mengubah tanda tangan
// method-nya, dan supaya Truncated punya tempat.
type OptionSet struct {
	Category []mastertipesparepart.Category

	// Truncated menyatakan daftarnya terpotong pada MaxLookupRows.
	//
	// Tanpa penanda ini, pemotongan terjadi DIAM-DIAM: pengguna membuka dropdown, tidak
	// menemukan kategori yang dicarinya, dan menyimpulkan kategorinya belum dibuat. Ia
	// keadaan yang sangat tidak mungkin pada tabel penggolongan berorde puluhan baris —
	// dan justru karena tidak mungkin, ketiadaannya tidak akan pernah diperiksa siapa pun
	// bila tidak dilaporkan.
	Truncated bool
}

// List mengembalikan baris satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status mastertipesparepart.ApprovalStatus,
	keyword string,
) ([]mastertipesparepart.PartType, error) {
	if !status.Known() {
		return nil, mastertipesparepart.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, mastertipesparepart.Filter{
		Status:  status,
		Keyword: strings.TrimSpace(keyword),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
func (l *Service) Get(
	ctx context.Context,
	portalAlias, id string,
) (mastertipesparepart.PartType, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Choices mengembalikan daftar pilihan untuk dropdown Kategori.
//
// Namanya Choices dan bukan Options supaya tidak bertabrakan dengan struct Options yang
// sudah menjadi bahan pembentuk Service di paket ini.
func (l *Service) Choices(ctx context.Context, portalAlias string) (OptionSet, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return OptionSet{}, err
	}

	category, err := store.ListCategories(ctx)
	if err != nil {
		return OptionSet{}, err
	}
	return OptionSet{
		Category:  category,
		Truncated: len(category) >= mastertipesparepart.MaxLookupRows,
	}, nil
}

// Create menyisipkan satu tipe baru.
//
// # Urutannya mengikuti sistem lama
//
//	ValidateMasterTipeSparepart        tolak bila PART_SECTION_NAME ganda
//	InsertMasterSparepartType_sql      terbitkan ID max+1 lalu sisipkan
//	UpdateTypeSparepart_act2           APPROVAL := "0"
//
// Satu perbedaan urutan yang disengaja: penerbitan ID DAN pemeriksaan nama berada di dalam
// Repo.Insert, bukan sebagai dua langkah terpisah sebelumnya. Alasannya ada pada doc
// comment mastertipesparepart.IDSource — memecahnya membuka kembali balapan yang justru
// sedang ditutup.
//
// # Satu langkah yang DITAMBAHKAN terhadap sistem lama
//
// **Kategori yang dipilih diperiksa keberadaannya.** Layar lama hanya menawarkan kategori
// dari daftarnya sendiri dan tidak pernah memeriksa ulang saat menyimpan — cukup selama
// satu-satunya jalan masuk adalah layar itu. API dapat ditembak tanpa melewati layar, dan
// tipe yang menunjuk kategori yang tidak ada akan tampil tanpa nama kategori di setiap grid
// yang menampilkannya. Lihat resolveCategory.
//
// # Satu hal yang TIDAK dibawa dari sistem lama, dan itu disengaja
//
// **Surel pemberitahuan.** `Activity/UpdateTypeSparepart_act2` menyusun badan surel lewat
// `Property-Set-HTML` lalu memanggil `SendEmailNotification` — activity yang **tidak ada di
// export** (`R-07`, dipanggil 15 activity). Penerimanya pun tidak dapat dibaca. `D-67`
// melarang akun pribadi dibawa apa adanya, dan seam Notifier (`S-3`) belum ada. Peristiwanya
// dicatat di log supaya ketiadaannya terlihat, bukan tersamar — perlakuan yang sama dipakai
// Master Kategori Sparepart dan Master Sparepart.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input mastertipesparepart.Input,
	by Actor,
	logger *slog.Logger,
) (mastertipesparepart.PartType, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastertipesparepart.PartType{}, err
	}

	category, err := resolveCategory(ctx, store, clean.CategoryID)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}

	saved, err := store.Insert(ctx, mastertipesparepart.PartType{
		Name:       clean.Name,
		CategoryID: clean.CategoryID,
		Status:     mastertipesparepart.StatusPending,
	})
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}
	if strings.TrimSpace(saved.ID) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan diam-diam:
		// baris tanpa kunci tidak dapat dibuka, disunting, maupun disetujui — dan pada modul
		// ini ia juga tidak dapat ditunjuk oleh sparepart mana pun.
		return mastertipesparepart.PartType{}, errors.New(
			"mastertipesparepart/usecase: ID tipe yang diterbitkan kosong")
	}

	// Nama kategori diisi dari daftar yang SUDAH dibaca untuk memeriksa keberadaannya, bukan
	// lewat pembacaan kedua. Tanpa ini, layar menerima baris baru dengan kolom Kategori
	// kosong dan harus memuat ulang daftarnya hanya untuk mengisinya.
	saved.CategoryName = category.Name

	notifyOmitted(logger, portalAlias, saved, by, "tipe sparepart baru diajukan")
	return saved, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
//
// Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
// `Activity/UpdateTypeSparepart_act2` menetapkan `InputKategori.NO_ACCOUNT := "0"` tanpa
// syarat apa pun, dan properti itulah yang dipetakan ke kolom APPROVAL oleh
// `UpdateMasterSparepartType_sql2`.
//
// # Akibatnya menyentuh layar LAIN, dan itu harus disadari
//
// Tipe yang sudah disetujui lalu disunting kembali menunggu — dan selama menunggu ia
// **hilang dari dropdown Tipe pada layar Master Sparepart**, yang menyaring
// `APPROVAL = '1'`. Sparepart yang sudah menunjuk tipe itu tetap menyimpan ID-nya, tetapi
// namanya tidak lagi dapat ditampilkan.
//
// Itu perilaku sistem lama apa adanya (`P-5`), bukan pilihan modul ini. Ia dicatat di sini
// karena satu-satunya cara mengetahuinya adalah membaca dua modul sekaligus.
//
// # Kategori BOLEH dipindahkan
//
// `UpdateMasterSparepartType_sql2` menulis PART_CATEGORY_ID pada setiap penyimpanan, jadi
// memindahkan sebuah tipe ke kategori lain memang didukung sistem lama. Sparepart yang
// sudah menunjuk tipe itu ikut berpindah golongan tanpa satu pun pemberitahuan — perilaku
// yang ditiru apa adanya, dan yang dinyatakan di muka pada form supaya tidak ditemukan
// setelah tombol ditekan.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input mastertipesparepart.Input,
	by Actor,
	logger *slog.Logger,
) (mastertipesparepart.PartType, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return mastertipesparepart.PartType{}, mastertipesparepart.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastertipesparepart.PartType{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}

	category, err := resolveCategory(ctx, store, clean.CategoryID)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}

	if err := ensureNameFree(ctx, store, key, clean.Name); err != nil {
		return mastertipesparepart.PartType{}, err
	}

	updated := mastertipesparepart.PartType{
		ID:           stored.ID,
		Name:         clean.Name,
		CategoryID:   clean.CategoryID,
		CategoryName: category.Name,
		Status:       mastertipesparepart.StatusPending,
	}
	if err := store.Update(ctx, updated); err != nil {
		return mastertipesparepart.PartType{}, err
	}

	notifyOmitted(logger, portalAlias, updated, by, "tipe sparepart diubah")
	return updated, nil
}

// resolveCategory memastikan kategori yang dipilih ada DAN sudah disetujui.
//
// # Kenapa daftar yang dibaca, bukan satu baris
//
// Karena daftar itu memang satu-satunya bentuk yang tersedia: seam LookupRepo hanya punya
// ListCategories, dan menambah "ambil satu kategori" ke sana berarti membuka jalan kedua ke
// tabel milik modul lain — dua kueri yang dapat berbeda penyaringnya tanpa ada yang
// memberi tahu.
//
// Harganya terjangkau: daftarnya berorde puluhan baris, dibatasi MaxLookupRows, dan jalur
// ini hanya dilewati saat menyimpan.
//
// # Yang DITOLAK di sini
//
// Kategori yang tidak ada, DAN kategori yang belum disetujui — keduanya sama saja dari
// sudut pandang daftar ini, karena ListCategories memang hanya mengembalikan yang
// disetujui. Pengguna yang memilih dari dropdown tidak akan pernah menemuinya; yang
// menemuinya adalah permintaan yang menembak API langsung, dan permintaan yang formnya
// terbuka lama sementara kategorinya ditolak petugas lain di tab sebelah.
//
// Pesannya menyebut kemungkinan kedua itu, karena itulah satu-satunya yang dapat terjadi
// pada pengguna yang tidak berbuat salah apa pun.
func resolveCategory(
	ctx context.Context,
	store mastertipesparepart.Store,
	id string,
) (mastertipesparepart.Category, error) {
	list, err := store.ListCategories(ctx)
	if err != nil {
		return mastertipesparepart.Category{}, err
	}

	wanted := strings.TrimSpace(id)
	for _, c := range list {
		if strings.TrimSpace(c.ID) == wanted {
			return c, nil
		}
	}
	return mastertipesparepart.Category{}, mastertipesparepart.ErrCategoryNotFound
}

// ensureNameFree menolak nama yang sudah dipakai baris LAIN.
//
// Baris yang sedang disunting dikecualikan: menyimpan tanpa mengubah namanya sama sekali
// adalah hal yang wajar dilakukan pengguna — misalnya untuk memindahkan tipe ke kategori
// lain, atau untuk mengembalikan baris yang ditolak ke antrean — dan menolaknya karena
// "namanya sudah dipakai" oleh dirinya sendiri adalah kegagalan yang tidak dapat
// dijelaskan kepada siapa pun.
//
// Perbandingannya case-insensitive, meniru `upper(...)` pada
// `RDB List/ValidationSparepartType-SQL.xml`, dan TIDAK menyaring kategori — nama tipe unik
// di seluruh tabel, bukan per kategori. Lihat mastertipesparepart.ErrNameTaken.
//
// Perbandingan kuncinya memakai nilai yang sudah dipangkas di kedua sisi — ID pada basis
// data dapat berisi spasi ujung, dan tanpa pemangkasan baris yang sedang disunting tidak
// akan dikenali sebagai dirinya sendiri.
func ensureNameFree(
	ctx context.Context,
	store mastertipesparepart.Store,
	key, name string,
) error {
	found, err := store.FindByName(ctx, name)
	switch {
	case errors.Is(err, mastertipesparepart.ErrNotFound):
		return nil
	case err != nil:
		return err
	case strings.TrimSpace(found.ID) == strings.TrimSpace(key):
		return nil
	default:
		return mastertipesparepart.ErrNameTaken
	}
}

// Decide menetapkan status persetujuan sejumlah baris sekaligus.
//
// # Kenapa borongan, dan bukan satu per satu
//
// Karena di sistem lama memang borongan.
// `Section/ApprovalMasterTipeSparepartHE-Section.xml` menggambar grid bercentang dengan
// tombol **Approve** dan **Reject** — bentuk yang sama persis dengan layar persetujuan
// kategori, bengkel, panel, dan sparepart.
//
// # Yang MEMBEDAKANNYA dari bengkel, panel, dan sparepart
//
// Tipe **tidak ikut** `Activity/SetApprovalAllMaster`. Activity itu hanya melayani
// `M_BENGKEL_HE`, `M_PANEL_HE`, dan `M_SPAREPART_HE`. Tipe punya jalurnya sendiri lewat
// `Activity/UpdateTipeSparepart_act`, dan rule SQL yang benar-benar menjalankannya **tidak
// ada di export** (`R-16`); bentuknya direkonstruksi — lihat
// mastertipesparepart.Repo.SetStatus.
//
// # TANPA alasan penolakan, dan TANPA pencatat pelaku
//
// Tabelnya hanya punya empat kolom. Tidak ada tempat menuliskan mengapa sebuah tipe
// ditolak, dan tidak ada tempat menuliskan siapa yang menolaknya. Layar baru karena itu
// TIDAK menggambar isian catatan sama sekali — menggambar isian yang diam-diam membuang
// isinya lebih buruk daripada tidak menggambarnya.
//
// Keputusannya tetap dicatat di LOG beserta pelakunya, supaya jejaknya ada di suatu tempat
// selama kolomnya belum ada. Itu bukan pengganti jejak audit `S-5`, dan tidak diklaim
// demikian.
func (l *Service) Decide(
	ctx context.Context,
	portalAlias string,
	id []string,
	status mastertipesparepart.ApprovalStatus,
	by Actor,
	logger *slog.Logger,
) (int, error) {
	if !status.Known() {
		return 0, mastertipesparepart.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return 0, err
	}

	wanted := make([]string, 0, len(id))
	seen := make(map[string]bool, len(id))
	for _, one := range id {
		clean := strings.TrimSpace(one)
		// Kunci ganda dibuang: layar dapat mengirim baris yang sama dua kali bila daftarnya
		// dimuat ulang saat centang masih terpasang, dan jumlah baris berubah yang dilaporkan
		// harus mencerminkan baris, bukan centang.
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		wanted = append(wanted, clean)
	}
	if len(wanted) == 0 {
		return 0, mastertipesparepart.OneViolation("id_tipe_sparepart",
			"Pilih dulu tipe sparepart yang akan diputuskan.")
	}

	changed, err := store.SetStatus(ctx, wanted, status)
	if err != nil {
		return 0, err
	}

	if changed != len(wanted) && logger != nil {
		// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau sudah
		// berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang sudah basi,
		// dan itu layak terbaca.
		logger.Warn("keputusan master tipe sparepart tidak menyentuh seluruh baris yang dipilih",
			slog.String("portal", portalAlias),
			slog.Int("dipilih", len(wanted)),
			slog.Int("berubah", changed),
			slog.String("oleh", by.Login))
	}

	notifyDecisionOmitted(logger, portalAlias, changed, status, by)
	return changed, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("mastertipesparepart/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// notifyOmitted mencatat surel pemberitahuan yang TIDAK dikirim.
//
// Ia sengaja bertingkat Info dan bukan Warn: ketiadaannya adalah keadaan yang DIKETAHUI dan
// diputuskan (`D-67`, seam `S-3` belum ada, dan `SendEmailNotification` sendiri tidak ada
// di export — `R-07`), bukan gangguan yang perlu ditindaklanjuti seseorang malam ini. Yang
// dicatat adalah peristiwanya, supaya saat Notifier tiba, penerimanya dapat dipasang tanpa
// menebak peristiwa mana saja yang seharusnya memicunya.
func notifyOmitted(
	logger *slog.Logger,
	portalAlias string,
	t mastertipesparepart.PartType,
	by Actor,
	event string,
) {
	if logger == nil {
		return
	}
	logger.Info("pemberitahuan master tipe sparepart tidak dikirim; seam Notifier belum ada",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_tipe", t.ID),
		slog.String("nama_tipe", t.Name),
		slog.String("id_kategori", t.CategoryID),
		slog.String("oleh", by.Login))
}

// notifyDecisionOmitted mencatat pemberitahuan keputusan yang TIDAK dikirim.
//
// Terpisah dari notifyOmitted karena isinya berbeda: yang penting pada keputusan adalah
// BERAPA baris dan MENJADI APA, bukan baris mana satu per satu. Ia juga satu-satunya tempat
// pelaku keputusan terekam — tabelnya tidak punya kolom untuk itu.
func notifyDecisionOmitted(
	logger *slog.Logger,
	portalAlias string,
	changed int,
	status mastertipesparepart.ApprovalStatus,
	by Actor,
) {
	if logger == nil {
		return
	}
	logger.Info("keputusan master tipe sparepart tersimpan; pemberitahuan tidak dikirim",
		slog.String("portal", portalAlias),
		slog.Int("berubah", changed),
		slog.String("status", string(status)),
		slog.String("status_label", status.Label()),
		slog.String("oleh", by.Login))
}
