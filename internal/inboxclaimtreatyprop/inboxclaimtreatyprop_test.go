package inboxclaimtreatyprop_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

// caller adalah pemanggil yang sah, dipakai hampir seluruh uji di berkas ini.
func caller() inboxclaimtreatyprop.Caller {
	return inboxclaimtreatyprop.Caller{Login: "ADMINTREATY1"}
}

func TestTabBawaanAdalahPilihanYangSelaluAda(t *testing.T) {
	// Layar terbuka pada "Prop Treaty-in Admin", karena itulah satu-satunya pilihan yang
	// SELALU ada di dropdown Pega: `FilterWorkBasket_Act` menambahkannya tanpa syarat,
	// sedangkan pilihan Teknik hanya menyusul bila pemanggil anggota antreannya.
	//
	// Membuka layar pada pilihan yang bisa jadi tidak ada berarti layar kosong tanpa sebab
	// yang terbaca.
	tab, found := inboxclaimtreatyprop.FindTab(inboxclaimtreatyprop.DefaultTab)
	require.True(t, found)
	require.Equal(t, "Prop Treaty-in Admin", tab.Name)
	require.False(t, tab.Blocked)
}

func TestTidakAdaTabYangMenyaringPemanggil(t *testing.T) {
	// Kedua Report Definition yang memasok layar ini TIDAK punya satu pun filter selain
	// kondisi join — tidak ada `pxAssignedOperatorID = <pemanggil>`, dan keenam
	// parameternya dideklarasikan tetapi tidak pernah dirujuk.
	//
	// Uji ini menjaga agar penyaring itu tidak kembali diam-diam. Bila suatu saat ia memang
	// harus kembali, yang berubah bukan hanya penanda di sini melainkan juga kueri dan
	// penyimpanan memori — dan uji ini yang memaksa ketiganya berubah bersama.
	for _, tab := range inboxclaimtreatyprop.Tabs() {
		require.Falsef(t, tab.ScopedToCaller,
			"tab %s menyaring pemanggil; Report Definition-nya tidak", tab.Code)
		require.Falsef(t, tab.SupportsSeeAll,
			"tab %s menggambar \"See All Claim\" yang tidak melepas apa pun", tab.Code)
	}
}

func TestSetiapTabPunyaKodeDanNamaYangBerbeda(t *testing.T) {
	codes := map[string]bool{}
	names := map[string]bool{}

	for _, tab := range inboxclaimtreatyprop.Tabs() {
		require.NotEmpty(t, tab.Code)
		require.NotEmpty(t, tab.Name)
		require.NotEmpty(t, tab.Description)

		require.Falsef(t, codes[tab.Code], "kode tab ganda: %s", tab.Code)
		require.Falsef(t, names[tab.Name], "nama tab ganda: %s", tab.Name)
		codes[tab.Code] = true
		names[tab.Name] = true
	}
}

func TestHanyaTabAntreanTeknikYangPunyaKolomSubjectivity(t *testing.T) {
	// Hanya kueri antrean teknik yang membawa Subjectivity. Kolom yang digambar pada tab
	// lain akan selalu kosong, dan kolom yang selalu kosong membuat pengguna menduga
	// datanya hilang.
	for _, tab := range inboxclaimtreatyprop.Tabs() {
		has := false
		for _, column := range tab.Columns {
			if column.Key == inboxclaimtreatyprop.FieldSubjectivity {
				has = true
			}
		}
		require.Equalf(t, tab.Code == inboxclaimtreatyprop.TabTechnical, has,
			"tab %s (%s) salah dalam memuat kolom Subjectivity", tab.Code, tab.Name)
	}
}

