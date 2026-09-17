package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/masterrekening"
)

// Komite adalah anggota komite yang memutuskan.
type Komite struct {
	Identitas string
	Nama      string

	// Email dipakai sebagai tujuan peringatan bila pendaftaran ke Kasir gagal.
	// Boleh kosong: non-karyawan tidak punya surel di POOLDATA.M_LOGIN_PNC, dan
	// peringatan yang tidak punya tujuan dilewati, bukan menggagalkan apa pun.
	Email string
}

// Keputusan adalah apa yang diputuskan komite atas satu rekening.
type Keputusan struct {
	// Status wajib StatusDisetujui atau StatusDitolak. StatusMenunggu ditolak —
	// "belum memutuskan" bukan sebuah keputusan.
	Status masterrekening.StatusApproval

	// Catatan adalah keterangan approval atasan. Wajib terisi saat menyetujui.
	Catatan string

	// IDDokumen mengisi lampiran buku rekening bila belum terisi saat pengajuan.
	IDDokumen string
}

// portalYangDidaftarkanKeKasir menyebut portal yang rekening setujuannya diteruskan ke
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
var portalYangDidaftarkanKeKasir = map[string]bool{
	"ASM":      true,
	"SIMASNET": true,
}

// Putuskan mencatat keputusan komite atas satu rekening.
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
// pendaftaran ke Kasir sebagai akibatnya, yang kegagalannya dicatat di StatusLayanan
// dan diberitahukan ke PIC, bukan disembunyikan dengan membatalkan keputusannya.
func (l *Layanan) Putuskan(
	ctx context.Context,
	k masterrekening.Kunci,
	putusan Keputusan,
	oleh Komite,
	logger *slog.Logger,
) (masterrekening.Rekening, error) {
	if putusan.Status != masterrekening.StatusDisetujui && putusan.Status != masterrekening.StatusDitolak {
		return masterrekening.Rekening{}, masterrekening.ErrStatusTidakDikenal
	}

	r, err := l.repo.Ambil(ctx, k)
	if err != nil {
		return masterrekening.Rekening{}, err
	}
	if !r.MenungguKeputusan() {
		return masterrekening.Rekening{}, masterrekening.ErrSudahDiputuskan
	}

	if c := strings.TrimSpace(putusan.Catatan); c != "" {
		r.Catatan = c
	}
	if d := rapikan(putusan.IDDokumen); d != "" {
		r.IDDokumen = d
	}

	// Dua syarat tambahan hanya berlaku saat menyetujui. Menolak tidak menuntut buku
	// rekening: rekening ditolak justru sering karena buktinya tidak ada.
	if putusan.Status == masterrekening.StatusDisetujui {
		if err := r.PeriksaSebelumDisetujui(); err != nil {
			return masterrekening.Rekening{}, err
		}
	}

	sekarang := l.jam.Sekarang()
	r.Status = putusan.Status
	r.KomiteApproval = oleh.Identitas
	r.DiputuskanPada = &sekarang
	r.DiubahOleh = oleh.Identitas

	if err := l.repo.Perbarui(ctx, r); err != nil {
		return masterrekening.Rekening{}, fmt.Errorf("masterrekening/usecase: menyimpan keputusan komite: %w", err)
	}

	if putusan.Status != masterrekening.StatusDisetujui {
		return r, nil
	}
	return l.daftarkanKeKasir(ctx, r, oleh, logger), nil
}

