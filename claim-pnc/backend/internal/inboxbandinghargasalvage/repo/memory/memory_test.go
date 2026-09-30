package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
)

// acuan adalah hari acuan perhitungan umur, supaya ujinya deterministik.
var acuan = time.Date(2026, time.September, 29, 0, 0, 0, 0, time.UTC)

func hari(lalu int) *time.Time {
	at := acuan.AddDate(0, 0, -lalu)
	return &at
}

func kueri(t *testing.T, tab, login, cari string) inboxbandinghargasalvage.Query {
	t.Helper()

	query, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{Tab: tab, Keyword: cari},
		inboxbandinghargasalvage.Caller{Login: login},
	)
	require.NoError(t, err)
	return query
}

func halamanPenuh() inboxbandinghargasalvage.Pagination {
	return inboxbandinghargasalvage.Pagination{Page: 1, Size: inboxbandinghargasalvage.MaxPageSize}
}

// nomorKlaim menyerahkan nomor klaim tiap baris, berurutan seperti tampilnya.
func nomorKlaim(rows []inboxbandinghargasalvage.AppealRow) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ClaimNo)
	}
	return result
}

// ============================================================================
// Penyaring grid Request
// ============================================================================

// Keempat penyaring diuji dalam satu penyimpanan, supaya yang lolos benar-benar HANYA baris
// yang memenuhi keempatnya — bukan baris yang kebetulan lolos karena penyaring lain tidak
// sempat berjalan.
func TestGridRequestHanyaMenampilkanBandingYangMemenuhiKeempatPenyaring(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0001", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Belum dibanding — HARGAREQUEST kosong.
			ClaimNo: "PNCN.26.0002", DetailObject: "D2", CommitteeName: "KOMITESAYA",
			RequestPrice: "", RequestDate: hari(3), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Barangnya tidak ada di DETAIL_PNC_SALVAGE.
			ClaimNo: "PNCN.26.0003", DetailObject: "D3", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: false,
		},
		memory.CheckerRecord{
			// Sudah diputus — TGLAPPROVE terisi.
			ClaimNo: "PNCN.26.0004", DetailObject: "D4", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), ApprovedAt: hari(1),
			ApprovalStatus: "1", InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Milik komite lain.
			ClaimNo: "PNCN.26.0005", DetailObject: "D5", CommitteeName: "KOMITELAIN",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0001"}, nomorKlaim(page.Items))
	require.Equal(t, 1, page.Total)
}

// ============================================================================
// Urutan umur — selisih terencana nomor 1
// ============================================================================

// Inilah uji yang menangkap cacat bila ia kembali.
//
// Di Pega umur disusun sebagai TEKS lalu diurutkan sebagai teks, sehingga '9 days'
// didahulukan dari '30 days'. Bila seseorang kelak mengembalikan bentuk itu, urutan di bawah
// akan terbalik dan uji ini gagal — bukan diam-diam salah di layar.
func TestUmurDiurutkanSebagaiAngkaBukanSebagaiTeks(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.SEMBILAN", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(9), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.TIGAPULUH", DetailObject: "D2", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(30), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.SERATUS", DetailObject: "D3", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(100), InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{
		"PNCN.26.SERATUS", "PNCN.26.TIGAPULUH", "PNCN.26.SEMBILAN",
	}, nomorKlaim(page.Items), "yang paling lama menunggu harus di ATAS")

	require.Equal(t, 100, page.Items[0].AgingDays)
	require.Equal(t, 30, page.Items[1].AgingDays)
	require.Equal(t, 9, page.Items[2].AgingDays)
}

func TestBarisTanpaTanggalRequestBerumurNolDanTidakMenggagalkan(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0001", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: nil, InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, 0, page.Items[0].AgingDays)
	require.Nil(t, page.Items[0].RequestDate)
}

// ============================================================================
// Penyaring giliran komite — aturan bernama orang
// ============================================================================

// Baris yang komite sebelumnya BELUM putuskan tidak boleh tampil; yang sudah, tampil.
//
// Keduanya diuji bersamaan supaya penyaring yang terlalu longgar (semua tampil) maupun yang
// terlalu ketat (tidak ada yang tampil) sama-sama ketahuan.
func TestGiliranKomiteMenahanBarisYangKomiteSebelumnyaBelumPutuskan(t *testing.T) {
	store := memory.NewStore(
		// Barang A — giliran BAMBANG masih menggantung.
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.TERTAHAN", DetailObject: "DA", CommitteeName: "DANIELLISWANDI",
			RequestPrice: "100", RequestDate: hari(5), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.TERTAHAN", DetailObject: "DA",
			CommitteeName: "BAMBANGSETIADJIGUNAWAN",
			RequestPrice:  "100", RequestDate: hari(5), InDetailSalvage: true,
		},

		// Barang B — giliran BAMBANG sudah lewat.
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.LOLOS", DetailObject: "DB", CommitteeName: "DANIELLISWANDI",
			RequestPrice: "100", RequestDate: hari(5), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.LOLOS", DetailObject: "DB",
			CommitteeName: "BAMBANGSETIADJIGUNAWAN",
			RequestPrice:  "100", RequestDate: hari(5), ApprovedAt: hari(2),
			ApprovalStatus: "1", InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "DANIELLISWANDI", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.LOLOS"}, nomorKlaim(page.Items))
}

