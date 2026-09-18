// Package usecase mengorkestrasi alur Master Rekening.
//
// Ia memegang urutan langkah dan aturan yang melibatkan lebih dari satu seam; aturan
// yang hanya menyangkut satu rekening tetap tinggal di paket domain.
//
// Lapisan Aplikasi — boleh mengimpor paket domain, dilarang mengimpor HTTP dan SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/platform/waktu"
)

// Pengaju adalah orang yang mengajukan rekening.
//
// Ia dipisahkan dari isian formulir supaya siapa yang menginput tidak pernah dapat
// dikirim dari peramban — nilainya selalu berasal dari sesi.
type Pengaju struct {
	// Identitas adalah NIK karyawan atau LOGIN_ID non-karyawan.
	Identitas string
	Nama      string
	Email     string
}

// Pengajuan adalah isian formulir master rekening.
//
// Ia sengaja bukan masterrekening.Rekening: field yang dimiliki sistem — status
// persetujuan, komite, waktu keputusan, dan seluruh jejak Kasir — tidak boleh dapat
// diisi dari luar.
type Pengajuan struct {
	NomorRekening string
	NamaPemilik   string
	NamaBank      string
	CabangBank    string
	AlamatBank    string
	KodeBank      string
	TipeRekening  string
	Email         string
	Telepon       string
	NIK           string
	IDDokumen     string
	Catatan       string
	Aktif         bool

	// Tiga field berikut hanya terisi bila pengajuan ini menggantikan rekening lama
	// pada klaim yang sedang berjalan.
	KodeBankLama      string
	NomorRekeningLama string
	NamaPemilikLama   string
}

// Layanan menjalankan alur Master Rekening.
type Layanan struct {
	repo     masterrekening.Repo
	bank     masterrekening.BankRepo
	kasir    masterrekening.Kasir
	notifier masterrekening.Notifier
	jam      waktu.Jam

	// portalAlias adalah alias portal yang sedang dilayani. Ia menentukan apakah
	// rekening yang disetujui didaftarkan ke Kasir — lihat Putuskan.
	portalAlias string

	// komiteBaku adalah identitas komite yang menerima pengajuan bila tidak ada yang
	// ditunjuk. Lihat catatan di Ajukan.
	komiteBaku string
}

// Opsi adalah bahan pembentuk Layanan.
type Opsi struct {
	Repo     masterrekening.Repo
	Bank     masterrekening.BankRepo
	Kasir    masterrekening.Kasir
	Notifier masterrekening.Notifier
	Jam      waktu.Jam

	PortalAlias string
	KomiteBaku  string
}

// LayananBaru membentuk layanan; Repo dan Jam wajib terisi.
func LayananBaru(o Opsi) *Layanan {
	return &Layanan{
		repo:        o.Repo,
		bank:        o.Bank,
		kasir:       o.Kasir,
		notifier:    o.Notifier,
		jam:         o.Jam,
		portalAlias: strings.ToUpper(strings.TrimSpace(o.PortalAlias)),
		komiteBaku:  o.KomiteBaku,
	}
}

