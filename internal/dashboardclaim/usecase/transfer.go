package usecase

import (
	"context"
	"errors"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

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

// Transfer MEMINDAHKAN PIC Teknik — tidak ada antrean, persis seperti Pega.
//
// # Kenapa tanpa tabel permintaan
//
// Bentuk pertama mencatat permintaan dan menyerahkan pelaksanaannya ke Pega, atas dasar
// `P-1`. Dua kenyataan membatalkannya:
//
//  1. **Antreannya tidak punya pelaksana.** Tidak ada satu pun job Pega yang membaca tabel
//     itu; kelima job terjadwal (`D-57`) seluruhnya lebih tua daripadanya. Permintaan
//     menumpuk berstatus `menunggu`, dan pengguna menunggu sesuatu yang tidak akan datang.
//
//  2. **`P-1` tidak berlaku di jalur ini.** Assign per baris di Pega TIDAK memanggil
//     `pxTransferAssignment` — `PNC_ReassignPNCTeknik` hanya mengubah
//     `ClaimData.UserTeknis`. Antrean tugas `PC_ASSIGN_WORKLIST` tidak disentuh sama
//     sekali, sehingga tidak ada dua sistem yang berebut menulisnya.
//
// Work Owner memutuskan (2026-10-06) jalur ini mengikuti Pega apa adanya, termasuk tanpa
// tabel permintaan.
//
// # Yang hilang bersamanya, dan itu disengaja
//
// Pega tidak mencatat siapa memindahkan klaim ke siapa, dan sekarang kita pun tidak.
// `D-28` dan `D-59` menghendaki jejak atas perubahan bernilai bisnis; jejak itu TIDAK ADA
// pada jalur ini, dan ketiadaannya adalah keputusan Work Owner — bukan kelalaian.
//
// Bila kelak jejak itu dituntut audit, yang dikembalikan bukan antrean melainkan tabel log
// tersendiri yang ditulis SESUDAH pemindahan berhasil.
func (s *Service) Transfer(ctx context.Context, cmd TransferCommand) (dashboardclaim.TransferRequest, error) {
	if cmd.Caller.Login == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, ia cacat pemrograman.
		return dashboardclaim.TransferRequest{}, errors.New("dashboardclaim/usecase: identitas pemohon kosong")
	}
	if s.assignments == nil {
		return dashboardclaim.TransferRequest{}, dashboardclaim.ErrAssignmentUnavailable
	}

	if err := cmd.Request.Validate(); err != nil {
		return dashboardclaim.TransferRequest{}, err
	}

	// Penulis dipilih menurut portal SEBELUM satu kolom pun berubah. Mengubah klaim milik
	// satu badan hukum di basis data badan hukum lain bukan sekadar menampilkan yang keliru
	// (`R-20`) melainkan MENGUBAH yang keliru.
	writer, err := s.assignments(cmd.PortalAlias)
	if err != nil {
		return dashboardclaim.TransferRequest{}, err
	}

	hasil := dashboardclaim.TransferRequest{
		Scope:           cmd.Request.Scope,
		ClaimID:         cmd.Request.ClaimID,
		ClaimNumber:     cmd.Request.ClaimNumber,
		FromOperator:    cmd.Request.FromOperator,
		ToOperator:      cmd.Request.ToOperator,
		UserType:        cmd.Request.UserType,
		Reason:          cmd.Request.Reason,
		Status:          dashboardclaim.TransferExecuted,
		RequestedBy:     cmd.Caller.Login,
		RequestedByName: cmd.Caller.Name,
		RequestedAt:     s.clock.Now(),
	}

	if cmd.Request.Scope == dashboardclaim.TransferFilter {
		// "Select All" — seluruh klaim yang cocok penyaring layar, lintas halaman.
		//
		// Penyaringnya datang dari permintaan, bukan disusun di sini: ia harus sama persis
		// dengan yang dipakai daftar, atau tombolnya memindahkan himpunan yang berbeda dari
		// yang dilihat pengguna — tanpa galat apa pun.
		dipindah, err := writer.MoveAllMatching(ctx, cmd.Request.Filter, cmd.Request.ToOperator)
		if err != nil {
			return dashboardclaim.TransferRequest{}, err
		}
		hasil.MovedCount = dipindah
		return hasil, nil
	}

	if cmd.Request.Scope == dashboardclaim.TransferBulk {
		// "Transfer All Case By UserID" — seluruh pekerjaan satu operator sekaligus.
		//
		// Satu pernyataan, bukan perulangan per klaim: jumlah klaim seorang petugas tidak
		// diketahui di muka, dan memutarnya satu per satu membuat kegagalan di tengah
		// meninggalkan sebagian berpindah dan sebagian tidak.
		dipindah, err := writer.MoveAllForOperator(ctx,
			cmd.Request.FromOperator, cmd.Request.ToOperator)
		if err != nil {
			return dashboardclaim.TransferRequest{}, err
		}
		hasil.MovedCount = dipindah
		return hasil, nil
	}

	moved, err := writer.MovePIC(ctx, dashboardclaim.PICMove{
		ClaimID:     cmd.Request.ClaimID,
		ClaimNumber: cmd.Request.ClaimNumber,
		ToOperator:  cmd.Request.ToOperator,
	})
	if err != nil {
		return dashboardclaim.TransferRequest{}, err
	}

	// PIC lama dibaca dari hasil pemindahan, bukan dari layar: layar memegang nilai yang
	// dibacanya saat halaman dimuat, dan nilai itu dapat sudah berubah.
	hasil.FromOperator = moved.FromOperator
	hasil.MovedCount = 1
	return hasil, nil
}

// TechnicalPICQuery adalah permintaan daftar PIC Teknik untuk layar Transfer.
type TechnicalPICQuery struct {
	PortalAlias string

	// BusinessType TIDAK lagi menyaring apa pun — lihat `picteknik.sql`.
	//
	// Ia dibiarkan ada supaya pemanggil tidak perlu berubah saat penyaringnya kelak
	// dikembalikan bersama master pemetaan pengguna ke lini bisnis (`F-4`).
	BusinessType string

	Search string
	Limit  int
	Offset int
}

// TechnicalPICResult membawa satu halaman daftar PIC beserta portal asalnya.
//
// Portalnya ikut dikembalikan, bukan diasumsikan: layar menampilkannya, dan daftar petugas
// satu badan hukum yang terbaca di portal badan hukum lain adalah persis `R-20`.
type TechnicalPICResult struct {
	Page   dashboardclaim.TechnicalPICPage
	Portal string
}

// TechnicalPIC membaca daftar PIC Teknik pada portal yang sedang dibuka.
func (s *Service) TechnicalPIC(
	ctx context.Context,
	q TechnicalPICQuery,
) (TechnicalPICResult, error) {
	if s.picReaders == nil {
		return TechnicalPICResult{}, dashboardclaim.ErrAssignmentUnavailable
	}

	reader, err := s.picReaders(q.PortalAlias)
	if err != nil {
		return TechnicalPICResult{}, err
	}

	page, err := reader.ListTechnicalPIC(ctx, dashboardclaim.TechnicalPICFilter{
		BusinessType: q.BusinessType,
		Search:       q.Search,
		Limit:        q.Limit,
		Offset:       q.Offset,
	})
	if err != nil {
		return TechnicalPICResult{}, err
	}

	return TechnicalPICResult{Page: page, Portal: q.PortalAlias}, nil
}
