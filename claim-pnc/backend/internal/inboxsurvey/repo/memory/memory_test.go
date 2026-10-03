package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/repo/memory"
)

// identity menerjemahkan login contoh menjadi identitas surveyor.
func identity(t *testing.T, login string) inboxsurvey.SurveyorIdentity {
	t.Helper()

	result, err := memory.NewSampleStore().ResolveSurveyor(context.Background(), login)
	require.NoError(t, err)
	return result
}

// list menjalankan kueri satu tab dan mengembalikan halamannya.
func list(t *testing.T, login string, tab inboxsurvey.Tab) inboxsurvey.Page {
	t.Helper()

	page, err := memory.NewSampleStore().List(
		context.Background(),
		identity(t, login),
		inboxsurvey.Filter{Tab: tab, Limit: 100},
	)
	require.NoError(t, err)
	return page
}

// numbers mengambil nomor klaim tiap baris, supaya penegasan terbaca sebagai daftar.
func numbers(page inboxsurvey.Page) []string {
	result := make([]string, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		result = append(result, task.ClaimNumber)
	}
	return result
}

// TestPemanggilBukanSurveyorDitolak, dan BUKAN diberi daftar kosong.
//
// Keduanya terlihat sama di layar dan berarti hal yang sangat berbeda. Daftar kosong terbaca
// sebagai "tidak ada pekerjaan hari ini", sehingga salah pasang kewenangan akan bertahan
// sampai ada orang yang kebetulan bertanya.
func TestPemanggilBukanSurveyorDitolak(t *testing.T) {
	_, err := memory.NewSampleStore().
		ResolveSurveyor(context.Background(), memory.SampleOutsiderLogin)

	require.ErrorIs(t, err, inboxsurvey.ErrNotSurveyor)
}

// TestLeaderMelihatPekerjaanAnggotanya menguji hierarki `LOGINLEADER`.
//
// Tanpa hierarki, seorang leader melihat antrean yang jauh lebih sedikit dari seharusnya —
// dan antrean yang kurang lengkap tidak pernah terlihat sebagai kerusakan.
func TestLeaderMelihatPekerjaanAnggotanya(t *testing.T) {
	leader := identity(t, memory.SampleLeaderLogin)

	require.True(t, leader.IsLeader)
	require.Contains(t, leader.Scope, memory.SampleLeaderName)
	require.Contains(t, leader.Scope, memory.SampleMemberName)

	// PNCN.26.0102 milik anggota, dan leader harus melihatnya.
	require.Contains(t,
		numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered)),
		"PNCN.26.0102")
}

// TestAnggotaTidakMelihatPekerjaanLeader menjaga arah hierarkinya.
//
// Hierarki hanya berlaku SATU ARAH. Membalikkannya akan membuat setiap anggota melihat
// seluruh antrean timnya — dan itu tampak seperti layar yang bekerja dengan baik.
func TestAnggotaTidakMelihatPekerjaanLeader(t *testing.T) {
	member := identity(t, memory.SampleMemberLogin)

	require.False(t, member.IsLeader)
	require.Equal(t, []string{memory.SampleMemberName}, member.Scope)

	require.Equal(t, []string{"PNCN.26.0102"},
		numbers(list(t, memory.SampleMemberLogin, inboxsurvey.TabNotAnswered)),
		"anggota hanya melihat barisnya sendiri")
}

// TestNamaBerawalanSamaTidakIkutTerbawa menguji pembatas cakupan.
//
// `SampleNearMissName` memuat `SampleLeaderName` sebagai awalan. Bila pencocokannya sebagian,
// baris miliknya bocor ke antrean leader — terisi wajar, tanpa satu pun tanda.
func TestNamaBerawalanSamaTidakIkutTerbawa(t *testing.T) {
	require.NotContains(t,
		numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered)),
		"PNCN.26.0103", "baris itu milik surveyor lain yang namanya berawalan sama")
}

