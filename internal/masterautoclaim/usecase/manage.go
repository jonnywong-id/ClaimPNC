// Package usecase mengorkestrasi perkara master auto claim.
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

	"claim-pnc/internal/masterautoclaim"
)

// Service adalah pintu masuk seluruh perkara master auto claim.
type Service struct {
	repoSelector masterautoclaim.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterautoclaim.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterautoclaim/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Satu field saja, dan itu disengaja. Yang dibutuhkan modul ini adalah OPERATOR ID —
// nilai yang sama yang dipakai `OperatorID.pyUserIdentifier` di Pega, dan yang sama
// yang tersimpan di `POOLDATA.EMAILKOMITE.OPERATOR_ID`.
//
// Ia login yang DIKETIK pengguna, bukan NIK. Memakai NIK di sini akan membuat penyaring
// tab Komite Approval (`KOMITE = <saya>`) tidak pernah cocok, karena kolom KOMITE berisi
// operator ID. Modul menu memakai nilai yang sama dengan alasan yang sama.
type Actor struct {
	Login string
}

// List mengembalikan baris master auto claim satu portal yang cocok dengan penyaring.
//
// Tab Komite Approval ditandai CommitteeOnly, dan CommitteeID-nya diisi DI SINI dari
// pemanggil — bukan diterima dari layar. Membiarkan layar mengirimkannya berarti siapa
// pun dapat melihat antrean persetujuan komite lain hanya dengan mengganti satu nilai
// di permintaan.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	status masterautoclaim.ApprovalStatus,
	committeeOnly bool,
	by Actor,
) ([]masterautoclaim.AutoClaim, error) {
	if !status.Known() {
		return nil, masterautoclaim.ErrUnknownStatus
	}

	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	filter := masterautoclaim.Filter{Status: status}
	if committeeOnly {
		filter.CommitteeOnly = true
		filter.CommitteeID = strings.TrimSpace(by.Login)
	}
	return store.List(ctx, filter)
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
//
// Padanan `Activity/UpdateMstAutoClaim_act1`, yang membaca satu baris lalu menyalin
// kedua belas kolomnya ke page form.
func (l *Service) Get(ctx context.Context, portalAlias, initial string) (masterautoclaim.AutoClaim, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterautoclaim.AutoClaim{}, err
	}
	return store.Get(ctx, strings.TrimSpace(initial))
}

// Create menyisipkan satu baris master auto claim baru.
//
// # Urutannya mengikuti Activity/InsertMstAutoClaim_act apa adanya
//
//	langkah 1–3  periksa isian, termasuk kode bank      -> Input.Check
//	langkah 4–5  tolak bila INISIALID sudah dipakai     -> Repo.Insert
//	langkah 6    cari penyetuju komite                  -> LookupRepo.Committee
//	langkah 7    isi USERINPUT, KOMITE, APPROVAL        -> di bawah
//	langkah 8    sisip                                  -> Repo.Insert
//
// Satu perbedaan urutan yang disengaja: pemeriksaan duplikat di sini berada DI DALAM
// Repo.Insert, bukan sebagai langkah terpisah sebelumnya. Alasannya ada pada doc
// comment masterautoclaim.Repo.Insert — memecahnya membuka lubang balapan yang justru
// sedang dipersempit.
//
// # Tiga nilai yang TIDAK diterima dari layar
//
//	APPROVAL       selalu StatusPending. Layar lama pun memanggil dengan Approval="0".
//	CLAIM_ALLOWED  selalu ClaimAllowedYes (keputusan Work Owner 2026-09-19).
//	KOMITE         hasil lookup, bukan pilihan pengguna.
//	USERINPUT      operator yang sedang masuk, bukan nilai yang dikirim.
func (l *Service) Create(
	ctx context.Context,
	portalAlias string,
	input masterautoclaim.Input,
	by Actor,
	logger *slog.Logger,
) (masterautoclaim.AutoClaim, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterautoclaim.AutoClaim{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}
	if err := ensureBankKnown(ctx, store, clean.BankName); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}

	committee, err := store.Committee(ctx)
	if err != nil {
		return masterautoclaim.AutoClaim{}, fmt.Errorf("masterautoclaim/usecase: mencari penyetuju komite: %w", err)
	}
	if strings.TrimSpace(committee) == "" && logger != nil {
		// Bukan galat — sistem lama pun menyimpan KOMITE kosong dalam keadaan ini.
		// Tetapi akibatnya tidak terlihat siapa pun: barisnya tidak akan pernah muncul
		// di tab Komite Approval, sehingga tertahan di Waiting Approval selamanya.
		// Kegagalan senyap seperti itu harus terbaca di log, bukan ditemukan berbulan
		// kemudian oleh petugas yang bertanya kenapa klaim otomatisnya tidak jalan.
		logger.Warn("penyetuju komite master auto claim tidak ditemukan",
			slog.String("portal", portalAlias),
			slog.String("akibat", "baris baru tidak akan muncul di tab Komite Approval dan tertahan di Waiting Approval"),
			slog.String("periksa", "POOLDATA.EMAILKOMITE dengan TYPE_BUSINESS='BONDING' dan STS_AKTIF='1'"))
	}

	fresh := masterautoclaim.AutoClaim{
		Initial:         clean.Initial,
		ReceiverName:    clean.ReceiverName,
		BankName:        clean.BankName,
		AccountNumber:   clean.AccountNumber,
		MaxPercent:      clean.MaxPercent,
		ReporterPIC:     clean.ReporterPIC,
		ReporterEmail:   clean.ReporterEmail,
		ClaimAllowed:    masterautoclaim.ClaimAllowedYes,
		ReceiverAddress: clean.ReceiverAddress,
		SubmittedBy:     strings.TrimSpace(by.Login),
		Committee:       strings.TrimSpace(committee),
		Status:          masterautoclaim.StatusPending,
		ClientID:        clean.ClientID,
		ClientName:      clean.ClientName,
	}

	if err := store.Insert(ctx, fresh); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}
	return fresh, nil
}

