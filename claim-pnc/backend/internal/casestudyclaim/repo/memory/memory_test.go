package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/casestudyclaim/repo/memory"
)

// Penyaring di penyimpanan memori harus berperilaku PERSIS seperti penyaring di SQL.
//
// Bila keduanya menyimpang, layar yang benar saat pengembangan akan salah di produksi — dan
// tidak ada satu pun uji yang menembak keduanya sekaligus tanpa Oracle.

// wideRange adalah rentang yang mencakup seluruh tahun pada data contoh.
func wideRange() casestudyclaim.Filter {
	return casestudyclaim.Filter{FromYear: "2000", ToYear: "2099"}
}

func claimNumbers(page casestudyclaim.Page) []string {
	numbers := make([]string, 0, len(page.Rows))
	for _, row := range page.Rows {
		numbers = append(numbers, row.ClaimNumber)
	}
	return numbers
}

// Pada rentang tahun yang lebar, yang tertolak hanyalah yang tertolak AMBANG.
//
// Dua dari sepuluh baris contoh: satu karena nilainya di bawah ambang, satu karena
// totalnya besar tetapi terpecah. Ketiga baris "tertolak" lainnya tertolak oleh penyaring
// yang berbeda — tahun dan bisnis — dan karena itu tetap muncul di sini.
//
// Tanpa baris yang tertolak, penyimpanan contoh tidak membuktikan apa pun: penyaring yang
// rusak pun akan tampak benar.
func TestHanyaKlaimDiAtasAmbangYangMuncul(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(context.Background(), wideRange())
	require.NoError(t, err)

	require.Equal(t, []string{"STD-0001", "STD-0002", "STD-0003", "STD-0004", "STD-0005",
		"STD-0008", "STD-0009", "STD-0010"}, claimNumbers(page))

	require.NotContains(t, claimNumbers(page), "STD-0006", "nilainya di bawah ambang")
	require.NotContains(t, claimNumbers(page), "STD-0007",
		"totalnya di atas ambang, tetapi tidak ada SATU baris settlement yang melampauinya")
}

// Klaim bertotal Rp 8 miliar yang terpecah menjadi dua baris Rp 4 miliar TIDAK muncul.
//
// Inilah saksi yang membedakan pembacaan ambang yang benar — `EXISTS` atas SATU baris —
// dari pembacaan "jumlah seluruh settlement". Keduanya menghasilkan daftar yang tampak
// wajar, dan hanya baris ini yang memisahkannya.
func TestKlaimYangTotalnyaBesarTetapiTerpecahTidakMuncul(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(context.Background(), wideRange())
	require.NoError(t, err)
	require.NotContains(t, claimNumbers(page), "STD-0007")
}

func TestRentangTahunMenyaring(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(context.Background(),
		casestudyclaim.Filter{FromYear: "2024", ToYear: "2024"})
	require.NoError(t, err)

	require.Equal(t, []string{"STD-0001", "STD-0002", "STD-0009"}, claimNumbers(page))
}

// Rentang yang kosong ditolak, bukan dilayani sebagai "seluruh tahun".
func TestRentangKosongDitolak(t *testing.T) {
	store := memory.NewSampleStore()

	_, err := store.List(context.Background(), casestudyclaim.Filter{})
	require.ErrorIs(t, err, casestudyclaim.ErrPeriodRequired)
}

