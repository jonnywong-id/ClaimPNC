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
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanID: " TP01 ", PlanName: " Silver ", CoverageID: " TC01 ", CoverageName: " Medis "},
		},
	}.Clean()

	require.Equal(t, "100001", clean.DocumentID)
	require.Equal(t, "Paspor", clean.DocumentName)
	require.Equal(t, "TP01", clean.Coverages[0].PlanID)
	require.Equal(t, "Silver", clean.Coverages[0].PlanName)
	require.Equal(t, "TC01", clean.Coverages[0].CoverageID)
	require.Equal(t, "Medis", clean.Coverages[0].CoverageName)
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

func TestBarisJaminanYangSeluruhnyaKosongDibuang(t *testing.T) {
	// Grid di layar selalu menyisakan baris yang baru ditambahkan tetapi belum diisi.
	// Menyimpannya berarti menulis pembatasan plan yang tidak membatasi apa pun, lalu
	// membacanya kembali sebagai baris hantu di form berikutnya.
	clean := daftardetaildokumentravel.Input{
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanName: "Silver"},
			{PlanID: "  ", PlanName: "  ", CoverageID: "  ", CoverageName: "  "},
			{CoverageName: "Medis"},
		},
	}.Clean()

	require.Len(t, clean.Coverages, 2)
	require.Equal(t, "Silver", clean.Coverages[0].PlanName)
	require.Equal(t, "Medis", clean.Coverages[1].CoverageName)
}

func TestBarisJaminanYangHanyaTerisiSebagianTETAPDisimpan(t *testing.T) {
	// Ini yang membedakan "membaca maksud" dari "memvalidasi". Baris berisi plan tanpa
	// jaminan adalah isian yang sah di layar lama — autocomplete-nya tidak menuntut
	// keduanya terisi — dan membuangnya akan menghilangkan isian yang benar-benar
	// diketik petugas.
	clean := daftardetaildokumentravel.Input{
		Coverages: []daftardetaildokumentravel.CoverageInput{{PlanID: "TP01", PlanName: "Silver"}},
	}.Clean()

	require.Len(t, clean.Coverages, 1)
	require.Equal(t, "", clean.Coverages[0].CoverageID)
}

func TestMinimalUnggahNegatifDiratakanMenjadiNol(t *testing.T) {
	// "Paling sedikit minus satu berkas" bukan aturan yang dapat dipenuhi maupun
	// dilanggar. Ia diratakan, bukan ditolak, supaya perlakuannya tetap sejalan dengan
	// "tanpa validasi".
	require.Equal(t, 0, daftardetaildokumentravel.Input{MinUpload: -3}.Clean().MinUpload)
	require.Equal(t, 2, daftardetaildokumentravel.Input{MinUpload: 2}.Clean().MinUpload)
}

func TestDaftarTidakMembawaJaminannya(t *testing.T) {
	// Kueri daftar memang tidak membacanya. Adapter memori yang membawanya akan membuat
	// uji layar lulus di sini lalu gagal terhadap Oracle, karena di sana daftarnya
	// benar-benar kosong.
	repo := memory.NewRepo(memory.SampleList()...)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, row := range list {
		require.Empty(t, row.Coverages, "baris %s membawa jaminan pada daftar", row.ID)
	}
}

func TestAmbilSatuBarisMembawaJaminannya(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)

	row, err := repo.Get(context.Background(), "00003")
	require.NoError(t, err)
	require.Len(t, row.Coverages, 2)
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

func TestPenyimpananMenggantiSeluruhDaftarJaminan(t *testing.T) {
	// Grid di form mengirim susunan akhir yang dikehendaki petugas, dan tidak ada satu
	// pun penanda di sana yang menyatakan baris mana yang baru, mana yang berubah, dan
	// mana yang dibuang. Penggantian menyeluruh adalah satu-satunya tafsiran yang tidak
	// menebak — dan uji ini yang menjaganya tetap begitu.
	ctx := context.Background()
	repo := memory.NewRepo(memory.SampleList()...)

	_, err := repo.Update(ctx, "00003", daftardetaildokumentravel.Input{
		DocumentID:   "100004",
		DocumentName: "Laporan Kehilangan Bagasi",
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanID: "TP03", PlanName: "Platinum", CoverageID: "TC04", CoverageName: "Pembatalan"},
		},
	})
	require.NoError(t, err)

	row, err := repo.Get(ctx, "00003")
	require.NoError(t, err)
	require.Len(t, row.Coverages, 1)
	require.Equal(t, "TP03", row.Coverages[0].PlanID)
}

func TestPenambahanMenerbitkanIDYangBelumTerpakai(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepo(memory.SampleList()...)

	saved, err := repo.InsertNew(ctx, daftardetaildokumentravel.Input{DocumentID: "100006"})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)

	// ID baris jaminan diterbitkan dari urutan yang SAMA dengan ID induknya, meniru
	// adapter SQL yang memakai satu urutan untuk kedua tabel. Karena itu keduanya tidak
	// boleh pernah bertabrakan.
	withCoverage, err := repo.InsertNew(ctx, daftardetaildokumentravel.Input{
		DocumentID: "100006",
		Coverages:  []daftardetaildokumentravel.CoverageInput{{PlanName: "Silver"}},
	})
	require.NoError(t, err)
	require.NotEqual(t, withCoverage.ID, withCoverage.Coverages[0].ID)
	require.NotEqual(t, saved.ID, withCoverage.ID)
}

func TestMengubahBarisYangTidakAdaMenghasilkanErrNotFound(t *testing.T) {
	_, err := memory.NewRepo().Update(context.Background(), "99999", daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
}

func TestMengubahSenaraiHasilTidakMengubahIsiRepo(t *testing.T) {
	// Tanpa salinan, pemanggil memegang senarai yang sama dengan yang tersimpan, dan
	// mengubah satu elemennya akan mengubah isi repo tanpa melewati Update sama sekali.
	// Adapter SQL tidak punya kelemahan itu karena ia selalu menyusun senarai baru dari
	// hasil kueri — adapter memori harus menirunya, bukan sekadar bekerja.
	ctx := context.Background()
	repo := memory.NewRepo(memory.SampleList()...)

	row, err := repo.Get(ctx, "00003")
	require.NoError(t, err)
	row.Coverages[0].PlanName = "DIUBAH DARI LUAR"

	again, err := repo.Get(ctx, "00003")
	require.NoError(t, err)
	require.NotEqual(t, "DIUBAH DARI LUAR", again.Coverages[0].PlanName)
}