func TestTabTerhalangTidakMenggambarGrid(t *testing.T) {
	// Tab terhalang digambar, tetapi TANPA kolom. Menyebut kolomnya akan membuat layar
	// menggambar tabel kosong — yang terbaca sebagai "tidak ada pekerjaan" padahal yang
	// benar adalah "belum dapat dibaca".
	blocked := 0
	for _, tab := range inboxclaimtreatyprop.Tabs() {
		if !tab.Blocked {
			require.NotEmptyf(t, tab.Columns, "tab %s tidak punya kolom", tab.Name)
			continue
		}
		blocked++
		require.Emptyf(t, tab.Columns, "tab terhalang %s tidak boleh punya kolom", tab.Name)
		require.NotEmptyf(t, tab.BlockedReason, "tab %s terhalang tanpa alasan", tab.Name)
		require.NotEmptyf(t, tab.BlockedOwner, "tab %s terhalang tanpa pemilik", tab.Name)
	}
	require.Equal(t, 1, blocked, "hanya tab Komite Treaty ASM yang terhalang")
}

func TestPermintaanTanpaIdentitasDitolak(t *testing.T) {
	// Tab pertama menyaring menurut login pemanggil. Tanpa identitas, ia tidak dapat
	// membedakan "antrean saya" dari "antrean semua orang".
	_, err := inboxclaimtreatyprop.NewQuery(
		inboxclaimtreatyprop.QueryInput{},
		inboxclaimtreatyprop.Caller{Login: "   "},
	)
	require.ErrorIs(t, err, inboxclaimtreatyprop.ErrCallerUnknown)
}

func TestTabKosongJatuhKeTabBawaan(t *testing.T) {
	query, err := inboxclaimtreatyprop.NewQuery(inboxclaimtreatyprop.QueryInput{}, caller())
	require.NoError(t, err)
	require.Equal(t, inboxclaimtreatyprop.DefaultTab, query.Tab.Code)
}

func TestTabTidakDikenalDitolakDenganPelanggaran(t *testing.T) {
	_, err := inboxclaimtreatyprop.NewQuery(
		inboxclaimtreatyprop.QueryInput{Tab: "99"}, caller())

	var validation *inboxclaimtreatyprop.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxclaimtreatyprop.FieldTab, validation.Violations[0].Field)
}

func TestTabTerhalangDitolakDenganAlasannya(t *testing.T) {
	// Alasannya sampai ke layar apa adanya. Kalau tab terhalang dibiarkan lolos sampai ke
	// penyimpanan, ia akan gagal sebagai galat internal 500 — jawaban yang tidak menyebut
	// sebabnya kepada siapa pun.
	tab, found := inboxclaimtreatyprop.FindTab(inboxclaimtreatyprop.TabCommittee)
	require.True(t, found)

	_, err := inboxclaimtreatyprop.NewQuery(
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabCommittee}, caller())

	var validation *inboxclaimtreatyprop.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, tab.BlockedReason, validation.Violations[0].Message)
}

func TestLihatSemuaDiabaikanPadaTabYangTidakMengenalnya(t *testing.T) {
	// Tanpa penetapan ini, dua permintaan yang hasilnya pasti sama akan tampak berbeda di
	// log dan di kunci cache layar.
	query, err := inboxclaimtreatyprop.NewQuery(
		inboxclaimtreatyprop.QueryInput{
			Tab:    inboxclaimtreatyprop.TabTechnical,
			SeeAll: true,
		}, caller())

	require.NoError(t, err)
	require.False(t, query.SeeAll)
}

func TestLihatSemuaDiabaikanKarenaTakAdaYangDilepas(t *testing.T) {
	// "See All Claim" dikirim layar lama dan mungkin masih dikirim klien yang belum
	// diperbarui. Ia DIABAIKAN, bukan ditolak: permintaannya sah, hanya tidak lagi
	// mengubah apa pun karena tab Admin sendiri sudah tidak menyaring pemanggil.
	//
	// Diabaikan diam-diam lebih baik daripada ditolak di sini — menolaknya akan membuat
	// klien lama gagal membuka layar yang sebenarnya dapat dilayani.
	for _, seeAll := range []bool{false, true} {
		query, err := inboxclaimtreatyprop.NewQuery(
			inboxclaimtreatyprop.QueryInput{
				Tab:    inboxclaimtreatyprop.TabWorkList,
				SeeAll: seeAll,
			}, caller())
		require.NoError(t, err)
		require.Falsef(t, query.SeeAll,
			"See All=%v diteruskan padahal tabnya tidak mendukungnya", seeAll)
		require.False(t, query.ScopedToCaller())
	}
}

