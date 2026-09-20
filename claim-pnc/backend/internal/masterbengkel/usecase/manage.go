// Package usecase mengorkestrasi perkara master bengkel.
//
// Ia yang mengetahui urutan langkah; aturan isian ada di paket domain, dan cara
// membacanya dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/masterbengkel"
)

// Service adalah pintu masuk seluruh perkara master bengkel.
type Service struct {
	repoSelector masterbengkel.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterbengkel.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterbengkel/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field saja, dan itu disengaja: `POOLDATA.BENGKEL_HE` **tidak punya satu pun kolom
// pencatat pelaku maupun waktu**. Identitas pemanggil karena itu tidak pernah tersimpan
// di tabelnya — ia hanya dipakai untuk log.
//
// Itu keterbatasan tabelnya, bukan pilihan. Jejak audit perubahan master (`D-28`, modul
// `S-5`) belum dapat disandarkan padanya, dan menambah kolom menuntut persetujuan Work
// Owner serta pelaksanaan DBA (`D-63`).
type Actor struct {
	Login string
}

// List mengembalikan baris master bengkel satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status masterbengkel.ApprovalStatus,
	keyword string,
) ([]masterbengkel.Workshop, error) {
	if !status.Known() {
		return nil, masterbengkel.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, masterbengkel.Filter{
		Status:  status,
		Keyword: strings.TrimSpace(keyword),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
func (l *Service) Get(ctx context.Context, portalAlias, id string) (masterbengkel.Workshop, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterbengkel.Workshop{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Create menyisipkan satu baris master bengkel baru.
//
// # Urutannya mengikuti sistem lama
//
//	ValidateMasterBengkel         tolak bila NAMA_BENGKEL sudah dipakai
//	ValidationLoginBengkel_act    tolak bila LOGIN_APLIKASI sudah dipakai
//	PEGA_M_BENGKEL_HE.prc:11,19   terbitkan ID_BENGKEL
//	UpdateBengkelHE_act step 7    APPROVAL := "0"
//	UpdateBengkelHE-SQL           simpan
//
// Satu perbedaan urutan yang disengaja: kedua pemeriksaan ganda di sini berada DI DALAM
// Repo.Insert, bukan sebagai langkah terpisah sebelumnya. Alasannya ada pada doc comment
// masterbengkel.Repo.Insert — memecahnya membuka lubang balapan yang justru sedang
// dipersempit.
//
// # Dua hal yang TIDAK dibawa dari sistem lama, dan keduanya disengaja
//
//  1. **Pembuatan akun bengkel.** `ValidationLoginBengkel_act` step 9 memanggil
//     `GCNMCreateOperator` dengan kata sandi yang sama untuk setiap bengkel —
//     `Local.password := "123456"`. Sistem baru tidak punya operator Pega, dan kontrak
//     identitasnya belum ada (`F-3`, `R-14`); mereplikasinya berarti menerbitkan akun
//     dengan kata sandi yang sudah diketahui siapa pun yang pernah membaca rule itu.
//     LOGIN_APLIKASI tetap disimpan dan diperiksa keunikannya.
//  2. **Surel pemberitahuan.** `UpdateBengkelHE_act` step 18–19 mengirimnya ke satu
//     alamat yang tertanam di dalam rule — alamat pribadi seseorang, bukan mailbox
//     fungsional. `D-67` melarangnya dibawa, dan seam Notifier (`S-3`) belum ada.
//     Peristiwanya dicatat di log supaya ketiadaannya terlihat, bukan tersamar.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input masterbengkel.Input,
	by Actor,
	logger *slog.Logger,
) (masterbengkel.Workshop, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterbengkel.Workshop{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterbengkel.Workshop{}, err
	}

	id, err := store.NextID(ctx)
	if err != nil {
		return masterbengkel.Workshop{}, fmt.Errorf("masterbengkel/usecase: menerbitkan ID bengkel: %w", err)
	}
	if strings.TrimSpace(id) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan
		// diam-diam: baris tanpa kunci tidak dapat dibuka, disunting, maupun disetujui.
		return masterbengkel.Workshop{}, errors.New("masterbengkel/usecase: ID bengkel yang diterbitkan kosong")
	}

	fresh := apply(masterbengkel.Workshop{ID: strings.TrimSpace(id)}, clean)
	fresh.Status = masterbengkel.StatusPending

	if err := store.Insert(ctx, fresh); err != nil {
		return masterbengkel.Workshop{}, err
	}

	notifyOmitted(logger, portalAlias, fresh, by, "bengkel baru diajukan")
	return fresh, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
//
// Itu bukan tafsiran, melainkan langkah tersendiri di sistem lama:
// `Activity/UpdateBengkelHE_act` step 7 menetapkan `APPROVAL := "0"` tanpa syarat apa
// pun. Bengkel yang sudah disetujui lalu disunting kembali menunggu — dan itu benar
// untuk master yang menentukan diskon, pajak, dan rekening tujuan pembayaran.
//
// # Yang TIDAK ikut tersimpan
//
//	ID_BENGKEL  kunci baris, diambil dari jalur URL, bukan dari badan permintaan
//	DOKUMENID   dipertahankan dari baris yang tersimpan; unggah lampiran tidak dibawa
//	APPROVAL    bukan isian melainkan akibat — selalu StatusPending di sini
//
// DOKUMENID patut diperhatikan. Sistem lama mengisinya dari hasil `PNCSaveAttachmentToDB`
// pada setiap penyimpanan, dan penyimpanan TANPA lampiran menimpanya dengan kosong —
// lampiran yang sudah ada lenyap hanya karena barisnya disunting. Di sini ia dibaca dari
// baris yang tersimpan dan ditulis kembali apa adanya, sehingga jalur yang menghapus
// lampirannya sendiri tidak ikut dibawa.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input masterbengkel.Input,
	by Actor,
	logger *slog.Logger,
) (masterbengkel.Workshop, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterbengkel.Workshop{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterbengkel.Workshop{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return masterbengkel.Workshop{}, err
	}

	if err := ensureUnique(ctx, store, key, clean); err != nil {
		return masterbengkel.Workshop{}, err
	}

	updated := apply(stored, clean)
	updated.Status = masterbengkel.StatusPending

	if err := store.Update(ctx, updated); err != nil {
		return masterbengkel.Workshop{}, err
	}

	notifyOmitted(logger, portalAlias, updated, by, "bengkel diubah")
	return updated, nil
}

// Decide menetapkan status persetujuan sejumlah baris sekaligus.
//
// # Kenapa borongan, dan bukan satu per satu
//
// Karena di sistem lama memang borongan. `Activity/SetApprovalAllMaster` menelusuri
// baris yang dicentang (`.pySelected=="true"`) lalu menetapkan `APPROVAL := Param.approval`
// pada masing-masing. Activity yang sama melayani tiga master — bengkel, panel, dan
// sparepart — dibedakan hanya oleh `Param.TIPE2`.
//
// # Satu hal yang TIDAK dapat direplikasi, dan harus dinyatakan
//
// Rule SQL yang benar-benar menjalankan penetapannya **tidak ada di export** (`R-16`):
// `SetApprovalAllMaster` dirujuk empat berkas tetapi tidak punya berkas sendiri. Yang
// terbaca hanyalah kolom yang disentuhnya — `ID_BENGKEL`, `APPROVAL`, dan `MAIL` yang
// diisi `USER_UPDATE`.
//
// Kolom MAIL itu SENGAJA TIDAK ikut ditulis di sini. Ia kolom surel bengkel; menimpanya
// dengan identitas petugas yang menyetujui akan menghapus alamat surel bengkel — dan
// itu satu lagi kolom berarti ganda seperti dua yang sudah dicatat di paket domain.
func (l *Service) Decide(
	ctx context.Context,
	portalAlias string,
	id []string,
	status masterbengkel.ApprovalStatus,
	by Actor,
	logger *slog.Logger,
) (int, error) {
	if !status.Known() {
		return 0, masterbengkel.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return 0, err
	}

	wanted := make([]string, 0, len(id))
	seen := make(map[string]bool, len(id))
	for _, one := range id {
		clean := strings.TrimSpace(one)
		// Kunci ganda dibuang: layar dapat mengirim baris yang sama dua kali bila
		// daftarnya dimuat ulang saat centang masih terpasang, dan jumlah baris berubah
		// yang dilaporkan harus mencerminkan baris, bukan centang.
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		wanted = append(wanted, clean)
	}
	if len(wanted) == 0 {
		return 0, masterbengkel.OneViolation("id_bengkel", "Pilih dulu bengkel yang akan diputuskan.")
	}

	changed, err := store.SetStatus(ctx, wanted, status)
	if err != nil {
		return 0, err
	}

	if changed != len(wanted) && logger != nil {
		// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau
		// sudah berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang
		// sudah basi, dan itu layak terbaca.
		logger.Warn("keputusan master bengkel tidak menyentuh seluruh baris yang dipilih",
			slog.String("portal", portalAlias),
			slog.Int("dipilih", len(wanted)),
			slog.Int("berubah", changed),
			slog.String("oleh", by.Login))
	}

	notifyDecisionOmitted(logger, portalAlias, changed, status, by)
	return changed, nil
}

// ListBranches melayani dropdown Cabang pada form.
func (l *Service) ListBranches(ctx context.Context, portalAlias string) ([]masterbengkel.Branch, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.ListBranches(ctx)
}

// SearchCities melayani lookup Kota pada form.
//
// Kata kunci yang terlalu pendek dijawab daftar kosong, BUKAN galat: pengguna yang baru
// mengetik satu huruf belum melakukan kesalahan apa pun, dan pesan galat di bawah kotak
// pencarian yang sedang diketik hanya akan mengganggu.
func (l *Service) SearchCities(ctx context.Context, portalAlias, keyword string) ([]masterbengkel.City, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	clean := strings.TrimSpace(keyword)
	if len(clean) < masterbengkel.MinLookupKeyword {
		return nil, nil
	}
	return store.SearchCities(ctx, clean)
}

// ListBanks melayani dropdown Bank pada form.
func (l *Service) ListBanks(ctx context.Context, portalAlias string) ([]masterbengkel.Bank, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.ListBanks(ctx)
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterbengkel/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// apply menyalin isian yang dikirim layar ke atas baris yang sudah ada.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya kedua jalur tidak pernah berbeda soal
// kolom mana yang ikut berubah. Tiga kolom sengaja tidak disentuh — ID, DocumentID, dan
// Status — dan ketiganya diatur pemanggil.
func apply(base masterbengkel.Workshop, in masterbengkel.Input) masterbengkel.Workshop {
	base.Name = in.Name
	base.Address = in.Address
	base.Phone = in.Phone
	base.Mobile = in.Mobile
	base.Email = in.Email
	base.WorkOrderEmail = in.WorkOrderEmail

	base.BranchID = in.BranchID
	base.BranchName = in.BranchName
	base.CityID = in.CityID
	base.CityName = in.CityName

	base.PartnerStatus = in.PartnerStatus
	base.WorkshopStatus = in.WorkshopStatus
	base.StatusReason = in.StatusReason
	base.StatusDate = in.StatusDate

	base.Login = in.Login

	base.BankID = in.BankID
	base.BankName = in.BankName
	base.AccountNumber = in.AccountNumber
	base.AccountName = in.AccountName
	base.AccountID = in.AccountID

	base.TaxName = in.TaxName
	base.TaxNumber = in.TaxNumber
	base.TaxAddress = in.TaxAddress
	base.IncomeTaxType = in.IncomeTaxType

	base.ValueAddedTax = in.ValueAddedTax
	base.ServiceDiscount = in.ServiceDiscount
	base.PartDiscount = in.PartDiscount
	base.MaterialPercent = in.MaterialPercent
	base.PriceListGapPercent = in.PriceListGapPercent
	base.SLA = in.SLA

	base.SuppliedByASM = in.SuppliedByASM
	base.EClaimStatus = in.EClaimStatus
	base.AutoAcceptStatus = in.AutoAcceptStatus
	base.PaymentStatus = in.PaymentStatus
	base.AutoPaymentStatus = in.AutoPaymentStatus
	base.TeknoStatus = in.TeknoStatus
	base.OrderStatus = in.OrderStatus
	base.Supplier = in.Supplier

	return base
}

// ensureUnique menolak nama dan login yang sudah dipakai BARIS LAIN.
//
// Ia dipakai jalur simpan saja. Jalur tambah memeriksanya di dalam Repo.Insert, yang
// dapat menguncinya lebih dulu; di sini penguncian itu tidak diperlukan karena kunci
// barisnya sudah ada, dan bentrok yang muncul di antara pemeriksaan dan penyimpanan
// hanya dapat berasal dari petugas lain yang menyimpan nama yang sama pada saat yang
// sama — keadaan yang tertutup constraint unik, bukan oleh urutan langkah (R-08, D-63).
//
// # Kenapa pemeriksaannya ada di jalur simpan juga
//
// Karena di sistem lama pun ada. `Activity/ValidateMasterBengkel` dipanggil dari layar
// tanpa memandang tambah atau ubah, dan `Activity/UpdateBengkelHE_act` memanggil
// `ValidationLoginBengkel_act` pada KETIGA tabnya. Melewatkannya di sini akan
// memperbolehkan dua bengkel bernama sama — cukup dengan menyunting salah satunya.
//
// Baris yang sedang disunting dikecualikan: menyimpan tanpa mengubah namanya tidak boleh
// ditolak karena namanya sendiri sudah dipakai oleh dirinya sendiri.
func ensureUnique(
	ctx context.Context,
	store masterbengkel.Store,
	id string,
	in masterbengkel.Input,
) error {
	switch other, err := store.FindByName(ctx, in.Name); {
	case err == nil && strings.TrimSpace(other.ID) != strings.TrimSpace(id):
		return masterbengkel.OneViolation("nama_bengkel",
			"Nama tersebut telah digunakan bengkel "+other.ID+". Silakan ganti dengan nama yang lain.")
	case err != nil && !errors.Is(err, masterbengkel.ErrNotFound):
		return fmt.Errorf("masterbengkel/usecase: memeriksa nama %q: %w", in.Name, err)
	}

	if strings.TrimSpace(in.Login) == "" {
		return nil
	}

	switch other, err := store.FindByLogin(ctx, in.Login); {
	case err == nil && strings.TrimSpace(other.ID) != strings.TrimSpace(id):
		return masterbengkel.OneViolation("login_aplikasi",
			"Login aplikasi tersebut telah dipakai bengkel "+other.ID+", tolong ubah login aplikasi.")
	case err != nil && !errors.Is(err, masterbengkel.ErrNotFound):
		return fmt.Errorf("masterbengkel/usecase: memeriksa login %q: %w", in.Login, err)
	}
	return nil
}

// notifyOmitted mencatat pemberitahuan yang di sistem lama dikirim, dan di sini tidak.
//
// Ia sengaja dicatat sebagai Info dan bukan didiamkan: yang hilang adalah satu langkah
// proses yang nyata — PIC tidak lagi diberi tahu bahwa ada bengkel menunggu persetujuan
// — dan ketiadaannya harus terbaca di log alih-alih ditemukan berbulan kemudian oleh
// petugas yang bertanya kenapa antreannya menumpuk.
func notifyOmitted(logger *slog.Logger, portalAlias string, w masterbengkel.Workshop, by Actor, event string) {
	if logger == nil {
		return
	}
	logger.Info("pemberitahuan master bengkel tidak dikirim",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_bengkel", w.ID),
		slog.String("nama_bengkel", w.Name),
		slog.String("oleh", by.Login),
		slog.String("sebab", "seam Notifier (S-3) belum ada, dan penerima di rule lama berupa alamat pribadi yang D-67 larang dibawa"))
}

// notifyDecisionOmitted mencatat hal yang sama untuk jalur keputusan.
func notifyDecisionOmitted(logger *slog.Logger, portalAlias string, changed int, status masterbengkel.ApprovalStatus, by Actor) {
	if logger == nil || changed == 0 {
		return
	}
	logger.Info("keputusan master bengkel tersimpan tanpa pemberitahuan",
		slog.String("portal", portalAlias),
		slog.Int("berubah", changed),
		slog.String("status", string(status)),
		slog.String("status_label", status.Label()),
		slog.String("oleh", by.Login))
}
