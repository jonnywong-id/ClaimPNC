// Package usecase mengorkestrasi perkara master sparepart.
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

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/platform/clock"
)

// Service adalah pintu masuk seluruh perkara master sparepart.
type Service struct {
	repoSelector mastersparepart.RepoSelector
	clock        clock.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastersparepart.RepoSelector

	// Clock menstempel TGL_UPDATE_HARGA. Wajib.
	//
	// Berbeda dari Master Panel dan Master Bengkel yang tidak menerimanya sama sekali:
	// `POOLDATA.SPAREPART_HE` PUNYA kolom waktu, dan `Activity/UpdateSparepartHE_act`
	// mengisinya dengan `@DateTime.CurrentDateTime()`.
	//
	// Ia seam dan bukan `time.Now()` langsung supaya penyimpanan dapat diuji
	// deterministik, dan supaya tidak ada satu pun penambahan 7 jam manual yang menyelinap
	// masuk (`F-5`, `08-TECHNICAL-STRATEGY.md` §4.4).
	Clock clock.Clock
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastersparepart/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("mastersparepart/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field, dan berbeda dari Master Panel ia BENAR-BENAR TERSIMPAN:
// `POOLDATA.SPAREPART_HE` punya kolom `USER_UPDATE`, dan
// `Activity/UpdateSparepartHE_act` mengisinya dengan `OperatorID.pyUserIdentifier`.
//
// Yang tersimpan adalah login yang DIKETIK pengguna, bukan NIK-nya — kolomnya sudah berisi
// login pada baris-baris lama, dan menuliskan NIK ke kolom yang sama akan membuat dua
// bentuk identitas hidup berdampingan tanpa satu pun cara membedakannya.
type Actor struct {
	Login string
}

// LookupSet adalah kedua daftar acuan yang dibutuhkan form.
//
// Keduanya dikirim bersamaan, bukan lewat dua endpoint terpisah, karena layar Pega pun
// menariknya dalam satu jalan: `Activity/BrowseTipeKategoriPart-Act.xml` menjalankan
// kedua RDB-List berturut-turut. Dua perjalanan jaringan untuk membuka satu form adalah
// biaya yang tidak perlu dibayar.
type LookupSet struct {
	Category []mastersparepart.Category
	Type     []mastersparepart.PartType
}

// List mengembalikan baris master sparepart satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status mastersparepart.ApprovalStatus,
	keyword string,
) ([]mastersparepart.Sparepart, error) {
	if !status.Known() {
		return nil, mastersparepart.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, mastersparepart.Filter{
		Status:  status,
		Keyword: strings.TrimSpace(keyword),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
func (l *Service) Get(
	ctx context.Context,
	portalAlias, id string,
) (mastersparepart.Sparepart, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersparepart.Sparepart{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Options mengembalikan kedua daftar acuan yang dipakai form.
//
// Keduanya dibaca dari basis data ENTITAS yang sedang dibuka, bukan dari daftar tetap
// milik aplikasi — kategori dan tipe suku cadang berbeda antarentitas, dan menyajikan
// daftar satu entitas kepada entitas lain adalah kebocoran yang justru dicegah `R-20`.
func (l *Service) Options(ctx context.Context, portalAlias string) (LookupSet, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return LookupSet{}, err
	}

	category, err := store.ListCategories(ctx)
	if err != nil {
		return LookupSet{}, fmt.Errorf("mastersparepart/usecase: membaca kategori: %w", err)
	}
	partType, err := store.ListTypes(ctx)
	if err != nil {
		return LookupSet{}, fmt.Errorf("mastersparepart/usecase: membaca tipe: %w", err)
	}

	return LookupSet{Category: category, Type: partType}, nil
}

// Create menyisipkan satu baris master sparepart baru.
//
// # Urutannya mengikuti sistem lama
//
//	ValidateMasterSparepart           tolak bila NO_SPART, NAMA_SPART, atau KODE_SPART ganda
//	PEGA_M_SPAREPART_HE.prc:12,21     terbitkan ID
//	UpdateSparepartHE_act             USER_UPDATE, TGL_UPDATE_HARGA, APPROVAL := "0"
//	UpdateSparepartHE-SQL             simpan
//
// Satu perbedaan urutan yang disengaja: pemeriksaan ketiga kunci alami di sini berada DI
// DALAM Repo.Insert, bukan sebagai langkah terpisah sebelumnya. Alasannya ada pada doc
// comment mastersparepart.Repo.Insert — memecahnya membuka kembali lubang balapan yang
// justru sedang dipersempit.
//
// # Dua hal yang TIDAK dibawa dari sistem lama, dan keduanya disengaja
//
//  1. **Surel pemberitahuan.** `UpdateSparepartHE_act` mengirimnya lewat
//     `SendAlertApprovalMaster` dan `SendEmailNotification`, dengan penerima yang diambil
//     dari `.USER_UPDATE` (`Activity/SetApprovalAllMaster-Act.xml` menetapkan
//     `InputBengkel.MAIL := .USER_UPDATE`). Seam Notifier (`S-3`) belum ada, dan `D-67`
//     melarang penerima berupa akun pribadi dibawa apa adanya. Peristiwanya dicatat di log
//     supaya ketiadaannya terlihat, bukan tersamar.
//  2. **Unggah lampiran.** `PNCSaveAttachmentToDB` berada di luar lingkup modul ini; kolom
//     DOKUMENID baris yang sudah ada dipertahankan apa adanya.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input mastersparepart.Input,
	by Actor,
	logger *slog.Logger,
) (mastersparepart.Sparepart, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersparepart.Sparepart{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastersparepart.Sparepart{}, err
	}

	id, err := store.NextID(ctx)
	if err != nil {
		return mastersparepart.Sparepart{}, fmt.Errorf(
			"mastersparepart/usecase: menerbitkan ID sparepart: %w", err)
	}
	if strings.TrimSpace(id) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan
		// diam-diam: baris tanpa kunci tidak dapat dibuka, disunting, maupun disetujui.
		return mastersparepart.Sparepart{}, errors.New(
			"mastersparepart/usecase: ID sparepart yang diterbitkan kosong")
	}

	fresh := apply(mastersparepart.Sparepart{ID: strings.TrimSpace(id)}, clean)
	fresh.Status = mastersparepart.StatusPending
	fresh.UpdatedBy = strings.TrimSpace(by.Login)
	l.stampPrice(&fresh)

	if err := store.Insert(ctx, fresh); err != nil {
		return mastersparepart.Sparepart{}, err
	}

	notifyOmitted(logger, portalAlias, fresh, by, "sparepart baru diajukan")
	return fresh, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
//
// Itu bukan tafsiran, melainkan langkah tersendiri di sistem lama:
// `Activity/UpdateSparepartHE_act` menetapkan `TempSparepart.APPROVAL := "0"` tanpa syarat
// apa pun. Sparepart yang sudah disetujui lalu disunting kembali menunggu.
//
// # Yang TIDAK ikut tersimpan dari badan permintaan
//
//	ID                kunci baris, diambil dari jalur URL, bukan dari badan permintaan
//	APPROVAL          bukan isian melainkan akibat — selalu StatusPending di sini
//	USER_UPDATE       diambil dari identitas pemanggil
//	TGL_UPDATE_HARGA  distempel di sini; lihat stampPrice
//	DOKUMENID         dipertahankan dari baris yang tersimpan
//
// DOKUMENID patut diperhatikan. Sistem lama mengisinya dari hasil `PNCSaveAttachmentToDB`
// pada setiap penyimpanan, dan penyimpanan TANPA lampiran menimpanya dengan kosong —
// lampiran yang sudah ada lenyap hanya karena barisnya disunting. Di sini ia dibaca dari
// baris yang tersimpan dan ditulis kembali apa adanya, sehingga jalur yang menghapus
// lampirannya sendiri tidak ikut dibawa.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input mastersparepart.Input,
	by Actor,
	logger *slog.Logger,
) (mastersparepart.Sparepart, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return mastersparepart.Sparepart{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return mastersparepart.Sparepart{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return mastersparepart.Sparepart{}, err
	}

	if err := ensureUnique(ctx, store, key, clean); err != nil {
		return mastersparepart.Sparepart{}, err
	}

	updated := apply(stored, clean)
	updated.Status = mastersparepart.StatusPending
	updated.UpdatedBy = strings.TrimSpace(by.Login)
	l.stampPrice(&updated)

	if err := store.Update(ctx, updated); err != nil {
		return mastersparepart.Sparepart{}, err
	}

	notifyOmitted(logger, portalAlias, updated, by, "sparepart diubah")
	return updated, nil
}

// stampPrice menstempel TGL_UPDATE_HARGA bila harga jualnya terisi.
//
// # Ia distempel pada SETIAP penyimpanan, bukan hanya saat harganya berubah
//
// Itu meniru sistem lama apa adanya (`P-5`). `Activity/UpdateSparepartHE_act` menetapkan
// `TempSparepart.TGL_UPDATE_HARGA := @DateTime.CurrentDateTime()` dengan precondition
// `@PropertyHasValue(TempSparepart.HARGA_JUAL)` — yang diperiksa hanyalah harganya TERISI,
// bukan harganya BERUBAH.
//
// Akibatnya kolom bernama "tanggal update harga" ikut bergerak ketika yang berubah hanya
// nama sparepartnya. Itu ganjil, dan ia TIDAK diperbaiki di sini: memperbaikinya berarti
// menambah butir ke daftar perbaikan eksplisit `P-5`, dan daftar itu milik `D-49` —
// keputusan Work Owner, bukan tafsiran modul.
//
// Karena harga jual WAJIB diisi (`pyRequired=true`), preconditionnya praktis selalu benar
// pada jalur layar ini. Ia tetap diperiksa supaya baris yang harganya kosong — baris lama
// yang disimpan sebelum kewajiban itu ada — tidak mendapat stempel yang tidak bermakna.
func (l *Service) stampPrice(s *mastersparepart.Sparepart) {
	if strings.TrimSpace(s.SellingPrice) == "" {
		return
	}
	now := l.clock.Now().UTC()
	s.PriceUpdatedAt = &now
}

// Decide menetapkan status persetujuan sejumlah baris sekaligus.
//
// # Kenapa borongan, dan bukan satu per satu
//
// Karena di sistem lama memang borongan.
// `Section/ApprovalMasterSparepartHE-Section.xml` menyediakan Select All, Deselect All,
// Approve, dan Reject, dan `Activity/SetApprovalAllMaster` menelusuri baris yang dicentang
// (`.pySelected=="true"`) lalu menetapkan `APPROVAL := Param.approval` pada masing-masing.
//
// Activity yang sama melayani TIGA master — bengkel, panel, dan sparepart — dibedakan oleh
// nilai yang ditulis ke `InputBengkel.ALASAN_STS_BGKL`, yang untuk modul ini bernilai
// `"M_SPAREPART_HE"`. Perhatikan nama propertinya: satu properti bernama "alasan status
// bengkel" dipakai memikul JENIS MASTER, dan itu satu lagi contoh utang penamaan yang
// `03-CURRENT-ARCHITECTURE.md` §4.2 catat.
//
// # TANPA alasan penolakan, berbeda dari Master Panel
//
// `POOLDATA.SPAREPART_HE` tidak punya kolom penampungnya: kedua puluh tiga kolomnya
// terbaca lengkap dari `BrowseSparepartHE_RD`, dan tidak satu pun menampung catatan.
//
// Layar persetujuan Pega memang punya isian `pyNote`, tetapi di modul ini ia tidak menuju
// ke mana pun — `Activity/SetValueSparepartHE` justru mengosongkannya
// (`TempSparepart.pyNote := ""`) dan `UpdateSparepartHE_act` memakainya sebagai penampung
// sementara hasil penyimpanan (`TempSparepart.pyNote := OutputSparepart.ID`).
//
// Karena itu layar baru TIDAK menggambar isian catatan sama sekali. Menggambar isian yang
// diam-diam membuang isinya lebih buruk daripada tidak menggambarnya.
//
// # Satu hal yang TIDAK dapat direplikasi, dan harus dinyatakan
//
// Rule SQL yang benar-benar menjalankan penetapannya **tidak ada di export** (`R-16`):
// `SetApprovalAllMaster` dirujuk sebagai `ASM-FW-GCNMFW-Int-PANEL_HE GCNM
// SetApprovalAllMaster` tetapi tidak punya berkas sendiri. Yang terbaca hanyalah kolom
// yang disentuhnya.
func (l *Service) Decide(
	ctx context.Context,
	portalAlias string,
	id []string,
	status mastersparepart.ApprovalStatus,
	by Actor,
	logger *slog.Logger,
) (int, error) {
	if !status.Known() {
		return 0, mastersparepart.ErrUnknownStatus
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
		return 0, mastersparepart.OneViolation("id_sparepart",
			"Pilih dulu sparepart yang akan diputuskan.")
	}

	changed, err := store.SetStatus(ctx, wanted, status)
	if err != nil {
		return 0, err
	}

	if changed != len(wanted) && logger != nil {
		// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau
		// sudah berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang
		// sudah basi, dan itu layak terbaca.
		logger.Warn("keputusan master sparepart tidak menyentuh seluruh baris yang dipilih",
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
		return fmt.Errorf("mastersparepart/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// apply menyalin isian yang dikirim layar ke atas baris yang sudah ada.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya kedua jalur tidak pernah berbeda soal
// kolom mana yang ikut berubah. Lima kolom sengaja tidak disentuh — ID, UpdatedBy,
// PriceUpdatedAt, DocumentID, dan Status — dan keempat yang terakhir diatur pemanggil atau
// jalur lain.
func apply(base mastersparepart.Sparepart, in mastersparepart.Input) mastersparepart.Sparepart {
	base.Name = in.Name
	base.Number = in.Number
	base.Code = in.Code

	base.SellingPrice = in.SellingPrice

	base.CategoryID = in.CategoryID
	base.TypeID = in.TypeID

	base.Weight = in.Weight
	base.Length = in.Length
	base.Width = in.Width
	base.Height = in.Height
	base.MinStock = in.MinStock
	base.MaxStock = in.MaxStock
	base.OrderQuantity = in.OrderQuantity

	base.ProductionDate = in.ProductionDate
	base.Substitute = in.Substitute

	base.Kind = in.Kind
	base.Unit = in.Unit
	base.ActiveStatus = in.ActiveStatus
	base.PartStatus = in.PartStatus

	return base
}

// ensureUnique menolak ketiga kunci alami yang sudah dipakai BARIS LAIN.
//
// Ia dipakai jalur simpan saja. Jalur tambah memeriksanya di dalam Repo.Insert, yang dapat
// menguncinya lebih dulu; di sini penguncian itu tidak diperlukan karena kunci barisnya
// sudah ada, dan bentrok yang muncul di antara pemeriksaan dan penyimpanan hanya dapat
// berasal dari petugas lain yang menyimpan nilai yang sama pada saat yang sama — keadaan
// yang tertutup constraint unik, bukan oleh urutan langkah (`R-08`, `D-63`).
//
// # Kenapa pemeriksaannya ada di jalur simpan juga
//
// Karena di sistem lama pun ada. `Activity/ValidateMasterSparepart` dipanggil dari layar
// tanpa memandang tambah atau ubah. Melewatkannya di sini akan memperbolehkan dua
// sparepart bernomor sama — cukup dengan menyunting salah satunya.
//
// # KETIGANYA diperiksa, dan seluruh pelanggarannya dilaporkan bersamaan
//
// Bukan berhenti pada yang pertama. `ValidateMasterSparepart` menyusun tiga pesan
// berdampingan — `Local.errmsg`, `errmsg1`, `errmsg2` — dan menampilkan ketiganya
// sekaligus. Petugas yang salah pada nomor DAN kode karena itu tahu keduanya dalam satu
// kali simpan, bukan dua.
//
// Baris yang sedang disunting dikecualikan: menyimpan tanpa mengubah nomornya tidak boleh
// ditolak karena nomornya sendiri sudah dipakai oleh dirinya sendiri.
func ensureUnique(
	ctx context.Context,
	store mastersparepart.Store,
	id string,
	in mastersparepart.Input,
) error {
	type probe struct {
		field, label, value string
		find                func(context.Context, string) (mastersparepart.Sparepart, error)
	}

	var violation []mastersparepart.Violation

	for _, p := range []probe{
		{"nomor_sparepart", "Nomor", in.Number, store.FindByNumber},
		{"nama_sparepart", "Nama", in.Name, store.FindByName},
		{"kode_sparepart", "Kode", in.Code, store.FindByCode},
	} {
		if p.value == "" {
			continue
		}

		other, err := p.find(ctx, p.value)
		switch {
		case err == nil && strings.TrimSpace(other.ID) != strings.TrimSpace(id):
			violation = append(violation, mastersparepart.Violation{
				Field: p.field,
				Message: p.label + " tersebut telah digunakan sparepart " + other.ID +
					". Silakan ganti dengan yang lain.",
			})
		case err != nil && !errors.Is(err, mastersparepart.ErrNotFound):
			return fmt.Errorf("mastersparepart/usecase: memeriksa %s %q: %w",
				p.field, p.value, err)
		}
	}

	if len(violation) > 0 {
		return &mastersparepart.ValidationError{Violation: violation}
	}
	return nil
}

// notifyOmitted mencatat pemberitahuan yang di sistem lama dikirim, dan di sini tidak.
//
// Ia sengaja dicatat sebagai Info dan bukan didiamkan: yang hilang adalah satu langkah
// proses yang nyata — PIC tidak lagi diberi tahu bahwa ada sparepart menunggu persetujuan
// — dan ketiadaannya harus terbaca di log alih-alih ditemukan berbulan kemudian oleh
// petugas yang bertanya kenapa antreannya menumpuk.
func notifyOmitted(
	logger *slog.Logger,
	portalAlias string,
	s mastersparepart.Sparepart,
	by Actor,
	event string,
) {
	if logger == nil {
		return
	}
	logger.Info("pemberitahuan master sparepart tidak dikirim",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_sparepart", s.ID),
		slog.String("nomor_sparepart", s.Number),
		slog.String("oleh", by.Login),
		slog.String("sebab", "seam Notifier (S-3) belum ada, dan penerima di rule lama diambil dari USER_UPDATE yang D-67 larang dibawa apa adanya"))
}

// notifyDecisionOmitted mencatat hal yang sama untuk jalur keputusan.
func notifyDecisionOmitted(
	logger *slog.Logger,
	portalAlias string,
	changed int,
	status mastersparepart.ApprovalStatus,
	by Actor,
) {
	if logger == nil || changed == 0 {
		return
	}
	logger.Info("keputusan master sparepart tersimpan tanpa pemberitahuan",
		slog.String("portal", portalAlias),
		slog.Int("berubah", changed),
		slog.String("status", string(status)),
		slog.String("status_label", status.Label()),
		slog.String("oleh", by.Login))
}
