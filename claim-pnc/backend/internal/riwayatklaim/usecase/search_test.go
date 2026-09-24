package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/internal/riwayatklaim/repo/memory"
	"claim-pnc/internal/riwayatklaim/usecase"
)

const portalContoh = "ASM"

// jamTetap membuat waktu dapat diuji tanpa bergantung pada jam mesin.
type jamTetap struct{ pada time.Time }

func (j jamTetap) Now() time.Time { return j.pada }

// rakit menyusun layanan beserta kedua penyimpanan memorinya.
func rakit(t *testing.T, protections ...riwayatklaim.Protection) (
	*usecase.Service, *memory.ProtectionRepo,
) {
	t.Helper()

	claims := memory.NewRepo(memory.SampleClaims()...)
	gate := memory.NewProtectionRepo(protections...)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (riwayatklaim.Repo, error) {
			require.Equal(t, portalContoh, alias)
			return claims, nil
		},
		ProtectionSelector: func(alias string) (riwayatklaim.ProtectionRepo, error) {
			require.Equal(t, portalContoh, alias)
			return gate, nil
		},
		Clock: jamTetap{pada: time.Date(2026, time.September, 20, 3, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)

	return service, gate
}

func penggunaTerdaftar(jatah int) riwayatklaim.Protection {
	return riwayatklaim.Protection{
		Login:       "adminpnc",
		SearchQuota: jatah,
		SubModules:  []string{riwayatklaim.ModuleKey},
	}
}

var pemanggil = riwayatklaim.Caller{Login: "adminpnc"}

// Membuka layar MEMAKAI satu jatah — perilaku sistem lama yang dibangun penuh atas
// keputusan Work Owner 2026-09-20.
func TestBukaLayarMemakaiSatuJatah(t *testing.T) {
	service, gate := rakit(t, penggunaTerdaftar(5))

	opened, err := service.Open(context.Background(), portalContoh, pemanggil)
	require.NoError(t, err)

	require.Equal(t, 5, opened.Access.QuotaTotal)
	require.Equal(t, 1, opened.Access.QuotaUsed)
	require.Equal(t, 4, opened.Access.QuotaRemaining)
	require.Len(t, opened.SearchTypes, 12)

	jejak := gate.Usage()
	require.Len(t, jejak, 1)
	require.True(t, jejak[0].ConsumesQuota, "pembukaan layar mengurangi jatah")
	require.Empty(t, jejak[0].SearchTypeCode, "belum ada tipe pencarian saat layar dibuka")
	require.Equal(t, riwayatklaim.ModuleKey, jejak[0].Module)
}

// Membuka layar berulang kali menghabiskan jatah, dan itu memang perilaku sistem lama.
//
// Uji ini ada supaya akibatnya terlihat sebagai keputusan, bukan ditemukan penguji
// sebagai kejutan: layar WAJIB memanggil Open sekali per kunjungan, bukan pada setiap
// penggambaran ulang.
func TestBukaLayarBerulangMenghabiskanJatah(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(2))
	ctx := context.Background()

	_, err := service.Open(ctx, portalContoh, pemanggil)
	require.NoError(t, err)

	_, err = service.Open(ctx, portalContoh, pemanggil)
	require.NoError(t, err)

	_, err = service.Open(ctx, portalContoh, pemanggil)
	require.ErrorIs(t, err, riwayatklaim.ErrQuotaExhausted)
}

// Pengguna yang belum terdaftar tidak dapat membuka layar sama sekali.
func TestBukaLayarDitolakBilaBelumTerdaftar(t *testing.T) {
	service, gate := rakit(t) // tanpa satu pun baris proteksi

	_, err := service.Open(context.Background(), portalContoh, pemanggil)
	require.ErrorIs(t, err, riwayatklaim.ErrNotRegistered)

	require.Empty(t, gate.Usage(),
		"penolakan tidak boleh ikut tercatat sebagai pemakaian jatah")
}

// Identitas yang tidak terbaca menghentikan permintaan sebelum penyimpanan disentuh.
func TestPemanggilTanpaLoginDitolak(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(5))

	_, err := service.Open(context.Background(), portalContoh, riwayatklaim.Caller{})
	require.ErrorIs(t, err, riwayatklaim.ErrCallerUnknown)
}

// Pencarian TIDAK mengurangi jatah, tetapi tetap dicatat.
//
// Keduanya perilaku yang berbeda dan keduanya disengaja — lihat
// riwayatklaim.Usage.ConsumesQuota.
func TestPencarianTidakMengurangiJatahTetapiDicatat(t *testing.T) {
	service, gate := rakit(t, penggunaTerdaftar(5))
	ctx := context.Background()

	_, err := service.Open(ctx, portalContoh, pemanggil)
	require.NoError(t, err)

	found, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{
			Type: riwayatklaim.TypeInsuredName,
			Text: "CONTOH",
		},
		riwayatklaim.Pagination{Page: 1, Size: 20},
	)
	require.NoError(t, err)
	require.NotEmpty(t, found.Page.Claims)

	require.Equal(t, 1, found.Access.QuotaUsed, "hanya pembukaan layar yang terhitung")
	require.Equal(t, 4, found.Access.QuotaRemaining)

	jejak := gate.Usage()
	require.Len(t, jejak, 2)
	require.False(t, jejak[1].ConsumesQuota)
	require.Equal(t, riwayatklaim.TypeInsuredName, jejak[1].SearchTypeCode)
	require.Equal(t, "CONTOH", jejak[1].SearchValue,
		"nilai yang dicari ikut tercatat — itulah gunanya jejak ini (D-59)")
}

