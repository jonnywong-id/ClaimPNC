package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/inboxinvestigator"
)

// OpenInvestigation menyiapkan formulir investigasi satu pekerjaan.
//
// # Tanggal Investigasi diisi DI SINI, bukan di layar
//
// `Activity/PresetInvestigation-Act.xml` adalah pra-proses Flow Action `InputInvestigator`,
// dan isinya satu langkah:
//
//	.ClaimData.SurveyResults(1).SurveyList(1).TanggalInvestigasi := @CurrentDateTime()
//
// Jadi tanggalnya terbit saat formulir DIBUKA, bukan saat disimpan. Menaruhnya di layar
// akan membuat nilainya bergantung pada jam mesin pengguna; menaruhnya di simpan akan
// mengubah artinya menjadi "tanggal selesai", yang bukan arti aslinya.
//
// # Ia hanya diisi bila BELUM pernah ada
//
// Formulir yang dibuka kembali untuk dikoreksi mempertahankan tanggal investigasi
// pertamanya. Menimpanya setiap kali dibuka akan membuat tanggal itu selalu menunjuk hari
// ini, dan kolom "Tanggal Investigasi" pada berkas ekspor kehilangan maknanya.
func (s *Service) OpenInvestigation(
	ctx context.Context,
	portalAlias, claimRef string,
) (inboxinvestigator.Investigation, error) {
	repo, err := s.investigationSelector(portalAlias)
	if err != nil {
		return inboxinvestigator.Investigation{}, err
	}

	one, found, err := repo.Load(ctx, claimRef)
	if err != nil {
		return inboxinvestigator.Investigation{}, fmt.Errorf(
			"membuka formulir investigasi: %w", err)
	}

	if !found {
		one = inboxinvestigator.Investigation{ClaimRef: claimRef}
	}
	one.ClaimRef = claimRef
	if one.SurveyIndex < 1 {
		one.SurveyIndex = 1
	}
	if one.Index < 1 {
		one.Index = 1
	}
	if one.InvestigatedAt == nil {
		now := s.now()
		one.InvestigatedAt = &now
	}
	return one, nil
}

// SubmitInvestigation menyimpan formulir dan memindahkan klaimnya ke Analyst.
//
// # Apa yang terjadi, dan kenapa itu satu peristiwa
//
// `Activity/SetStatusInvestigator_Act-Act.xml` menyetel status lalu menyimpan objek
// kerjanya. Di sini keduanya satu transaksi (lihat InvestigationRepo.Save), karena klaim
// yang berpindah tanpa hasil investigasinya tersimpan tidak dapat ditindaklanjuti siapa
// pun.
//
// # Pekerjaan HILANG dari antrean setelah ini
//
// Itu perilaku yang benar — ciri kedua Inbox pada `D-79`. Transition dikembalikan supaya
// lapisan transport dapat menyebutkannya kepada pengguna; tanpa itu, barisnya terbaca
// seperti data yang lenyap.
//
// # Yang TIDAK dikerjakan di sini, dan itu disengaja
//
// Sistem lama juga memanggil `InsertHistoryClaimPNC` dan `InsertJsonClaimNonMBU_act` —
// pencatatan riwayat dan penyalinan ulang dokumen JSON klaim. Keduanya menulis ke tabel
// yang **dimiliki Pega** selama masa paralel (`P-1`), sehingga tidak dibawa. Jejak
// perubahan nilai bisnis adalah tugas modul `S-5` Jejak Audit, yang belum dibangun —
// dicatat, bukan dikarang.
func (s *Service) SubmitInvestigation(
	ctx context.Context,
	portalAlias string,
	form inboxinvestigator.Investigation,
	inpatient bool,
	by string,
) (inboxinvestigator.Transition, error) {
	repo, err := s.investigationSelector(portalAlias)
	if err != nil {
		return inboxinvestigator.Transition{}, err
	}

	cleaned := form.Clean(inpatient)
	if err := cleaned.Validate(); err != nil {
		return inboxinvestigator.Transition{}, err
	}

	now := s.now()
	if cleaned.InvestigatedAt == nil {
		cleaned.InvestigatedAt = &now
	}

	move := inboxinvestigator.Transition{
		ClaimRef:     cleaned.ClaimRef,
		SurveyStatus: inboxinvestigator.SurveyStatusInvestigated,
		PNCStatus:    inboxinvestigator.PNCStatusInvestigated,
		ClaimStatus:  inboxinvestigator.ClaimStatusAnalyst,
		At:           now,
	}

	if err := repo.Save(ctx, cleaned, move, by); err != nil {
		if errors.Is(err, inboxinvestigator.ErrInvestigationStoreMissing) ||
			errors.Is(err, inboxinvestigator.ErrInvestigationInvalid) {
			return inboxinvestigator.Transition{}, err
		}
		return inboxinvestigator.Transition{}, fmt.Errorf(
			"menyimpan hasil investigasi: %w", err)
	}
	return move, nil
}

// now mengembalikan waktu kini lewat seam Clock bila ada, atau jam sistem bila tidak.
//
// # Kenapa modul ini akhirnya punya jam, padahal sempat dinyatakan tidak perlu
//
// Catatan lama pada paket ini menyatakan modul ini tidak bergantung pada jam dinding, dan
// itu benar SELAMA ia hanya membaca. Formulir investigasi mengubahnya: tiga nilai lahir
// dari waktu kini — Tanggal Investigasi, `InvestTfDate`, dan `AnalystTransferDate`.
//
// Jamnya dapat diganti supaya ketiganya dapat diuji tanpa menunggu hari berganti, dan
// supaya satu permintaan tidak dapat menghasilkan tiga waktu yang berbeda sepersekian
// detik.
func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now().UTC()
}
