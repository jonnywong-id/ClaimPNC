package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/repo/memory"
	"claim-pnc/internal/inboxservicecenter/usecase"
)

const portalUtama = "asm"

func newService(t *testing.T, store inboxservicecenter.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxservicecenter.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

func list(
	t *testing.T,
	service *usecase.Service,
	tab, keyword string,
	page inboxservicecenter.Pagination,
) usecase.Listed {
	t.Helper()

	listed, err := service.List(
		context.Background(),
		portalUtama,
		inboxservicecenter.Caller{Login: memory.SampleOwner},
		inboxservicecenter.QueryInput{Tab: tab, Keyword: keyword},
		page,
	)
	require.NoError(t, err)
	return listed
}

func ids(page inboxservicecenter.Page) []string {
	result := make([]string, 0, len(page.Items))
	for _, claim := range page.Items {
		result = append(result, claim.ID)
	}
	return result
}

// ---------------------------------------------------------------------------
// Penyaring tab
// ---------------------------------------------------------------------------

// TestSetiapTabMembawaBarisnyaSendiri membuktikan ketiga bentuk penyaring `STS_APPROVAL`
// bekerja di ujung ke ujung, bukan hanya benar sebagai nilai.
func TestSetiapTabMembawaBarisnyaSendiri(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	cases := []struct {
		tab  string
		want []string
	}{
		// Dua baris milik PIC yang login; yang ketiga milik PIC lain dan tidak boleh ikut.
		{inboxservicecenter.TabRegistration, []string{"SC-000101", "SC-000102"}},
		{inboxservicecenter.TabWaitingApproval, []string{"SC-000201"}},
		{inboxservicecenter.TabApproved, []string{"SC-000301"}},
		// TLO (kode 2) dan REJECT (kode 3) berkumpul di satu tab.
		{inboxservicecenter.TabRejected, []string{"SC-000401", "SC-000402"}},
	}

	for _, c := range cases {
		t.Run(c.tab, func(t *testing.T) {
			listed := list(t, service, c.tab, "", inboxservicecenter.Pagination{})

			require.Equal(t, c.want, ids(listed.Page))
			require.Equal(t, len(c.want), listed.Page.Total)
		})
	}
}

// TestBarisMilikPICLainTidakPernahTampil adalah uji kebocoran, bukan uji penyaring.
//
// Bila penyaring PIC lupa dipasang, SC-000103 akan muncul di tab Registrasi SC — dan
// barisnya memuat nama nasabah beserta IMEI perangkat milik petugas lain.
func TestBarisMilikPICLainTidakPernahTampil(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	for _, tab := range inboxservicecenter.Tabs() {
		listed := list(t, service, tab.Code, "", inboxservicecenter.Pagination{})
		require.NotContainsf(t, ids(listed.Page), "SC-000103",
			"baris milik PIC lain bocor di tab %s", tab.Name)
	}
}

// ---------------------------------------------------------------------------
// Urutan
// ---------------------------------------------------------------------------

// TestUrutanTerbaruDiAtasDanTanpaTanggalDiBawah mengunci `ORDER BY INPUTDATE DESC, ID`
// beserta penempatan NULL-nya — Oracle menaruh NULL terakhir pada DESC secara bawaan, dan
// adapter memori harus sama supaya uji di sini menyatakan sesuatu tentang yang di Oracle.
func TestUrutanTerbaruDiAtasDanTanpaTanggalDiBawah(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed := list(t, service, inboxservicecenter.TabRegistration, "", inboxservicecenter.Pagination{})

	require.Equal(t, []string{
		"SC-000101", // 22 September
		"SC-000102", // tanpa tanggal input
	}, ids(listed.Page))
}

// ---------------------------------------------------------------------------
// Pencarian — termasuk cacat yang sengaja direplikasi
// ---------------------------------------------------------------------------

// TestPencarianMenemukanLewatNoPolisNasabahDanIMEI — ketiga isian inilah irisan kedua
// kelompok penyaring, sehingga hanya ketiganya yang benar-benar bekerja.
func TestPencarianMenemukanLewatNoPolisNasabahDanIMEI(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	cases := map[string]string{
		"no polis": "90-001-2026-00000101",
		"nasabah":  "Nasabah Satu",
		"imei":     "IMEI-CONTOH-001",
	}

	for nama, keyword := range cases {
		t.Run(nama, func(t *testing.T) {
			listed := list(t, service, inboxservicecenter.TabRegistration, keyword,
				inboxservicecenter.Pagination{})

			require.Equal(t, []string{"SC-000101"}, ids(listed.Page))
		})
	}
}

// TestPencarianDenganIDSajaTidakMenghasilkanBaris mengunci CACAT sistem lama, bukan
// perilaku yang diinginkan.
//
// `Activity/DataServiceCenter-Act.xml` memasang DUA potongan pencarian yang digabung dengan
// AND: langkah 17 menyebut ID, langkah 18 menyebut CLAIMNO, dan keduanya hanya berbagi
// NOPOLIS/QQNAME/IMEI. Akibatnya kata kunci yang hanya cocok di ID gugur pada kelompok kedua.
//
// Ia DIREPLIKASI sesuai `P-5` dan dinyatakan terbuka lewat Limitations. Bila kelak Work Owner
// memutuskan memperbaikinya, uji inilah yang harus diubah lebih dulu — sehingga perbaikannya
// menjadi keputusan yang tercatat, bukan perubahan yang tidak sengaja.
func TestPencarianDenganIDSajaTidakMenghasilkanBaris(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed := list(t, service, inboxservicecenter.TabRegistration, "SC-000101",
		inboxservicecenter.Pagination{})

	require.Empty(t, listed.Page.Items,
		"kata kunci yang hanya cocok di ID gugur pada kelompok penyaring kedua")
	require.Zero(t, listed.Page.Total)
}

// TestPencarianMematikanPaginasi — klausa `rn` adalah satu-satunya paginasi kueri lama, dan
// ia dikosongkan begitu kotak cari terisi.
func TestPencarianMematikanPaginasi(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	tanpaCari := list(t, service, inboxservicecenter.TabRegistration, "",
		inboxservicecenter.Pagination{Page: 1, Size: 1})
	require.True(t, tanpaCari.Page.Paginated)
	require.Len(t, tanpaCari.Page.Items, 1, "dipotong menurut ukuran halaman")
	require.Equal(t, 2, tanpaCari.Page.Total)
	require.Equal(t, 2, tanpaCari.Page.TotalPages())

	denganCari := list(t, service, inboxservicecenter.TabRegistration, "Contoh Nasabah",
		inboxservicecenter.Pagination{Page: 1, Size: 1})
	require.False(t, denganCari.Page.Paginated)
	require.Len(t, denganCari.Page.Items, 2,
		"ukuran halaman diabaikan saat mencari — seluruh baris yang cocok ikut")
	require.Equal(t, 1, denganCari.Page.TotalPages())
}

// ---------------------------------------------------------------------------
// Paginasi
// ---------------------------------------------------------------------------

func TestHalamanDiluarJangkauanMenghasilkanDaftarKosongBukanGalat(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed := list(t, service, inboxservicecenter.TabRegistration, "",
		inboxservicecenter.Pagination{Page: 99, Size: 25})

	require.Empty(t, listed.Page.Items)
	require.NotNil(t, listed.Page.Items, "senarai kosong, bukan nil")
	require.Equal(t, 2, listed.Page.Total, "total tetap menyebut seluruh baris yang cocok")
}

// ---------------------------------------------------------------------------
// Penyaring yang dikembalikan
// ---------------------------------------------------------------------------

// TestPenyaringYangDipakaiDikembalikanKeLayar — kotak cari di layar harus memperlihatkan
// kata kunci yang BENAR-BENAR menyaring, bukan yang sempat diketik.
func TestPenyaringYangDipakaiDikembalikanKeLayar(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed := list(t, service, inboxservicecenter.TabApproved, "  90-001  ",
		inboxservicecenter.Pagination{})

	require.Equal(t, "90-001", listed.Query.Keyword)
	require.Equal(t, inboxservicecenter.TabApproved, listed.Query.Tab.Code)
}

// ---------------------------------------------------------------------------
// Portal
// ---------------------------------------------------------------------------

// TestPortalTidakDikenalMenghasilkanGalat — mengembalikan repo portal utama sebagai jalan
// pintas berarti menampilkan klaim satu badan hukum kepada petugas badan hukum lain tanpa
// satu pun pesan galat (`R-20`).
func TestPortalTidakDikenalMenghasilkanGalat(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(
		context.Background(),
		"portal-karangan",
		inboxservicecenter.Caller{Login: memory.SampleOwner},
		inboxservicecenter.QueryInput{},
		inboxservicecenter.Pagination{},
	)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Keterangan layar
// ---------------------------------------------------------------------------

func TestMetadataMenyebutEmpatTabDanKeterbatasannya(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	meta := service.Metadata()

	require.Len(t, meta.Tabs, 4)
	require.Equal(t, inboxservicecenter.DefaultTab, meta.DefaultTab)
	require.NotEmpty(t, meta.Limitations)
}

func TestServiceMenolakDibentukTanpaPemilihRepo(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}
