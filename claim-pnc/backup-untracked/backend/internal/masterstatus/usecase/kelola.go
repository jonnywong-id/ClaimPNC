// Package usecase mengorkestrasi pengelolaan Master Status Klaim: melihat daftar,
// membuka satu baris, menambah, dan mengubah.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket masterstatus dan
// tidak tahu apa pun soal HTTP maupun SQL.
//
// # Empat aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian: layar Pega pun tidak punya tombol hapus
// (`Section/BrowseStatusClaim-Section.xml` — `pyDeleteActivityExists=false`), dan
// `Database/PEGA_M_STS_CLAIM.prc` hanya mengenal INSERT dan UPDATE. Menghapus satu
// baris master akan membuat setiap klaim lama yang menyimpan kode itu kehilangan
// artinya — persis alasan `ADR-0012` menetapkan master tidak dihapus permanen.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterstatus"
)

// Layanan mengelola master status klaim di atas satu seam penyimpanan.
type Layanan struct {
	repo masterstatus.Repo
}

// Opsi adalah bahan pembentuk Layanan.
type Opsi struct {
	Repo masterstatus.Repo
}

// LayananBaru membentuk Layanan dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func LayananBaru(o Opsi) (*Layanan, error) {
	if o.Repo == nil {
		return nil, errors.New("masterstatus/usecase: seam penyimpanan wajib diisi")
	}
	return &Layanan{repo: o.Repo}, nil
}

// Daftar mengembalikan seluruh status klaim, terurut menurut kode.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan angka: isinya 33 baris dan
// bertambah beberapa baris per tahun. Report Definition lama pun memuat seluruhnya
// sekaligus dengan batas `pyMaxRecords=500`. Menambahkan paginasi server di sini akan
// menambah kerumitan yang tidak menyelesaikan satu pun masalah nyata; penyaringan dan
// pengurutan cukup dikerjakan di layar atas 33 baris yang sudah di tangan.
func (l *Layanan) Daftar(ctx context.Context) ([]masterstatus.StatusKlaim, error) {
	daftar, err := l.repo.Daftar(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterstatus/usecase: membaca daftar status: %w", err)
	}
	return daftar, nil
}

// Ambil mengembalikan satu status klaim.
//
// Ia menggantikan `SetStsClaimValue_act(lscid)`, yang menjalankan Report Definition
// `SelectVStsClaim_RD` lalu menyalin hasilnya ke halaman `TempStsClaim`.
func (l *Layanan) Ambil(ctx context.Context, kode string) (masterstatus.StatusKlaim, error) {
	status, err := l.repo.Ambil(ctx, bersihkan(kode))
	if err != nil {
		if errors.Is(err, masterstatus.ErrTidakDitemukan) {
			return masterstatus.StatusKlaim{}, err
		}
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/usecase: membaca status %q: %w", kode, err)
	}
	return status, nil
}

// Tambah menyisipkan status baru dan mengembalikannya lengkap dengan kode yang dibuat
// penyimpanan.
//
// Kode TIDAK diterima dari pemanggil. Sistem lama pun demikian: layar mengirim sentinel
// `"UnknownID"` dan procedure yang menentukan kodenya
// (`Activity/CNMUpdateStsclaim_act-Act.xml` → `PEGA_M_STS_CLAIM`). Menerima kode dari
// luar akan membuat dua status berbeda dapat memperebutkan nomor yang sama.
func (l *Layanan) Tambah(ctx context.Context, label string) (masterstatus.StatusKlaim, error) {
	if err := masterstatus.GalatValidasiBaru(masterstatus.PeriksaLabel(label)); err != nil {
		return masterstatus.StatusKlaim{}, err
	}
	if err := l.pastikanLabelBelumDipakai(ctx, label, ""); err != nil {
		return masterstatus.StatusKlaim{}, err
	}

	status, err := l.repo.Sisip(ctx, bersihkan(label))
	if err != nil {
		// Kedua galat di bawah datang dari penegakan di basis data, yang menang atas
		// pemeriksaan di atas bila dua permintaan tiba bersamaan. Ia diteruskan apa
		// adanya supaya pengguna melihat sebab yang benar.
		if errors.Is(err, masterstatus.ErrLabelSudahAda) || errors.Is(err, masterstatus.ErrKodeSudahAda) {
			return masterstatus.StatusKlaim{}, err
		}
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/usecase: menambah status: %w", err)
	}
	return status, nil
}

// Ubah mengganti label status yang sudah ada.
//
// Hanya label yang dapat berubah. Kode bersifat tetap seumur hidup baris itu — layar
// Pega menandainya read-only (`pyEditOptions=Read-only`), dan mengubahnya akan
// memutus setiap klaim lama yang menyimpan kode tersebut.
func (l *Layanan) Ubah(ctx context.Context, kode, label string) (masterstatus.StatusKlaim, error) {
	kode = bersihkan(kode)

	if err := masterstatus.GalatValidasiBaru(masterstatus.PeriksaLabel(label)); err != nil {
		return masterstatus.StatusKlaim{}, err
	}
	// Kode diperiksa lebih dulu supaya mengubah status yang tidak ada dijawab "tidak
	// ditemukan", bukan "label sudah dipakai" yang menyesatkan.
	if _, err := l.repo.Ambil(ctx, kode); err != nil {
		if errors.Is(err, masterstatus.ErrTidakDitemukan) {
			return masterstatus.StatusKlaim{}, err
		}
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/usecase: membaca status %q: %w", kode, err)
	}
	if err := l.pastikanLabelBelumDipakai(ctx, label, kode); err != nil {
		return masterstatus.StatusKlaim{}, err
	}

	status, err := l.repo.Perbarui(ctx, kode, bersihkan(label))
	if err != nil {
		if errors.Is(err, masterstatus.ErrTidakDitemukan) || errors.Is(err, masterstatus.ErrLabelSudahAda) {
			return masterstatus.StatusKlaim{}, err
		}
		return masterstatus.StatusKlaim{}, fmt.Errorf("masterstatus/usecase: mengubah status %q: %w", kode, err)
	}
	return status, nil
}

// pastikanLabelBelumDipakai menolak label yang sudah dipakai status lain.
//
// kecualiKode dikosongkan saat menambah, dan diisi kode yang sedang diubah saat
// mengubah — tanpa itu, menyimpan ulang status tanpa mengubah labelnya akan ditolak
// karena bentrok dengan dirinya sendiri.
//
// Pemeriksaan ini adalah KENYAMANAN, bukan jaminan: dua permintaan yang tiba bersamaan
// dapat sama-sama lolos di sini. Jaminannya ada di indeks unik basis data
// (`UX_M_STS_CLAIM_LABEL` pada migrasi 0002), dan galatnya diterjemahkan kembali
// menjadi ErrLabelSudahAda oleh repo. Yang di sini hanya membuat pesannya tiba lebih
// cepat dan lebih jelas.
func (l *Layanan) pastikanLabelBelumDipakai(ctx context.Context, label, kecualiKode string) error {
	daftar, err := l.repo.Daftar(ctx)
	if err != nil {
		return fmt.Errorf("masterstatus/usecase: memeriksa keunikan label: %w", err)
	}

	dicari := masterstatus.KunciLabel(label)
	for _, s := range daftar {
		if s.Kode == kecualiKode {
			continue
		}
		if masterstatus.KunciLabel(s.Label) == dicari {
			return masterstatus.ErrLabelSudahAda
		}
	}
	return nil
}

// bersihkan membuang spasi tepi dari masukan pengguna sebelum ia menyentuh penyimpanan.
func bersihkan(s string) string { return strings.TrimSpace(s) }