// Save menyimpan perubahan atas baris yang sudah ada, SEKALIGUS menetapkan statusnya.
//
// # Kenapa menyimpan dan memutuskan adalah satu operasi
//
// Karena di sistem lama memang satu: `Activity/UpdateMstAutoClaim_act` melayani ketiga
// tombol, dibedakan hanya oleh parameternya.
//
//	tab Master Auto Klaim / Reject, tombol Update   stsapprove="0"
//	tab Komite Approval, tombol Approve             stsapprove="1"
//	tab Komite Approval, tombol Reject              stsapprove="2"
//
// Ketiganya menulis dua belas kolom yang sama dari isi form. Keputusan Work Owner
// 2026-09-19 mempertahankan bentuk itu: badan permintaan memuat seluruh isian, dan
// status yang dikehendaki ikut di dalamnya.
//
// Akibat yang mengikuti, dan memang dikehendaki: MENYUNTING BARIS YANG SUDAH DISETUJUI
// MENGEMBALIKANNYA KE MENUNGGU, karena tombol Update pada tab Master mengirim status
// "0". Persetujuan lama tidak berlaku atas isi yang sudah berubah — dan itu benar untuk
// master yang menentukan ke mana uang dikirim.
//
// # Yang TIDAK ikut tersimpan
//
//	INISIALID      kunci baris, diambil dari jalur URL, bukan dari badan permintaan
//	NAMA_PENERIMA  tidak disebut UpdateAutoClaim-SQL.xml; lihat AutoClaim.ReceiverName
//	KOMITE         dipertahankan dari baris yang tersimpan
//
// KOMITE patut diperhatikan. Pega menulisnya dari `TempInputAutoClaim.FlagASO`, yaitu
// nilai yang ADA DI FORM — dan form itu hanya terisi bila barisnya lebih dulu dimuat
// lewat tombol Update. Bila komite menekan Approve tanpa memuatnya, FlagASO kosong dan
// kolom KOMITE tertimpa kosong, sehingga barisnya lenyap dari tab Komite Approval.
//
// Di sini KOMITE tidak pernah datang dari permintaan: ia dibaca dari baris yang
// tersimpan dan ditulis kembali apa adanya. Hasil yang teramati pada jalur normal sama;
// jalur yang menghapus penyetujunya sendiri tidak ikut dibawa.
func (l *Service) Save(
	ctx context.Context,
	portalAlias, initial string,
	input masterautoclaim.Input,
	by Actor,
) (masterautoclaim.AutoClaim, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterautoclaim.AutoClaim{}, err
	}

	key := strings.TrimSpace(initial)
	if key == "" {
		return masterautoclaim.AutoClaim{}, masterautoclaim.ErrNotFound
	}

	clean := input.Clean()
	if !clean.Status.Known() {
		return masterautoclaim.AutoClaim{}, masterautoclaim.ErrUnknownStatus
	}
	// CheckEditable, bukan Check: Sumber Bisnis dan nama penerima tidak datang dari
	// permintaan pada jalur ini, dan memeriksanya akan menolak permintaan yang benar.
	if err := clean.CheckEditable(); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}
	if err := ensureBankKnown(ctx, store, clean.BankName); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}

	stored, err := store.Get(ctx, key)
	if err != nil {
		return masterautoclaim.AutoClaim{}, err
	}

	stored.BankName = clean.BankName
	stored.AccountNumber = clean.AccountNumber
	stored.MaxPercent = clean.MaxPercent
	stored.ReporterPIC = clean.ReporterPIC
	stored.ReporterEmail = clean.ReporterEmail
	stored.ReceiverAddress = clean.ReceiverAddress
	stored.ClientID = clean.ClientID
	stored.ClientName = clean.ClientName
	stored.Status = clean.Status
	stored.SubmittedBy = strings.TrimSpace(by.Login)
	// Selalu "1" — pada jalur ubah pun, dan itu selisih yang direncanakan terhadap
	// Pega. Lihat masterautoclaim.ClaimAllowedYes.
	stored.ClaimAllowed = masterautoclaim.ClaimAllowedYes

	if err := store.Update(ctx, stored); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}
	return stored, nil
}

