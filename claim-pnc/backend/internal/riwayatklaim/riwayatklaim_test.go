package riwayatklaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
)

func tanggal(year int, month time.Month, day int) *time.Time {
	moment := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &moment
}

// Kedua belas tipe pencarian sistem lama ada, dan kode 10 TIDAK ada.
//
// Lompatan dari "9" ke "11" adalah keadaan sistem lama, bukan salah ketik: tidak ada satu
// pun langkah `Search Type 10` di `Activity/PNCSearchHistoryKlaim_Act-Act.xml`. Uji ini
// menjaganya supaya tidak ada yang "merapikannya" menjadi 1..12 di kemudian hari —
// perapian itu akan membuat setiap rujukan ke nomor tipe menunjuk tipe yang berbeda.
func TestSearchTypesMengikutiKodeSistemLama(t *testing.T) {
	list := riwayatklaim.SearchTypes()
	require.Len(t, list, 12, "sistem lama punya 12 tipe pencarian")

	codes := make([]string, 0, len(list))
	for _, searchType := range list {
		codes = append(codes, searchType.Code)
	}
	require.Equal(t,
		[]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "11", "12", "13"},
		codes,
		"kode 10 tidak ada di sistem lama dan tidak boleh diadakan di sini")

	_, ada := riwayatklaim.FindSearchType("10")
	require.False(t, ada, "kode 10 tidak boleh dikenali")
}

// Tipe No Rekening ditandai belum tersedia, beserta alasannya.
//
// Ia tetap TAMPIL di dropdown — supaya kemajuan migrasi terbaca dari layar — tetapi
// permintaannya ditolak di lapisan domain, jauh sebelum menyentuh basis data.
func TestTipeNoRekeningBelumTersedia(t *testing.T) {
	searchType, ada := riwayatklaim.FindSearchType(riwayatklaim.TypeAccountNumber)
	require.True(t, ada, "pilihannya tetap ada di dropdown")
	require.False(t, searchType.Available)
	require.NotEmpty(t, searchType.Unavailable, "alasan wajib disebut supaya terbaca di layar")

	_, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type:       riwayatklaim.TypeAccountNumber,
		Text:       "0029763305",
		SearchDate: tanggal(2026, time.March, 1),
	})

	var validation *riwayatklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, riwayatklaim.FieldSearchType, validation.Violations[0].Field)
}

// CACAT YANG DIREPLIKASI — pencarian Tanggal Lahir memakai isian yang salah.
//
// Ini uji terpenting di berkas ini, dan ia menguji sebuah CACAT. Keberadaannya disengaja:
// Work Owner memutuskan 2026-09-20 cacat ini direplikasi demi kesetaraan `P-5`, dan
// keputusan yang tidak diuji akan tertimbun menjadi kelalaian yang "diperbaiki" seseorang
// di kemudian hari tanpa menyadari ia sedang mengubah perilaku.
//
// Aturannya di `Activity/PNCSearchHistoryKlaim_Act-Act.xml`: langkah 7 menyetel
// `CARI4 := TempSearch.DateOfSendInputor` untuk tipe 6, 9, dan 12 — sementara isian yang
// TAMPAK untuk tipe 9 adalah `TempSearch.SearchDate`.
func TestPencarianTanggalLahirMemakaiIsianYangSalah(t *testing.T) {
	criteria, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type:      riwayatklaim.TypeBirthDate,
		BirthDate: tanggal(1990, time.July, 17),
		// "Tanggal Pencarian" tidak tampak untuk tipe 9, sehingga layar tidak pernah
		// mengirimnya. Begitu pula di sistem lama.
		SearchDate: nil,
	})
	require.NoError(t, err, "isian yang tampak sudah diisi, jadi validasinya lolos")

	require.Equal(t, tanggal(1990, time.July, 17), criteria.BirthDate,
		"isian yang diketik pengguna tetap tersimpan")

	_, date, isDate := criteria.QueryValue()
	require.True(t, isDate)
	require.Nil(t, date,
		"yang dikirim ke kueri adalah Tanggal Pencarian yang kosong — "+
			"inilah cacat sistem lama yang direplikasi, dan itulah sebabnya pencarian "+
			"Tanggal Lahir tidak pernah mengembalikan baris")
}

