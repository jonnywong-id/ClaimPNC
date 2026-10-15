package main

import (
	"log/slog"

	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/inboxmanager"
	inboxmanagernotification "claim-pnc/internal/inboxmanager/notification"
	inboxmanagerdocumentlink "claim-pnc/internal/inboxmanager/repo/documentlink"
	inboxmanagerusecase "claim-pnc/internal/inboxmanager/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/registrasi"
)

// newInboxManagerService menyusun layanan Inbox Manager.
//
// Ia dipanggil DUA KALI — pola yang sama dengan Master Panel dan Master Bengkel:
// buildExtraServices menyusunnya tanpa linker lampiran karena documentService belum ada, lalu
// assembleInboxManager menyusunnya ulang setelah documentService ada dan menimpa
// extra.inboxManager. Linker dan notifier boleh nil (lihat usecase.Options).
func newInboxManagerService(
	store storage,
	logger *slog.Logger,
	linker inboxmanager.DocumentLinker,
	notifier inboxmanager.PaymentNotifier,
) (*inboxmanagerusecase.Service, error) {
	return inboxmanagerusecase.NewService(inboxmanagerusecase.Options{
		RepoSelector:         store.extra.inboxManager,
		LineBusinessSelector: store.extra.inboxManagerLine,

		// Clock menentukan bulan bawaan tab Produktivitas Klaim. Tab itu tidak dapat berjalan
		// tanpa periode — kedelapan pencacahnya dibangun dari perbandingan dua periode.
		Clock: clock.System{},

		// Logger WAJIB terisi. Modul ini MENULIS, dan `D-59` menjadikan jejak audit
		// satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas formal: siapa pun
		// yang dapat membuka layar ini dapat menyetujui pengajuan yang boleh jadi diajukannya
		// sendiri. Lihat inboxmanager/usecase.logDecision.
		Logger: logger,

		// Tipe Pembayaran pada panel detail Payment Klaim Akseptasi dinamai aturan modul
		// Registrasi — satu sumber untuk kode 1-6 di seluruh aplikasi.
		PaymentRules:    inboxmanager.PaymentRules{PaymentTypeName: registrasi.PaymentTypeName},
		DocumentLinker:  linker,
		PaymentNotifier: notifier,
	})
}

// assembleInboxManager menyusun ulang layanan Inbox Manager lengkap dengan linker lampiran
// (modul Dokumen Penunjang — alamat diperpanjang bila lewat EXPDATE) dan pengirim surel SIMPAN.
func assembleInboxManager(
	cfg config.Config,
	store storage,
	logger *slog.Logger,
	documentService *dokumenpenunjangusecase.Service,
) (*inboxmanagerusecase.Service, error) {
	return newInboxManagerService(store, logger,
		inboxmanagerdocumentlink.New(documentService),
		buildInboxManagerPaymentNotifier(cfg, store, logger))
}

// buildInboxManagerPaymentNotifier menyusun pengirim surel SIMPAN Payment Klaim Akseptasi, atau
// nil bila SMTP belum siap. Nil SAH: SIMPAN tetap tertulis, dan hasilnya menyatakan surel tidak
// terkirim.
func buildInboxManagerPaymentNotifier(
	cfg config.Config,
	store storage,
	logger *slog.Logger,
) inboxmanager.PaymentNotifier {
	smtpConfig := inboxmanagernotification.Config{
		Account:       emailAccountSource(store),
		Host:          cfg.SMTP.Host,
		Port:          cfg.SMTP.Port,
		User:          cfg.SMTP.User,
		Password:      cfg.SMTP.Password,
		From:          cfg.SMTP.From,
		Timeout:       cfg.SMTP.Timeout,
		SkipTLSVerify: cfg.SMTP.SkipTLSVerify,
	}
	if !smtpConfig.Complete() {
		logger.Warn("surel SIMPAN Payment Klaim Akseptasi tidak aktif",
			slog.String("akibat", "keputusan tetap tersimpan, tetapi PIC tidak diberi tahu lewat surel"),
			slog.String("perbaikan", "isi POOLDATA.M_EMAIL_SERVER_PNC atau SMTP_*"))
		return nil
	}
	return inboxmanagernotification.NewSender(smtpConfig)
}
