// Package usecase mengorkestrasi alur modul auth: masuk, pemeriksaan sesi,
// perpanjangan, dan keluar.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket auth dan tidak memuat
// aturan modul sendiri. Ia juga tidak tahu apa pun soal HTTP.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/platform/waktu"
)

// Layanan menyatukan sistem identitas, catatan pengguna lokal, dan penyimpanan sesi.
type Layanan struct {
	identitas       auth.Identitas
	penggunaRepo    auth.PenggunaRepo
	sesiRepo        auth.SesiRepo
	jam             waktu.Jam
	masaBerlakuSesi time.Duration
	sumberAcak      io.Reader
}

// Opsi adalah bahan pembentuk Layanan. Seluruhnya wajib kecuali SumberAcak.
type Opsi struct {
	Identitas       auth.Identitas
	PenggunaRepo    auth.PenggunaRepo
	SesiRepo        auth.SesiRepo
	Jam             waktu.Jam
	MasaBerlakuSesi time.Duration
	SumberAcak      io.Reader
}

// LayananBaru membentuk Layanan dan menolak bahan yang tidak lengkap — kegagalan di
// sini terjadi saat start, bukan saat pengguna pertama mencoba masuk.
func LayananBaru(o Opsi) (*Layanan, error) {
	switch {
	case o.Identitas == nil:
		return nil, errors.New("usecase: seam identitas wajib diisi")
	case o.PenggunaRepo == nil:
		return nil, errors.New("usecase: penyimpanan pengguna wajib diisi")
	case o.SesiRepo == nil:
		return nil, errors.New("usecase: penyimpanan sesi wajib diisi")
	case o.Jam == nil:
		return nil, errors.New("usecase: seam jam wajib diisi")
	case o.MasaBerlakuSesi <= 0:
		return nil, errors.New("usecase: masa berlaku sesi harus lebih besar dari nol")
	}
	return &Layanan{
		identitas:       o.Identitas,
		penggunaRepo:    o.PenggunaRepo,
		sesiRepo:        o.SesiRepo,
		jam:             o.Jam,
		masaBerlakuSesi: o.MasaBerlakuSesi,
		sumberAcak:      o.SumberAcak,
	}, nil
}

// Hasil adalah yang didapat pemanggil setelah berhasil masuk. Token di dalamnya adalah
// satu-satunya kesempatan membacanya — setelah ini yang tersimpan hanyalah sidiknya.
type Hasil struct {
	Token    auth.Token
	Sesi     auth.Sesi
	Pengguna auth.Pengguna
}

// Konteks adalah identitas pemanggil yang sudah terbukti, dipakai seluruh lapisan lain.
type Konteks struct {
	Pengguna auth.Pengguna
	Sesi     auth.Sesi
}

// Masuk memverifikasi kredensial ke sistem identitas, menyegarkan catatan pengguna
// lokal, lalu menerbitkan sesi milik aplikasi.
//
// Urutannya disengaja: pemanggilan sistem luar selesai lebih dulu, sebelum satu pun
// baris basis data disentuh — kegagalan jaringan tidak boleh menahan kunci baris.
func (l *Layanan) Masuk(ctx context.Context, k auth.Kredensial) (Hasil, error) {
	if !k.Lengkap() {
		// Kredensial kosong dijawab sama dengan kredensial salah. Membedakannya
		// memberi tahu penyerang bahwa formatnya sudah benar.
		return Hasil{}, auth.ErrKredensialSalah
	}

	profil, err := l.identitas.Verifikasi(ctx, k)
	if err != nil {
		return Hasil{}, err
	}
	if err := profil.Periksa(); err != nil {
		return Hasil{}, err
	}

	p, err := l.segarkanPengguna(ctx, profil)
	if err != nil {
		return Hasil{}, err
	}
	if !p.Aktif {
		return Hasil{}, auth.ErrPenggunaTidakAktif
	}

	token, err := auth.TerbitkanToken(l.sumberAcak)
	if err != nil {
		return Hasil{}, fmt.Errorf("usecase: menerbitkan token: %w", err)
	}
	idSesi, err := auth.IDSesiBaru(l.sumberAcak)
	if err != nil {
		return Hasil{}, fmt.Errorf("usecase: menerbitkan pengenal sesi: %w", err)
	}
	sekarang := l.jam.Sekarang().UTC()
	s := auth.Sesi{
		ID:              idSesi,
		SidikToken:      token.Sidik(),
		Identitas:       p.Identitas,
		DiterbitkanPada: sekarang,
		BerlakuSampai:   sekarang.Add(l.masaBerlakuSesi),
	}
	if err := l.sesiRepo.Simpan(ctx, s); err != nil {
		return Hasil{}, fmt.Errorf("usecase: menyimpan sesi: %w", err)
	}
	return Hasil{Token: token, Sesi: s, Pengguna: p}, nil
}

