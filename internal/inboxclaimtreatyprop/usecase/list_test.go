package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatyprop"
	"claim-pnc/internal/inboxclaimtreatyprop/repo/memory"
	"claim-pnc/internal/inboxclaimtreatyprop/usecase"
)

const portalUtama = "ASM"

// newService merakit layanan di atas penyimpanan memori berisi contoh bawaan.
func newService(t *testing.T) *usecase.Service {
	t.Helper()

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxclaimtreatyprop.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak tersedia")
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

// list memanggil layanan dengan paginasi yang cukup besar untuk memuat seluruh contoh.
func list(
	t *testing.T,
	service *usecase.Service,
	caller string,
	input inboxclaimtreatyprop.QueryInput,
) usecase.Listed {
	t.Helper()

	listed, err := service.List(
		context.Background(),
		portalUtama,
		inboxclaimtreatyprop.Caller{Login: caller},
		input,
		inboxclaimtreatyprop.Pagination{Page: 1, Size: 100},
	)
	require.NoError(t, err)
	return listed
}

// claimIDs mengambil nomor klaim setiap baris, supaya perbandingan di bawah terbaca.
func claimIDs(page inboxclaimtreatyprop.Page) []string {
	result := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		result = append(result, item.ClaimID)
	}
	return result
}

func TestLayananMenolakTanpaRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestKeteranganLayarTidakMenyentuhPenyimpanan(t *testing.T) {
	// Daftar tab dan kolomnya sama di seluruh entitas: ia bentuk layar, bukan data
	// entitas. Kalau ia menyentuh penyimpanan, layar akan gagal dibuka saat basis data
	// satu portal sedang mati — padahal tidak ada satu pun isinya yang berasal dari sana.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxclaimtreatyprop.Repo, error) {
			t.Fatal("Metadata tidak boleh memilih repo")
			return nil, nil
		},
	})
	require.NoError(t, err)

	meta := service.Metadata()
	require.Len(t, meta.Tabs, 3)
	require.Equal(t, inboxclaimtreatyprop.DefaultTab, meta.DefaultTab)
	require.NotEmpty(t, meta.PlannedDifferences)
}

func TestAntreanAdminMemuatPekerjaanSeluruhPetugas(t *testing.T) {
	// Report Definition `InboxKlaimPropAdmin` TIDAK menyaring menurut petugas, sehingga
	// tabnya memuat seluruh penugasan klaim treaty di entitas itu — termasuk milik
	// ADMINTREATY2.
	listed := list(t, newService(t), "ADMINTREATY1",
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabWorkList})

	// Terbaru lebih dulu: CLMP-1002 (20 Sep), CLMP-1003 (19 Sep), CLMP-1001 (18 Sep).
	require.Equal(t, []string{"CLMP-1002", "CLMP-1003", "CLMP-1001"}, claimIDs(listed.Page))
	require.Equal(t, 3, listed.Page.Total)
}

func TestLihatSemuaTidakMengubahApaPun(t *testing.T) {
	// Centangnya tidak lagi punya penyaring untuk dilepas. Permintaan yang masih
	// mengirimnya — klien lama — DILAYANI dengan hasil yang sama, bukan ditolak.
	service := newService(t)

	tanpa := list(t, service, "ADMINTREATY1",
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabWorkList})

	dengan := list(t, service, "ADMINTREATY1", inboxclaimtreatyprop.QueryInput{
		Tab:    inboxclaimtreatyprop.TabWorkList,
		SeeAll: true,
	})

	require.Equal(t, claimIDs(tanpa.Page), claimIDs(dengan.Page))
	require.Equal(t, tanpa.Page.Total, dengan.Page.Total)
	require.False(t, dengan.Query.SeeAll)
}

func TestAntreanTeknikTidakBergantungSiapaYangMasuk(t *testing.T) {
	// Antrean bersama menampilkan isi yang sama bagi siapa pun. Kalau ia ikut disaring
	// menurut pemanggil, antrean yang belum diambil siapa pun akan tampak kosong bagi
	// semua orang.
	service := newService(t)

	satu := list(t, service, "ADMINTREATY1",
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabTechnical})
	dua := list(t, service, "SIAPAPUN",
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabTechnical})

	require.Equal(t, []string{"CLMP-3001", "CLMP-2001", "CLMP-2002"}, claimIDs(satu.Page))
	require.Equal(t, claimIDs(satu.Page), claimIDs(dua.Page))
}

func TestAntreanTeknikMemuatSeluruhAntreanBersama(t *testing.T) {
	// Baris contoh `CLMP-3001` berada di workbasket tetapi milik antrean LAIN, bukan
	// `TreatyinPNCTeknik`. Ia harus MUNCUL: Report Definition `InboxKlaimPropTeknik` tidak
	// menyaring menurut nama antrean.
	//
	// Perannya berbalik dari sebelumnya, dan itu disengaja — lihat catatan di sample.go.
	listed := list(t, newService(t), "ADMINTREATY1",
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabTechnical})

	require.Contains(t, claimIDs(listed.Page), "CLMP-3001")
}

