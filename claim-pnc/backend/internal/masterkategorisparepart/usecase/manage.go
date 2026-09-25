// Package usecase mengorkestrasi perkara master kategori sparepart.
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

	"claim-pnc/internal/masterkategorisparepart"
)

// Service adalah pintu masuk seluruh perkara master kategori sparepart.
type Service struct {
	repoSelector masterkategorisparepart.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterkategorisparepart.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// TANPA Clock, berbeda dari Master Sparepart dan Master Penolakan Klaim:
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya satu pun kolom waktu, sehingga tidak
// ada yang perlu distempel. Menerimanya "untuk jaga-jaga" berarti menerima bahan yang
// tidak pernah dipakai, dan itu menyesatkan pembaca berikutnya.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterkategorisparepart/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field, dan berbeda dari Master Sparepart ia **TIDAK TERSIMPAN**:
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya kolom pencatat pelaku sama sekali.
//
// Ia tetap diterima karena dipakai LOG — peristiwa yang mengubah tabel acuan yang
// menyuapi Master Sparepart layak terbaca di log beserta pelakunya, terlebih karena basis
// datanya sendiri tidak menyimpannya. Perlakuan yang sama dipakai Master Panel.
type Actor struct {
	Login string
}

// List mengembalikan baris satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status masterkategorisparepart.ApprovalStatus,
	keyword string,
) ([]masterkategorisparepart.PartCategory, error) {
	if !status.Known() {
		return nil, masterkategorisparepart.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, masterkategorisparepart.Filter{
		Status:  status,
		Keyword: strings.TrimSpace(keyword),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
func (l *Service) Get(
	ctx context.Context,
	portalAlias, id string,
) (masterkategorisparepart.PartCategory, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Create menyisipkan satu kategori baru.
//
// # Urutannya mengikuti sistem lama
//
//	ValidateMasterKategoriSparepart       tolak bila PART_CATEGORY_NAME ganda
//	InsertMasterSparepartCategory_sql     terbitkan ID max+1 lalu sisipkan
//	UpdateKategoriSparepart_act2          APPROVAL := "0"
//
// Satu perbedaan urutan yang disengaja: penerbitan ID DAN pemeriksaan nama berada di
// dalam Repo.Insert, bukan sebagai dua langkah terpisah sebelumnya. Alasannya ada pada doc
// comment masterkategorisparepart.IDSource — memecahnya membuka kembali balapan yang
// justru sedang ditutup.
//
// # Satu hal yang TIDAK dibawa dari sistem lama, dan itu disengaja
//
// **Surel pemberitahuan.** `Activity/UpdateKategoriSparepart_act2` memanggil
// `SendAlertApprovalMaster` dan `SendEmailNotification` dengan
// `TempMaster.NamaDokumen := "KATEGORI SPAREPART"` dan penerima yang **di-hardcode sebagai
// satu alamat surel pribadi** (`local.send`). `D-67` melarang akun pribadi dibawa apa
// adanya, dan seam Notifier (`S-3`) belum ada. Peristiwanya dicatat di log supaya
// ketiadaannya terlihat, bukan tersamar — perlakuan yang sama dipakai Master Sparepart.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input masterkategorisparepart.Input,
	by Actor,
	logger *slog.Logger,
) (masterkategorisparepart.PartCategory, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	saved, err := store.Insert(ctx, masterkategorisparepart.PartCategory{
		Name:   clean.Name,
		Status: masterkategorisparepart.StatusPending,
	})
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}
	if strings.TrimSpace(saved.ID) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan
		// diam-diam: baris tanpa kunci tidak dapat dibuka, disunting, maupun disetujui —
		// dan pada modul ini ia juga tidak dapat ditunjuk oleh sparepart mana pun.
		return masterkategorisparepart.PartCategory{}, errors.New(
			"masterkategorisparepart/usecase: ID kategori yang diterbitkan kosong")
	}

	notifyOmitted(logger, portalAlias, saved, by, "kategori sparepart baru diajukan")
	return saved, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
//
// Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
// `Activity/UpdateKategoriSparepart_act2` menetapkan `InputKategori.LOGIN_APLIKASI := "0"`
// tanpa syarat apa pun, dan properti itulah yang dipetakan ke kolom APPROVAL oleh
// `UpdateMasterSparepartCategory_sql2`.
//
// # Akibatnya menyentuh layar LAIN, dan itu harus disadari
//
// Kategori yang sudah disetujui lalu disunting kembali menunggu — dan selama menunggu ia
// **hilang dari dropdown Kategori pada layar Master Sparepart**, yang menyaring
// `APPROVAL = '1'`. Sparepart yang sudah menunjuk kategori itu tetap menyimpan ID-nya,
// tetapi namanya tidak lagi dapat ditampilkan.
//
// Itu perilaku sistem lama apa adanya (`P-5`), bukan pilihan modul ini. Ia dicatat di sini
// karena satu-satunya cara mengetahuinya adalah membaca dua modul sekaligus.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input masterkategorisparepart.Input,
	by Actor,
	logger *slog.Logger,
) (masterkategorisparepart.PartCategory, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return masterkategorisparepart.PartCategory{}, masterkategorisparepart.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	if err := ensureNameFree(ctx, store, key, clean.Name); err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	updated := masterkategorisparepart.PartCategory{
		ID:     stored.ID,
		Name:   clean.Name,
		Status: masterkategorisparepart.StatusPending,
	}
	if err := store.Update(ctx, updated); err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	notifyOmitted(logger, portalAlias, updated, by, "kategori sparepart diubah")
	return updated, nil
}

// ensureNameFree menolak nama yang sudah dipakai baris LAIN.
//
// Baris yang sedang disunting dikecualikan: menyimpan tanpa mengubah namanya sama sekali
// adalah hal yang wajar dilakukan pengguna — misalnya untuk mengembalikan baris yang
// ditolak ke antrean — dan menolaknya karena "namanya sudah dipakai" oleh dirinya sendiri
// adalah kegagalan yang tidak dapat dijelaskan kepada siapa pun.
//
// Perbandingannya case-insensitive, meniru `upper(...)` pada
// `RDB List/ValidationSparepartCat-SQL.xml`. Perbandingan kuncinya memakai nilai yang
// sudah dipangkas di kedua sisi — ID pada basis data dapat berisi spasi ujung, dan tanpa
// pemangkasan baris yang sedang disunting tidak akan dikenali sebagai dirinya sendiri.
func ensureNameFree(
	ctx context.Context,
	store masterkategorisparepart.Store,
	key, name string,
) error {
	found, err := store.FindByName(ctx, name)
	switch {
	case errors.Is(err, masterkategorisparepart.ErrNotFound):
		return nil
	case err != nil:
		return err
	case strings.TrimSpace(found.ID) == strings.TrimSpace(key):
		return nil
	default:
		return masterkategorisparepart.ErrNameTaken
	}
}

// Decide menetapkan status persetujuan sejumlah baris sekaligus.
//
// # Kenapa borongan, dan bukan satu per satu
//
// Karena di sistem lama memang borongan.
// `Section/ApprovalMasterKategoriSparepartHE-Section.xml` menggambar grid bercentang
// dengan tombol **Approve**, **Reject**, dan **DETAILS** — bentuk yang sama persis dengan
// layar persetujuan bengkel, panel, dan sparepart.
//
// # Yang MEMBEDAKANNYA dari ketiga master itu
//
// Kategori **tidak ikut** `Activity/SetApprovalAllMaster`. Activity itu hanya melayani
// `M_BENGKEL_HE`, `M_PANEL_HE`, dan `M_SPAREPART_HE`. Kategori punya jalurnya sendiri lewat
// `Activity/UpdateKategoriSparepart_act`, dan rule SQL yang benar-benar menjalankannya
// **tidak ada di export** (`R-16`); bentuknya direkonstruksi — lihat
// masterkategorisparepart.Repo.SetStatus.
//
// # TANPA alasan penolakan, dan TANPA pencatat pelaku
//
// Tabelnya hanya punya tiga kolom. Tidak ada tempat menuliskan mengapa sebuah kategori
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
	status masterkategorisparepart.ApprovalStatus,
	by Actor,
	logger *slog.Logger,
) (int, error) {
	if !status.Known() {
		return 0, masterkategorisparepart.ErrUnknownStatus
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
		// dimuat ulang saat centang masih terpasang, dan jumlah baris berubah yang
		// dilaporkan harus mencerminkan baris, bukan centang.
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		wanted = append(wanted, clean)
	}
	if len(wanted) == 0 {
		return 0, masterkategorisparepart.OneViolation("id_kategori_sparepart",
			"Pilih dulu kategori sparepart yang akan diputuskan.")
	}

	changed, err := store.SetStatus(ctx, wanted, status)
	if err != nil {
		return 0, err
	}

	if changed != len(wanted) && logger != nil {
		// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau
		// sudah berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang
		// sudah basi, dan itu layak terbaca.
		logger.Warn("keputusan master kategori sparepart tidak menyentuh seluruh baris yang dipilih",
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
		return fmt.Errorf("masterkategorisparepart/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// notifyOmitted mencatat surel pemberitahuan yang TIDAK dikirim.
//
// Ia sengaja bertingkat Info dan bukan Warn: ketiadaannya adalah keadaan yang DIKETAHUI dan
// diputuskan (`D-67`, seam `S-3` belum ada), bukan gangguan yang perlu ditindaklanjuti
// seseorang malam ini. Yang dicatat adalah peristiwanya, supaya saat Notifier tiba,
// penerimanya dapat dipasang tanpa menebak peristiwa mana saja yang seharusnya memicunya.
//
// Alamat surel yang di-hardcode di `Activity/UpdateKategoriSparepart_act2` TIDAK ditulis
// ke log maupun ke berkas mana pun (`D-69`).
func notifyOmitted(
	logger *slog.Logger,
	portalAlias string,
	c masterkategorisparepart.PartCategory,
	by Actor,
	event string,
) {
	if logger == nil {
		return
	}
	logger.Info("pemberitahuan master kategori sparepart tidak dikirim; seam Notifier belum ada",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_kategori", c.ID),
		slog.String("nama_kategori", c.Name),
		slog.String("oleh", by.Login))
}

// notifyDecisionOmitted mencatat pemberitahuan keputusan yang TIDAK dikirim.
//
// Terpisah dari notifyOmitted karena isinya berbeda: yang penting pada keputusan adalah
// BERAPA baris dan MENJADI APA, bukan baris mana satu per satu. Ia juga satu-satunya
// tempat pelaku keputusan terekam — tabelnya tidak punya kolom untuk itu.
func notifyDecisionOmitted(
	logger *slog.Logger,
	portalAlias string,
	changed int,
	status masterkategorisparepart.ApprovalStatus,
	by Actor,
) {
	if logger == nil {
		return
	}
	logger.Info("keputusan master kategori sparepart tersimpan; pemberitahuan tidak dikirim",
		slog.String("portal", portalAlias),
		slog.Int("berubah", changed),
		slog.String("status", string(status)),
		slog.String("status_label", status.Label()),
		slog.String("oleh", by.Login))
}