// Ajukan mendaftarkan rekening baru dengan status menunggu keputusan komite.
//
// Urutannya mengikuti CNMUpdateMasterRekening_act dan tidak boleh dibalik:
//
//  1. Periksa kelengkapan isian.
//  2. Cari nomor rekening yang sama. Bila ada dan BUKAN bekas penolakan komite,
//     pengajuan ditolak.
//  3. Bila ada dan bekas penolakan, baris lama dibuang lalu pengajuan disisipkan.
//  4. Rekening baru selalu lahir dengan status menunggu — tidak ada jalan bagi
//     pengaju untuk menerbitkan rekening yang langsung disetujui.
func (l *Layanan) Ajukan(ctx context.Context, p Pengajuan, oleh Pengaju) (masterrekening.Rekening, error) {
	sekarang := l.jam.Sekarang()

	r := masterrekening.Rekening{
		NomorRekening:     rapikan(p.NomorRekening),
		NamaPemilik:       rapikan(p.NamaPemilik),
		NamaBank:          rapikan(p.NamaBank),
		CabangBank:        rapikan(p.CabangBank),
		AlamatBank:        rapikan(p.AlamatBank),
		KodeBank:          rapikan(p.KodeBank),
		TipeRekening:      rapikan(p.TipeRekening),
		Email:             rapikan(p.Email),
		Telepon:           rapikan(p.Telepon),
		NIK:               rapikan(p.NIK),
		IDDokumen:         rapikan(p.IDDokumen),
		Catatan:           strings.TrimSpace(p.Catatan),
		Aktif:             p.Aktif,
		KodeBankLama:      rapikan(p.KodeBankLama),
		NomorRekeningLama: rapikan(p.NomorRekeningLama),
		NamaPemilikLama:   rapikan(p.NamaPemilikLama),

		// Ditetapkan sistem, bukan dikirim peramban.
		Status:         masterrekening.StatusMenunggu,
		KomiteApproval: l.komiteBaku,
		DiinputOleh:    oleh.Identitas,
		DiinputPada:    sekarang,
		DiubahOleh:     oleh.Identitas,
		EmailPenginput: rapikan(oleh.Email),
	}

	if err := r.Periksa(); err != nil {
		return masterrekening.Rekening{}, err
	}

	serupa, err := l.repo.CariNomor(ctx, r.NomorRekening)
	if err != nil {
		return masterrekening.Rekening{}, fmt.Errorf("masterrekening/usecase: memeriksa duplikasi: %w", err)
	}
	if !masterrekening.BolehDidaftarkanUlang(serupa) {
		return masterrekening.Rekening{}, masterrekening.ErrSudahAda
	}

	// Baris bekas penolakan dibuang lebih dulu. Perilaku sistem lama dipertahankan
	// apa adanya (P-5); utang tekniknya dicatat di masterrekening.BolehDidaftarkanUlang.
	for _, lama := range serupa {
		if lama.Status != masterrekening.StatusDitolak {
			continue
		}
		if err := l.repo.HapusYangDitolak(ctx, lama.KunciDari()); err != nil {
			return masterrekening.Rekening{}, fmt.Errorf("masterrekening/usecase: membuang pengajuan yang ditolak: %w", err)
		}
	}

	if err := l.repo.Simpan(ctx, r); err != nil {
		return masterrekening.Rekening{}, fmt.Errorf("masterrekening/usecase: menyimpan rekening: %w", err)
	}
	return r, nil
}

// Ubah memperbarui rekening yang sudah ada.
//
// Rekening yang sudah diputuskan komite tidak dapat diubah lewat jalur ini: mengubah
// nomor rekening yang sudah disetujui berarti uang klaim berpindah tujuan tanpa
// seorang pun menyetujuinya. Pengajuan baru adalah jalannya.
func (l *Layanan) Ubah(ctx context.Context, k masterrekening.Kunci, p Pengajuan, oleh Pengaju) (masterrekening.Rekening, error) {
	ada, err := l.repo.Ambil(ctx, k)
	if err != nil {
		return masterrekening.Rekening{}, err
	}
	if !ada.MenungguKeputusan() {
		return masterrekening.Rekening{}, masterrekening.ErrSudahDiputuskan
	}

	ada.NamaPemilik = rapikan(p.NamaPemilik)
	ada.NamaBank = rapikan(p.NamaBank)
	ada.CabangBank = rapikan(p.CabangBank)
	ada.AlamatBank = rapikan(p.AlamatBank)
	ada.TipeRekening = rapikan(p.TipeRekening)
	ada.Email = rapikan(p.Email)
	ada.Telepon = rapikan(p.Telepon)
	ada.NIK = rapikan(p.NIK)
	ada.Aktif = p.Aktif
	ada.DiubahOleh = oleh.Identitas
	if d := rapikan(p.IDDokumen); d != "" {
		ada.IDDokumen = d
	}
	if c := strings.TrimSpace(p.Catatan); c != "" {
		ada.Catatan = c
	}

	if err := ada.Periksa(); err != nil {
		return masterrekening.Rekening{}, err
	}
	if err := l.repo.Perbarui(ctx, ada); err != nil {
		return masterrekening.Rekening{}, fmt.Errorf("masterrekening/usecase: memperbarui rekening: %w", err)
	}
	return ada, nil
}

// Daftar membaca rekening yang cocok dengan filter.
func (l *Layanan) Daftar(ctx context.Context, f masterrekening.Filter) ([]masterrekening.Rekening, int, error) {
	baris, jumlah, err := l.repo.Daftar(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("masterrekening/usecase: membaca daftar rekening: %w", err)
	}
	return baris, jumlah, nil
}

// Ambil membaca satu rekening.
func (l *Layanan) Ambil(ctx context.Context, k masterrekening.Kunci) (masterrekening.Rekening, error) {
	return l.repo.Ambil(ctx, k)
}

// DaftarBank membaca daftar bank dari GENERAL.LST_BANK_GROUP.
func (l *Layanan) DaftarBank(ctx context.Context) ([]masterrekening.Bank, error) {
	if l.bank == nil {
		return nil, errors.New("masterrekening/usecase: sumber daftar bank belum dipasang")
	}
	daftar, err := l.bank.Daftar(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterrekening/usecase: membaca daftar bank: %w", err)
	}
	return daftar, nil
}

func rapikan(s string) string { return strings.TrimSpace(s) }