// daftarkanKeKasir meneruskan rekening yang baru disetujui ke sistem Kasir.
//
// Ia TIDAK pernah mengembalikan galat. Keputusan komite sudah tersimpan dan sah; yang
// gagal hanyalah akibatnya. Kegagalannya dicatat pada rekening dan diberitahukan ke
// PIC — meneruskannya sebagai galat HTTP akan membuat komite mengira keputusannya
// tidak tersimpan, lalu memutuskan ulang.
func (l *Layanan) daftarkanKeKasir(
	ctx context.Context,
	r masterrekening.Rekening,
	oleh Komite,
	logger *slog.Logger,
) masterrekening.Rekening {
	if l.kasir == nil || !portalYangDidaftarkanKeKasir[l.portalAlias] {
		return r
	}

	// Rekening yang menggantikan rekening lama dikirim lewat jalur pembaruan, bukan
	// pendaftaran baru.
	//
	// ASUMSI YANG DISADARI. Activity lama memanggil kedua Connect-REST dengan
	// prasyarat yang sama persis (komite="ya" && APPROVAL="1" && portal ASM/SIMASNET)
	// tanpa syarat pembeda di antara keduanya, sehingga export tidak menunjukkan mana
	// yang dipakai kapan. Pembedaan berdasarkan ada-tidaknya nomor rekening lama
	// diambil dari nama servicenya — Inject untuk data baru, UpdateSearch untuk yang
	// menggantikan — dan dicatat sebagai asumsi yang menunggu konfirmasi Work Owner.
	var (
		hasil masterrekening.HasilKasir
		err   error
	)
	if r.NomorRekeningLama != "" {
		hasil, err = l.kasir.Perbarui(ctx, r)
	} else {
		hasil, err = l.kasir.Daftarkan(ctx, r)
	}

	if err != nil {
		r.StatusLayanan = "GAGAL"
		r.ResponsKasir = "Sistem Kasir tidak dapat dihubungi."

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
				slog.String("nomor_rekening", r.NomorRekening),
				slog.String("kode_bank", r.KodeBank),
				slog.String("pesan", err.Error()),
			)
		}
		l.simpanJejakKasir(ctx, r, logger)
		return r
	}

	r.StatusLayanan = statusLayanan(hasil)
	r.ResponsKasir = masterrekening.PangkasResponsKasir(hasil.Pesan)
	if hasil.IDRekening != "" {
		r.IDRekeningKasir = hasil.IDRekening
	}

	// Kode "9" adalah satu-satunya kegagalan yang di sistem lama memicu surel ke PIC
	// (SendEmailAlertRekening, prasyarat CekStatus.pxResults(1).ResponseCode == "9").
	if !hasil.Berhasil && hasil.Kode == "9" {
		l.catatKegagalanKasir(ctx, r, hasil.Pesan, hasil.Kode, oleh, logger)
	}

	l.simpanJejakKasir(ctx, r, logger)
	return r
}

func statusLayanan(h masterrekening.HasilKasir) string {
	if h.Berhasil {
		return "BERHASIL"
	}
	return "GAGAL"
}

// simpanJejakKasir menuliskan hasil pendaftaran Kasir ke rekening.
//
// Kegagalannya hanya dicatat di log: keputusan komite sudah tersimpan pada penulisan
// sebelumnya, dan yang hilang di sini hanyalah jejak hasil Kasir — merugikan, tetapi
// tidak sebanding dengan membatalkan keputusan yang sudah sah.
func (l *Layanan) simpanJejakKasir(ctx context.Context, r masterrekening.Rekening, logger *slog.Logger) {
	if err := l.repo.Perbarui(ctx, r); err != nil && logger != nil {
		logger.Error("gagal menyimpan jejak pendaftaran Kasir",
			slog.String("nomor_rekening", r.NomorRekening),
			slog.String("kode_bank", r.KodeBank),
			slog.String("galat", err.Error()),
		)
	}
}

func (l *Layanan) catatKegagalanKasir(
	ctx context.Context,
	r masterrekening.Rekening,
	pesan string,
	kode string,
	oleh Komite,
	logger *slog.Logger,
) {
	if logger != nil {
		logger.Warn("pendaftaran rekening ke Kasir gagal",
			slog.String("nomor_rekening", r.NomorRekening),
			slog.String("kode_bank", r.KodeBank),
			slog.String("pesan", pesan),
		)
	}
	if l.notifier == nil {
		return
	}
	peringatan := masterrekening.Peringatan{
		Rekening:   r,
		Pesan:      pesan,
		Kode:       kode,
		Diputuskan: masterrekening.Penerima{Nama: oleh.Nama, Email: oleh.Email},
	}
	if err := l.notifier.PeringatkanKegagalanKasir(ctx, peringatan); err != nil && logger != nil {
		logger.Error("gagal mengirim peringatan kegagalan Kasir",
			slog.String("nomor_rekening", r.NomorRekening),
			slog.String("galat", err.Error()),
		)
	}
}
