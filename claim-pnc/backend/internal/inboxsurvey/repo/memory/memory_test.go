package memory_test

import (
	"context"
	"testing"

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
	require.Contains(t, numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabAll)),
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

	require.NotContains(t, numbers(list(t, memory.SampleMemberLogin, inboxsurvey.TabAll)),
		"PNCN.26.0103")
}

// TestNamaBerawalanSamaTidakIkutTerbawa menguji pembatas cakupan.
//
// `SampleNearMissName` memuat `SampleLeaderName` sebagai awalan. Bila pencocokannya sebagian,
// baris miliknya bocor ke antrean leader — terisi wajar, tanpa satu pun tanda.
func TestNamaBerawalanSamaTidakIkutTerbawa(t *testing.T) {
	require.NotContains(t, numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabAll)),
		"PNCN.26.0103", "baris itu milik surveyor lain yang namanya berawalan sama")
}

// TestSurveyorInternalMelihatAntreannyaSendiri mengunci keputusan Work Owner 2026-09-28.
//
// Layar ini melayani DUA populasi, dan yang membedakannya adalah identitas yang masuk — bukan
// penyaring yang dipilih pengguna.
func TestSurveyorInternalMelihatAntreannyaSendiri(t *testing.T) {
	page := list(t, memory.SampleInternalLogin, inboxsurvey.TabOutstanding)

	require.Equal(t, []string{"PNCN.26.0110"}, numbers(page))
	require.Equal(t, inboxsurvey.SurveyorTypeInternal, page.Tasks[0].SurveyorType)
}

// TestTabOutstandingMemakaiIsNull adalah uji yang paling mudah terbalik.
//
// Data contoh memuat satu baris ber-`AdjusterAccept = "0"`. Di Pega baris seperti itu tidak
// masuk tab mana pun, karena lawan Outstanding adalah IS NULL — bukan <> "1".
func TestTabOutstandingMemakaiIsNull(t *testing.T) {
	outstanding := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabOutstanding))
	all := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabAll))

	require.Contains(t, outstanding, "PNCN.26.0101")
	require.NotContains(t, outstanding, "PNCN.26.0106", "baris itu bernilai \"0\", bukan NULL")
	require.NotContains(t, all, "PNCN.26.0106")
}

// TestTabInvoiceAdalahIrisanBukanKeranjangTersendiri.
//
// `SetTempLostAdjuster` MENAMBAHKAN penyaring status di atas penyaring Confirm, tidak
// menggantikannya. Baris Invoice karena itu muncul di tab ALL juga.
func TestTabInvoiceAdalahIrisanBukanKeranjangTersendiri(t *testing.T) {
	invoice := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabInvoice))
	all := numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabAll))

	require.Equal(t, []string{"PNCN.26.0104"}, invoice)
	require.Contains(t, all, "PNCN.26.0104")
}

// TestTabCloseMemakaiStatusAdjuster mengunci selisih terencana.
//
// Di Pega tab ini memakai status ALUR KERJA objek survei; di sini ia memakai
// `ADJUSTERSTATUS_1 = 'Close Case'` karena kolom itulah yang tersedia.
func TestTabCloseMemakaiStatusAdjuster(t *testing.T) {
	page := list(t, memory.SampleLeaderLogin, inboxsurvey.TabClose)

	require.Equal(t, []string{"PNCN.26.0105"}, numbers(page))
	require.Equal(t, inboxsurvey.StatusCloseCase, page.Tasks[0].ASMStatus)
}

// TestKetigaTabKomunikasiTidakTertukar adalah uji yang paling berharga di berkas ini.
//
// Ketiganya dibedakan HANYA oleh arah pengirim dan status pesan. Tertukarnya menghasilkan
// tiga layar yang sama-sama masuk akal, sehingga tidak ada seorang pun yang akan
// melaporkannya.
func TestKetigaTabKomunikasiTidakTertukar(t *testing.T) {
	require.Equal(t, []string{"PNCN.26.0107"},
		numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotAnswered)),
		"pesan dari pihak LAIN yang belum dijawab")

	require.Equal(t, []string{"PNCN.26.0108"},
		numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabNotReplied)),
		"pesan dari DIRI SENDIRI yang belum dibalas")

	require.Equal(t, []string{"PNCN.26.0109"},
		numbers(list(t, memory.SampleLeaderLogin, inboxsurvey.TabReplied)),
		"pesan dari diri sendiri yang SUDAH dibalas")
}

// TestUrutanMenaikDenganPemutusSeri mengunci urutan antrean kerja.
//
// Yang tertua lebih dulu, sesuai `ORDER BY … ASC` pada kedua Browse rule yang ada. Dua baris
// contoh berwaktu input SAMA PERSIS, sehingga pemutus serinya ikut teruji.
func TestUrutanMenaikDenganPemutusSeri(t *testing.T) {
	page, err := memory.NewSampleStore().List(
		context.Background(),
		identity(t, memory.SampleLeaderLogin),
		inboxsurvey.Filter{Tab: inboxsurvey.TabAll, Limit: 100},
	)
	require.NoError(t, err)

	got := numbers(page)
	require.NotEmpty(t, got)

	// PNCN.26.0104 (2026-09-04) mendahului PNCN.26.0105 (2026-09-05).
	require.Less(t, indexOf(got, "PNCN.26.0104"), indexOf(got, "PNCN.26.0105"))

	// Keduanya berwaktu sama; SRV-0008 mendahului SRV-0009 lewat pemutus seri.
	require.Less(t, indexOf(got, "PNCN.26.0108"), indexOf(got, "PNCN.26.0109"))
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

// TestPencarianMenyentuhClaimNoDanReferenceNo.
//
// Kedua kolom itu, bukan karangan: `SetTempLostAdjuster` menyusun penyaring carinya atas
// kunci klaim dan `REFNO_1`.
func TestPencarianMenyentuhClaimNoDanReferenceNo(t *testing.T) {
	store := memory.NewSampleStore()
	who := identity(t, memory.SampleLeaderLogin)

	byClaim, err := store.List(context.Background(), who,
		inboxsurvey.Filter{Tab: inboxsurvey.TabAll, Search: "0104", Limit: 100})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0104"}, numbers(byClaim))

	// `REF-0005` milik klaim `PNCN.26.0105` — keduanya sengaja BERBEDA nomornya, supaya uji
	// ini benar-benar membuktikan kolom Reference No ikut dicari dan bukan kebetulan cocok
	// dengan nomor klaimnya.
	byReference, err := store.List(context.Background(), who,
		inboxsurvey.Filter{Tab: inboxsurvey.TabAll, Search: "ref-0005", Limit: 100})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0105"}, numbers(byReference),
		"pencarian harus tidak peka huruf besar-kecil")
}

// TestAgingKosongBukanNolHari.
//
// Kolom `AGING` boleh NULL, dan NULL berbeda artinya dari nol. Menyamakannya akan
// menampilkan "0" pada baris yang sebenarnya tidak punya angka.
func TestAgingKosongBukanNolHari(t *testing.T) {
	page := list(t, memory.SampleLeaderLogin, inboxsurvey.TabOutstanding)

	for _, task := range page.Tasks {
		if task.ClaimNumber == "PNCN.26.0101" {
			require.Nil(t, task.AgingDays)
			return
		}
	}
	t.Fatal("baris ber-Aging kosong tidak ditemukan di data contoh")
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
	require.Len(t, counts, len(inboxsurvey.Tabs()))

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