// TestSurveyorInternalMelihatAntreannyaSendiri mengunci keputusan Work Owner 2026-09-28.
//
// Layar ini melayani DUA populasi, dan yang membedakannya adalah identitas yang masuk — bukan
// penyaring yang dipilih pengguna.
func TestSurveyorInternalMelihatAntreannyaSendiri(t *testing.T) {
	page := list(t, memory.SampleInternalLogin, inboxsurvey.TabNotAnswered)

	require.Equal(t, []string{"PNCN.26.0110"}, numbers(page))
	require.Equal(t, inboxsurvey.SurveyorTypeInternal, page.Tasks[0].SurveyorType)
}

// TestKetigaTabKomunikasiTidakTertukar adalah uji yang paling berharga di berkas ini.
//
// Ketiganya dibedakan HANYA oleh arah pengirim dan status pesan. Tertukarnya menghasilkan
// tiga layar yang sama-sama masuk akal, sehingga tidak ada seorang pun yang akan
// melaporkannya.
func TestKetigaTabKomunikasiTidakTertukar(t *testing.T) {
	notAnswered := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered))
	notReplied := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotReplied))
	replied := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabReplied))

	// Pesan dari pihak LAIN yang belum dijawab. Termasuk baris milik anggota.
	require.Equal(t,
		[]string{"PNCN.26.0101", "PNCN.26.0102", "PNCN.26.0107", "PNCN.26.0108"},
		notAnswered, "pesan dari pihak LAIN yang belum dijawab")

	require.Equal(t, []string{"PNCN.26.0104"}, notReplied,
		"pesan dari DIRI SENDIRI yang belum dibalas")

	require.Equal(t, []string{"PNCN.26.0105"}, replied,
		"pesan dari diri sendiri yang SUDAH dibalas")

	// Ketiganya harus SALING LEPAS — satu baris tidak boleh muncul di dua tab sekaligus.
	for _, ref := range notReplied {
		require.NotContains(t, notAnswered, ref)
		require.NotContains(t, replied, ref)
	}
}

// TestBarisTanpaPesanTidakMunculDiTabManaPun membuktikan penyaring tab benar-benar menyaring.
//
// Ketiga tab yang dapat dihitung seluruhnya berbasis komunikasi. Tanpa uji ini, penyaring yang
// meloloskan segalanya akan lulus setiap uji lain di berkas ini.
func TestBarisTanpaPesanTidakMunculDiTabManaPun(t *testing.T) {
	for _, tab := range inboxsurvey.Tabs() {
		require.NotContainsf(t, numbers(list(t, memory.SampleLeaderLogin, tab)),
			"PNCN.26.0106", "baris tanpa pesan muncul di tab %q", tab)
	}
}

// TestTabYangBelumTersediaMengembalikanHalamanKosong mengunci pencegatan di Repo.List.
//
// Keempat tab itu bergantung pada kolom yang tidak ada. Mengembalikan halaman kosong BUKAN
// penyamaran: metadata layar sudah lebih dulu menyatakan tab itu belum tersedia beserta
// sebabnya, sehingga kosongnya tidak pernah terbaca sebagai "tidak ada pekerjaan".
func TestTabYangBelumTersediaMengembalikanHalamanKosong(t *testing.T) {
	hasUnavailable := false

	for _, tab := range inboxsurvey.Tabs() {
		if tab.Available() {
			continue
		}
		hasUnavailable = true

		page := list(t, memory.SampleLeaderLogin, tab)
		require.Emptyf(t, page.Tasks, "tab %q belum tersedia tetapi mengembalikan baris", tab)
		require.Zerof(t, page.Total, "tab %q belum tersedia tetapi mengembalikan jumlah", tab)
	}

	// Bila kelak keempat kolomnya tiba, uji ini kehilangan isinya. Kegagalan di sini adalah
	// TANDA BAIK: hidupkan tabnya dan hapus uji ini, jangan longgarkan.
	require.True(t, hasUnavailable,
		"tidak ada lagi tab yang belum tersedia — hidupkan keempat tab dan hapus uji ini")
}

