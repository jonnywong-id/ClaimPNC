package usecase

import (
	"context"
	"errors"
	"log/slog"

	"claim-pnc/internal/inboxrclpucl"
)

// ErrReminderUnavailable menyatakan pengirim pengingat belum terpasang.
//
// Ia dijawab ke layar sebagai galat yang terbaca, bukan panik: peladen SMTP belum tentu
// terkonfigurasi di setiap lingkungan, dan antrean tetap harus dapat dibuka tanpanya.
var ErrReminderUnavailable = errors.New(
	"inboxrclpucl/usecase: pengirim Reminder PUCL belum terpasang")

// SendReminder mengirim pengingat kelengkapan dokumen untuk klaim yang dipilih.
//
// # Ia TIDAK gagal seluruhnya karena satu klaim gagal
//
// Pengiriman sebagian adalah hasil yang LAZIM di sini: sebagian klaim punya alamat surel,
// sebagian belum. Satu kalimat "berhasil" atau "gagal" untuk seluruh permintaan
// menyembunyikan tepat hal yang perlu ditindak — klaim mana yang belum punya alamat.
//
// Karena itu hasilnya selalu PER KLAIM, dan galat hanya dikembalikan untuk kegagalan yang
// menyangkut seluruh permintaan: permintaan kosong, terlalu banyak, pengirim belum
// terpasang, atau basis data tidak terbaca.
//
// # Nomor klaim dibaca ULANG di sini
//
// Alamat surel dan nama tertanggung TIDAK diterima dari permintaan melainkan dibaca dari
// basis data lewat Repo.ReminderRecipients, yang sekaligus menegakkan penyaring antrean
// Kelengkapan Dokumen. Ini tindakan yang mengirim sesuatu keluar; memercayai alamat dari
// badan permintaan berarti siapa pun yang dapat memanggil API dapat memakai layar ini
// sebagai pengirim surel.
func (s *Service) SendReminder(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	claimNumbers []string,
) (inboxrclpucl.ReminderResult, error) {
	wanted, err := inboxrclpucl.NormalizeReminderRequest(claimNumbers)
	if err != nil {
		return inboxrclpucl.ReminderResult{}, err
	}

	if s.reminders == nil {
		return inboxrclpucl.ReminderResult{}, ErrReminderUnavailable
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxrclpucl.ReminderResult{}, err
	}

	recipients, err := repo.ReminderRecipients(ctx, wanted)
	if err != nil {
		return inboxrclpucl.ReminderResult{}, err
	}

	// Dipetakan menurut nomor klaim supaya nomor yang TIDAK kembali dari basis data tetap
	// mendapat barisnya sendiri di hasil. Tanpa itu, klaim yang tidak memenuhi syarat
	// hilang diam-diam dari laporan — dan petugas mengira pengingatnya terkirim.
	found := make(map[string]inboxrclpucl.ReminderRecipient, len(recipients))
	for _, recipient := range recipients {
		found[recipient.ClaimNumber] = recipient
	}

	result := inboxrclpucl.ReminderResult{
		Outcomes: make([]inboxrclpucl.ReminderOutcome, 0, len(wanted)),
	}

	for _, number := range wanted {
		recipient, ok := found[number]
		switch {
		case !ok:
			result.Outcomes = append(result.Outcomes, inboxrclpucl.ReminderOutcome{
				ClaimNumber: number,
				Status:      inboxrclpucl.ReminderNotEligible,
			})
			continue

		case recipient.Email == "":
			result.Outcomes = append(result.Outcomes, inboxrclpucl.ReminderOutcome{
				ClaimNumber: number,
				Status:      inboxrclpucl.ReminderNoEmail,
			})
			continue
		}

		if err := s.reminders.SendReminder(ctx, recipient); err != nil {
			// Galat MENTAH tidak diteruskan ke layar: ia dapat memuat nama host internal
			// peladen SMTP. Yang dicatat lengkap adalah lognya, yang dibaca tim teknis.
			s.logReminderFailure(caller, recipient, err)

			result.Outcomes = append(result.Outcomes, inboxrclpucl.ReminderOutcome{
				ClaimNumber: number,
				Status:      inboxrclpucl.ReminderFailed,
				Reason:      reminderFailureReason(err),
			})
			continue
		}

		result.Outcomes = append(result.Outcomes, inboxrclpucl.ReminderOutcome{
			ClaimNumber: number,
			Status:      inboxrclpucl.ReminderSent,
		})
	}

	return result, nil
}

// reminderFailureReason menyusun kalimat yang dibaca petugas dari sebuah kegagalan.
//
// # Kenapa dibedakan, dan bukan satu kalimat untuk semuanya
//
// Karena "tidak dapat dihubungi" dan "ditolak" menuntut tindakan yang berbeda, dan menyebut
// keduanya dengan kalimat yang sama menyuruh orang mencari di tempat yang keliru.
//
// Itu benar-benar terjadi 2026-10-12. Layar berbunyi "Pengiriman ditolak peladen surel"
// padahal nama host peladennya — `SMTP_HOST` — tidak teresolusi sama sekali di jaringan
// tempat aplikasi berjalan. Peladennya tidak pernah dihubungi, apalagi menolak. Yang dicari
// orang menjadi akun surel dan alamat tertanggung, padahal yang salah satu baris
// konfigurasi.
//
// Galat MENTAH tetap tidak diteruskan: ia dapat memuat nama host internal. Yang berbeda
// hanyalah KELASNYA, dan kelas itu sudah cukup mengarahkan.
func reminderFailureReason(err error) string {
	switch {
	case errors.Is(err, inboxrclpucl.ErrReminderUnreachable):
		return "Peladen surel tidak dapat dihubungi. Ini soal konfigurasi atau jaringan, " +
			"bukan soal klaim ini — laporkan ke tim teknis."

	case errors.Is(err, inboxrclpucl.ErrReminderRejected):
		return "Peladen surel menolak pengiriman. Laporkan ke tim teknis."

	default:
		return "Pengiriman gagal. Coba lagi beberapa saat lagi; bila berulang, " +
			"laporkan ke tim teknis."
	}
}

// logReminderFailure mencatat kegagalan satu pengiriman.
//
// Alamat surelnya TIDAK ikut dicatat. `D-69` melarang data nasabah ditulis ke artefak yang
// disimpan, dan log termasuk di dalamnya — Cross-Cutting §2.4 menyebut alamat secara
// eksplisit. Nomor klaimnya cukup untuk menelusuri.
func (s *Service) logReminderFailure(
	caller inboxrclpucl.Caller,
	recipient inboxrclpucl.ReminderRecipient,
	err error,
) {
	if s.logger == nil {
		return
	}
	s.logger.Error("reminder pucl gagal dikirim",
		slog.String("nomor_klaim", recipient.ClaimNumber),
		slog.String("pemanggil", caller.Login),
		slog.Any("galat", err),
	)
}