func TestAntreanTeknikBukanAntreanMilikPemanggil(t *testing.T) {
	// Antrean bersama TIDAK pernah disaring menurut pemanggil, berapa pun keadaan
	// checkbox. Menyaringnya akan membuat antrean yang belum diambil siapa pun tampak
	// kosong bagi semua orang.
	query, err := inboxclaimtreatyprop.NewQuery(
		inboxclaimtreatyprop.QueryInput{Tab: inboxclaimtreatyprop.TabTechnical}, caller())

	require.NoError(t, err)
	require.False(t, query.ScopedToCaller())
}

func TestPaginasiDibetulkanKeRentangYangSah(t *testing.T) {
	// Nilai di luar rentang DIBETULKAN, tidak ditolak: keduanya datang dari parameter
	// query yang mudah salah ketik.
	for _, c := range []struct {
		name  string
		given inboxclaimtreatyprop.Pagination
		want  inboxclaimtreatyprop.Pagination
	}{
		{
			name:  "halaman nol menjadi satu",
			given: inboxclaimtreatyprop.Pagination{Page: 0, Size: 10},
			want:  inboxclaimtreatyprop.Pagination{Page: 1, Size: 10},
		},
		{
			name:  "halaman negatif menjadi satu",
			given: inboxclaimtreatyprop.Pagination{Page: -5, Size: 10},
			want:  inboxclaimtreatyprop.Pagination{Page: 1, Size: 10},
		},
		{
			name:  "ukuran kosong menjadi bawaan",
			given: inboxclaimtreatyprop.Pagination{Page: 2},
			want: inboxclaimtreatyprop.Pagination{
				Page: 2, Size: inboxclaimtreatyprop.DefaultPageSize,
			},
		},
		{
			name:  "ukuran berlebih dipotong ke batas",
			given: inboxclaimtreatyprop.Pagination{Page: 1, Size: 5000},
			want: inboxclaimtreatyprop.Pagination{
				Page: 1, Size: inboxclaimtreatyprop.MaxPageSize,
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, c.given.Normalize())
		})
	}
}

func TestOffsetTidakPernahNegatif(t *testing.T) {
	// Offset negatif di Oracle bukan galat melainkan halaman pertama, sehingga cacatnya
	// tidak pernah terlihat. Ia dicegah di sini.
	require.Equal(t, 0, inboxclaimtreatyprop.Pagination{Page: -3, Size: 10}.Offset())
	require.Equal(t, 20, inboxclaimtreatyprop.Pagination{Page: 3, Size: 10}.Offset())
}

func TestJumlahHalamanMinimalSatu(t *testing.T) {
	// Layar tidak boleh pernah menggambar "halaman 1 dari 0".
	kosong := inboxclaimtreatyprop.Page{
		Pagination: inboxclaimtreatyprop.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 1, kosong.TotalPages())
}

func TestJumlahHalamanMembulatKeAtas(t *testing.T) {
	page := inboxclaimtreatyprop.Page{
		Total:      26,
		Pagination: inboxclaimtreatyprop.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 2, page.TotalPages())
}

func TestSliceMemotongHalamanYangBenar(t *testing.T) {
	all := []inboxclaimtreatyprop.WorkItem{
		{ClaimID: "CLMP-1"}, {ClaimID: "CLMP-2"}, {ClaimID: "CLMP-3"},
		{ClaimID: "CLMP-4"}, {ClaimID: "CLMP-5"},
	}

	page := inboxclaimtreatyprop.Slice(all, inboxclaimtreatyprop.Pagination{Page: 2, Size: 2})
	require.Equal(t, 5, page.Total)
	require.Len(t, page.Items, 2)
	require.Equal(t, "CLMP-3", page.Items[0].ClaimID)
	require.Equal(t, "CLMP-4", page.Items[1].ClaimID)
}

