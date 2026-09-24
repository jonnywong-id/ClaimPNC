// Package usecase mengorkestrasi modul Inbox Analyst Doctor.
//
// Dua operasi, dan keduanya hanya MEMBACA:
//
//	Metadata  judul kolom, selisih terencana, dan keterbatasan yang berlaku
//	List      satu halaman antrean milik pemanggil
//
// Tidak ada operasi yang menulis. Menyelesaikan tugas Analyst Doctor berarti menjalankan
// Flow Action `SendAnalystDoctor`, yang memindahkan penugasan — dan penugasan masih dimiliki
// Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxanalystdoctor"
)

// Service melayani modul Inbox Analyst Doctor.
type Service struct {
	repoSelector inboxanalystdoctor.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxanalystdoctor.RepoSelector
}

// NewService membentuk layanan modul Inbox Analyst Doctor.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxanalystdoctor/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Column adalah satu judul kolom pada layar.
//
// # Kenapa judul kolom datang dari server
//
// Karena kedelapannya adalah HASIL PEMBACAAN `Harness/inboxAnalystDoctor_Harness-Harness.xml`
// (rule `pyCaption …`), dan tempat pembacaan itu tercatat adalah backend. Menyalinnya ke
// layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang
// lain diperbaiki.
type Column struct {
	// Key adalah nama field pada baris JSON yang diisi kolom ini.
	Key string

	// Title adalah judul yang DILIHAT pengguna, apa adanya dari harness (`D-13`).
	Title string

	// Note adalah keterangan yang ditempelkan pada judul, kosong bila tidak ada.
	//
	// Dipakai kolom yang isinya belum dapat terbawa, supaya sel yang kosong tidak terbaca
	// sebagai data yang hilang.
	Note string
}

// columns adalah kedelapan kolom layar, dalam urutan tampilnya.
//
// Urutannya mengikuti pembacaan grid pada `Section/InboxAnalystDoctor_Section-Section.xml`,
// yang memuat delapan sel berkepala (`pyCellHeader = true`) ditambah satu sel tautan baris.
var columns = []Column{
	{Key: "nomor_case", Title: "Nomor Case"},
	{Key: "nomor_polis", Title: "No Polis"},
	{Key: "nama_tertanggung", Title: "Nama Tertanggung"},
	{Key: "nama_cabang", Title: "Nama Cabang"},
	{Key: "tanggal_pendaftaran", Title: "Tanggal Pendaftaran"},
	{Key: "nama_admin", Title: "Nama Admin"},
	{
		Key:   "komentar_pic_teknis",
		Title: "Komentar dari PIC Teknis",
		Note: "Belum terbawa. Properti `ClaimData.AnalystDoctorRemaks` ditandai Pega " +
			"sendiri sebagai tidak terekspos, sehingga ia tidak punya kolom SQL yang " +
			"dapat dibaca.",
	},
	{
		Key:   "lama_hari",
		Title: "Lama Waktu Klaim",
		Note:  "Umur tugas dalam hari, dihitung sampai hari ini.",
	},
}

// Columns menyerahkan salinan daftar kolom.
func Columns() []Column {
	result := make([]Column, len(columns))
	copy(result, columns)
	return result
}

// PlannedDifferences adalah perbedaan yang DISENGAJA terhadap layar Pega.
//
// Ia dikirim ke layar dan ditampilkan, bukan disimpan sebagai catatan teknis. `D-54`
// menetapkan selisih di luar 13 butir `P-5` menuntut persetujuan Work Owner tertulis, dan
// menyatakannya di layar itulah yang membuat keputusan itu terlihat oleh orang yang memakai
// layarnya — bukan hanya oleh orang yang membaca kodenya.
func PlannedDifferences() []string {
	return []string{
		"Kolom \"Lama Waktu Klaim\" berisi umur tugas dalam hari. Report Definition layar " +
			"lama tidak mengambil satu pun properti durasi, sehingga angkanya dihitung — " +
			"sama seperti pada Inbox Close Claim.",
		"Halaman dipotong basis data, bukan setelah seluruh baris ditarik. Layar lama " +
			"menarik semuanya, memotongnya di 500 baris, lalu menomori halamannya di " +
			"memori; antrean yang melampaui 500 karena itu tidak pernah terlihat utuh.",
		"Kotak cari Nomor Case dan No Polis adalah TAMBAHAN. Layar lama tidak punya " +
			"penyaring apa pun, dan tanpa pencarian sisi server satu klaim menjadi sulit " +
			"ditemukan begitu antreannya dipaginasi.",
	}
}