// Tipe Tgl Kejadian TIDAK terkena cacat yang sama: isian yang tampak dan isian yang
// dipakai kebetulan sama, yaitu "Tanggal Pencarian".
func TestPencarianTanggalKejadianMemakaiIsianYangBenar(t *testing.T) {
	criteria, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type:       riwayatklaim.TypeLossDate,
		SearchDate: tanggal(2026, time.March, 12),
	})
	require.NoError(t, err)

	_, date, isDate := criteria.QueryValue()
	require.True(t, isDate)
	require.Equal(t, tanggal(2026, time.March, 12), date)
}

// Isian teks dibesarkan hurufnya, meniru `@toUpperCase(TempSearch.SearchName)`.
func TestIsianTeksDibesarkanHurufnya(t *testing.T) {
	criteria, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type: riwayatklaim.TypeInsuredName,
		Text: "  pt sumber contoh  ",
	})
	require.NoError(t, err)

	text, _, isDate := criteria.QueryValue()
	require.False(t, isDate)
	require.Equal(t, "PT SUMBER CONTOH", text)
}

// Isian yang tidak tampak untuk sebuah tipe DIBUANG, tidak dibawa diam-diam.
//
// Tanpa ini, nilai sisa dari tipe yang dipilih sebelumnya ikut masuk ke kriteria — dan
// pada tipe tanggal, sisa itu akan membuat pencarian menemukan baris yang tidak diminta.
func TestIsianYangTidakTampakDibuang(t *testing.T) {
	criteria, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type:       riwayatklaim.TypeClaimNumber,
		Text:       "PNC-9001",
		SearchDate: tanggal(2026, time.March, 12),
		BirthDate:  tanggal(1990, time.July, 17),
	})
	require.NoError(t, err)

	require.Equal(t, "PNC-9001", criteria.Text)
	require.Nil(t, criteria.SearchDate, "tipe No Klaim tidak menampilkan isian tanggal")
	require.Nil(t, criteria.BirthDate)
}

// Isian yang tampak WAJIB diisi — aturan yang TIDAK ada di sistem lama.
//
// Ia pengaman, bukan perubahan aturan bisnis: tanpa isian, `QQNAME like '%%'` menarik
// seluruh isi T_CLAIM_PNC yang berpuluh juta baris (`D-10`).
func TestIsianYangTampakWajibDiisi(t *testing.T) {
	_, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type: riwayatklaim.TypeInsuredName,
		Text: "   ",
	})

	var validation *riwayatklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, riwayatklaim.FieldSearchValue, validation.Violations[0].Field)
}

// Tipe yang tidak dikenal ditolak sebelum isian lain dinilai.
func TestTipeTidakDikenalDitolak(t *testing.T) {
	_, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{Type: "99"})

	var validation *riwayatklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, riwayatklaim.FieldSearchType, validation.Violations[0].Field)
}

// Jam dibuang dari isian tanggal, meniru `TRUNC` pada kueri lama.
func TestTanggalDipotongMenjadiTanggalKalender(t *testing.T) {
	siang := time.Date(2026, time.March, 12, 13, 45, 30, 0, time.UTC)

	criteria, err := riwayatklaim.NewCriteria(riwayatklaim.CriteriaInput{
		Type:       riwayatklaim.TypeLossDate,
		SearchDate: &siang,
	})
	require.NoError(t, err)

	require.Equal(t, tanggal(2026, time.March, 12), criteria.SearchDate)
}

// ---------------------------------------------------------------------------
// Gerbang proteksi data
// ---------------------------------------------------------------------------

// Pengguna yang belum terdaftar ditolak — "Input Data Proteksi Terlebih Dahulu".
func TestGerbangMenolakPenggunaBelumTerdaftar(t *testing.T) {
	_, err := riwayatklaim.Check(riwayatklaim.Protection{}, false, 0)
	require.ErrorIs(t, err, riwayatklaim.ErrNotRegistered)
}

