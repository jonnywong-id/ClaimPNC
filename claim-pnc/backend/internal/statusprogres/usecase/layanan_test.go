package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/statusprogres"
	"claim-pnc/internal/statusprogres/repo/memori"
	"claim-pnc/internal/statusprogres/usecase"
)

// duaPortal menyiapkan layanan dengan DUA penyimpanan terpisah, satu per entitas.
//
// Ini bentuk pengujian yang sengaja dipilih: cacat yang paling ingin dicegah modul ini
// adalah data satu badan hukum masuk ke basis data badan hukum lain (R-20), dan cacat
// itu hanya terlihat bila ada lebih dari satu penyimpanan.
func duaPortal(t *testing.T) (*usecase.Layanan, *memori.Repo, *memori.Repo) {
	t.Helper()

	asm := memori.RepoBaru(memori.DaftarContoh()...)
	asi := memori.RepoBaru()

	layanan, err := usecase.LayananBaru(usecase.Opsi{
		PemilihRepo: func(alias string) (statusprogres.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrBelumSiap
			}
		},
	})
	require.NoError(t, err)
	return layanan, asm, asi
}

func TestLayananMenolakBahanTidakLengkap(t *testing.T) {
	_, err := usecase.LayananBaru(usecase.Opsi{})
	require.Error(t, err, "rakitan setengah jadi harus gagal saat start, bukan saat pengguna bekerja")
}

func TestDaftarDibacaDariPortalYangDiminta(t *testing.T) {
	layanan, _, _ := duaPortal(t)

	dariASM, err := layanan.Daftar(context.Background(), "ASM")
	require.NoError(t, err)
	require.Len(t, dariASM, 6)

	dariASI, err := layanan.Daftar(context.Background(), "ASI")
	require.NoError(t, err)
	require.Empty(t, dariASI, "entitas lain punya basis datanya sendiri (ADR-0030)")
}

// Penambahan pada satu entitas TIDAK boleh terlihat di entitas lain. Inilah pemisahan
// yang ditetapkan ADR-0030 dan yang dilindungi R-20.
func TestPenambahanTidakBocorAntarPortal(t *testing.T) {
	layanan, _, _ := duaPortal(t)
	ctx := context.Background()

	tersimpan, err := layanan.Tambah(ctx, "ASI", statusprogres.Isian{
		Nama:       "MENUNGGU BERKAS",
		KodePosisi: "002",
	})
	require.NoError(t, err)
	require.Equal(t, "01", tersimpan.ID, "tabel ASI masih kosong, nomor mulai dari 1")

	diASI, err := layanan.Daftar(ctx, "ASI")
	require.NoError(t, err)
	require.Len(t, diASI, 1)

	diASM, err := layanan.Daftar(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, diASM, 6, "isi ASM tidak berubah sama sekali")
	for _, sp := range diASM {
		require.NotEqual(t, "MENUNGGU BERKAS", sp.Nama)
	}
}

// Portal yang tidak dapat dilayani menghasilkan galat — TIDAK dialihkan ke portal utama.
func TestPortalTidakDikenalDitolakBukanDialihkan(t *testing.T) {
	layanan, _, _ := duaPortal(t)
	ctx := context.Background()

	_, err := layanan.Daftar(ctx, "SMAS")
	require.ErrorIs(t, err, portal.ErrBelumSiap)

	_, err = layanan.Tambah(ctx, "SMAS", statusprogres.Isian{Nama: "APA SAJA", KodePosisi: "002"})
	require.ErrorIs(t, err, portal.ErrBelumSiap)

	_, err = layanan.Ubah(ctx, "SMAS", "01", statusprogres.Isian{Nama: "APA SAJA", KodePosisi: "002"})
	require.ErrorIs(t, err, portal.ErrBelumSiap)

	// Yang terpenting: tidak ada satu baris pun yang masuk ke entitas mana pun.
	diASM, err := layanan.Daftar(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, diASM, 6)
	diASI, err := layanan.Daftar(ctx, "ASI")
	require.NoError(t, err)
	require.Empty(t, diASI)
}

func TestNomorBaruMeneruskanYangTertinggi(t *testing.T) {
	layanan, _, _ := duaPortal(t)

	tersimpan, err := layanan.Tambah(context.Background(), "ASM", statusprogres.Isian{
		Nama:       "MENUNGGU PEMBAYARAN",
		KodePosisi: "007",
	})
	require.NoError(t, err)
	require.Equal(t, "07", tersimpan.ID, "contoh berisi 01..06, berikutnya 07")
	require.Equal(t, "MENUNGGU PEMBAYARAN", tersimpan.Nama)
	require.Equal(t, "007", tersimpan.KodePosisi)
}

// Isian tidak sah ditolak SEBELUM menyentuh penyimpanan.
func TestIsianTidakSahTidakMenyentuhPenyimpanan(t *testing.T) {
	layanan, _, _ := duaPortal(t)
	ctx := context.Background()

	_, err := layanan.Tambah(ctx, "ASM", statusprogres.Isian{Nama: "", KodePosisi: "999"})
	var validasi *statusprogres.GalatValidasi
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Pelanggaran, 2)

	daftar, err := layanan.Daftar(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, daftar, 6, "tidak ada baris yang tersisip")
}

func TestUbahMenyimpanNamaDanPosisiTanpaMengubahID(t *testing.T) {
	layanan, _, _ := duaPortal(t)
	ctx := context.Background()

	hasil, err := layanan.Ubah(ctx, "ASM", "03", statusprogres.Isian{
		Nama:       "SURVEI DIJADWALKAN",
		KodePosisi: "006",
	})
	require.NoError(t, err)
	require.Equal(t, "03", hasil.ID, "ID adalah kunci baris, bukan isian")
	require.Equal(t, "SURVEI DIJADWALKAN", hasil.Nama)
	require.Equal(t, "006", hasil.KodePosisi)

	dibaca, err := layanan.Ambil(ctx, "ASM", "03")
	require.NoError(t, err)
	require.Equal(t, "SURVEI DIJADWALKAN", dibaca.Nama)

	// Jumlah baris tidak bertambah: penyuntingan bukan penambahan.
	daftar, err := layanan.Daftar(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, daftar, 6)
}

func TestUbahBarisYangTidakAda(t *testing.T) {
	layanan, _, _ := duaPortal(t)

	_, err := layanan.Ubah(context.Background(), "ASM", "99", statusprogres.Isian{
		Nama:       "APA SAJA",
		KodePosisi: "002",
	})
	require.ErrorIs(t, err, statusprogres.ErrTidakDitemukan)
}

// Galat penyimpanan diteruskan apa adanya, tidak ditelan menjadi daftar kosong.
func TestGalatPenyimpananDiteruskan(t *testing.T) {
	layanan, asm, _ := duaPortal(t)
	gagal := errors.New("basis data tidak dapat dihubungi")
	asm.SetGalat(gagal)

	_, err := layanan.Daftar(context.Background(), "ASM")
	require.ErrorIs(t, err, gagal)
}

func TestPosisiDilayaniDariSatuTempat(t *testing.T) {
	layanan, _, _ := duaPortal(t)
	require.Equal(t, statusprogres.DaftarPosisi(), layanan.Posisi())
}
