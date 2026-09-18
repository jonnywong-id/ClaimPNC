package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/masterrekening"
)

// Committee adalah anggota komite yang memutuskan.
type Committee struct {
	Identity string
	Name     string

	// Email dipakai sebagai tujuan peringatan bila pendaftaran ke Kasir gagal.
	// Boleh kosong: non-karyawan tidak punya surel di POOLDATA.M_LOGIN_PNC, dan
	// peringatan yang tidak punya tujuan dilewati, bukan menggagalkan apa pun.
	Email string
}

// Decision adalah apa yang decided komite atas satu rekening.
type Decision struct {
	// Status wajib StatusApproved atau StatusRejected. StatusPending ditolak —
	// "belum memutuskan" bukan sebuah keputusan.
	Status masterrekening.ApprovalStatus

	// Note adalah keterangan approval atasan. Wajib terisi saat menyetujui.
	Note string

	// DocumentID mengisi lampiran buku rekening bila belum terisi saat pengajuan.
	DocumentID string
}

// portalsRegisteredWithCashier menyebut portal yang rekening setujuannya diteruskan ke
// sistem Kasir.
//
// Daftarnya diambil apa adanya dari prasyarat activity lama:
//
//	TempGetApp.LSC_ID=="ASM" || TempGetApp.LSC_ID=="SIMASNET"
//
// CATATAN. Ini persis bentuk hardcode yang ADR-0025 perintahkan menjadi master atau
// konfigurasi. Ia dibiarkan di sini sebagai konstanta yang TERLIHAT dan bernama, bukan
// tersebar di dalam percabangan — supaya saat master portal siap, yang perlu diubah
// hanya satu tempat ini. Dicatat sebagai utang teknis di docs/keputusan-implementasi.md.
var portalsRegisteredWithCashier = map[string]bool{
	"ASM":      true,
	"SIMASNET": true,
}

// Decide mencatat keputusan komite atas satu rekening.
//
// Urutannya mengikuti CNMUpdateMasterRekening_act:
//
//  1. Rekening harus masih menunggu. Keputusan yang sudah diambil tidak dianulir di sini.
//  2. Bila menyetujui: buku rekening wajib ada dan keterangan approval wajib terisi.
//  3. Keputusan disimpan lebih dulu, SEBELUM Kasir dihubungi.
//  4. Baru setelah itu rekening yang disetujui didaftarkan ke Kasir — dan hanya untuk
//     portal yang memang memakai Kasir.
//
// Kenapa langkah 3 mendahului langkah 4. Kasir adalah sistem lain di ujung jaringan;
// ia dapat lambat, dapat menolak, dapat mati. Bila keputusan komite baru disimpan
// setelah Kasir menjawab, satu kegagalan jaringan akan membuang keputusan yang sudah
// benar-benar diambil orang, dan komite harus memutuskan ulang tanpa tahu kenapa.
// Karena itu urutannya: keputusan dulu — itu fakta bisnis yang sudah terjadi — lalu
// pendaftaran ke Kasir sebagai akibatnya, yang kegagalannya dicatat di ServiceStatus
// dan diberitahukan ke PIC, bukan disembunyikan dengan membatalkan keputusannya.
func (l *Service) Decide(
	ctx context.Context,
	k masterrekening.Key,
	decision Decision,
	oleh Committee,
	logger *slog.Logger,
) (masterrekening.Account, error) {
	if decision.Status != masterrekening.StatusApproved && decision.Status != masterrekening.StatusRejected {
		return masterrekening.Account{}, masterrekening.ErrUnknownStatus
	}

	r, err := l.repo.Get(ctx, k)
	if err != nil {
		return masterrekening.Account{}, err
	}
	if !r.AwaitingDecision() {
		return masterrekening.Account{}, masterrekening.ErrAlreadyDecided
	}

	if c := strings.TrimSpace(decision.Note); c != "" {
		r.Note = c
	}
	if d := tidy(decision.DocumentID); d != "" {
		r.DocumentID = d
	}

	// Dua syarat tambahan hanya berlaku saat menyetujui. Menolak tidak menuntut buku
	// rekening: rekening ditolak justru sering karena buktinya tidak ada.
	if decision.Status == masterrekening.StatusApproved {
		if err := r.CheckBeforeApproval(); err != nil {
			return masterrekening.Account{}, err
		}
	}

	sekarang := l.clock.Now()
	r.Status = decision.Status
	r.CommitteeApproval = oleh.Identity
	r.DecidedAt = &sekarang
	r.UpdatedBy = oleh.Identity

	if err := l.repo.Update(ctx, r); err != nil {
		return masterrekening.Account{}, fmt.Errorf("masterrekening/usecase: menyimpan keputusan komite: %w", err)
	}

	if decision.Status != masterrekening.StatusApproved {
		return r, nil
	}
	return l.registerWithCashier(ctx, r, oleh, logger), nil
}

