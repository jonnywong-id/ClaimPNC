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
type Service struct {
	repo      masterpicteknik.Repo
	directory masterpicteknik.OperatorDirectory
}

// Opsi adalah bahan pembentuk Layanan.
type Options struct {
	Repo      masterpicteknik.Repo
	Directory masterpicteknik.OperatorDirectory
}

// LayananBaru membentuk Layanan dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.Repo == nil {
		return nil, errors.New("masterpicteknik/usecase: seam penyimpanan wajib diisi")
	}
	if o.Directory == nil {
		return nil, errors.New("masterpicteknik/usecase: seam direktori operator wajib diisi")
	}
	return &Service{repo: o.Repo, directory: o.Directory}, nil
}

// Daftar mengembalikan seluruh petugas, terurut menurut ID operator.
//
// Tidak dipaginasi, sama dengan Master Status Klaim dan dengan alasan yang sama: isinya
// puluhan baris, bukan puluhan ribu. Report Definition lama pun memuat seluruhnya
// sekaligus. Penyaringan dan pengurutan cukup dikerjakan di layar atas data yang sudah
// di tangan.
func (l *Service) List(ctx context.Context) ([]masterpicteknik.PICTeknik, error) {
	list, err := l.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterpicteknik/usecase: membaca daftar PIC teknik: %w", err)
	}
	return list, nil
}

// Ambil mengembalikan satu petugas.
//
// Ia menggantikan `SetMstUserTeknisValue_act`, yang menjalankan `GetMasterPICTeknis`
// lalu menyalin hasilnya ke halaman `TempDcol`.
func (l *Service) Get(ctx context.Context, operatorID string) (masterpicteknik.PICTeknik, error) {
	p, err := l.repo.Get(ctx, masterpicteknik.IDKey(operatorID))
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrNotFound) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: membaca PIC %q: %w", operatorID, err)
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
func (l *Service) Create(ctx context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	p = p.Clean()

	if err := masterpicteknik.NewValidationError(masterpicteknik.Check(p)); err != nil {
		return masterpicteknik.PICTeknik{}, err
	}

	name, err := l.nameFromDirectory(ctx, p.OperatorID)
	if err != nil {
		return masterpicteknik.PICTeknik{}, err
	}
	p.Name = name

	// GrupPanel tidak pernah ditulis procedure lama, pada cabang INSERT maupun UPDATE.
	// Ia dikosongkan di sini supaya nilai yang terlanjur dikirim klien tidak diam-diam
	// menjadi perilaku baru.
	p.GrupPanel = ""

	stored, err := l.repo.Insert(ctx, p)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrAlreadyExists) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: menambah PIC teknik: %w", err)
	}
	return stored, nil
}

// Ubah mengganti data petugas yang sudah ada.
//
// ID operator dan nama tidak dapat berubah. ID adalah kunci alaminya — mengubahnya akan
// memutus setiap klaim lama yang menyimpannya. Nama dimiliki direktori operator, bukan
// master ini; ia disegarkan ulang dari sana setiap kali disimpan, supaya master tidak
// perlahan menyimpang dari sumbernya.
func (l *Service) Update(ctx context.Context, operatorID string, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	operatorID = masterpicteknik.IDKey(operatorID)

	exists, err := l.repo.Get(ctx, operatorID)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrNotFound) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: membaca PIC %q: %w", operatorID, err)
	}

	// ID diambil dari jalur URL, bukan dari badan permintaan: dua sumber untuk satu
	// nilai berarti keduanya dapat berbeda, dan yang menang menjadi soal urutan baca.
	p = p.Clean()
	p.OperatorID = exists.OperatorID

	if err := masterpicteknik.NewValidationError(masterpicteknik.Check(p)); err != nil {
		return masterpicteknik.PICTeknik{}, err
	}

	name, err := l.nameFromDirectory(ctx, p.OperatorID)
	if err != nil {
		return masterpicteknik.PICTeknik{}, err
	}
	p.Name = name
	p.GrupPanel = exists.GrupPanel

	stored, err := l.repo.Update(ctx, p)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrNotFound) {
			return masterpicteknik.PICTeknik{}, err
		}
		return masterpicteknik.PICTeknik{}, fmt.Errorf("masterpicteknik/usecase: mengubah PIC %q: %w", operatorID, err)
	}
	return stored, nil
}

// namaDariDirektori mencari nama petugas dan mengubah ketiadaannya menjadi galat
// validasi pada kolom yang benar.
//
// Ketiadaan di direktori BUKAN galat sistem melainkan isian yang salah: yang perlu
// diperbaiki pengguna adalah ID operatornya. Karena itu ia dilaporkan sebagai
// pelanggaran pada field id_operator, sehingga layar menandai kolom itu — bukan
// menampilkan pesan umum yang tidak menunjuk ke mana pun.
func (l *Service) nameFromDirectory(ctx context.Context, operatorID string) (string, error) {
	name, err := l.directory.OperatorName(ctx, operatorID)
	switch {
	case err == nil:
		return name, nil

	case errors.Is(err, masterpicteknik.ErrUnknownOperator):
		return "", masterpicteknik.NewValidationError([]masterpicteknik.Violation{{
			Field: masterpicteknik.FieldOperatorID,
			Message: "ID operator tidak terdaftar di direktori operator.",
		}})

	case errors.Is(err, masterpicteknik.ErrDirectoryUnreachable):
		// Diteruskan apa adanya. Ini bukan kesalahan pengguna, dan menyamarkannya
		// sebagai isian salah akan menyuruhnya memperbaiki sesuatu yang sudah benar.
		return "", err

	default:
		return "", fmt.Errorf("masterpicteknik/usecase: mencari nama operator %q: %w", operatorID, err)
	}
}