// TestUrutanMenaikDenganPemutusSeri mengunci urutan antrean kerja.
//
// Yang tertua lebih dulu, sesuai `ORDER BY … ASC` pada kedua Browse rule yang ada. Dua baris
// contoh berwaktu input SAMA PERSIS, sehingga pemutus serinya ikut teruji.
func TestUrutanMenaikDenganPemutusSeri(t *testing.T) {
	got := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered))
	require.NotEmpty(t, got)

	// PNCN.26.0102 (2026-09-02) mendahului PNCN.26.0107 (2026-09-08).
	require.Less(t, indexOf(got, "PNCN.26.0102"), indexOf(got, "PNCN.26.0107"))

	// PNCN.26.0107 dan PNCN.26.0108 berwaktu SAMA PERSIS; urutannya ditentukan pemutus seri
	// CASEID — SRV-0007 mendahului SRV-0008.
	require.Less(t, indexOf(got, "PNCN.26.0107"), indexOf(got, "PNCN.26.0108"))
}

// TestTabTidakDikenalJatuhKeBawaan mengunci Filter.Normalize.
//
// Tab datang dari URL, dan URL yang tertinggal versi lama sebaiknya membuka halaman yang
// masuk akal — bukan layar galat.
func TestTabTidakDikenalJatuhKeBawaan(t *testing.T) {
	page, err := memory.NewSampleStore().List(
		context.Background(),
		identity(t, memory.SampleLeaderLogin),
		inboxsurvey.Filter{Tab: "tab-yang-tidak-ada", Limit: 100},
	)
	require.NoError(t, err)

	require.Equal(t, numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.DefaultTab)),
		numbers(page))
}

// TestPencarianMenyentuhClaimNo.
//
// HANYA Claim No, sama dengan kuerinya. Di Pega ia mencari pada dua kolom — yang kedua
// `REFNO_1`, dan kolom itu belum tersedia di tabel mana pun yang dibaca modul ini.
func TestPencarianMenyentuhClaimNo(t *testing.T) {
	store := memory.NewSampleStore()
	who := identity(t, memory.SampleLeaderLogin)

	byClaim, err := store.List(context.Background(), who, inboxsurvey.Filter{
		Tab: inboxsurvey.TabNotAnswered, Search: "0107", Limit: 100,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0107"}, numbers(byClaim))

	// Huruf kecil harus tetap cocok dengan nomor klaim berhuruf besar.
	lowerCase, err := store.List(context.Background(), who, inboxsurvey.Filter{
		Tab: inboxsurvey.TabNotAnswered, Search: "pncn.26.0107", Limit: 100,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0107"}, numbers(lowerCase),
		"pencarian harus tidak peka huruf besar-kecil")
}

// TestUmurTidakDapatDihitungBukanNolHari.
//
// `TGLINPUT` boleh kosong, dan "tidak dapat dihitung" berbeda artinya dari nol hari.
// Menyamakannya akan menampilkan "0" pada baris yang sebenarnya tidak punya angka.
func TestUmurTidakDapatDihitungBukanNolHari(t *testing.T) {
	page := list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered)
	now := time.Date(2026, time.September, 10, 3, 0, 0, 0, time.UTC)

	for _, task := range page.Tasks {
		if task.ClaimNumber == "PNCN.26.0101" {
			require.Nil(t, task.AgingDays(now, time.UTC))
			return
		}
	}
	t.Fatal("baris tanpa tanggal masuk tidak ditemukan di data contoh")
}

// TestUmurDihitungTerhadapTanggalWIB mengunci alasan zona waktu ikut masuk.
//
// Baris SRV-0002 masuk 2026-09-02 pukul 09.00 UTC. Dilihat pada 2026-09-03 pukul 01.00 UTC —
// yang di WIB sudah 2026-09-03 pukul 08.00 — umurnya SATU hari, bukan nol.
//
// Perbedaannya tidak akan terlihat pada baris yang sudah berumur berminggu-minggu; ia hanya
// terlihat pada baris yang baru masuk, dan itulah baris yang paling sering dilihat orang.
func TestUmurDihitungTerhadapTanggalWIB(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)
	now := time.Date(2026, time.September, 3, 1, 0, 0, 0, time.UTC)

	page := list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered)

	for _, task := range page.Tasks {
		if task.ClaimNumber != "PNCN.26.0102" {
			continue
		}

		age := task.AgingDays(now, wib)
		require.NotNil(t, age)
		require.Equal(t, 1, *age, "selisihnya 16 jam, tetapi tanggal WIB-nya sudah berganti")
		return
	}
	t.Fatal("baris PNCN.26.0102 tidak ditemukan di data contoh")
}