// ensureBankKnown memastikan nama bank yang akan tersimpan ada di master bank.
//
// Ia padanan pemeriksaan "Nama bank jangan diketik manual" (InsertMstAutoClaim_act
// step 3), dipindahkan ke lapisan ini karena menuntut pembacaan basis data.
//
// Galatnya dikembalikan sebagai PELANGGARAN ISIAN, bukan galat teknis: yang salah
// adalah isian pengguna, dan ia dapat memperbaikinya dengan memilih ulang dari daftar.
// Galat selain "tidak ditemukan" diteruskan apa adanya — basis data yang tidak dapat
// dihubungi bukan kesalahan pengguna dan tidak boleh tampil sebagai isian yang salah.
func ensureBankKnown(ctx context.Context, store masterautoclaim.Store, name string) error {
	_, err := store.FindBankByName(ctx, name)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, masterautoclaim.ErrBankNotFound):
		return masterautoclaim.OneViolation("nama_bank",
			"Bank penerima harus dipilih dari daftar, tidak boleh diketik manual.")
	default:
		return fmt.Errorf("masterautoclaim/usecase: memeriksa bank %q: %w", name, err)
	}
}

// SearchBusinessSources melayani lookup Sumber Bisnis pada form.
//
// Kata kunci yang terlalu pendek dijawab daftar kosong, BUKAN galat: pengguna yang baru
// mengetik satu huruf belum melakukan kesalahan apa pun, dan pesan galat di bawah kotak
// pencarian yang sedang diketik hanya akan mengganggu.
func (l *Service) SearchBusinessSources(ctx context.Context, portalAlias, keyword string) ([]masterautoclaim.BusinessSource, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	clean := strings.TrimSpace(keyword)
	if len(clean) < masterautoclaim.MinLookupKeyword {
		return nil, nil
	}
	return store.SearchBusinessSources(ctx, clean)
}

// SearchClients melayani lookup Client pada form.
func (l *Service) SearchClients(ctx context.Context, portalAlias, keyword string) ([]masterautoclaim.Client, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	clean := strings.TrimSpace(keyword)
	if len(clean) < masterautoclaim.MinLookupKeyword {
		return nil, nil
	}
	return store.SearchClients(ctx, clean)
}

// ListBanks melayani dropdown Bank Penerima pada form.
func (l *Service) ListBanks(ctx context.Context, portalAlias string) ([]masterautoclaim.Bank, error) {
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
		return fmt.Errorf("masterautoclaim/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
