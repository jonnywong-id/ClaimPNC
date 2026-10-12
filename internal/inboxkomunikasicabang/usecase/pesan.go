package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Branches mengembalikan daftar cabang yang dapat dipilih sebagai tujuan pesan baru.
//
// # Kenapa ia TIDAK menurunkan batas cabang pemanggil
//
// Karena yang dibatasi adalah percakapan, bukan daftar cabang. Petugas cabang Surabaya boleh
// mengirim pesan ke cabang Bandung; yang tidak boleh adalah MEMBACA percakapan cabang
// Bandung dengan pihak lain.
//
// Menurunkannya di sini juga akan menembus DB Link untuk sebuah daftar yang tidak
// bergantung pada hasilnya — satu perjalanan yang tidak mengubah apa pun.
func (s *Service) Branches(
	ctx context.Context,
	portalAlias string,
	caller inboxkomunikasicabang.Caller,
) ([]inboxkomunikasicabang.BranchOption, error) {
	if caller.Clean().Login == "" {
		return nil, inboxkomunikasicabang.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	branches, err := repo.Branches(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca daftar cabang: %w", err)
	}

	return branches, nil
}

// SendMessage membuat percakapan BARU — tombol "Kirim Pesan".
//
// # Cabang ASAL diturunkan di sini, bukan diterima dari layar
//
// Ini yang membuat kolom `COMMUNICATE_FROM` dapat dipercaya. Ia menentukan siapa melihat
// percakapannya; menerimanya dari permintaan berarti pengirim dapat menyatakan dirinya
// berasal dari cabang mana pun dan menaruh percakapan di kotak masuk yang bukan haknya
// (`R-20`).
//
// Penurunannya memakai BranchResolver yang sama dengan daftar, sehingga pesan yang dikirim
// seorang petugas pasti tampil di daftarnya sendiri — asal dan penyaring berasal dari satu
// sumber.
//
// # Petugas yang cabangnya TIDAK terbaca tetap dapat mengirim
//
// Ia dilayani sebagai kantor pusat (`P-5`), persis seperti pada pembacaan. Menolaknya di
// sini akan membuat layar yang dapat DIBACA tetapi tidak dapat DIPAKAI — keadaan yang jauh
// lebih membingungkan daripada daftar yang isinya kantor pusat.
//
// Keadaan itu tetap DICATAT pada jejak, karena di sini akibatnya melekat permanen: pesan
// yang terkirim membawa asal `"1"` selamanya, dan tidak ada layar yang dapat mengubahnya.
func (s *Service) SendMessage(
	ctx context.Context,
	portalAlias string,
	caller inboxkomunikasicabang.Caller,
	input inboxkomunikasicabang.NewMessageInput,
) (string, error) {
	cleanCaller := caller.Clean()

	command, err := inboxkomunikasicabang.NewMessageCommandOf(input, cleanCaller, s.clock.Now())
	if err != nil {
		return "", err
	}

	filter, err := s.resolveBranch(ctx, cleanCaller)
	if err != nil {
		return "", err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return "", err
	}

	id, err := repo.SendMessage(ctx, command, filter.Code)
	if err != nil {
		return "", fmt.Errorf("mengirim pesan baru: %w", err)
	}

	if s.logger != nil {
		s.logger.Info(
			"pesan komunikasi cabang baru dikirim",
			slog.String("modul", "inbox-komunikasi-cabang"),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias),
			slog.String("komunikasi", id),
			slog.String("asal", filter.Code),
			slog.Bool("cabang_terbaca", filter.Resolved),
			slog.String("tujuan", command.RecipientCode()),
			// Isi pesannya TIDAK dicatat — ia percakapan tentang klaim nasabah, dan log
			// disimpan lebih longgar daripada basis data (`11-CROSSCUTTING.md` §2.4).
			slog.Int("panjang_pesan", len([]rune(command.Message))),
		)
	}

	return id, nil
}