// Penyaring giliran itu berlaku bagi SATU nama saja. Komite lain melihat barisnya apa adanya,
// dan uji ini menjaga agar penyaringnya tidak diam-diam meluas ke semua orang — yang akan
// membuat antrean tampak kosong bagi setiap komite yang barangnya ditangani berdua.
func TestGiliranKomiteTidakBerlakuBagiKomiteLain(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0001", DetailObject: "DA", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(5), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0001", DetailObject: "DA",
			CommitteeName: "BAMBANGSETIADJIGUNAWAN",
			RequestPrice:  "100", RequestDate: hari(5), InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
}

// ============================================================================
// Pencarian
// ============================================================================

// Pencarian COCOK PERSIS, bukan mengandung — persis seperti kuerinya di Pega.
func TestPencarianCocokPersisBukanMengandung(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0451", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0452", DetailObject: "D2", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: true,
		},
	).WithNow(acuan)

	utuh, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", "PNCN.26.0451"),
		halamanPenuh(),
	)
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0451"}, nomorKlaim(utuh.Items))

	separuh, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", "PNCN.26"),
		halamanPenuh(),
	)
	require.NoError(t, err)
	require.Empty(t, separuh.Items, "separuh nomor klaim TIDAK menghasilkan baris")
}

func TestPencarianTidakPekaHurufBesarKecil(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0451", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", "pncn.26.0451"),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
}

// ============================================================================
// Grid History
// ============================================================================

// Ia menampilkan pengajuan salvage milik klaim yang bandingnya SUDAH diputus komite ini —
// dan tanpa DISTINCT, sehingga satu klaim berpengajuan ganda muncul dua kali.
func TestGridHistoryMenampilkanSetiapPengajuanTanpaMenggabungkanKlaim(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0440", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(40), ApprovedAt: hari(30),
			ApprovalStatus: "1", InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Belum diputus — klaimnya tidak boleh muncul di History.
			ClaimNo: "PNCN.26.0451", DetailObject: "D2", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(5), InDetailSalvage: true,
		},
	).WithSalvages(
		memory.SalvageRecord{
			ClaimNo: "PNCN.26.0440", SalvageID: "S2", SalvageType: "Suku Cadang",
			SalvageLocation: "Gudang A", PIC: "PICSATU",
		},
		memory.SalvageRecord{
			ClaimNo: "PNCN.26.0440", SalvageID: "S1", SalvageType: "Alat Berat",
			SalvageLocation: "Gudang A", PIC: "PICSATU",
		},
		memory.SalvageRecord{
			ClaimNo: "PNCN.26.0451", SalvageID: "S9", SalvageType: "Mesin",
			SalvageLocation: "Gudang B", PIC: "PICDUA",
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabHistory, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0440", "PNCN.26.0440"}, nomorKlaim(page.Items))

	// Urut menurut IDSALVAGE di dalam satu klaim — paginasi menuntut urutan yang pasti.
	require.Equal(t, "Alat Berat", page.Items[0].SalvageType)
	require.Equal(t, "Suku Cadang", page.Items[1].SalvageType)
}

// Putusan komite LAIN tidak membuka History bagi pemanggil.
func TestGridHistoryTidakTerbukaOlehPutusanKomiteLain(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0440", DetailObject: "D1", CommitteeName: "KOMITELAIN",
			RequestPrice: "100", RequestDate: hari(40), ApprovedAt: hari(30),
			ApprovalStatus: "1", InDetailSalvage: true,
		},
	).WithSalvages(
		memory.SalvageRecord{ClaimNo: "PNCN.26.0440", SalvageID: "S1"},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabHistory, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Empty(t, page.Items)
}

// Baris yang TGLAPPROVE-nya terisi tetapi STATUSAPPROVE kosong belum dianggap diputus —
// kuerinya menuntut KEDUANYA terisi.
func TestHistoryMenuntutKeduaPenandaPutusanTerisi(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0440", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(40), ApprovedAt: hari(30),
			ApprovalStatus: "", InDetailSalvage: true,
		},
	).WithSalvages(
		memory.SalvageRecord{ClaimNo: "PNCN.26.0440", SalvageID: "S1"},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabHistory, "KOMITESAYA", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Empty(t, page.Items)
}