// TestJumlahTabSepadanDenganIsiTabnya.
//
// Bilah tab dan isi tab dihitung oleh dua jalur yang berbeda. Bila keduanya berbeda, pengguna
// melihat angka "3" pada tab yang isinya dua baris — dan itu meruntuhkan kepercayaan pada
// seluruh layar.
func TestJumlahTabSepadanDenganIsiTabnya(t *testing.T) {
	store := memory.NewSampleStore()
	who := identity(t, memory.SampleLeaderLogin)

	counts, err := store.Counts(context.Background(), who)
	require.NoError(t, err)

	// Hanya tab TERSEDIA yang dihitung — bukan ketujuhnya.
	tersedia := 0
	for _, tab := range inboxsurvey.Tabs() {
		if tab.Available() {
			tersedia++
		}
	}
	require.Len(t, counts, tersedia)

	for _, count := range counts {
		page, err := store.List(context.Background(), who,
			inboxsurvey.Filter{Tab: count.Tab, Limit: 100})
		require.NoError(t, err)

		require.Equalf(t, page.Total, count.Total,
			"jumlah tab %q berbeda dari isinya", count.Tab)
	}
}

// TestKPIDisaringCakupan mencegah papan penilaian menjadi bocor.
func TestKPIDisaringCakupan(t *testing.T) {
	rows, err := memory.NewSampleStore().KPI(
		context.Background(),
		identity(t, memory.SampleLeaderLogin),
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIFinal},
	)
	require.NoError(t, err)

	for _, row := range rows {
		require.NotEqual(t, memory.SampleNearMissName, row.Group,
			"penilaian adjuster di luar cakupan ikut terbawa")
	}
}

// TestKPIMenghitungRataRataBukanJumlah.
//
// Dua baris contoh milik leader pada tahun yang sama bernilai 90 dan 80 pada Penjadwalan
// Survey. Rata-ratanya 85; jumlahnya 170. Satu baris saja tidak dapat membedakan keduanya.
func TestKPIMenghitungRataRataBukanJumlah(t *testing.T) {
	rows, err := memory.NewSampleStore().KPI(
		context.Background(),
		identity(t, memory.SampleLeaderLogin),
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIFinal},
	)
	require.NoError(t, err)

	for _, row := range rows {
		if row.Group == memory.SampleLeaderName {
			require.Equal(t, 85.0, row.SurveyScheduling)
			return
		}
	}
	t.Fatal("baris KPI milik leader tidak ditemukan")
}

// TestRingkasanFinalMenolakKategoriLain.
//
// KPIFinal dan KPIQuarterly keduanya mematok `tipe = 'FINAL'` di dalam rule-nya sendiri.
// Menerima kategori dari layar akan membuat layar menampilkan angka yang di Pega tidak pernah
// dapat ditampilkan.
func TestRingkasanFinalMenolakKategoriLain(t *testing.T) {
	rows, err := memory.NewSampleStore().KPI(
		context.Background(),
		identity(t, memory.SampleLeaderLogin),
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIFinal, Category: "OUTSTANDING"},
	)
	require.NoError(t, err)

	for _, row := range rows {
		if row.Group == memory.SampleLeaderName {
			require.Equal(t, 85.0, row.SurveyScheduling,
				"kategori OUTSTANDING seharusnya tidak ikut terhitung")
		}
	}
}

// indexOf mengembalikan posisi sebuah nilai, atau -1.
func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}
