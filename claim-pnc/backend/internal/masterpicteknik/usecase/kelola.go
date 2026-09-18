// Package usecase mengorkestrasi pengelolaan Master PIC Teknik: melihat daftar,
// membuka satu baris, menambah, dan mengubah.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket masterpicteknik dan
// tidak tahu apa pun soal HTTP maupun SQL.
//
// # Empat aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Procedure lama `PEGA_MST_USER_TEKNIS` hanya mengenal INSERT dan
// UPDATE, dan menghapus satu petugas akan membuat setiap klaim lama yang menyimpan
// `OPERATOR_ID` itu kehilangan rujukan penugasannya — persis alasan `ADR-0012`
// menetapkan master tidak dihapus permanen. Petugas yang berhenti **dinonaktifkan**
// lewat kolom aktif, dan itu memang yang disediakan tabelnya.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterpicteknik"
)

// Layanan mengelola master PIC teknik di atas seam penyimpanan dan direktori operator.
type Layanan struct {
	repo      masterpicteknik.Repo
	direktori masterpicteknik.DirektoriOperator
}

// Opsi adalah bahan pembentuk Layanan.
type Opsi struct {
	Repo      masterpicteknik.Repo
	Direktori masterpicteknik.DirektoriOperator
}

// LayananBaru membentuk Layanan dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func LayananBaru(o Opsi) (*Layanan, error) {
	if o.Repo == nil {
		return nil, errors.New("masterpicteknik/usecase: seam penyimpanan wajib diisi")
	}
	if o.Direktori == nil {
		return nil, errors.New("masterpicteknik/usecase: seam direktori operator wajib diisi")
	}
	return &Layanan{repo: o.Repo, direktori: o.Direktori}, nil
}

// Daftar mengembalikan seluruh petugas, terurut menurut ID operator.
//
// Tidak dipaginasi, sama dengan Master Status Klaim dan dengan alasan yang sama: isinya
// puluhan baris, bukan puluhan ribu. Report Definition lama pun memuat seluruhnya
// sekaligus. Penyaringan dan pengurutan cukup dikerjakan di layar atas data yang sudah
// di tangan.
func (l *Layanan) Daftar(ctx context.Context) ([]masterpicteknik.PICTeknik, error) {
	daftar, err := l.repo.Daftar(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterpicteknik/usecase: membaca daftar PIC teknik: %w", err)
	}
	return daftar, nil
}

// Ambil mengembalikan satu petugas.
//
// Ia menggantikan `SetMstUserTeknisValue_act`, yang menjalankan `GetMasterPICTeknis`
// lalu menyalin hasilnya ke halaman `TempDcol`.
func (l *Layanan) Ambil(ctx context.Context, idOperator string) (masterpicteknik.PICTeknik, error) {
	p, err := l.repo.Ambil(ctx, masterpicteknik.KunciID(idOperator))
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrTidakDitemukan) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: membaca PIC %q: %w", idOperator, err)
	}
	return p, nil
}

// Tambah mendaftarkan petugas baru.
//
// Urutannya mengikuti `CNMInsertMstUserTeknis_act` dan tidak boleh dibalik:
//
//  1. Periksa kelengkapan isian.
//  2. Cari namanya di direktori operator. Bila tidak ditemukan, pengajuan DITOLAK —
//     inilah langkah "set error kalau tidak ditemukan di service".
//  3. Baru simpan, dengan nama yang berasal dari direktori, bukan dari isian.
//
// Nama TIDAK diterima dari pemanggil. Menerimanya berarti master ini dapat memuat nama
// yang tidak cocok dengan direktori operator, dan setiap layar yang menampilkan
// penugasan akan menyebut orang yang berbeda dari yang sesungguhnya bertugas.
func (l *Layanan) Tambah(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	p = p.Bersih()

	if err := masterpicteknik.GalatValidasiBaru(masterpicteknik.Periksa(p)); err != nil {
		return masterpicteknik.PICTeknik{}, err
	}

	nama, err := l.namaDariDirektori(ctx, p.IDOperator)
	if err != nil {
		return masterpicteknik.PICTeknik{}, err
	}
	p.Nama = nama

	// GrupPanel tidak pernah ditulis procedure lama, pada cabang INSERT maupun UPDATE.
	// Ia dikosongkan di sini supaya nilai yang terlanjur dikirim klien tidak diam-diam
	// menjadi perilaku baru.
	p.GrupPanel = ""

	tersimpan, err := l.repo.Sisip(ctx, p)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrSudahAda) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: menambah PIC teknik: %w", err)
	}
	return tersimpan, nil
}

// Ubah mengganti data petugas yang sudah ada.
//
// ID operator dan nama tidak dapat berubah. ID adalah kunci alaminya — mengubahnya akan
// memutus setiap klaim lama yang menyimpannya. Nama dimiliki direktori operator, bukan
// master ini; ia disegarkan ulang dari sana setiap kali disimpan, supaya master tidak
// perlahan menyimpang dari sumbernya.
func (l *Layanan) Ubah(ctx context.Context, idOperator string, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	idOperator = masterpicteknik.KunciID(idOperator)

	ada, err := l.repo.Ambil(ctx, idOperator)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrTidakDitemukan) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: membaca PIC %q: %w", idOperator, err)
	}

	// ID diambil dari jalur URL, bukan dari badan permintaan: dua sumber untuk satu
	// nilai berarti keduanya dapat berbeda, dan yang menang menjadi soal urutan baca.
	p = p.Bersih()
	p.IDOperator = ada.IDOperator

	if err := masterpicteknik.GalatValidasiBaru(masterpicteknik.Periksa(p)); err != nil {
		return masterpicteknik.PICTeknik{}, err
	}

	nama, err := l.namaDariDirektori(ctx, p.IDOperator)
	if err != nil {
		return masterpicteknik.PICTeknik{}, err
	}
	p.Nama = nama
	p.GrupPanel = ada.GrupPanel

	tersimpan, err := l.repo.Perbarui(ctx, p)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrTidakDitemukan) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: mengubah PIC %q: %w", idOperator, err)
	}
	return tersimpan, nil
}

// namaDariDirektori mencari nama petugas dan mengubah ketiadaannya menjadi galat
// validasi pada kolom yang benar.
//
// Ketiadaan di direktori BUKAN galat sistem melainkan isian yang salah: yang perlu
// diperbaiki pengguna adalah ID operatornya. Karena itu ia dilaporkan sebagai
// pelanggaran pada field id_operator, sehingga layar menandai kolom itu — bukan
// menampilkan pesan umum yang tidak menunjuk ke mana pun.
func (l *Layanan) namaDariDirektori(ctx context.Context, idOperator string) (string, error) {
	nama, err := l.direktori.NamaOperator(ctx, idOperator)
	switch {
	case err == nil:
		return nama, nil

	case errors.Is(err, masterpicteknik.ErrOperatorTidakDikenal):
		return "", masterpicteknik.GalatValidasiBaru([]masterpicteknik.Pelanggaran{{
			Field: masterpicteknik.FieldIDOperator,
			Pesan: "ID operator tidak terdaftar di direktori operator.",
		}})

	case errors.Is(err, masterpicteknik.ErrDirektoriTidakTerhubung):
		// Diteruskan apa adanya. Ini bukan kesalahan pengguna, dan menyamarkannya
		// sebagai isian salah akan menyuruhnya memperbaiki sesuatu yang sudah benar.
		return "", err

	default:
		return "", fmt.Errorf("masterpicteknik/usecase: mencari nama operator %q: %w", idOperator, err)
	}
}