// Keempat cakupan bisnis, masing-masing diperiksa terhadap barisnya sendiri.
func TestCakupanBisnisMenyaring(t *testing.T) {
	store := memory.NewSampleStore()

	for name, kasus := range map[string]struct {
		scope    casestudyclaim.BusinessScope
		expected []string
	}{
		"PA": {casestudyclaim.ScopePA, []string{"STD-0002"}},

		"TRAVEL": {casestudyclaim.ScopeTravel, []string{"STD-0003"}},

		// NONMBU mencakup panel 003/004/006/009 KECUALI lima kode bisnis. STD-0009 berada
		// di panel 003 tetapi kode bisnisnya termasuk yang dikecualikan, sehingga ia hilang
		// di sini padahal muncul pada "semua".
		//
		// STD-0004 IKUT muncul meski ia klaim Bonding — lihat uji tumpang tindih di bawah.
		"NONMBU": {casestudyclaim.ScopeNonMBU,
			[]string{"STD-0001", "STD-0004", "STD-0005", "STD-0008"}},

		// BONDING berada di panel 003 yang SAMA dengan sebagian NONMBU; yang memisahkannya
		// hanyalah kesepuluh kode bisnisnya.
		"BONDING": {casestudyclaim.ScopeBonding, []string{"STD-0004"}},
	} {
		t.Run(name, func(t *testing.T) {
			filter := wideRange()
			filter.Business = kasus.scope

			page, err := store.List(context.Background(), filter)
			require.NoError(t, err)
			require.Equal(t, kasus.expected, claimNumbers(page))
		})
	}
}

// Kode bisnis yang DIKECUALIKAN dari NONMBU tetap muncul saat Bisnis dibiarkan kosong.
//
// Perbedaan antara "semua" dan "NONMBU" justru di situ, dan tanpa saksi ini penyaring yang
// keliru mengartikan kosong sebagai NONMBU akan lolos.
func TestKodeBisnisYangDikecualikanTetapMunculPadaSemua(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(context.Background(), wideRange())
	require.NoError(t, err)
	require.Contains(t, claimNumbers(page), "STD-0009")

	filter := wideRange()
	filter.Business = casestudyclaim.ScopeNonMBU
	page, err = store.List(context.Background(), filter)
	require.NoError(t, err)
	require.NotContains(t, claimNumbers(page), "STD-0009")
}

// NONMBU dan BONDING BERTUMPANG TINDIH, dan itu memang perilaku kueri lama.
//
// # Kenapa ini perlu uji tersendiri
//
// Keempat pilihan Bisnis terbaca seperti pembagian yang saling lepas — PA, TRAVEL, NONMBU,
// BONDING — dan pembaca berikutnya wajar mengira klaim Bonding tidak muncul di NONMBU.
// Kenyataannya:
//
//	NONMBU   panel 003/004/006/009 MINUS lima kode bisnis
//	BONDING  panel 003 DAN sepuluh kode bisnis tertentu
//
// Kesepuluh kode Bonding itu TIDAK ada di daftar lima yang dikecualikan NONMBU, sehingga
// setiap klaim Bonding memenuhi keduanya. Menjumlahkan hasil keempat pilihan karena itu
// menghitung sebagian klaim dua kali.
//
// Perilakunya direplikasi apa adanya (`P-5`) dan dicatat sebagai temuan, bukan diperbaiki:
// memperbaikinya berarti menghilangkan baris yang di layar lama terlihat.
func TestCakupanNONMBUDanBONDINGBertumpangTindih(t *testing.T) {
	store := memory.NewSampleStore()

	nonMBU := wideRange()
	nonMBU.Business = casestudyclaim.ScopeNonMBU
	page, err := store.List(context.Background(), nonMBU)
	require.NoError(t, err)
	require.Contains(t, claimNumbers(page), "STD-0004", "klaim Bonding IKUT muncul di NONMBU")

	bonding := wideRange()
	bonding.Business = casestudyclaim.ScopeBonding
	page, err = store.List(context.Background(), bonding)
	require.NoError(t, err)
	require.Contains(t, claimNumbers(page), "STD-0004")
}

// Status "CLAIM ON PROGRESS/ ACCEPT" adalah `!= '3'`, bukan `IN ('0','1')`.
func TestStatusMenyaringDenganOperatorYangSamaDenganSQL(t *testing.T) {
	store := memory.NewSampleStore()

	filter := wideRange()
	filter.Status = casestudyclaim.StatusRejected
	page, err := store.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, []string{"STD-0003"}, claimNumbers(page))

	filter.Status = casestudyclaim.StatusInProgressOrAccepted
	page, err = store.List(context.Background(), filter)
	require.NoError(t, err)
	require.NotContains(t, claimNumbers(page), "STD-0003")
	require.Contains(t, claimNumbers(page), "STD-0001")
}