// ============================================================================
// Pencacah dan paginasi
// ============================================================================

// Selisih terencana nomor 4: angka ringkas HARUS sama dengan jumlah baris gridnya.
func TestPencacahMenghitungPopulasiYangSamaDenganDaftarnya(t *testing.T) {
	// Jamnya TIDAK dipatok di sini. Tanggal pada baris contoh dibentuk relatif terhadap hari
	// ini (lihat sample.go), sehingga memasangkannya dengan jam tetap akan menghasilkan umur
	// negatif begitu hari berganti — uji yang lulus hari ini dan gagal besok tanpa satu pun
	// baris kode berubah.
	store := memory.NewSampleStore()

	for _, tab := range inboxbandinghargasalvage.Tabs() {
		query := kueri(t, tab.Code, memory.SampleOwner, "")

		page, err := store.List(context.Background(), query, halamanPenuh())
		require.NoError(t, err)

		count, err := store.Count(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, page.Total, count, "tab %s", tab.Code)
		require.Len(t, page.Items, count, "tab %s", tab.Code)
	}
}

func TestPaginasiMemotongHalamanDanMenyatakanJumlahSeluruhnya(t *testing.T) {
	records := make([]memory.CheckerRecord, 0, 5)
	for index := 0; index < 5; index++ {
		records = append(records, memory.CheckerRecord{
			ClaimNo:      "PNCN.26.000" + string(rune('1'+index)),
			DetailObject: "D" + string(rune('1'+index)),

			CommitteeName: "KOMITESAYA",
			RequestPrice:  "100",
			// Umur menurun, sehingga urutannya dapat diperiksa dengan mata.
			RequestDate:     hari(50 - index),
			InDetailSalvage: true,
		})
	}

	store := memory.NewStore(records...).WithNow(acuan)
	query := kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", "")

	pertama, err := store.List(
		context.Background(), query,
		inboxbandinghargasalvage.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Len(t, pertama.Items, 2)
	require.Equal(t, 5, pertama.Total)
	require.Equal(t, 3, pertama.TotalPages())

	ketiga, err := store.List(
		context.Background(), query,
		inboxbandinghargasalvage.Pagination{Page: 3, Size: 2})
	require.NoError(t, err)
	require.Len(t, ketiga.Items, 1, "halaman terakhir berisi sisa barisnya")
}

func TestHalamanDiLuarJangkauanMenghasilkanDaftarKosongBukanGalat(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0001", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			RequestPrice: "100", RequestDate: hari(3), InDetailSalvage: true,
		},
	).WithNow(acuan)

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITESAYA", ""),
		inboxbandinghargasalvage.Pagination{Page: 99, Size: 25},
	)

	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Equal(t, 1, page.Total, "jumlah seluruhnya tetap dilaporkan")
}

// ============================================================================
// Baris contoh
// ============================================================================

// Baris contoh ADA gunanya hanya bila ia benar-benar melatih penyaringnya. Uji ini menjaga
// agar seseorang yang menyederhanakan sample.go tidak diam-diam menghapus kasus yang
// membuat penyaringnya teruji.
func TestBarisContohMelatihSetiapPenyaring(t *testing.T) {
	// Jamnya tidak dipatok — lihat catatan pada uji pencacah di atas.
	store := memory.NewSampleStore()
	ctx := context.Background()

	milikSaya, err := store.List(
		ctx, kueri(t, inboxbandinghargasalvage.TabRequest, memory.SampleOwner, ""),
		halamanPenuh())
	require.NoError(t, err)
	require.NotEmpty(t, milikSaya.Items, "SampleOwner harus punya antrean")

	for _, row := range milikSaya.Items {
		require.Equal(t, memory.SampleOwner, row.CommitteeName)
		require.NotEmpty(t, row.RequestPrice)
	}

	// Satu baris contoh sengaja tanpa tanggal request.
	tanpaTanggal := 0
	for _, row := range milikSaya.Items {
		if row.RequestDate == nil {
			tanpaTanggal++
		}
	}
	require.Equal(t, 1, tanpaTanggal)

	// Komite lain melihat antrean yang berbeda.
	milikOrangLain, err := store.List(
		ctx, kueri(t, inboxbandinghargasalvage.TabRequest, "KOMITELAIN", ""),
		halamanPenuh())
	require.NoError(t, err)
	require.NotEmpty(t, milikOrangLain.Items)
	require.NotEqual(t, nomorKlaim(milikSaya.Items), nomorKlaim(milikOrangLain.Items))

	// History SampleOwner berisi klaim yang sudah diputus, dua kali karena dua pengajuan.
	riwayat, err := store.List(
		ctx, kueri(t, inboxbandinghargasalvage.TabHistory, memory.SampleOwner, ""),
		halamanPenuh())
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0440", "PNCN.26.0440"}, nomorKlaim(riwayat.Items))
}