func TestPekerjaanBukanTreatyTersaringDiSeluruhTab(t *testing.T) {
	// Pembatas `w.PXOBJCLASS` yang memisahkan objek kerja klaim treaty dari objek kerja
	// lain di tabel penugasan yang sama. Baris contoh `PNC-9001` ada di worklist milik
	// ADMINTREATY1 dan tetap tidak boleh muncul.
	service := newService(t)

	for _, input := range []inboxclaimtreatyprop.QueryInput{
		{Tab: inboxclaimtreatyprop.TabWorkList},
		{Tab: inboxclaimtreatyprop.TabWorkList, SeeAll: true},
		{Tab: inboxclaimtreatyprop.TabTechnical},
	} {
		listed := list(t, service, "ADMINTREATY1", input)
		require.NotContains(t, claimIDs(listed.Page), "PNC-9001")
	}
}

func TestKolomSubjectivitySelaluKosong(t *testing.T) {
	// Report Definition MEMILIH `WorkPage.ClaimData.IsSubjectivity`, tetapi nama kolom
	// tereksposnya tidak dapat ditemukan di export — dan menebaknya menggagalkan seluruh
	// kueri dengan ORA-00904, bukan satu kolom.
	//
	// Kolomnya tetap digambar karena Pega produksi punya; isinya yang belum ada. Uji ini
	// menjaga agar penyimpanan memori tidak diam-diam mengisinya, yang akan membuat uji
	// modul menyatakan hal yang tidak benar tentang Oracle.
	service := newService(t)

	for _, input := range []inboxclaimtreatyprop.QueryInput{
		{Tab: inboxclaimtreatyprop.TabWorkList},
		{Tab: inboxclaimtreatyprop.TabTechnical},
	} {
		listed := list(t, service, "ADMINTREATY1", input)
		require.NotEmpty(t, listed.Page.Items)
		for _, item := range listed.Page.Items {
			require.Emptyf(t, item.Subjectivity,
				"tab %s mengisi Subjectivity; kuerinya mengirim NULL", input.Tab)
		}
	}
}

func TestTanggalKejadianTerisiDiSeluruhTab(t *testing.T) {
	// Inilah perbaikan P-5: di sistem lama kolom ini SELALU kosong pada antrean teknik.
	service := newService(t)

	for _, tab := range []string{
		inboxclaimtreatyprop.TabWorkList,
		inboxclaimtreatyprop.TabTechnical,
	} {
		listed := list(t, service, "ADMINTREATY1", inboxclaimtreatyprop.QueryInput{Tab: tab})
		require.NotEmpty(t, listed.Page.Items)
		for _, item := range listed.Page.Items {
			require.NotEmptyf(t, item.LossDate,
				"tab %s: Tanggal Kejadian kosong pada %s", tab, item.ClaimID)
		}
	}
}

func TestTabTerhalangDitolakSebelumMenyentuhPenyimpanan(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxclaimtreatyprop.Repo, error) {
			t.Fatal("tab terhalang tidak boleh sampai ke penyimpanan")
			return nil, nil
		},
	})
	require.NoError(t, err)

	_, err = service.List(
		context.Background(),
		portalUtama,
		inboxclaimtreatyprop.Caller{Login: "ADMINTREATY1"},
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabCommittee},
		inboxclaimtreatyprop.Pagination{},
	)

	var validation *inboxclaimtreatyprop.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestPortalYangTidakSiapMenghasilkanGalat(t *testing.T) {
	// Portal yang tidak dikenal WAJIB gagal, tidak pernah dialihkan ke portal utama
	// sebagai cadangan — jatuh ke koneksi bawaan berarti menampilkan antrean satu badan
	// hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
	_, err := newService(t).List(
		context.Background(),
		"ENTITAS-LAIN",
		inboxclaimtreatyprop.Caller{Login: "ADMINTREATY1"},
		inboxclaimtreatyprop.QueryInput{},
		inboxclaimtreatyprop.Pagination{},
	)
	require.Error(t, err)
}

func TestPaginasiMemotongHalamanDanMenjagaTotal(t *testing.T) {
	listed, err := newService(t).List(
		context.Background(),
		portalUtama,
		inboxclaimtreatyprop.Caller{Login: "ADMINTREATY1"},
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabWorkList, SeeAll: true},
		inboxclaimtreatyprop.Pagination{Page: 2, Size: 2},
	)
	require.NoError(t, err)

	require.Len(t, listed.Page.Items, 1)
	require.Equal(t, 3, listed.Page.Total, "total adalah seluruh baris, bukan isi halaman")
	require.Equal(t, 2, listed.Page.TotalPages())
}

func TestPermintaanYangDipakaiDikembalikanApaAdanya(t *testing.T) {
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim.
	listed := list(t, newService(t), "ADMINTREATY1", inboxclaimtreatyprop.QueryInput{
		Tab:    inboxclaimtreatyprop.TabTechnical,
		SeeAll: true,
	})

	require.Equal(t, inboxclaimtreatyprop.TabTechnical, listed.Query.Tab.Code)
	require.False(t, listed.Query.SeeAll,
		"tab antrean teknik tidak mengenal See All Claim, sehingga centangnya dimatikan")
}