// Paginasi memotong halaman dan tetap menyebut jumlah SELURUH baris yang cocok.
func TestPaginasiMemotongHalamanTanpaMengubahTotal(t *testing.T) {
	store := memory.NewSampleStore()

	filter := wideRange()
	filter.Limit = 3
	page, err := store.List(context.Background(), filter)
	require.NoError(t, err)

	require.Len(t, page.Rows, 3)
	require.Equal(t, 8, page.Total, "total adalah seluruh yang cocok, bukan yang tampil")

	filter.Offset = 6
	page, err = store.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, []string{"STD-0009", "STD-0010"}, claimNumbers(page))
	require.Equal(t, 8, page.Total)
}

// Halaman di luar jangkauan mengembalikan daftar kosong, bukan galat.
func TestHalamanDiLuarJangkauanMengembalikanDaftarKosong(t *testing.T) {
	store := memory.NewSampleStore()

	filter := wideRange()
	filter.Offset = 999
	page, err := store.List(context.Background(), filter)

	require.NoError(t, err)
	require.Empty(t, page.Rows)
	require.Equal(t, 8, page.Total)
}

// Nilai yang KOSONG tetap kosong sampai ke pemanggil.
//
// STD-0003 sengaja tidak punya Deductible, ASM Share, dan Lack of Doc — keadaan yang nyata
// ketika kolom sumbernya NULL pada seluruh baris settlement.
func TestNilaiKosongTidakBerubahMenjadiNol(t *testing.T) {
	store := memory.NewSampleStore()

	filter := wideRange()
	filter.Status = casestudyclaim.StatusRejected
	page, err := store.List(context.Background(), filter)
	require.NoError(t, err)
	require.Len(t, page.Rows, 1)

	row := page.Rows[0]
	require.Nil(t, row.Deductible)
	require.Nil(t, row.ASMSharePercent)
	require.Nil(t, row.LackOfDoc)

	// Yang terisi tetap terisi — supaya uji ini tidak lulus hanya karena barisnya kosong.
	require.NotNil(t, row.ClaimValue100)
}

func TestSimpanCatatanMengubahBarisnya(t *testing.T) {
	store := memory.NewSampleStore()

	saved, err := store.SaveRemark(context.Background(), "STD-0001", "  sudah ditelaah  ")
	require.NoError(t, err)
	require.True(t, saved)

	page, err := store.List(context.Background(), wideRange())
	require.NoError(t, err)
	require.Equal(t, "sudah ditelaah", page.Rows[0].Remark)
}

// Catatan boleh DIKOSONGKAN kembali.
func TestCatatanDapatDikosongkan(t *testing.T) {
	store := memory.NewSampleStore()

	saved, err := store.SaveRemark(context.Background(), "STD-0002", "")
	require.NoError(t, err)
	require.True(t, saved)

	page, err := store.List(context.Background(), wideRange())
	require.NoError(t, err)
	require.Equal(t, "", page.Rows[1].Remark)
}

// Nomor klaim yang tidak ada mengembalikan false, BUKAN galat.
//
// Baris dapat hilang di antara saat daftar dibaca dan saat Save ditekan, dan itu keadaan
// yang sah — pantas dijawab "muat ulang daftarnya", bukan "terjadi kesalahan sistem".
func TestSimpanCatatanPadaKlaimYangTidakAda(t *testing.T) {
	store := memory.NewSampleStore()

	saved, err := store.SaveRemark(context.Background(), "STD-9999", "apa pun")
	require.NoError(t, err)
	require.False(t, saved)
}

// Penyimpanan kosong tetap melayani, dan tidak ada baris yang tertolak menjadi galat.
func TestPenyimpananKosongMengembalikanDaftarKosong(t *testing.T) {
	store := memory.NewStore()

	page, err := store.List(context.Background(), wideRange())
	require.NoError(t, err)
	require.Empty(t, page.Rows)
	require.Equal(t, 0, page.Total)
}
