package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/casestudyclaim/repo/memory"
	"claim-pnc/internal/casestudyclaim/usecase"
	"claim-pnc/internal/platform/logging"
)

const portalUtama = "ASM"

// newService membentuk layanan di atas penyimpanan contoh satu portal.
//
// Portal lain DITOLAK, sama seperti di produksi: menjalankan tanpa Oracle tidak boleh
// mengubah aturan pemisahan entitas, karena justru di lingkungan itulah pelanggarannya
// paling mudah lolos (`R-20`).
func newService(t *testing.T) (*usecase.Service, *memory.Store) {
	t.Helper()

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (casestudyclaim.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak tersedia")
			}
			return store, nil
		},
		Logger: logging.New(0),
	})
	require.NoError(t, err)
	return service, store
}

func TestServiceMenolakDibentukTanpaPemilihRepo(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Logger: logging.New(0)})
	require.Error(t, err)
}

// Logger WAJIB, dan penolakannya disengaja.
//
// `POOLDATA.T_CLAIM_PNC` tidak punya kolom pelaku maupun waktu untuk
// `REMARKRECOMENDATION`. Selama `S-5` belum ada, log adalah satu-satunya tempat perubahan
// itu meninggalkan jejak — dan layanan yang dibentuk tanpa logger akan menulis tanpa jejak
// sama sekali.
func TestServiceMenolakDibentukTanpaLogger(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (casestudyclaim.Repo, error) { return nil, nil },
	})
	require.Error(t, err)
}

func TestListMembacaHalamanPortalYangDipilih(t *testing.T) {
	service, _ := newService(t)

	page, err := service.List(context.Background(), usecase.Query{
		PortalAlias: portalUtama,
		FromYear:    "2024",
		ToYear:      "2024",
	})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
}

// Portal yang tidak tersedia menghasilkan GALAT, tidak pernah dialihkan ke portal utama.
//
// Jatuh ke koneksi lain berarti menampilkan klaim di atas Rp 5 miliar milik satu badan
// hukum di layar badan hukum lain, tanpa satu pun galat (`R-20`).
func TestListMenolakPortalYangTidakTersedia(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), usecase.Query{
		PortalAlias: "SIMASNET",
		FromYear:    "2024",
		ToYear:      "2024",
	})
	require.Error(t, err)
}

// Periode diperiksa SEBELUM penyimpanan dipilih.
//
// Urutan itu bukan kerapian: bila periode kosong dilayani lebih dulu memilih penyimpanan,
// galat portal akan menutupi galat periode — dan pengguna diberi tahu sebab yang salah.
func TestListMemeriksaPeriodeSebelumMemilihPenyimpanan(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), usecase.Query{
		PortalAlias: "PORTAL-YANG-TIDAK-ADA",
	})
	require.ErrorIs(t, err, casestudyclaim.ErrPeriodRequired)
}

// Galat periode dikembalikan APA ADANYA, tidak dibungkus.
//
// Transport mengenalinya dengan errors.Is dan menjawabnya 422. Membungkusnya dengan fmt
// tetap dapat dikenali errors.Is, tetapi uji ini menjaga tipenya tidak berganti menjadi
// galat teknis yang dijawab 500.
func TestGalatPeriodeDapatDikenaliTransport(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), usecase.Query{
		PortalAlias: portalUtama,
		FromYear:    "2026",
		ToYear:      "2024",
	})
	require.ErrorIs(t, err, casestudyclaim.ErrPeriodReversed)
}

func TestSimpanCatatanMengubahBarisnya(t *testing.T) {
	service, store := newService(t)

	err := service.SaveRemark(context.Background(), portalUtama,
		usecase.Caller{Login: "budi"}, "STD-0001", "sudah ditelaah")
	require.NoError(t, err)

	page, err := store.List(context.Background(),
		casestudyclaim.Filter{FromYear: "2024", ToYear: "2024"})
	require.NoError(t, err)
	require.Equal(t, "sudah ditelaah", page.Rows[0].Remark)
}

// Klaim yang tidak ada dijawab ErrClaimNotFound, bukan galat teknis.
//
// Transport memetakannya ke 404 beserta pesan "muat ulang daftarnya" — bukan 500 beserta
// pesan yang menyuruh pengguna menghubungi tim teknis.
func TestSimpanCatatanPadaKlaimYangTidakAda(t *testing.T) {
	service, _ := newService(t)

	err := service.SaveRemark(context.Background(), portalUtama,
		usecase.Caller{Login: "budi"}, "STD-9999", "apa pun")
	require.ErrorIs(t, err, casestudyclaim.ErrClaimNotFound)
}

// Isian diperiksa SEBELUM penyimpanan disentuh.
//
// Catatan yang terlalu panjang ditolak di sini, bukan dijawab ORA-12899 yang sampai ke
// pengguna sebagai "terjadi kesalahan sistem" setelah ia mengetik satu halaman penuh.
func TestSimpanCatatanTerlaluPanjangDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	service, store := newService(t)

	long := make([]rune, casestudyclaim.MaxRemarkLength+1)
	for i := range long {
		long[i] = 'a'
	}

	err := service.SaveRemark(context.Background(), portalUtama,
		usecase.Caller{Login: "budi"}, "STD-0001", string(long))
	require.ErrorIs(t, err, casestudyclaim.ErrRemarkTooLong)

	page, listErr := store.List(context.Background(),
		casestudyclaim.Filter{FromYear: "2024", ToYear: "2024"})
	require.NoError(t, listErr)
	require.Equal(t, "", page.Rows[0].Remark, "penyimpanan tidak boleh tersentuh")
}

// Menyimpan ke portal yang tidak tersedia DITOLAK.
//
// Taruhannya di jalur tulis lebih besar daripada jalur baca: catatan telaah akan masuk ke
// tabel klaim badan hukum lain, dan tidak ada apa pun yang akan menandainya.
func TestSimpanCatatanMenolakPortalYangTidakTersedia(t *testing.T) {
	service, _ := newService(t)

	err := service.SaveRemark(context.Background(), "SIMASNET",
		usecase.Caller{Login: "budi"}, "STD-0001", "apa pun")
	require.Error(t, err)
	require.NotErrorIs(t, err, casestudyclaim.ErrClaimNotFound)
}

// Menyimpan dua kali menghasilkan keadaan yang SAMA.
//
// Itulah yang membuat jalur ini tidak membutuhkan kunci idempotensi: ia menimpa satu kolom
// dengan nilai yang sama, bukan menambah baris.
func TestSimpanCatatanBersifatIdempoten(t *testing.T) {
	service, store := newService(t)

	for i := 0; i < 2; i++ {
		require.NoError(t, service.SaveRemark(context.Background(), portalUtama,
			usecase.Caller{Login: "budi"}, "STD-0001", "catatan"))
	}

	page, err := store.List(context.Background(),
		casestudyclaim.Filter{FromYear: "2024", ToYear: "2024"})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total, "tidak ada baris yang bertambah")
	require.Equal(t, "catatan", page.Rows[0].Remark)
}