// Jatah yang habis ditolak, dan sisanya dilaporkan nol — bukan negatif.
func TestGerbangMenolakJatahHabis(t *testing.T) {
	protection := riwayatklaim.Protection{Login: "adminpnc", SearchQuota: 3}

	access, err := riwayatklaim.Check(protection, true, 3)
	require.ErrorIs(t, err, riwayatklaim.ErrQuotaExhausted)
	require.Equal(t, 0, access.QuotaRemaining)

	// Pemakaian yang melampaui jatah — mungkin terjadi bila jatah di master DITURUNKAN
	// setelah pemakaian berjalan — tetap dilaporkan nol, tidak negatif.
	access, err = riwayatklaim.Check(protection, true, 10)
	require.ErrorIs(t, err, riwayatklaim.ErrQuotaExhausted)
	require.Equal(t, 0, access.QuotaRemaining)
}

// Grant memperhitungkan pemakaian yang sedang berjalan; Check tidak.
//
// Pembedaannya menentukan angka yang dibaca pengguna: melaporkan sisa sebelum pengurangan
// membuat angka di layar selalu satu lebih besar daripada kenyataannya.
func TestGrantMenghitungPemakaianYangSedangBerjalan(t *testing.T) {
	protection := riwayatklaim.Protection{Login: "adminpnc", SearchQuota: 5}

	diperiksa, err := riwayatklaim.Check(protection, true, 2)
	require.NoError(t, err)
	require.Equal(t, 2, diperiksa.QuotaUsed)
	require.Equal(t, 3, diperiksa.QuotaRemaining)

	dipakai, err := riwayatklaim.Grant(protection, true, 2)
	require.NoError(t, err)
	require.Equal(t, 3, dipakai.QuotaUsed)
	require.Equal(t, 2, dipakai.QuotaRemaining)
}

// Pemakaian terakhir masih boleh — jatah 1 dengan 0 terpakai berarti satu kali lagi.
func TestGrantMeluluskanPemakaianTerakhir(t *testing.T) {
	access, err := riwayatklaim.Grant(riwayatklaim.Protection{SearchQuota: 1}, true, 0)
	require.NoError(t, err)
	require.Equal(t, 0, access.QuotaRemaining, "sesudah ini habis")
}

func TestSplitSubModulesMembuangPotonganKosong(t *testing.T) {
	require.Equal(t,
		[]string{"PNCSearchKlaim", "PNCViewClaim"},
		riwayatklaim.SplitSubModules(" PNCSearchKlaim , , PNCViewClaim "))
	require.Empty(t, riwayatklaim.SplitSubModules(""))
}

// ---------------------------------------------------------------------------
// Paginasi
// ---------------------------------------------------------------------------

func TestPaginasiDibetulkanKeRentangYangSah(t *testing.T) {
	require.Equal(t,
		riwayatklaim.Pagination{Page: 1, Size: riwayatklaim.DefaultPageSize},
		riwayatklaim.Pagination{Page: 0, Size: 0}.Normalize())

	require.Equal(t,
		riwayatklaim.Pagination{Page: 1, Size: riwayatklaim.MaxPageSize},
		riwayatklaim.Pagination{Page: -3, Size: 5000}.Normalize(),
		"ukuran yang melebihi batas dipotong ke batas, bukan dibiarkan")
}

func TestOffsetMengikutiHalaman(t *testing.T) {
	require.Equal(t, 0, riwayatklaim.Pagination{Page: 1, Size: 20}.Offset())
	require.Equal(t, 40, riwayatklaim.Pagination{Page: 3, Size: 20}.Offset())
}

// Halaman tidak pernah nol, supaya layar tidak menggambar "halaman 1 dari 0".
func TestTotalHalamanMinimalSatu(t *testing.T) {
	kosong := riwayatklaim.Page{Pagination: riwayatklaim.Pagination{Page: 1, Size: 20}}
	require.Equal(t, 1, kosong.TotalPages())

	pas := riwayatklaim.Page{Total: 40, Pagination: riwayatklaim.Pagination{Page: 1, Size: 20}}
	require.Equal(t, 2, pas.TotalPages())

	sisa := riwayatklaim.Page{Total: 41, Pagination: riwayatklaim.Pagination{Page: 1, Size: 20}}
	require.Equal(t, 3, sisa.TotalPages())
}
