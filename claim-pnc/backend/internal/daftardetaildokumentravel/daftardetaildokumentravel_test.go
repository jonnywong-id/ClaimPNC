package daftardetaildokumentravel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetaildokumentravel/repo/memory"
)

// Nama uji di berkas ini menyebut ATURANNYA, bukan nama fungsinya
// (`14-TESTING-STRATEGY.md` §3.2) — daftar uji yang lulus karena itu terbaca sebagai
// daftar aturan yang berlaku.

func TestIsianDipangkasSpasinyaDiKeduaUjung(t *testing.T) {
	// Kolomnya dibaca kembali dengan pemangkasan — kolom CHAR berlebar tetap memadatkan
	// nilainya dengan spasi tanpa memberi tanda apa pun. Tanpa memangkas saat menulis,
	// apa yang disimpan dan apa yang dibaca kembali dapat berbeda, dan selisih itu tidak
	// terlihat di layar karena spasi tidak tampak.
	clean := daftardetaildokumentravel.Input{
		DocumentID:   "  100001  ",
		DocumentName: "  Paspor  ",
	}.Clean()

	require.Equal(t, "100001", clean.DocumentID)
	require.Equal(t, "Paspor", clean.DocumentName)
}

func TestIsianKosongTetapDiterimaKarenaLayarIniTanpaValidasi(t *testing.T) {
	// Work Owner menetapkan 2026-09-21 layar ini meniru Pega apa adanya. Layar lama
	// tidak memuat satu pun `pyRequired` bernilai true maupun Validate rule, sehingga
	// nama dokumen kosong memang tersimpan.
	//
	// Uji ini ada supaya ketiadaan validasi menjadi keputusan yang terlihat, bukan
	// kelalaian yang kelak "diperbaiki" seseorang tanpa menyadari ia mengubah perilaku.
	clean := daftardetaildokumentravel.Input{DocumentID: "   ", DocumentName: "   "}.Clean()

	require.Equal(t, "", clean.DocumentID)
	require.Equal(t, "", clean.DocumentName)
}

func TestMinimalUnggahNegatifDiratakanMenjadiNol(t *testing.T) {
	// "Paling sedikit minus satu berkas" bukan aturan yang dapat dipenuhi maupun
	// dilanggar. Ia diratakan, bukan ditolak, supaya perlakuannya tetap sejalan dengan
	// "tanpa validasi".
	require.Equal(t, 0, daftardetaildokumentravel.Input{MinUpload: -3}.Clean().MinUpload)
	require.Equal(t, 2, daftardetaildokumentravel.Input{MinUpload: 2}.Clean().MinUpload)
}

func TestStatusWajibTerbawaApaAdanya(t *testing.T) {
	// Clean tidak menyentuh STSWAJIB — ia bukan isian teks, dan tidak ada aturan yang
	// berlaku padanya. Uji ini menjaga supaya pemangkasan kelak tidak merembet ke sana.
	require.True(t, daftardetaildokumentravel.Input{Mandatory: true}.Clean().Mandatory)
	require.False(t, daftardetaildokumentravel.Input{Mandatory: false}.Clean().Mandatory)
}

func TestDaftarTerurutMenurutIDLaluIDDokumen(t *testing.T) {
	// Urutannya mengikuti `BrowseLstDocTravel_RD-RD.xml`: ID menaik (pySortOrder 1),
	// lalu DOCID menaik (pySortOrder 2).
	repo := memory.NewRepo(
		daftardetaildokumentravel.Detail{ID: "00002", DocumentID: "100001"},
		daftardetaildokumentravel.Detail{ID: "00001", DocumentID: "100009"},
	)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "00001", list[0].ID)
	require.Equal(t, "00002", list[1].ID)
}

func TestPenambahanMenerbitkanIDYangBelumTerpakai(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepo(memory.SampleList()...)

	first, err := repo.InsertNew(ctx, daftardetaildokumentravel.Input{DocumentID: "100006"})
	require.NoError(t, err)
	require.NotEmpty(t, first.ID)

	second, err := repo.InsertNew(ctx, daftardetaildokumentravel.Input{DocumentID: "100006"})
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
}

func TestMengubahBarisYangTidakAdaMenghasilkanErrNotFound(t *testing.T) {
	_, err := memory.NewRepo().Update(context.Background(), "99999", daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
}

func TestPembatasanPlanDanJaminanTidakAdaDiModulIni(t *testing.T) {
	// Uji ini menjaga sebuah KEPUTUSAN, bukan sebuah perhitungan.
	//
	// `Section/BrowseDocumentTravel-Section.xml:3731` memuat grid berulang
	// `TempDTDocTravel.COVERAGELIST` tanpa kondisi yang menyembunyikannya, sehingga siapa
	// pun yang membaca export akan menyimpulkan layar ini punya grid Plan dan Jaminan.
	// Grid itu TIDAK ADA di aplikasi Pega yang berjalan — Work Owner memeriksa layarnya
	// langsung dan menetapkannya 2026-10-03, dan modul ini sempat membangunnya lalu
	// mencabutnya.
	//
	// Yang dijaga di sini: Detail dan Input tetap berisi LIMA dan EMPAT field saja.
	// Menambahkan kembali pembatasan plan menuntut keputusan Work Owner lebih dulu, dan
	// uji ini yang membuat penambahan diam-diam gagal di CI alih-alih lolos.
	saved, err := memory.NewRepo().InsertNew(context.Background(), daftardetaildokumentravel.Input{
		DocumentID:   "100001",
		DocumentName: "Paspor",
		Mandatory:    true,
		MinUpload:    1,
	})
	require.NoError(t, err)
	require.Equal(t, daftardetaildokumentravel.Detail{
		ID:           saved.ID,
		DocumentID:   "100001",
		DocumentName: "Paspor",
		Mandatory:    true,
		MinUpload:    1,
	}, saved)
}
