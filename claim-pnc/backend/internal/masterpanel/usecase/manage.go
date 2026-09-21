// Package usecase mengorkestrasi perkara master panel.
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

	"claim-pnc/internal/masterpanel"
)

// Service adalah pintu masuk seluruh perkara master panel.
type Service struct {
	repoSelector masterpanel.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpanel.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpanel/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field saja, dan itu disengaja: `POOLDATA.PANEL_HE` **tidak punya satu pun kolom
// pencatat pelaku maupun waktu** — keempat belas kolomnya terbaca lengkap dari
// `BrowseMasterPanel_HE_RD`, dan tidak satu pun di antaranya menyebut siapa atau kapan.
// Identitas pemanggil karena itu tidak pernah tersimpan di tabelnya — ia hanya dipakai
// untuk log.
//
// Itu keterbatasan tabelnya, bukan pilihan. Jejak audit perubahan master (`D-28`, modul
// `S-5`) belum dapat disandarkan padanya, dan menambah kolom menuntut persetujuan Work
// Owner serta pelaksanaan DBA (`D-63`).
type Actor struct {
	Login string
}

// List mengembalikan baris master panel satu portal yang cocok dengan penyaring.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status masterpanel.ApprovalStatus,
	keyword string,
) ([]masterpanel.Panel, error) {
	if !status.Known() {
		return nil, masterpanel.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx, masterpanel.Filter{
		Status:  status,
		Keyword: strings.TrimSpace(keyword),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
func (l *Service) Get(ctx context.Context, portalAlias, id string) (masterpanel.Panel, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpanel.Panel{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Create menyisipkan satu baris master panel baru beserta lokasinya.
//
// # Urutannya mengikuti sistem lama
//
//	ValidateMasterPanel          tolak bila NAME sudah dipakai
//	PEGA_M_PANEL_HE.prc:12,21    terbitkan ID_PANEL
//	CNMUpdatePanelHE_act         APPROVAL := "0"
//	UpdatePanel_HE-SQL           simpan
//
// Satu perbedaan urutan yang disengaja: pemeriksaan nama ganda di sini berada DI DALAM
// Repo.Insert, bukan sebagai langkah terpisah sebelumnya. Alasannya ada pada doc comment
// masterpanel.Repo.Insert — memecahnya membuka kembali lubang balapan yang justru sedang
// dipersempit.
//
// # Dua hal yang TIDAK dibawa dari sistem lama, dan keduanya disengaja
//
//  1. **Surel pemberitahuan.** `CNMUpdatePanelHE_act` mengirimnya lewat
//     `SendAlertApprovalMaster` dan `SendEmailNotification` ke satu alamat yang tertanam
//     di dalam rule — alamat pribadi seseorang, bukan mailbox fungsional. `D-67`
//     melarangnya dibawa, dan seam Notifier (`S-3`) belum ada. Peristiwanya dicatat di
//     log supaya ketiadaannya terlihat, bukan tersamar.
//  2. **Unggah lampiran.** `PNCSaveAttachmentToDB` dan
//     `SetUploadDocumentToPanelDocumentList` berada di luar lingkup modul ini; kolom
//     DOKUMENID baris yang sudah ada dipertahankan apa adanya.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input masterpanel.Input,
	by Actor,
	logger *slog.Logger,
) (masterpanel.Panel, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpanel.Panel{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpanel.Panel{}, err
	}

	id, err := store.NextID(ctx)
	if err != nil {
		return masterpanel.Panel{}, fmt.Errorf("masterpanel/usecase: menerbitkan ID panel: %w", err)
	}
	if strings.TrimSpace(id) == "" {
		// Bukan keadaan yang dapat diperbaiki pengguna, dan tidak boleh diteruskan
		// diam-diam: baris tanpa kunci tidak dapat dibuka, disunting, maupun disetujui —
		// dan baris ANAKNYA tidak punya apa pun untuk menunjuk induknya.
		return masterpanel.Panel{}, errors.New("masterpanel/usecase: ID panel yang diterbitkan kosong")
	}

	fresh := apply(masterpanel.Panel{ID: strings.TrimSpace(id)}, clean)
	fresh.Status = masterpanel.StatusPending

	if err := store.Insert(ctx, fresh); err != nil {
		return masterpanel.Panel{}, err
	}

	notifyOmitted(logger, portalAlias, fresh, by, "panel baru diajukan")
	return fresh, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Menyimpan SELALU mengembalikan baris ke antrean persetujuan
//
// Itu bukan tafsiran, melainkan langkah tersendiri di sistem lama:
// `Activity/CNMUpdatePanelHE_act` menetapkan `TempStsClaim.APPROVAL := "0"` dan
// `Local.APPROVAL := "0"` tanpa syarat apa pun. Panel yang sudah disetujui lalu disunting
// kembali menunggu.
//
// # Yang TIDAK ikut tersimpan
//
//	ID_PANEL      kunci baris, diambil dari jalur URL, bukan dari badan permintaan
//	STS_APPROVAL  tidak punya isian di layar mana pun; dipertahankan apa adanya
//	ALASAN_TOLAK  diisi pada jalur keputusan; dipertahankan apa adanya di sini
//	DOKUMENID     dipertahankan dari baris yang tersimpan; unggah lampiran tidak dibawa
//	APPROVAL      bukan isian melainkan akibat — selalu StatusPending di sini
//
// DOKUMENID patut diperhatikan. Sistem lama mengisinya dari hasil `PNCSaveAttachmentToDB`
// pada setiap penyimpanan, dan penyimpanan TANPA lampiran menimpanya dengan kosong —
// lampiran yang sudah ada lenyap hanya karena barisnya disunting. Di sini ia dibaca dari
// baris yang tersimpan dan ditulis kembali apa adanya, sehingga jalur yang menghapus
// lampirannya sendiri tidak ikut dibawa.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input masterpanel.Input,
	by Actor,
	logger *slog.Logger,
) (masterpanel.Panel, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpanel.Panel{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return masterpanel.Panel{}, masterpanel.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpanel.Panel{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return masterpanel.Panel{}, err
	}

	if err := ensureUniqueName(ctx, store, key, clean); err != nil {
		return masterpanel.Panel{}, err
	}

	updated := apply(stored, clean)
	updated.Status = masterpanel.StatusPending

	if err := store.Update(ctx, updated); err != nil {
		return masterpanel.Panel{}, err
	}

	notifyOmitted(logger, portalAlias, updated, by, "panel diubah")
	return updated, nil
}

// Decide menetapkan status persetujuan sejumlah baris sekaligus.
//
// # Kenapa borongan, dan bukan satu per satu
//
// Karena di sistem lama memang borongan. `Section/ApprovalMasterPanelHE-Section.xml`
// menyediakan Select All, Deselect All, Approve, dan Reject, dan
// `Activity/SetApprovalAllMaster` menelusuri baris yang dicentang (`.pySelected=="true"`)
// lalu menetapkan `APPROVAL := Param.approval` pada masing-masing. Activity yang sama
// melayani tiga master — bengkel, panel, dan sparepart — dibedakan hanya oleh
// `Param.TIPE2`, yang untuk modul ini bernilai `"M_PANEL_HE"`.
//
// # Alasan penolakan ikut tersimpan
//
// Layar lama menaruh isian "Catatan" (`TempStsClaim.pyNote`) berdampingan dengan tombol
// keputusan, dan `POOLDATA.PANEL_HE` punya kolom ALASAN_TOLAK yang
// `Activity/SetPanelHEValue-Act.xml` muat kembali ke form. Keduanya jelas sepasang.
//
// Alasan hanya ditulis pada keputusan TOLAK. Menuliskannya pada persetujuan akan mengisi
// kolom bernama "alasan tolak" pada baris yang justru disetujui — satu lagi kolom
// berarti ganda, yang justru sedang dihindari modul ini.
//
// # Satu hal yang TIDAK dapat direplikasi, dan harus dinyatakan
//
// Rule SQL yang benar-benar menjalankan penetapannya **tidak ada di export** (`R-16`):
// `SetApprovalAllMaster` dirujuk beberapa berkas tetapi tidak punya berkas sendiri. Yang
// terbaca hanyalah kolom yang disentuhnya.
func (l *Service) Decide(
	ctx context.Context,
	portalAlias string,
	id []string,
	status masterpanel.ApprovalStatus,
	reason string,
	by Actor,
	logger *slog.Logger,
) (int, error) {
	if !status.Known() {
		return 0, masterpanel.ErrUnknownStatus
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
		return 0, masterpanel.OneViolation("id_panel", "Pilih dulu panel yang akan diputuskan.")
	}

	note := strings.TrimSpace(reason)
	if len(note) > masterpanel.MaxReasonLength {
		return 0, masterpanel.OneViolation("catatan",
			fmt.Sprintf("Catatan paling panjang %d karakter.", masterpanel.MaxReasonLength))
	}
	if status != masterpanel.StatusRejected {
		note = ""
	}

	changed, err := store.SetStatus(ctx, wanted, status, note)
	if err != nil {
		return 0, err
	}

	if changed != len(wanted) && logger != nil {
		// Bukan galat: baris yang tidak berubah adalah baris yang sudah tidak ada, atau
		// sudah berstatus itu. Tetapi selisihnya berarti layar menampilkan daftar yang
		// sudah basi, dan itu layak terbaca.
		logger.Warn("keputusan master panel tidak menyentuh seluruh baris yang dipilih",
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
		return fmt.Errorf("masterpanel/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// apply menyalin isian yang dikirim layar ke atas baris yang sudah ada.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya kedua jalur tidak pernah berbeda soal
// kolom mana yang ikut berubah. Lima kolom sengaja tidak disentuh — ID, ApprovalMark,
// RejectReason, DocumentID, dan Status — dan ketiganya yang terakhir diatur pemanggil
// atau jalur lain.
//
// Daftar lokasi DISALIN, bukan dipakai bersama. Tanpa itu, pemanggil yang mengubah
// isiannya setelah apply akan ikut mengubah baris yang sudah dianggap tersimpan.
func apply(base masterpanel.Panel, in masterpanel.Input) masterpanel.Panel {
	base.Name = in.Name
	base.RepairStatus = in.RepairStatus
	base.EditQuantityStatus = in.EditQuantityStatus
	base.PremiumRepairStatus = in.PremiumRepairStatus
	base.ShatterStatus = in.ShatterStatus
	base.StickerStatus = in.StickerStatus
	base.SideStatus = in.SideStatus
	base.SevereDamageStatus = in.SevereDamageStatus
	base.ActiveStatus = in.ActiveStatus
	base.ExclusionC = in.ExclusionC

	base.Location = make([]masterpanel.PanelLocation, len(in.Location))
	copy(base.Location, in.Location)

	return base
}

// ensureUniqueName menolak nama yang sudah dipakai BARIS LAIN.
//
// Ia dipakai jalur simpan saja. Jalur tambah memeriksanya di dalam Repo.Insert, yang
// dapat menguncinya lebih dulu; di sini penguncian itu tidak diperlukan karena kunci
// barisnya sudah ada, dan bentrok yang muncul di antara pemeriksaan dan penyimpanan hanya
// dapat berasal dari petugas lain yang menyimpan nama yang sama pada saat yang sama —
// keadaan yang tertutup constraint unik, bukan oleh urutan langkah (R-08, D-63).
//
// # Kenapa pemeriksaannya ada di jalur simpan juga
//
// Karena di sistem lama pun ada. `Activity/ValidateMasterPanel` dipanggil dari layar
// tanpa memandang tambah atau ubah. Melewatkannya di sini akan memperbolehkan dua panel
// bernama sama — cukup dengan menyunting salah satunya.
//
// Baris yang sedang disunting dikecualikan: menyimpan tanpa mengubah namanya tidak boleh
// ditolak karena namanya sendiri sudah dipakai oleh dirinya sendiri.
func ensureUniqueName(
	ctx context.Context,
	store masterpanel.Store,
	id string,
	in masterpanel.Input,
) error {
	switch other, err := store.FindByName(ctx, in.Name); {
	case err == nil && strings.TrimSpace(other.ID) != strings.TrimSpace(id):
		return masterpanel.OneViolation("nama_panel",
			"Nama tersebut telah digunakan panel "+other.ID+". Silakan ganti dengan nama yang lain.")
	case err != nil && !errors.Is(err, masterpanel.ErrNotFound):
		return fmt.Errorf("masterpanel/usecase: memeriksa nama %q: %w", in.Name, err)
	}
	return nil
}

// notifyOmitted mencatat pemberitahuan yang di sistem lama dikirim, dan di sini tidak.
//
// Ia sengaja dicatat sebagai Info dan bukan didiamkan: yang hilang adalah satu langkah
// proses yang nyata — PIC tidak lagi diberi tahu bahwa ada panel menunggu persetujuan —
// dan ketiadaannya harus terbaca di log alih-alih ditemukan berbulan kemudian oleh
// petugas yang bertanya kenapa antreannya menumpuk.
func notifyOmitted(logger *slog.Logger, portalAlias string, p masterpanel.Panel, by Actor, event string) {
	if logger == nil {
		return
	}
	logger.Info("pemberitahuan master panel tidak dikirim",
		slog.String("peristiwa", event),
		slog.String("portal", portalAlias),
		slog.String("id_panel", p.ID),
		slog.String("nama_panel", p.Name),
		slog.Int("jumlah_lokasi", len(p.Location)),
		slog.String("oleh", by.Login),
		slog.String("sebab", "seam Notifier (S-3) belum ada, dan penerima di rule lama berupa alamat pribadi yang D-67 larang dibawa"))
}

// notifyDecisionOmitted mencatat hal yang sama untuk jalur keputusan.
func notifyDecisionOmitted(logger *slog.Logger, portalAlias string, changed int, status masterpanel.ApprovalStatus, by Actor) {
	if logger == nil || changed == 0 {
		return
	}
	logger.Info("keputusan master panel tersimpan tanpa pemberitahuan",
		slog.String("portal", portalAlias),
		slog.Int("berubah", changed),
		slog.String("status", string(status)),
		slog.String("status_label", status.Label()),
		slog.String("oleh", by.Login))
}