// Pencarian tetap menempuh gerbang meski jatahnya tidak dipakai.
//
// Tanpa ini, pencarian dapat dijalankan langsung ke endpoint-nya tanpa pernah membuka
// layar — dan gerbang yang hanya dijaga antarmuka bukan gerbang (`D-59`).
func TestPencarianTetapMenempuhGerbang(t *testing.T) {
	service, _ := rakit(t) // belum terdaftar

	_, err := service.Search(context.Background(), portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeInsuredName, Text: "CONTOH"},
		riwayatklaim.Pagination{},
	)
	require.ErrorIs(t, err, riwayatklaim.ErrNotRegistered)
}

// Isian yang cacat ditolak SEBELUM gerbang disentuh, sehingga salah ketik tidak
// terhitung sebagai pemakaian layar.
func TestIsianCacatDitolakSebelumGerbang(t *testing.T) {
	service, gate := rakit(t, penggunaTerdaftar(5))

	_, err := service.Search(context.Background(), portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeInsuredName, Text: ""},
		riwayatklaim.Pagination{},
	)

	var validation *riwayatklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Empty(t, gate.Usage(), "salah ketik bukan pemakaian")
}

// CACAT YANG DIREPLIKASI, diuji dari ujung ke ujung.
//
// Data contoh MEMUAT peserta bertanggal lahir 17 Juli 1990, sehingga pencariannya
// seharusnya menemukan satu baris. Ia tetap kosong — dan itulah yang membuktikan cacatnya
// direplikasi, bukan data contohnya yang kebetulan tidak ada.
func TestPencarianTanggalLahirSelaluKosong(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(5))

	found, err := service.Search(context.Background(), portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{
			Type:      riwayatklaim.TypeBirthDate,
			BirthDate: waktu(1990, time.July, 17),
		},
		riwayatklaim.Pagination{},
	)
	require.NoError(t, err, "tidak menghasilkan galat — persis sistem lama")
	require.Empty(t, found.Page.Claims,
		"peserta dengan tanggal lahir itu ADA di data contoh, tetapi kueri menerima "+
			"tanggal kosong karena isian yang dipakai bukan isian yang tampak")
	require.Equal(t, 0, found.Page.Total)
}

// CACAT YANG DIREPLIKASI — Posisi Klaim kosong pada pencarian Nama Objek.
func TestPencarianNamaObjekTanpaPosisiKlaim(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(5))
	ctx := context.Background()

	lewatNama, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeInsuredName, Text: "SUMBER CONTOH"},
		riwayatklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Len(t, lewatNama.Page.Claims, 1)
	require.NotEmpty(t, lewatNama.Page.Claims[0].ClaimPosition,
		"tipe lain membawa Posisi Klaim")

	lewatObjek, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeInsuredItemName, Text: "Gudang Contoh"},
		riwayatklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Len(t, lewatObjek.Page.Claims, 1)
	require.Empty(t, lewatObjek.Page.Claims[0].ClaimPosition,
		"kueri Nama Objek adalah satu-satunya yang tidak membawa subquery V_STS_CLAIM")
}

// Pencarian No Klaim menemukan KEDUA format nomor yang hidup berdampingan.
//
// Inilah yang dituntut `D-22`: kueri lama merangkai awalan kelas Pega ke kunci
// pencariannya, dan klaim terbitan sistem baru tidak pernah menulis awalan itu lagi.
func TestPencarianNoKlaimMenemukanKeduaFormat(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(9))
	ctx := context.Background()

	warisan, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeClaimNumber, Text: "PNC-9001"},
		riwayatklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Len(t, warisan.Page.Claims, 1)

	baru, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeClaimNumber, Text: "PNCN.26.0001"},
		riwayatklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Len(t, baru.Page.Claims, 1)
}

// Tipe ID Balai Lelang membawa dua kolom tambahannya.
func TestPencarianBalaiLelangMembawaKolomTambahan(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(5))

	found, err := service.Search(context.Background(), portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeAuctionHouseID, Text: "BL-CONTOH-77"},
		riwayatklaim.Pagination{},
	)
	require.NoError(t, err)
	require.Len(t, found.Page.Claims, 1)
	require.Equal(t, "AKS-CONTOH-0003", found.Page.Claims[0].AcceptanceNumber)
	require.Equal(t, "BL-CONTOH-77", found.Page.Claims[0].AuctionHouseID)
}

// Paginasi membagi hasil dan melaporkan jumlah seluruhnya.
func TestPaginasiMembagiHasil(t *testing.T) {
	service, _ := rakit(t, penggunaTerdaftar(9))
	ctx := context.Background()

	halamanSatu, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeInsuredName, Text: "CONTOH"},
		riwayatklaim.Pagination{Page: 1, Size: 2},
	)
	require.NoError(t, err)
	require.Len(t, halamanSatu.Page.Claims, 2)
	require.Greater(t, halamanSatu.Page.Total, 2)

	halamanJauh, err := service.Search(ctx, portalContoh, pemanggil,
		riwayatklaim.CriteriaInput{Type: riwayatklaim.TypeInsuredName, Text: "CONTOH"},
		riwayatklaim.Pagination{Page: 99, Size: 2},
	)
	require.NoError(t, err, "halaman di luar hasil bukan galat")
	require.Empty(t, halamanJauh.Page.Claims)
	require.Equal(t, halamanSatu.Page.Total, halamanJauh.Page.Total)
}

func waktu(year int, month time.Month, day int) *time.Time {
	moment := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &moment
}
