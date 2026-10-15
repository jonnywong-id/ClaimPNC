package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/masterreas/repo/memory"
	"claim-pnc/internal/masterreas/usecase"
	"claim-pnc/internal/portal"
)

// newService merakit layanan dengan dua portal: ASM berisi contoh, ASI kosong.
//
// ASI sengaja tidak diberi satu baris pun — ia yang membuktikan pemisahan antarentitas
// (`ADR-0030`, `R-20`). Portal lain menghasilkan galat, bukan jatuh ke ASM.
func newService(t *testing.T) (*usecase.Service, *memory.Repo, *memory.Repo) {
	t.Helper()

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{})

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterreas.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)
	return service, asm, asi
}

// TestNewServiceMenolakBahanKosong membuktikan rakitan setengah jadi gagal saat start,
// bukan saat pengguna sedang bekerja.
func TestNewServiceMenolakBahanKosong(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// TestListMengembalikanSeluruhBaris membuktikan cakupan daftarnya seluruh baris entitas.
//
// Cakupan itu adalah REKONSTRUKSI — section grid layar lama tidak ada di export (`R-16`) —
// dan uji ini yang membuat rekonstruksinya tetap terbaca bila kelak berubah.
func TestListMengembalikanSeluruhBaris(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), "ASM", "")
	require.NoError(t, err)
	require.Len(t, list, 7)
}

// TestListTerurutNamaLaluTipe membuktikan urutannya sama dengan `ORDER BY` pada kuerinya.
//
// Urutan yang stabil bukan kerapian: daftar ini dipaginasi di layar, dan urutan yang berubah
// antar permintaan membuat satu baris muncul di dua halaman sekaligus sementara yang lain
// tidak muncul sama sekali.
func TestListTerurutNamaLaluTipe(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), "ASM", "")
	require.NoError(t, err)

	for i := 1; i < len(list); i++ {
		sebelum, sekarang := list[i-1], list[i]
		if sebelum.ReinsurerName != sekarang.ReinsurerName {
			require.Less(t, sebelum.ReinsurerName, sekarang.ReinsurerName)
			continue
		}
		require.LessOrEqual(t, sebelum.Type, sekarang.Type)
	}
}

// TestListMenyaringEmpatKolom membuktikan kata kunci mencari kode, nama, login, dan email —
// dan TIDAK mencari negara.
//
// Negara sengaja dikecualikan; lihat masterreas.Filter. Tanpa uji ini, menambahkannya
// "karena kelihatan berguna" tidak akan tertahan oleh apa pun.
func TestListMenyaringEmpatKolom(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	byID, err := service.List(ctx, "ASM", "RE-004")
	require.NoError(t, err)
	require.Len(t, byID, 1)
	require.Equal(t, "Cakrawala Re Asia", byID[0].ReinsurerName)

	byName, err := service.List(ctx, "ASM", "nusantara")
	require.NoError(t, err)
	require.Len(t, byName, 3, "pencarian tidak boleh peka huruf besar-kecil")

	byLogin, err := service.List(ctx, "ASM", "AndalasRe")
	require.NoError(t, err)
	require.Len(t, byLogin, 2, "satu login dipakai dua kode reas; keduanya harus tampil")

	byEmail, err := service.List(ctx, "ASM", "xol.nusantara@contoh.invalid")
	require.NoError(t, err)
	require.Len(t, byEmail, 1)

	byCountry, err := service.List(ctx, "ASM", "Singapura")
	require.NoError(t, err)
	require.Empty(t, byCountry, "negara TIDAK ikut dicari; lihat masterreas.Filter")
}

// TestListMemangkasKataKunci membuktikan spasi ujung tidak membuat pencarian gagal diam-diam.
func TestListMemangkasKataKunci(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), "ASM", "  andalas  ")
	require.NoError(t, err)
	require.Len(t, list, 2)
}

// TestListTerpisahAntarPortal membuktikan satu entitas tidak pernah melihat data entitas
// lain.
//
// Ini kelas cacat yang paling ingin dicegah `R-20`: layarnya tampil normal, angkanya masuk
// akal, dan yang salah hanya milik siapa data itu.
func TestListTerpisahAntarPortal(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	asm, err := service.List(ctx, "ASM", "")
	require.NoError(t, err)
	require.NotEmpty(t, asm)

	asi, err := service.List(ctx, "ASI", "")
	require.NoError(t, err)
	require.Empty(t, asi, "ASI tidak diberi satu baris pun; ia tidak boleh melihat milik ASM")
}

// TestListMenolakPortalYangBelumSiap membuktikan portal yang koneksinya belum hidup ditolak,
// bukan dilayani portal utama sebagai cadangan.
func TestListMenolakPortalYangBelumSiap(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.List(context.Background(), "SMAS", "")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// TestListMeneruskanGalatRepo membuktikan kegagalan basis data tidak tersamar menjadi daftar
// kosong.
//
// Daftar kosong dan kegagalan membaca terlihat sama di layar bila galatnya ditelan, dan
// pengguna tidak punya cara membedakan "belum ada mitra" dari "tabelnya tidak terbaca".
func TestListMeneruskanGalatRepo(t *testing.T) {
	service, asm, _ := newService(t)

	kandas := errors.New("koneksi terputus")
	asm.SetError(kandas)

	_, err := service.List(context.Background(), "ASM", "")
	require.ErrorIs(t, err, kandas)
}

// TestEnsurePortalReady membuktikan penolakan lebih awal bekerja tanpa menyentuh satu baris
// pun.
func TestEnsurePortalReady(t *testing.T) {
	service, _, _ := newService(t)

	require.NoError(t, service.EnsurePortalReady("ASM"))
	require.Error(t, service.EnsurePortalReady("SMAS"))
}
