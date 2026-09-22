package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/masterstatusprogres/repo/memory"
	"claim-pnc/internal/masterstatusprogres/usecase"
	"claim-pnc/internal/portal"
)

// twoPortals2 merakit dua entitas yang benar-benar terpisah.
//
// ASI sengaja dibiarkan KOSONG. Itulah yang membuktikan pemisahan antarbadan hukum
// (R-20): bila layanan diam-diam jatuh ke portal utama, daftar ASI akan berisi baris ASM
// dan ujinya gagal.
func twoPortals2(t *testing.T) (*usecase.Service2, *memory.Repo, *memory.Repo2) {
	t.Helper()

	asmParent := memory.NewRepo(memory.SampleList()...)
	asiParent := memory.NewRepo()

	asm := memory.NewRepo2(asmParent, memory.SampleList2()...)
	asi := memory.NewRepo2(asiParent)

	service, err := usecase.NewService2(usecase.Options2{
		RepoSelector: func(alias string) (masterstatusprogres.Repo2, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		ParentSelector: func(alias string) (masterstatusprogres.Repo, error) {
			switch alias {
			case "ASM":
				return asmParent, nil
			case "ASI":
				return asiParent, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	return service, asmParent, asm
}

// Bahan yang tidak lengkap ditolak saat PERAKITAN, bukan saat permintaan pertama datang.
//
// Rakitan setengah jadi yang baru gagal ketika pengguna sedang bekerja jauh lebih sulit
// ditelusuri daripada proses yang menolak start.
func TestNewService2RejectsIncompleteOptions(t *testing.T) {
	repoSelector := func(string) (masterstatusprogres.Repo2, error) { return nil, nil }
	parentSelector := func(string) (masterstatusprogres.Repo, error) { return nil, nil }

	_, err := usecase.NewService2(usecase.Options2{ParentSelector: parentSelector})
	require.Error(t, err, "RepoSelector tingkat 2 wajib")

	_, err = usecase.NewService2(usecase.Options2{RepoSelector: repoSelector})
	require.Error(t, err, "ParentSelector wajib — penambahan tingkat 2 membaca baris induknya")

	_, err = usecase.NewService2(usecase.Options2{RepoSelector: repoSelector, ParentSelector: parentSelector})
	require.NoError(t, err)
}

// Setiap portal membaca isinya sendiri. Tidak ada jalur cadangan ke portal utama.
func TestService2ListIsPerPortal(t *testing.T) {
	service, _, _ := twoPortals2(t)

	asmList, err := service.List(context.Background(), "ASM")
	require.NoError(t, err)
	require.NotEmpty(t, asmList)

	asiList, err := service.List(context.Background(), "ASI")
	require.NoError(t, err)
	require.Empty(t, asiList, "portal lain tidak boleh melihat baris milik ASM")
}

// Portal yang tidak dikenal menghasilkan galat, TIDAK dialihkan ke portal utama.
//
// Ini inti R-20: kegagalannya tidak terlihat sebagai galat — layarnya tampil normal dan
// angkanya masuk akal; yang salah hanya milik siapa data itu.
func TestService2RejectsUnknownPortal(t *testing.T) {
	service, _, _ := twoPortals2(t)

	_, err := service.List(context.Background(), "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.ListParents(context.Background(), "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(context.Background(), "TIDAKADA", masterstatusprogres.Input2{
		Name: "APA SAJA", ParentID: "01",
	})
	require.ErrorIs(t, err, portal.ErrNotReady)

	require.Error(t, service.EnsurePortalReady("TIDAKADA"))
	require.NoError(t, service.EnsurePortalReady("ASM"))
}

// Dropdown induk dibaca dari tabel tingkat 1 milik portal yang SAMA.
//
// Bukan daftar tetap milik aplikasi seperti halnya posisi klaim pada tingkat 1: dua
// entitas punya Status Progres 1 yang berbeda.
func TestService2ListParentsIsPerPortal(t *testing.T) {
	service, _, _ := twoPortals2(t)

	asmParentList, err := service.ListParents(context.Background(), "ASM")
	require.NoError(t, err)
	require.NotEmpty(t, asmParentList)

	asiParentList, err := service.ListParents(context.Background(), "ASI")
	require.NoError(t, err)
	require.Empty(t, asiParentList)
}

// Induk yang baru ditambahkan lewat layar tingkat 1 LANGSUNG muncul di dropdown tingkat 2.
//
// Ia membuktikan keduanya membaca sumber yang sama, bukan dua salinan yang dapat
// menyimpang — perilaku yang sama dengan adapter SQL, yang membaca tabel induk di dalam
// transaksi yang sama.
func TestService2SeesNewlyAddedParent(t *testing.T) {
	service, asmParent, _ := twoPortals2(t)

	before, err := service.ListParents(context.Background(), "ASM")
	require.NoError(t, err)

	added, err := asmParent.InsertNew(context.Background(), masterstatusprogres.Input{
		Name:         "MENUNGGU PEMBAYARAN",
		PositionCode: "AKSEPTASI",
	})
	require.NoError(t, err)

	after, err := service.ListParents(context.Background(), "ASM")
	require.NoError(t, err)
	require.Len(t, after, len(before)+1)

	found := false
	for _, p := range after {
		if p.ID == added.ID {
			found = true
		}
	}
	require.True(t, found, "induk baru wajib terlihat tanpa restart")
}

// Isian dibersihkan lebih dulu, lalu diperiksa. Isian yang salah tidak pernah sampai ke
// penyimpanan.
func TestService2CreateValidatesBeforeStoring(t *testing.T) {
	service, _, repo := twoPortals2(t)

	before, err := repo.List(context.Background())
	require.NoError(t, err)

	_, err = service.Create(context.Background(), "ASM", masterstatusprogres.Input2{
		Name: "   ", ParentID: "01",
	})
	var validationError *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationError)

	after, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, after, len(before), "isian yang ditolak tidak boleh menyisakan baris")
}

// Penambahan menurunkan ID sendiri dan MENYALIN nama induk.
//
// Keduanya diterbitkan server, tidak diterima dari layar: nilainya sudah ada di basis
// data, dan menerimanya dari peramban berarti mempercayai klien atas hal yang tidak perlu
// dipercayakan kepadanya.
func TestService2CreateDerivesIDAndCopiesParentName(t *testing.T) {
	service, asmParent, _ := twoPortals2(t)

	parentList, err := asmParent.List(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, parentList)
	parent := parentList[0]

	saved, err := service.Create(context.Background(), "ASM", masterstatusprogres.Input2{
		Name:     "  DOKUMEN TAMBAHAN DITERIMA  ",
		ParentID: parent.ID,
	})
	require.NoError(t, err)

	require.NotEmpty(t, saved.ID)
	require.Equal(t, "DOKUMEN TAMBAHAN DITERIMA", saved.Name, "isian dipangkas lebih dulu")
	require.Equal(t, parent.ID, saved.ParentID)
	require.Equal(t, parent.Name, saved.ParentName, "nama induk disalin dari barisnya, bukan dari layar")
	require.Empty(t, saved.Kind, "TIPE tidak pernah ditulis")

	// ID tingkat 2 tidak pernah berawalan nol — lihat FormatID2.
	require.NotEqual(t, byte('0'), saved.ID[0])

	stored, err := service.Get(context.Background(), "ASM", saved.ID)
	require.NoError(t, err)
	require.Equal(t, saved, stored)
}

// Penyuntingan mengubah nama DAN memindahkan induk, lalu menyalin ulang nama induknya.
//
// Penyalinan ulang itu yang paling mudah terlewat: `STS_PROGRESS1` adalah salinan, dan
// induk yang berpindah tanpa namanya ikut berpindah meninggalkan baris yang menunjuk
// induk A sambil menyandang nama induk B.
func TestService2UpdateMovesParentAndRecopiesItsName(t *testing.T) {
	service, asmParent, _ := twoPortals2(t)

	parents, err := asmParent.List(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parents), 2)
	asal, tujuan := parents[0], parents[1]

	awal, err := service.Create(context.Background(), "ASM", masterstatusprogres.Input2{
		Name: "DOKUMEN AWAL", ParentID: asal.ID,
	})
	require.NoError(t, err)
	require.Equal(t, asal.Name, awal.ParentName)

	diubah, err := service.Update(context.Background(), "ASM", awal.ID, masterstatusprogres.Input2{
		Name: "  DOKUMEN SETELAH DIUBAH  ", ParentID: tujuan.ID,
	})
	require.NoError(t, err)

	require.Equal(t, awal.ID, diubah.ID, "ID tidak pernah berubah")
	require.Equal(t, "DOKUMEN SETELAH DIUBAH", diubah.Name, "isian dipangkas lebih dulu")
	require.Equal(t, tujuan.ID, diubah.ParentID)
	require.Equal(t, tujuan.Name, diubah.ParentName, "nama induk disalin ulang dari induk BARU")

	// Perubahannya benar-benar tersimpan, bukan hanya dikembalikan.
	tersimpan, err := service.Get(context.Background(), "ASM", awal.ID)
	require.NoError(t, err)
	require.Equal(t, diubah, tersimpan)
}

// Isian diperiksa lebih dulu; yang salah tidak pernah sampai ke penyimpanan.
func TestService2UpdateValidatesBeforeStoring(t *testing.T) {
	service, _, _ := twoPortals2(t)

	awal, err := service.Create(context.Background(), "ASM", masterstatusprogres.Input2{
		Name: "SEBELUM", ParentID: "01",
	})
	require.NoError(t, err)

	_, err = service.Update(context.Background(), "ASM", awal.ID, masterstatusprogres.Input2{
		Name: "   ", ParentID: "01",
	})
	var validationError *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationError)

	tetap, err := service.Get(context.Background(), "ASM", awal.ID)
	require.NoError(t, err)
	require.Equal(t, "SEBELUM", tetap.Name, "isian yang ditolak tidak boleh menyentuh baris")
}

// Baris yang tidak ada menghasilkan ErrNotFound, bukan diam-diam berhasil.
func TestService2UpdateReportsMissingRow(t *testing.T) {
	service, _, _ := twoPortals2(t)

	_, err := service.Update(context.Background(), "ASM", "9999", masterstatusprogres.Input2{
		Name: "APA SAJA", ParentID: "01",
	})
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)
}

// Induk baru yang tidak ada ditolak dengan galat yang MENYEBUT induknya.
func TestService2UpdateRejectsMissingParent(t *testing.T) {
	service, _, _ := twoPortals2(t)

	awal, err := service.Create(context.Background(), "ASM", masterstatusprogres.Input2{
		Name: "SEBELUM", ParentID: "01",
	})
	require.NoError(t, err)

	_, err = service.Update(context.Background(), "ASM", awal.ID, masterstatusprogres.Input2{
		Name: "SESUDAH", ParentID: "99",
	})
	require.ErrorIs(t, err, masterstatusprogres.ErrParentNotFound)
}

// Portal tetap ditegakkan pada jalur ubah — bukan hanya pada jalur baca dan tambah.
func TestService2UpdateRejectsUnknownPortal(t *testing.T) {
	service, _, _ := twoPortals2(t)

	_, err := service.Update(context.Background(), "TIDAKADA", "1", masterstatusprogres.Input2{
		Name: "APA SAJA", ParentID: "01",
	})
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// Induk yang tidak ada ditolak dengan galat yang MENYEBUT induknya, bukan galat umum.
func TestService2CreateRejectsMissingParent(t *testing.T) {
	service, _, _ := twoPortals2(t)

	_, err := service.Create(context.Background(), "ASM", masterstatusprogres.Input2{
		Name:     "APA SAJA",
		ParentID: "99",
	})
	require.ErrorIs(t, err, masterstatusprogres.ErrParentNotFound)
}

// Baris yang tidak ada menghasilkan ErrNotFound, bukan baris kosong.
func TestService2GetReportsMissingRow(t *testing.T) {
	service, _, _ := twoPortals2(t)

	_, err := service.Get(context.Background(), "ASM", "9999")
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)
}
