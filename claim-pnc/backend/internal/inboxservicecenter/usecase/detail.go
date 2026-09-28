package usecase

import (
	"context"
	"log/slog"

	"claim-pnc/internal/inboxservicecenter"
)

// Detailed adalah satu klaim beserta riwayat progresnya.
type Detailed struct {
	// Claim adalah seluruh isian layar rincian.
	Claim inboxservicecenter.ClaimDetail

	// Progress adalah riwayat catatan progres, terlama di atas.
	//
	// Senarai kosong berarti klaimnya belum pernah dicatat progresnya — keadaan biasa pada
	// tab Registrasi SC, bukan kegagalan.
	Progress []inboxservicecenter.ProgressNote

	// Groups adalah ketujuh kelompok isian beserta urutannya.
	Groups []inboxservicecenter.FieldGroup
}

// Detail mengambil satu klaim beserta riwayat progresnya.
//
// # Kenapa riwayatnya diambil dengan RepairID, bukan dengan ID yang diminta
//
// Karena `POOLDATA.PROGRESS_SERVICECENTER_CLAIM` dikunci `REPAIRID`, dan itu kolom yang
// BERBEDA dari `ID`. Keduanya sama-sama terbaca sebagai "nomor klaim" oleh mata yang membaca
// sekilas, dan menukarnya menghasilkan riwayat yang selalu kosong tanpa satu pun galat.
//
// RepairID karena itu diambil dari klaim yang SUDAH ditemukan, bukan dari parameter — dengan
// begitu ia tidak mungkin milik klaim lain.
//
// # Kenapa riwayat yang gagal TIDAK menggagalkan seluruh permintaan
//
// Karena rincian klaim tetap berguna tanpa riwayatnya, sedangkan riwayat tanpa rincian tidak
// berguna sama sekali. Kegagalan mengambil riwayat dicatat di log dan layar menampilkan
// rinciannya; menggagalkan keduanya berarti satu tabel pendukung dapat menutup seluruh layar.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxservicecenter.Caller,
	id string,
) (Detailed, error) {
	query, err := inboxservicecenter.NewDetailQuery(id, caller)
	if err != nil {
		return Detailed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Detailed{}, err
	}

	claim, err := repo.FindDetail(ctx, query)
	if err != nil {
		// ErrNotFound diteruskan apa adanya supaya transport dapat memetakannya ke 404;
		// membungkusnya akan membuat errors.Is di sana tetap bekerja, tetapi pesannya
		// menyebut kueri — dan itu bocoran yang tidak perlu.
		return Detailed{}, err
	}

	result := Detailed{
		Claim:    claim,
		Progress: []inboxservicecenter.ProgressNote{},
		Groups:   inboxservicecenter.DetailGroups(),
	}

	progress, err := repo.ListProgress(ctx, claim.RepairID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn(
				"riwayat progres Inbox Service Center tidak dapat dibaca",
				slog.String("id", claim.ID),
				slog.String("repair_id", claim.RepairID),
				slog.String("galat", err.Error()),
			)
		}
		return result, nil
	}

	result.Progress = progress
	return result, nil
}