func TestSliceMelewatiUjungMengembalikanSenaraiKosong(t *testing.T) {
	// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien.
	all := []inboxclaimtreatyprop.WorkItem{{ClaimID: "CLMP-1"}}

	page := inboxclaimtreatyprop.Slice(all, inboxclaimtreatyprop.Pagination{Page: 9, Size: 25})
	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)
	require.Equal(t, 1, page.Total)
}

func TestGalatTulisDikenaliSebagaiBelumTersedia(t *testing.T) {
	require.True(t, errors.Is(
		inboxclaimtreatyprop.ErrWriteNotAvailable,
		inboxclaimtreatyprop.ErrWriteNotAvailable,
	))
}

func TestSelisihTerencanaDinyatakanDiMuka(t *testing.T) {
	// Setiap selisih terhadap Pega WAJIB dapat dipetakan ke butir yang sudah diputuskan
	// (`D-54`). Daftar yang kosong berarti tidak ada yang dapat dipetakan — dan uji
	// kesetaraan gerbang 1 akan melaporkan seluruh selisihnya sebagai tidak terpetakan.
	require.NotEmpty(t, inboxclaimtreatyprop.PlannedDifferences)
	for _, line := range inboxclaimtreatyprop.PlannedDifferences {
		require.NotEmpty(t, line)
	}
}

func TestTabLabelComesFromTheDropdownNotTheGridCaption(t *testing.T) {
	// Keduanya teks yang BERBEDA dan tampil BERSAMAAN di layar yang sama: dropdown
	// bertuliskan "Prop Treaty-in Admin", grid di bawahnya berjudul "Work List Treatyin
	// Propotional".
	//
	// Uji ini ada karena keduanya SEMPAT tertukar di sini — judul kontainer dipakai sebagai
	// teks pilihan dropdown, sehingga pilihan yang dicari pengguna tidak ada dan judul
	// gridnya hilang. Kegagalannya tidak menghasilkan galat apa pun; ia hanya layar yang
	// tidak dikenali.
	//
	// Labelnya diambil dari `Data Transform/FilterWorkBasket_Act-DT.xml`, judul gridnya dari
	// `Section/InboxClaimTreaty_Section-Section.xml`.
	expected := map[string]struct{ name, gridTitle string }{
		inboxclaimtreatyprop.TabWorkList: {
			name:      "Prop Treaty-in Admin",
			gridTitle: "Work List Treatyin Propotional",
		},
		inboxclaimtreatyprop.TabTechnical: {
			name:      "Prop Treaty-in Teknik",
			gridTitle: "Work Teknik Treatyin",
		},
	}

	for _, tab := range inboxclaimtreatyprop.Tabs() {
		want, checked := expected[tab.Code]
		if !checked {
			continue
		}
		require.Equalf(t, want.name, tab.Name,
			"teks pilihan dropdown tab %s tidak sesuai FilterWorkBasket_Act", tab.Code)
		require.Equalf(t, want.gridTitle, tab.GridTitle,
			"judul grid tab %s tidak sesuai section", tab.Code)
		require.NotEqualf(t, tab.Name, tab.GridTitle,
			"tab %s memakai satu teks untuk dua hal yang berbeda", tab.Code)
	}
}

func TestOnlySelectableTabsHaveAGridTitle(t *testing.T) {
	// Tab terhalang tidak menggambar grid sama sekali, sehingga judul grid yang menganggur
	// di sana hanya menunggu dipakai layar yang kemudian menggambar tabel kosong.
	for _, tab := range inboxclaimtreatyprop.Tabs() {
		if tab.Blocked {
			require.Emptyf(t, tab.GridTitle,
				"tab %s terhalang tetapi punya judul grid", tab.Code)
			continue
		}
		require.NotEmptyf(t, tab.GridTitle, "tab %s tidak punya judul grid", tab.Code)
	}
}
