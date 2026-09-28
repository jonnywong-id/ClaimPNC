package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"claim-pnc/internal/inboxpladla"
)

// Berkas ini mengorkestrasi layar RINCIAN satu klaim — tombol **"Detail Claim"**.
//
// # Satu aturan berlaku pada SELURUH operasi di sini
//
// Setiap satunya berangkat dari inboxpladla.DetailScope, dan scope itu selalu disusun
// scopeOf — tidak pernah dirakit di tempat. Scope memuat kunci klaim, login pemanggil, dan
// kode reasuradurnya; ketiganya bersama-sama itulah yang menentukan apa yang boleh terbaca.
//
// Alasannya bukan kerapian. Seluruh alamat di layar ini dapat dipanggil langsung, dan kunci
// klaim berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx` — sebuah pola yang dapat ditebak. Tanpa
// batas reasuradur DI DALAM setiap pernyataan, satu tebakan yang beruntung membuka
// pemberitahuan, dokumen, dan percakapan milik mitra lain.

// Detail adalah seluruh isi layar rincian satu klaim.
type Detail struct {
	Claim inboxpladla.ClaimDetail

	// AdviceColumnsPLA dan AdviceColumnsDLA adalah kolom kedua grid pemberitahuan.
	//
	// Dikirim bersama isinya, sama seperti kolom daftar di layar induk: inventaris kolom
	// adalah hasil pembacaan export, dan menyalinnya ke layar berarti keputusan yang sama
	// hidup di dua tempat.
	AdviceColumnsPLA []inboxpladla.Column
	AdviceColumnsDLA []inboxpladla.Column

	DocumentColumns     []inboxpladla.Column
	ConversationColumns []inboxpladla.Column
}

// Detail mengambil seluruh isi layar rincian satu klaim.
//
// # Kenapa keempat bagiannya diambil SEKALIGUS
//
// Karena layar rincian menggambar keempatnya bersamaan, dan memisahkannya menjadi empat
// permintaan berarti empat kali perjalanan untuk satu layar yang tidak dapat dipakai
// sebelum keempatnya tiba. `10-API-STRATEGY.md` §1 menuntut satu layar dilayani satu
// permintaan, justru supaya layar baru tidak terasa lebih lambat daripada Pega.
//
// Yang TIDAK ikut adalah dokumen: ia baru diambil setelah pengguna menekan "Dokumen" pada
// salah satu baris, karena kuncinya adalah nomor pemberitahuan itu — bukan klaimnya.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	claimKey string,
) (Detail, error) {
	repo, scope, err := s.scopeOf(ctx, portalAlias, caller, claimKey)
	if err != nil {
		return Detail{}, err
	}

	header, err := repo.ClaimHeader(ctx, scope)
	if err != nil {
		return Detail{}, fmt.Errorf("mengambil keterangan klaim: %w", err)
	}

	pla, err := repo.Advices(ctx, scope, inboxpladla.AdviceKindPLA)
	if err != nil {
		return Detail{}, fmt.Errorf("mengambil daftar PLA: %w", err)
	}

	dla, err := repo.Advices(ctx, scope, inboxpladla.AdviceKindDLA)
	if err != nil {
		return Detail{}, fmt.Errorf("mengambil daftar DLA: %w", err)
	}

	conversations, err := repo.Conversations(ctx, scope)
	if err != nil {
		return Detail{}, fmt.Errorf("mengambil riwayat komunikasi: %w", err)
	}

	// SETIAP pembukaan rincian dicatat, dan di sini jejaknya lebih penting daripada di
	// layar induk: rincian memuat NILAI UANG per pemberitahuan dan ISI PERCAKAPAN, bukan
	// sekadar daftar nomor klaim.
	//
	// Nomor klaimnya TIDAK dicatat — ia data nasabah (`D-69`). Yang dicatat adalah kunci
	// kerjanya, yang memang sudah dipakai sebagai kunci teknis di seluruh log aplikasi.
	if s.logger != nil {
		s.logger.Info(
			"rincian klaim dibuka reasuradur",
			slog.String("modul", "inbox-pla-dla"),
			slog.String("pemanggil", scope.Login),
			slog.Any("kode_reasuradur", scope.ReinsurerCodes),
			slog.String("portal", portalAlias),
			slog.Int("pla", len(pla)),
			slog.Int("dla", len(dla)),
			slog.Int("percakapan", len(conversations)),
		)
	}

	return Detail{
		Claim: inboxpladla.ClaimDetail{
			Header:        header,
			PLA:           pla,
			DLA:           dla,
			Conversations: conversations,
		},
		AdviceColumnsPLA:    inboxpladla.AdviceColumns(inboxpladla.AdviceKindPLA),
		AdviceColumnsDLA:    inboxpladla.AdviceColumns(inboxpladla.AdviceKindDLA),
		DocumentColumns:     inboxpladla.DocumentColumns(),
		ConversationColumns: inboxpladla.ConversationColumns(),
	}, nil
}