// Baris contoh untuk penyaring giliran harus benar-benar menahan satu baris dan meloloskan
// satu baris — kalau tidak, aturan itu tidak teruji sama sekali di pengembangan lokal.
func TestBarisContohMelatihPenyaringGiliranKomite(t *testing.T) {
	// Jamnya tidak dipatok — lihat catatan pada uji pencacah di atas.
	store := memory.NewSampleStore()

	page, err := store.List(
		context.Background(),
		kueri(t, inboxbandinghargasalvage.TabRequest, "DANIELLISWANDI", ""),
		halamanPenuh(),
	)

	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0461"}, nomorKlaim(page.Items),
		"yang tertahan giliran BAMBANG tidak boleh tampil")
}

// ============================================================================
// Panel rincian History Cheker
// ============================================================================

func rincian(t *testing.T, klaim, login string) inboxbandinghargasalvage.DecisionQuery {
	t.Helper()

	query, err := inboxbandinghargasalvage.NewDecisionQuery(
		klaim, inboxbandinghargasalvage.Caller{Login: login})
	require.NoError(t, err)
	return query
}

// Panel menampilkan keputusan klaim itu SAJA, milik komite pemanggil, dan hanya yang sudah
// diputus. Ketiga penyaringnya diuji bersamaan supaya yang lolos benar-benar memenuhi
// ketiganya.
func TestPanelRincianMenampilkanKeputusanKlaimItuMilikKomitePemanggil(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0440", DetailObject: "D1", CommitteeName: "KOMITESAYA",
			ItemName: "Forklift", ItemPrice: "27000000", RequestPrice: "21000000",
			ApprovalStatus: "1", ApprovedAt: hari(10), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Klaim lain.
			ClaimNo: "PNCN.26.0441", DetailObject: "D2", CommitteeName: "KOMITESAYA",
			ApprovalStatus: "1", ApprovedAt: hari(10), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Komite lain.
			ClaimNo: "PNCN.26.0440", DetailObject: "D3", CommitteeName: "KOMITELAIN",
			ApprovalStatus: "1", ApprovedAt: hari(10), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			// Belum diputus.
			ClaimNo: "PNCN.26.0440", DetailObject: "D4", CommitteeName: "KOMITESAYA",
			ApprovalStatus: "", InDetailSalvage: true,
		},
	).WithNow(acuan)

	items, err := store.ListDecisions(
		context.Background(), rincian(t, "PNCN.26.0440", "KOMITESAYA"))

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "D1", items[0].DetailObject)
	require.Equal(t, "Forklift", items[0].ItemName)
	require.Equal(t, "1", items[0].Status)
}

// Satu klaim dapat punya BEBERAPA barang yang dibanding, dan urutannya terbaru di atas.
func TestPanelRincianMengurutkanKeputusanTerbaruDiAtas(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0440", DetailObject: "LAMA", CommitteeName: "KOMITESAYA",
			ApprovalStatus: "1", ApprovedAt: hari(30), InDetailSalvage: true,
		},
		memory.CheckerRecord{
			ClaimNo: "PNCN.26.0440", DetailObject: "BARU", CommitteeName: "KOMITESAYA",
			ApprovalStatus: "0", ApprovedAt: hari(2), InDetailSalvage: true,
		},
	).WithNow(acuan)

	items, err := store.ListDecisions(
		context.Background(), rincian(t, "PNCN.26.0440", "KOMITESAYA"))

	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "BARU", items[0].DetailObject)
	require.Equal(t, "LAMA", items[1].DetailObject)
}

// Klaim tanpa keputusan menghasilkan daftar KOSONG, bukan galat — panel ini dibuka dari baris
// yang sudah tergambar, sehingga klaimnya pasti ada.
func TestPanelRincianKlaimTanpaKeputusanMengembalikanDaftarKosong(t *testing.T) {
	store := memory.NewStore().WithNow(acuan)

	items, err := store.ListDecisions(
		context.Background(), rincian(t, "PNCN.26.9999", "KOMITESAYA"))

	require.NoError(t, err)
	require.Empty(t, items)
	require.NotNil(t, items, "senarai kosong, bukan nil")
}
