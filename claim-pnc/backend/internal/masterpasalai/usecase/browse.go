// Package usecase mengorkestrasi perkara Master Pasal AI.
//
// Ia yang mengetahui urutan langkah; aturan penyaring ada di paket domain, dan cara
// membacanya dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"

	"claim-pnc/internal/masterpasalai"
)

// Service adalah pintu masuk seluruh perkara Master Pasal AI.
//
// Satu method saja — modul ini baca-saja. Lihat doc comment masterpasalai.Repo.
type Service struct {
	repoSelector masterpasalai.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpasalai.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang: rakitan
// yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpasalai/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan satu halaman Master Pasal AI milik satu portal.
//
// # Urutannya mengikuti `Activity/GetListPasalAI_act.xml`
//
//	langkah 2-3  susun penyaring dari kata kunci   -> Filter.Clean, lalu repo
//	langkah 4    cacah seluruh baris yang cocok    -> Page.Total
//	langkah 5    hitung jendela halaman            -> Filter.Offset
//	langkah 6    baca barisnya                     -> Page.Clause
//
// Keempatnya dikerjakan Repo.List dalam satu pemanggilan; alasannya ada pada doc comment
// method itu.
//
// # Kata kunci kosong TIDAK ditolak
//
// Ia jalur yang normal, bukan kesalahan: layar lama pun membuka daftar tanpa penyaring apa
// pun pada pemuatan pertama — `TempQuery.AlasanKlaim := ""` ketika `TempSearch.Country` kosong
// (`Activity/GetListPasalAI_act.xml:379`, `:432`).
//
// Ini berbeda dari pencarian lini bisnis pada Master Pasal Kerugian, yang menolak kata kunci
// di bawah dua huruf. Di sana kata kunci kosong akan menarik seluruh tabel ke dalam sebuah
// daftar pilihan; di sini daftar penuh memang yang diminta, dan paginasi yang menahan
// besarnya.
func (l *Service) List(
	ctx context.Context,
	portalAlias string,
	filter masterpasalai.Filter,
) (masterpasalai.Page, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpasalai.Page{}, err
	}
	return store.List(ctx, filter.Clean())
}

// EnsurePortalReady memastikan penyimpanan portal dapat dipilih, tanpa membaca apa pun.
//
// Dipakai pemeriksaan kesiapan; ia memisahkan "portal ini belum siap" dari "pembacaannya
// gagal", dua hal yang di mata pengguna terlihat sama tetapi ditangani berbeda.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	_, err := l.repoSelector(portalAlias)
	return err
}