// registerWithCashier meneruskan rekening yang baru disetujui ke sistem Kasir.
//
// Ia TIDAK pernah mengembalikan galat. Keputusan komite sudah tersimpan dan sah; yang
// gagal hanyalah akibatnya. Kegagalannya dicatat pada rekening dan diberitahukan ke
// PIC — meneruskannya sebagai galat HTTP akan membuat komite mengira keputusannya
// tidak tersimpan, lalu memutuskan ulang.
func (l *Service) registerWithCashier(
	ctx context.Context,
	r masterrekening.Account,
	oleh Committee,
	logger *slog.Logger,
) masterrekening.Account {
	if l.cashier == nil || !portalsRegisteredWithCashier[l.portalAlias] {
		return r
	}

	// Account yang menggantikan rekening lama dikirim lewat jalur pembaruan, bukan
	// pendaftaran baru.
	//
	// ASUMSI YANG DISADARI. Activity lama memanggil kedua Connect-REST dengan
	// prasyarat yang sama persis (komite="ya" && APPROVAL="1" && portal ASM/SIMASNET)
	// tanpa syarat pembeda di antara keduanya, sehingga export tidak menunjukkan mana
	// yang dipakai kapan. Pembedaan berdasarkan ada-tidaknya nomor rekening lama
	// diambil dari nama servicenya — Inject untuk data baru, UpdateSearch untuk yang
	// menggantikan — dan dicatat sebagai asumsi yang menunggu konfirmasi Work Owner.
	var (
		result masterrekening.CashierResult
		err    error
	)
	if r.PreviousNumber != "" {
		result, err = l.cashier.Update(ctx, r)
	} else {
		result, err = l.cashier.Register(ctx, r)
	}

	if err != nil {
		r.ServiceStatus = "GAGAL"
		r.CashierResponse = "Sistem Cashier tidak dapat dihubungi."

		// Kegagalan menghubungi Kasir dicatat di log dan pada rekening, TETAPI TIDAK
		// mengirim surel. Sistem lama hanya mengirim surel pada ResponseCode "9", dan
		// alurnya dipertahankan apa adanya — tidak ada pemicu baru yang ditambahkan
		// (keputusan Work Owner 2026-09-17: "flow bisnis as-is").
		//
		// Akibat yang disadari: Kasir yang mati total tidak memicu pemberitahuan.
		// Yang memperlihatkannya adalah kolom Kasir di layar Master Rekening dan log
		// aplikasi. Dicatat di docs/keputusan-implementasi.md.
		if logger != nil {
			logger.Warn("pendaftaran rekening ke Kasir gagal",
				slog.String("nomor_rekening", r.Number),
				slog.String("kode_bank", r.BankCode),
				slog.String("pesan", err.Error()),
			)
		}
		l.saveCashierTrace(ctx, r, logger)
		return r
	}

	r.ServiceStatus = serviceStatus(result)
	r.CashierResponse = masterrekening.TrimCashierResponse(result.Message)
	if result.AccountID != "" {
		r.CashierAccountID = result.AccountID
	}

	// Kode "9" adalah satu-satunya kegagalan yang di sistem lama memicu surel ke PIC
	// (SendEmailAlertRekening, prasyarat CekStatus.pxResults(1).ResponseCode == "9").
	if !result.Succeeded && result.Code == "9" {
		l.recordCashierFailure(ctx, r, result.Message, result.Code, oleh, logger)
	}

	l.saveCashierTrace(ctx, r, logger)
	return r
}

func serviceStatus(h masterrekening.CashierResult) string {
	if h.Succeeded {
		return "BERHASIL"
	}
	return "GAGAL"
}

// saveCashierTrace menuliskan hasil pendaftaran Kasir ke rekening.
//
// Kegagalannya hanya dicatat di log: keputusan komite sudah tersimpan pada penulisan
// sebelumnya, dan yang hilang di sini hanyalah jejak hasil Kasir — merugikan, tetapi
// tidak sebanding dengan membatalkan keputusan yang sudah sah.
func (l *Service) saveCashierTrace(ctx context.Context, r masterrekening.Account, logger *slog.Logger) {
	if err := l.repo.Update(ctx, r); err != nil && logger != nil {
		logger.Error("gagal menyimpan jejak pendaftaran Cashier",
			slog.String("nomor_rekening", r.Number),
			slog.String("kode_bank", r.BankCode),
			slog.String("galat", err.Error()),
		)
	}
}

func (l *Service) recordCashierFailure(
	ctx context.Context,
	r masterrekening.Account,
	message string,
	code string,
	oleh Committee,
	logger *slog.Logger,
) {
	if logger != nil {
		logger.Warn("pendaftaran rekening ke Kasir gagal",
			slog.String("nomor_rekening", r.Number),
			slog.String("kode_bank", r.BankCode),
			slog.String("pesan", message),
		)
	}
	if l.notifier == nil {
		return
	}
	peringatan := masterrekening.Alert{
		Account:   r,
		Message:   message,
		Code:      code,
		DecidedBy: masterrekening.Recipients{Name: oleh.Name, Email: oleh.Email},
	}
	if err := l.notifier.WarnCashierFailure(ctx, peringatan); err != nil && logger != nil {
		logger.Error("gagal mengirim peringatan kegagalan Cashier",
			slog.String("nomor_rekening", r.Number),
			slog.String("galat", err.Error()),
		)
	}
}
