// Package usecase mengorkestrasi penyusunan menu untuk satu pemanggil.
//
// Tugasnya tiga, dan tidak lebih: mencari group milik login, mengumpulkan izin untuk
// group-group itu beserta loginnya sendiri, lalu menyusun pohon menunya. Ia tidak tahu
// apa pun tentang HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/menu"
)

// Service adalah pintu masuk seluruh perkara menu.
type Service struct {
	repo menu.Repo
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// Repo membaca peta menu dan kewenangannya. Wajib.
	Repo menu.Repo
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.Repo == nil {
		return nil, errors.New("menu/usecase: Repo wajib diisi")
	}
	return &Service{repo: o.Repo}, nil
}

// ForLogin menyusun menu yang boleh dilihat sebuah login.
//
// # Urutan langkahnya adalah aturan bisnis, bukan detail teknis
//
// Ditetapkan Work Owner 2026-09-18, dari login yang DIKETIK pengguna:
//
//  1. cari GROUP_ID apa saja yang diikutinya di M_LOGIN_GROUP_PNC;
//  2. cari izin menu untuk group-group itu di M_OTORISASI_PNC;
//  3. cari juga izin menu untuk loginnya sendiri di tabel yang sama.
//
// Langkah 2 dan 3 DIGABUNG, tidak saling menggantikan: izin yang diberikan langsung
// kepada seseorang berlaku di samping izin yang ia warisi dari groupnya. Itu terbaca
// langsung dari isi contohnya — login `JONNY` anggota group `IT`, dan keduanya
// memberikan butir menu yang berbeda.
//
// # Kenapa izin dibaca pada setiap permintaan, bukan sekali saat masuk
//
// `11-SECURITY.md` §2.2 menetapkan izin TIDAK ditanam di dalam token, supaya pencabutan
// hak berlaku hampir seketika alih-alih menunggu sesinya habis. Menyimpan hasil susunan
// ini di sesi akan mengembalikan persoalan yang sama.
//
// Login kosong dijawab menu kosong, bukan galat: pemanggilnya sudah lewat middleware
// sesi, jadi keadaan itu berarti catatan penggunanya yang tidak lengkap — dan menu
// kosong lebih jujur daripada menu penuh.
func (s *Service) ForLogin(ctx context.Context, loginID string) ([]menu.Node, error) {
	if strings.TrimSpace(loginID) == "" {
		return nil, nil
	}

	groups, err := s.repo.GroupsOf(ctx, loginID)
	if err != nil {
		return nil, fmt.Errorf("menu/usecase: membaca group login: %w", err)
	}

	subjects := menu.Subjects(loginID, groups)

	authorized, err := s.repo.AuthorizedIDs(ctx, menu.AppName, subjects)
	if err != nil {
		return nil, fmt.Errorf("menu/usecase: membaca otorisasi menu: %w", err)
	}
	if len(authorized) == 0 {
		// Tidak ada satu pun izin. Peta menunya tidak perlu dibaca sama sekali — dan
		// tanpa ini, pengguna tanpa izin tetap membebani basis data dengan pembacaan
		// yang hasilnya pasti tersaring habis.
		return nil, nil
	}

	items, err := s.repo.List(ctx, menu.AppName)
	if err != nil {
		return nil, fmt.Errorf("menu/usecase: membaca peta menu: %w", err)
	}

	return menu.BuildTree(items, authorized), nil
}
