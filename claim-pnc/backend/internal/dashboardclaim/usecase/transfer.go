package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// IDGenerator membangkitkan pengenal permintaan.
type IDGenerator interface {
	New() (string, error)
}

// Clock adalah seam ke jam.
//
// Waktu permintaan dicatat dari sini, bukan dari time.Now() di tengah kode, supaya jejaknya
// dapat diuji secara pasti — dan supaya tidak ada satu pun tempat yang menambah tujuh jam
// sendiri (`F-5`).
type Clock interface {
	Now() time.Time
}

// TransferCaller adalah identitas pemohon.
//
// Dua field, bukan satu: NAMA disimpan sebagai salinan di samping login. Jejak yang namanya
// diambil lewat join akan berubah ketika orangnya berganti nama, dan jejak yang dapat
// berubah bukan jejak.
type TransferCaller struct {
	Login string
	Name  string
}

// TransferCommand adalah permintaan transfer dari layar.
type TransferCommand struct {
	PortalAlias string
	Caller      TransferCaller
	Request     dashboardclaim.TransferCommand
}

// Transfer mencatat permintaan pemindahan penugasan.
//
// # Ia MENCATAT PERMINTAAN, bukan memindahkan penugasan
//
// `P-1` menetapkan `DATAPEGA.PC_ASSIGN_WORKLIST` ditulis Pega selama masa paralel, sehingga
// yang tercatat di sini adalah permintaan beserta pemohonnya; pelaksanaannya tetap di Pega.
// Penugasannya TIDAK berpindah, dan barisnya tetap ada di layar.
//
// Itu bukan setengah jalan melainkan satu-satunya bentuk yang aman: dua sistem yang
// sama-sama memindahkan penugasan menghasilkan tugas yang hilang atau terpegang dua orang.
func (s *Service) Transfer(ctx context.Context, cmd TransferCommand) (dashboardclaim.TransferRequest, error) {
	if cmd.Caller.Login == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, ia cacat pemrograman —
		// dan mencatat permintaan tanpa pemohon menghapus satu-satunya kontrol pengimbang
		// yang tersisa (`D-59`).
		return dashboardclaim.TransferRequest{}, errors.New("dashboardclaim/usecase: identitas pemohon kosong")
	}
	if s.transfers == nil {
		return dashboardclaim.TransferRequest{}, dashboardclaim.ErrTransferUnavailable
	}

	if err := cmd.Request.Validate(); err != nil {
		return dashboardclaim.TransferRequest{}, err
	}

	// Penyimpanan dipilih menurut portal SEBELUM apa pun ditulis. Permintaan atas klaim satu
	// badan hukum yang tercatat di basis data badan hukum lain adalah kebocoran yang persis
	// `R-20` larang — dan di sini yang tercatat adalah perintah yang akan dijalankan.
	store, err := s.transfers(cmd.PortalAlias)
	if err != nil {
		return dashboardclaim.TransferRequest{}, err
	}

	id, err := s.ids.New()
	if err != nil {
		return dashboardclaim.TransferRequest{}, fmt.Errorf(
			"dashboardclaim/usecase: membangkitkan pengenal permintaan: %w", err)
	}

	request := dashboardclaim.TransferRequest{
		ID:              id,
		Scope:           cmd.Request.Scope,
		ClaimID:         cmd.Request.ClaimID,
		ClaimNumber:     cmd.Request.ClaimNumber,
		FromOperator:    cmd.Request.FromOperator,
		ToOperator:      cmd.Request.ToOperator,
		UserType:        cmd.Request.UserType,
		Reason:          cmd.Request.Reason,
		Status:          dashboardclaim.TransferPending,
		RequestedBy:     cmd.Caller.Login,
		RequestedByName: cmd.Caller.Name,
		RequestedAt:     s.clock.Now(),
	}

	if err := store.Record(ctx, request); err != nil {
		return dashboardclaim.TransferRequest{}, fmt.Errorf(
			"dashboardclaim/usecase: mencatat permintaan transfer: %w", err)
	}
	return request, nil
}

// PendingTransfers membaca permintaan tertunda atas sekumpulan klaim.
//
// Galatnya TIDAK dikembalikan sebagai kegagalan layar: tabelnya dibuat migrasi `0014` yang
// belum dijalankan DBA di lingkungan mana pun, sehingga pembacaannya gagal di setiap portal
// hari ini. Mengembalikannya sebagai galat berarti DAFTARNYA IKUT MATI — padahal daftarnya
// sendiri sudah dapat dipakai.
//
// Yang dilakukan: peta kosong dikembalikan bersama galatnya, dan pemanggil WAJIB mencatat
// galat itu. Bila pemanggil mengabaikannya, kegagalan ini menjadi tidak terlihat oleh
// siapa pun.
func (s *Service) PendingTransfers(
	ctx context.Context,
	portalAlias string,
	claimIDs []string,
) (map[string][]dashboardclaim.TransferRequest, error) {
	empty := map[string][]dashboardclaim.TransferRequest{}
	if s.transfers == nil || len(claimIDs) == 0 {
		return empty, nil
	}

	store, err := s.transfers(portalAlias)
	if err != nil {
		return empty, err
	}

	pending, err := store.PendingFor(ctx, claimIDs)
	if err != nil {
		return empty, fmt.Errorf("dashboardclaim/usecase: membaca permintaan transfer tertunda: %w", err)
	}
	return pending, nil
}