// Documents mengambil dokumen satu nomor pemberitahuan.
//
// Ia meniru tombol **"Dokumen"** pada baris PLA atau DLA — yang di Pega menetapkan
// `TempFilePLADLA.NO_DLA` dan `.DLAType` lalu menjalankan ulang `SetViewAttachmentReas`.
func (s *Service) Documents(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	claimKey string,
	adviceNo string,
	kind inboxpladla.AdviceKind,
) ([]inboxpladla.DocumentRow, error) {
	if !kind.Valid() {
		return nil, inboxpladla.ErrAdviceKindUnknown
	}

	repo, scope, err := s.scopeOf(ctx, portalAlias, caller, claimKey)
	if err != nil {
		return nil, err
	}

	rows, err := repo.Documents(ctx, scope, adviceNo, kind)
	if err != nil {
		return nil, fmt.Errorf("mengambil dokumen %s: %w", kind, err)
	}
	return rows, nil
}

// DocumentContent mengambil ISI satu dokumen.
//
// # Kenapa ia DICATAT, sementara Documents tidak
//
// Karena yang satu memperlihatkan bahwa sebuah berkas ada, dan yang lain menyerahkan
// isinya ke luar perusahaan. Hanya yang kedua yang tidak dapat ditarik kembali.
func (s *Service) DocumentContent(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	claimKey string,
	documentID string,
) (inboxpladla.DocumentContent, error) {
	repo, scope, err := s.scopeOf(ctx, portalAlias, caller, claimKey)
	if err != nil {
		return inboxpladla.DocumentContent{}, err
	}

	content, err := repo.DocumentContent(ctx, scope, documentID)
	if err != nil {
		return inboxpladla.DocumentContent{}, err
	}

	if s.logger != nil {
		s.logger.Info(
			"dokumen klaim diunduh reasuradur",
			slog.String("modul", "inbox-pla-dla"),
			slog.String("pemanggil", scope.Login),
			slog.Any("kode_reasuradur", scope.ReinsurerCodes),
			slog.String("portal", portalAlias),
			slog.String("dokumen", documentID),
			slog.Int("bita", len(content.Content)),
		)
	}

	return content, nil
}

// Reply menyimpan balasan pihak luar atas satu percakapan.
//
// # Ia SATU-SATUNYA operasi yang menulis di modul ini
//
// Urutannya sengaja: perintahnya divalidasi LEBIH DULU, sebelum satu pun perjalanan ke
// basis data. Balasan kosong atau kelewat panjang ditolak tanpa menyentuh tabel yang
// dibaca petugas internal.
func (s *Service) Reply(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	claimKey string,
	input inboxpladla.ReplyInput,
	now time.Time,
) error {
	command, err := inboxpladla.NewReplyCommand(claimKey, input, caller, now)
	if err != nil {
		return err
	}

	repo, scope, err := s.scopeOf(ctx, portalAlias, caller, claimKey)
	if err != nil {
		return err
	}

	if err := repo.Reply(ctx, scope, command); err != nil {
		return err
	}

	// Dicatat SETELAH berhasil, bukan sebelum.
	//
	// Jejak yang ditulis lebih dulu akan mencatat balasan yang ditolak pemagarannya
	// sebagai balasan yang terjadi — dan pada tulisan yang pelakunya pihak luar, jejak
	// yang mengaku lebih banyak daripada yang terjadi lebih buruk daripada tidak ada
	// jejak.
	//
	// ISI balasannya TIDAK dicatat. Ia tulisan manusia yang dapat memuat apa saja,
	// termasuk nomor polis dan nama tertanggung (`D-69`); yang dicatat adalah panjangnya,
	// yang cukup untuk membuktikan sesuatu memang terkirim.
	if s.logger != nil {
		s.logger.Info(
			"balasan komunikasi ditulis reasuradur",
			slog.String("modul", "inbox-pla-dla"),
			slog.String("pemanggil", command.Replier.Login),
			slog.Any("kode_reasuradur", scope.ReinsurerCodes),
			slog.String("portal", portalAlias),
			slog.String("percakapan", command.ConversationID),
			slog.Int("panjang_balasan", len([]rune(command.Message))),
		)
	}

	return nil
}

// RejectAction menjawab tombol yang belum dibangun.
//
// Yang tersisa hanya "Download ALL PLA" dan "Download ALL DLA" — lihat
// inboxpladla.Action.
func (s *Service) RejectAction(action string) error {
	return inboxpladla.NewNotAvailable(action)
}

// scopeOf menyusun batas yang berlaku pada seluruh operasi rincian.
//
// Ia satu-satunya tempat DetailScope dirakit. Merakitnya di tiap operasi berarti lima
// kesempatan melupakan salah satu bagiannya — dan yang dilupakan tidak menghasilkan galat,
// hanya data mitra lain yang terbuka.
func (s *Service) scopeOf(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	claimKey string,
) (inboxpladla.Repo, inboxpladla.DetailScope, error) {
	repo, codes, err := s.reinsurerOf(ctx, portalAlias, caller)
	if err != nil {
		return nil, inboxpladla.DetailScope{}, err
	}

	scope := inboxpladla.DetailScope{
		ClaimKey:       claimKey,
		Login:          caller.Clean().Login,
		ReinsurerCodes: codes,
	}.Clean()

	if scope.ClaimKey == "" {
		return nil, inboxpladla.DetailScope{}, inboxpladla.ErrRowNotFound
	}

	return repo, scope, nil
}