// Limitations adalah keterbatasan yang berlaku hari ini dan akan hilang dengan sendirinya.
//
// Bedanya dengan PlannedDifferences: yang di atas adalah pilihan yang sudah diputuskan, yang
// di bawah adalah penghalang yang masih menunggu pihak lain. Keduanya dipisah supaya
// keterbatasan yang selesai dapat dihapus tanpa menyentuh keputusan yang masih berlaku.
func Limitations() []string {
	return []string{
		"Kolom \"Komentar dari PIC Teknis\" masih kosong. Properti Pega-nya tidak " +
			"terekspos sebagai kolom SQL, dan nama kolom penggantinya menunggu konfirmasi " +
			"DBA.",
		"Seluruh tugas tahap Analyst Doctor di sistem lama ditujukan ke SATU operator yang " +
			"tertanam di dalam alur (`Flow/Register_Flow.xml`, `Assignment13`). Selama " +
			"penugasannya belum dipindahkan ke master data (`D-15`), pengguna lain melihat " +
			"antrean kosong — dan kosong di sini berarti \"bukan milik Anda\", bukan " +
			"\"tidak ada pekerjaan\".",
		"Pemeriksaan kewenangan menu belum ada (`TKT-F3-005`). Yang menjaga layar ini " +
			"sekarang hanyalah sesi dan portal aktif.",
	}
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	Columns            []Column
	PlannedDifferences []string
	Limitations        []string

	// PageSize adalah ukuran halaman bawaan.
	PageSize int
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: judul kolom sama di
// seluruh entitas, karena ia bentuk layar — bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Columns:            Columns(),
		PlannedDifferences: PlannedDifferences(),
		Limitations:        Limitations(),
		PageSize:           inboxanalystdoctor.DefaultLimit,
	}
}

// Listed adalah satu halaman antrean beserta penyaring yang benar-benar dipakai.
type Listed struct {
	// Filter adalah penyaring setelah dinormalkan.
	//
	// Layar menggambar keadaan kotak cari dan bilah halaman dari sini, bukan dari isian yang
	// ia kirim: permintaan `batas=5000` dipangkas menjadi 100, dan tanpa mengembalikan
	// angka yang dipakai, bilah halamannya akan menghitung jumlah halaman yang salah.
	Filter inboxanalystdoctor.Filter

	Page inboxanalystdoctor.Page
}

// List mengambil satu halaman antrean milik pemanggil.
//
// # Kenapa identitas diperiksa lebih dulu, sebelum portal
//
// Karena tanpanya tidak ada antrean yang dapat dibentuk sama sekali, dan memilih repo untuk
// permintaan yang pasti ditolak hanyalah satu perjalanan yang terbuang. Yang lebih penting:
// urutan ini membuat kegagalan sesi terbaca sebagai kegagalan sesi, bukan sebagai antrean
// kosong.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxanalystdoctor.Caller,
	filter inboxanalystdoctor.Filter,
) (Listed, error) {
	if caller.Login == "" {
		return Listed{}, inboxanalystdoctor.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	clean := filter.Normalize()

	page, err := repo.List(ctx, caller.Login, clean)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil antrean Analyst Doctor: %w", err)
	}

	return Listed{Filter: clean, Page: page}, nil
}
