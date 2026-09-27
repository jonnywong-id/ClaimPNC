package masterpasalai_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasalai"
)

// TestPageSizeFollowsActivityNotSection menjaga angka yang paling mudah salah di modul ini.
//
// Section memuat `pyPageSize = 30`, activity memuat `PageSize := 25`, dan yang BERLAKU adalah
// yang kedua — grid-nya ber-`pyPageMode = None` sehingga tidak memaginasi apa pun sendiri.
//
// Uji ini ada supaya angka 30 tidak pernah masuk kembali dari membaca section saja.
func TestPageSizeFollowsActivityNotSection(t *testing.T) {
	require.Equal(t, 25, masterpasalai.PageSize,
		"PageSize mengikuti Activity/GetListPasalAI_act.xml:874, bukan pyPageSize pada section")
}

// TestFilterCleanTrimsKeyword membuktikan kata kunci dipangkas sebelum dipakai mencari.
func TestFilterCleanTrimsKeyword(t *testing.T) {
	clean := masterpasalai.Filter{Keyword: "   kebakaran  "}.Clean()
	require.Equal(t, "kebakaran", clean.Keyword)
}

// TestFilterCleanNormalisesPage membuktikan halaman tidak masuk akal dinaikkan, bukan ditolak.
//
// Nomor halaman datang dari tautan paginasi; tautan basi bukan kesalahan pengguna.
func TestFilterCleanNormalisesPage(t *testing.T) {
	for _, given := range []int{0, -1, -99} {
		clean := masterpasalai.Filter{Page: given}.Clean()
		require.Equalf(t, masterpasalai.FirstPage, clean.Page,
			"halaman %d seharusnya dinaikkan ke halaman pertama", given)
	}

	require.Equal(t, 4, masterpasalai.Filter{Page: 4}.Clean().Page)
}

// TestFilterOffset membuktikan jendela halaman sama dengan `FirstRow` pada activity.
//
//	Pagination.FirstRow := ((.CurrentIndex-1) * .PageSize) + 1
//
// `OFFSET` menghitung dari nol sedangkan `FirstRow` dari satu, sehingga Offset() harus
// menghasilkan `FirstRow - 1`.
func TestFilterOffset(t *testing.T) {
	for _, one := range []struct {
		page   int
		offset int
	}{
		{page: 1, offset: 0},
		{page: 2, offset: 25},
		{page: 3, offset: 50},
		{page: 10, offset: 225},
	} {
		got := masterpasalai.Filter{Page: one.page}.Clean().Offset()
		require.Equalf(t, one.offset, got, "halaman %d", one.page)

		// Dinyatakan ulang sebagai rumus aslinya, supaya uji ini tetap menjaga hal yang
		// sama bila PageSize kelak berubah.
		firstRow := ((one.page - 1) * masterpasalai.PageSize) + 1
		require.Equal(t, firstRow-1, got)
	}
}

// TestPageCount membuktikan cacah halaman dihitung dari TOTAL, bukan dari isi halaman.
func TestPageCount(t *testing.T) {
	for _, one := range []struct {
		total int
		count int
	}{
		// Nol baris tetap satu halaman: layar selalu menggambar paginator, dan
		// "halaman 1 dari 0" tidak dapat dibaca siapa pun.
		{total: 0, count: 1},
		{total: 1, count: 1},
		{total: 25, count: 1},
		{total: 26, count: 2},
		{total: 50, count: 2},
		{total: 51, count: 3},
	} {
		got := masterpasalai.Page{Total: one.total}.PageCount()
		require.Equalf(t, one.count, got, "total %d baris", one.total)
	}
}

// TestClauseIsEmpty membuktikan baris kosong dikenali hanya bila KETIGA isiannya kosong.
func TestClauseIsEmpty(t *testing.T) {
	require.True(t, masterpasalai.Clause{}.IsEmpty())
	require.True(t, masterpasalai.Clause{Number: "  ", Paragraph: "\t", Event: " "}.IsEmpty())

	require.False(t, masterpasalai.Clause{Number: "1"}.IsEmpty())
	require.False(t, masterpasalai.Clause{Paragraph: "2"}.IsEmpty())
	require.False(t, masterpasalai.Clause{Event: "kebakaran"}.IsEmpty())
}