// Periksa menguji token yang dibawa permintaan dan mengembalikan identitas pemiliknya.
func (l *Layanan) Periksa(ctx context.Context, token auth.Token) (Konteks, error) {
	if strings.TrimSpace(string(token)) == "" {
		return Konteks{}, auth.ErrSesiTidakDitemukan
	}
	s, err := l.sesiRepo.AmbilBySidikToken(ctx, token.Sidik())
	if err != nil {
		return Konteks{}, err
	}
	if err := s.Periksa(l.jam.Sekarang().UTC()); err != nil {
		return Konteks{}, err
	}
	p, err := l.penggunaRepo.AmbilByIdentitas(ctx, s.Identitas)
	if err != nil {
		return Konteks{}, fmt.Errorf("usecase: memuat pengguna sesi: %w", err)
	}
	if !p.Aktif {
		return Konteks{}, auth.ErrPenggunaTidakAktif
	}
	return Konteks{Pengguna: p, Sesi: s}, nil
}

// Perpanjang menggeser batas berlaku sesi yang masih hidup. Sesi yang sudah kedaluwarsa
// atau dicabut tidak dapat dihidupkan kembali — pengguna harus masuk ulang.
func (l *Layanan) Perpanjang(ctx context.Context, token auth.Token) (auth.Sesi, error) {
	konteks, err := l.Periksa(ctx, token)
	if err != nil {
		return auth.Sesi{}, err
	}
	batasBaru := l.jam.Sekarang().UTC().Add(l.masaBerlakuSesi)
	if err := l.sesiRepo.Perpanjang(ctx, konteks.Sesi.ID, batasBaru); err != nil {
		return auth.Sesi{}, fmt.Errorf("usecase: memperpanjang sesi: %w", err)
	}
	konteks.Sesi.BerlakuSampai = batasBaru
	return konteks.Sesi, nil
}

// Keluar mencabut sesi di server. Mencabut sesi yang sudah tidak berlaku bukan galat:
// hasil akhirnya sama, dan pengguna tidak perlu diberi tahu bedanya.
func (l *Layanan) Keluar(ctx context.Context, token auth.Token) error {
	s, err := l.sesiRepo.AmbilBySidikToken(ctx, token.Sidik())
	if err != nil {
		if errors.Is(err, auth.ErrSesiTidakDitemukan) {
			return nil
		}
		return err
	}
	if s.DicabutPada != nil {
		return nil
	}
	if err := l.sesiRepo.Cabut(ctx, s.ID, l.jam.Sekarang().UTC()); err != nil {
		return fmt.Errorf("usecase: mencabut sesi: %w", err)
	}
	return nil
}

// segarkanPengguna menyalin profil terbaru dari sistem identitas ke catatan lokal.
//
// Catatan: pembaruan catatan pengguna dan penyimpanan sesi belum berada dalam satu
// transaksi — kepemilikan transaksi di lapisan aplikasi adalah lingkup TKT-F2-003 yang
// belum dikerjakan. Dampak bila langkah kedua gagal terbatas: catatan pengguna
// tersegarkan tanpa sesi terbit, dan penyegaran itu idempoten.
func (l *Layanan) segarkanPengguna(ctx context.Context, profil auth.Profil) (auth.Pengguna, error) {
	sekarang := l.jam.Sekarang().UTC()

	p, err := l.penggunaRepo.AmbilByIdentitas(ctx, profil.Identitas)
	switch {
	case errors.Is(err, auth.ErrPenggunaTidakDitemukan):
		p = auth.DariProfil(profil, sekarang)
	case err != nil:
		return auth.Pengguna{}, fmt.Errorf("usecase: memuat pengguna: %w", err)
	default:
		// Status aktif dan OperatorID tidak ikut tersegarkan — keduanya dimiliki
		// administrator aplikasi ini, bukan sistem identitas luar.
		p.SegarkanDari(profil, sekarang)
	}

	if err := l.penggunaRepo.SimpanAtauPerbarui(ctx, p); err != nil {
		return auth.Pengguna{}, fmt.Errorf("usecase: menyimpan pengguna: %w", err)
	}
	return p, nil
}
