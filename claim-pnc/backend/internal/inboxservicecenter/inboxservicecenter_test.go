package inboxservicecenter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
)

// ---------------------------------------------------------------------------
// Tab — bentuk layar
// ---------------------------------------------------------------------------

// TestEmpatTabSesuaiWadahTABBED mengunci apa yang terbaca dari
// `Section/BrowseServiceCenter-Section.xml`: empat tab, judulnya apa adanya, dan nilai
// `stsapprove` yang dikirim masing-masing section.
func TestEmpatTabSesuaiWadahTABBED(t *testing.T) {
	tabs := inboxservicecenter.Tabs()
	require.Len(t, tabs, 4)

	type ringkas struct{ code, name, pega string }
	got := make([]ringkas, 0, len(tabs))
	for _, tab := range tabs {
		got = append(got, ringkas{tab.Code, tab.Name, tab.PegaParam})
	}

	require.Equal(t, []ringkas{
		{inboxservicecenter.TabRegistration, "Registrasi SC", ""},
		{inboxservicecenter.TabWaitingApproval, "Waiting Approval", "0"},
		{inboxservicecenter.TabApproved, "Approved", "1"},
		{inboxservicecenter.TabRejected, "Rejected", "2"},
	}, got)
}

// TestKeempatTabBerbagiEnamKolomYangSama — keempat section memuat `pyCaption` yang persis
// sama. Bila kelak seseorang menambah kolom pada satu tab saja, uji ini yang menangkapnya.
func TestKeempatTabBerbagiEnamKolomYangSama(t *testing.T) {
	judul := []string{"ID", "Tanggal Input", "No Polis", "Nasabah", "Tipe", "PIC"}

	for _, tab := range inboxservicecenter.Tabs() {
		require.Lenf(t, tab.Columns, 6, "tab %s", tab.Name)

		got := make([]string, 0, len(tab.Columns))
		for _, column := range tab.Columns {
			got = append(got, column.Title)
		}
		require.Equalf(t, judul, got, "urutan kolom tab %s", tab.Name)
	}
}

// TestDaftarKolomTidakDapatDiubahLewatHasilTabs — Tabs mengembalikan salinan.
func TestDaftarKolomTidakDapatDiubahLewatHasilTabs(t *testing.T) {
	tabs := inboxservicecenter.Tabs()
	tabs[0].Columns[0].Title = "DIUBAH"

	require.Equal(t, "ID", inboxservicecenter.Tabs()[0].Columns[0].Title)
}

func TestTabBawaanAdalahRegistrasiSC(t *testing.T) {
	require.Equal(t, inboxservicecenter.TabRegistration, inboxservicecenter.DefaultTab)

	_, known := inboxservicecenter.FindTab(inboxservicecenter.DefaultTab)
	require.True(t, known, "tab bawaan wajib terdaftar")
}

// ---------------------------------------------------------------------------
// Penyaring status persetujuan — tiga bentuk yang berbeda
// ---------------------------------------------------------------------------

// TestPenyaringStatusPersetujuanPerTab mengunci ketiga bentuk pada
// `Activity/DataServiceCenter-Act.xml` langkah 8, 12, dan 15.
func TestPenyaringStatusPersetujuanPerTab(t *testing.T) {
	caller := inboxservicecenter.Caller{Login: "PIC"}

	t.Run("Registrasi SC menyaring IS NULL, bukan kode apa pun", func(t *testing.T) {
		q := mustQuery(t, inboxservicecenter.TabRegistration, "", caller)

		require.True(t, q.Approval.MatchNull)
		require.Empty(t, q.Approval.Codes,
			"IS NULL tidak boleh dicampur dengan kode — `= NULL` tidak pernah benar di SQL")
	})

	t.Run("Waiting Approval menyaring kode 0", func(t *testing.T) {
		q := mustQuery(t, inboxservicecenter.TabWaitingApproval, "", caller)

		require.False(t, q.Approval.MatchNull)
		require.Equal(t, []string{inboxservicecenter.ApprovalPending}, q.Approval.Codes)
	})

	t.Run("Approved menyaring kode 1", func(t *testing.T) {
		q := mustQuery(t, inboxservicecenter.TabApproved, "", caller)

		require.Equal(t, []string{inboxservicecenter.ApprovalApproved}, q.Approval.Codes)
	})

	t.Run("Rejected membawa TLO dan REJECT sekaligus", func(t *testing.T) {
		q := mustQuery(t, inboxservicecenter.TabRejected, "", caller)

		require.False(t, q.Approval.MatchNull)
		require.Equal(t, []string{
			inboxservicecenter.ApprovalTotalLoss,
			inboxservicecenter.ApprovalRejected,
		}, q.Approval.Codes,
			"langkah 12 menimpa penyaring dasar dengan IN ('2','3')")
	})
}

// ---------------------------------------------------------------------------
// Permintaan
// ---------------------------------------------------------------------------

func TestTabKosongMenjadiTabBawaan(t *testing.T) {
	q, err := inboxservicecenter.NewQuery(
		inboxservicecenter.QueryInput{},
		inboxservicecenter.Caller{Login: "PIC"},
	)
	require.NoError(t, err)
	require.Equal(t, inboxservicecenter.DefaultTab, q.Tab.Code)
}

func TestTabTidakDikenalDitolakSebagaiGalatValidasi(t *testing.T) {
	_, err := inboxservicecenter.NewQuery(
		inboxservicecenter.QueryInput{Tab: "tab-karangan"},
		inboxservicecenter.Caller{Login: "PIC"},
	)

	var validation *inboxservicecenter.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxservicecenter.FieldTab, validation.Violations[0].Field)
}

// TestTanpaIdentitasPermintaanDitolak — keempat tab menyaring PIC menurut pemanggil, sehingga
// permintaan tanpa identitas tidak punya jawaban yang benar. Menampilkan seluruh klaim
// sebagai gantinya berarti membocorkan antrean orang lain.
func TestTanpaIdentitasPermintaanDitolak(t *testing.T) {
	for _, login := range []string{"", "   "} {
		_, err := inboxservicecenter.NewQuery(
			inboxservicecenter.QueryInput{},
			inboxservicecenter.Caller{Login: login},
		)
		require.ErrorIs(t, err, inboxservicecenter.ErrCallerUnknown)
	}
}

func TestKataKunciDipangkas(t *testing.T) {
	q := mustQuery(t, inboxservicecenter.TabApproved, "  90-001  ", inboxservicecenter.Caller{Login: "PIC"})
	require.Equal(t, "90-001", q.Keyword)
}

// TestPaginasiMatiSaatMencari mengunci prakondisi langkah 22 dan 23: klausa `rn` adalah
// satu-satunya paginasi kueri lama, dan ia dikosongkan begitu kotak cari terisi.
func TestPaginasiMatiSaatMencari(t *testing.T) {
	caller := inboxservicecenter.Caller{Login: "PIC"}

	require.True(t, mustQuery(t, inboxservicecenter.TabApproved, "", caller).Paginated(),
		"tanpa kata kunci, paginasi aktif")

	require.False(t, mustQuery(t, inboxservicecenter.TabApproved, "abc", caller).Paginated(),
		"dengan kata kunci, seluruh baris yang cocok ditampilkan sekaligus (P-5)")

	require.True(t, mustQuery(t, inboxservicecenter.TabApproved, "   ", caller).Paginated(),
		"spasi saja bukan pencarian")
}

// ---------------------------------------------------------------------------
// Paginasi
// ---------------------------------------------------------------------------

func TestPaginasiDibetulkanKeRentangYangSah(t *testing.T) {
	cases := []struct {
		nama       string
		masuk      inboxservicecenter.Pagination
		page, size int
	}{
		{"nol menjadi nilai bawaan", inboxservicecenter.Pagination{}, 1, inboxservicecenter.DefaultPageSize},
		{"halaman negatif menjadi 1", inboxservicecenter.Pagination{Page: -5, Size: 10}, 1, 10},
		{"ukuran di atas batas dipangkas", inboxservicecenter.Pagination{Page: 2, Size: 5000}, 2, inboxservicecenter.MaxPageSize},
		{"nilai sah dibiarkan", inboxservicecenter.Pagination{Page: 3, Size: 25}, 3, 25},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			clean := c.masuk.Normalize()
			require.Equal(t, c.page, clean.Page)
			require.Equal(t, c.size, clean.Size)
		})
	}
}

// TestUkuranHalamanBawaanDuaPuluhLima — angkanya dibaca dari langkah 21 activity, bukan
// disamakan dengan modul lain yang memakai 20.
func TestUkuranHalamanBawaanDuaPuluhLima(t *testing.T) {
	require.Equal(t, 25, inboxservicecenter.DefaultPageSize)
}

func TestOffsetDihitungDariHalamanYangSudahDibetulkan(t *testing.T) {
	require.Equal(t, 0, inboxservicecenter.Pagination{Page: 1, Size: 25}.Offset())
	require.Equal(t, 50, inboxservicecenter.Pagination{Page: 3, Size: 25}.Offset())
	require.Equal(t, 0, inboxservicecenter.Pagination{Page: -1, Size: 25}.Offset())
}

func TestJumlahHalamanMinimalSatu(t *testing.T) {
	kosong := inboxservicecenter.Page{
		Pagination: inboxservicecenter.Pagination{Page: 1, Size: 25},
		Paginated:  true,
	}
	require.Equal(t, 1, kosong.TotalPages(),
		"layar tidak boleh pernah menggambar \"halaman 1 dari 0\"")

	penuh := inboxservicecenter.Page{
		Total:      51,
		Pagination: inboxservicecenter.Pagination{Page: 1, Size: 25},
		Paginated:  true,
	}
	require.Equal(t, 3, penuh.TotalPages())
}

// TestHasilPencarianSelaluSatuHalaman — saat paginasi mati, seluruh baris ada di satu
// halaman, berapa pun jumlahnya.
func TestHasilPencarianSelaluSatuHalaman(t *testing.T) {
	hasil := inboxservicecenter.Page{
		Total:      900,
		Pagination: inboxservicecenter.Pagination{Page: 1, Size: 25},
		Paginated:  false,
	}
	require.Equal(t, 1, hasil.TotalPages())
}

// ---------------------------------------------------------------------------
// Label status
// ---------------------------------------------------------------------------

func TestLabelStatusPersetujuan(t *testing.T) {
	require.Equal(t, "Belum diajukan", inboxservicecenter.ApprovalStatusLabel(""))
	require.Equal(t, "Menunggu Approval", inboxservicecenter.ApprovalStatusLabel("0"))
	require.Equal(t, "APPROVED", inboxservicecenter.ApprovalStatusLabel("1"))
	require.Equal(t, "TLO", inboxservicecenter.ApprovalStatusLabel("2"))
	require.Equal(t, "REJECT", inboxservicecenter.ApprovalStatusLabel("3"))
}

func TestLabelStatusPerbaikanSembilanKode(t *testing.T) {
	harapan := map[string]string{
		"1": "Repair Submitted",
		"2": "Repair Assesment",
		"3": "Repair Cancel",
		"4": "Repair Indent",
		"5": "Repair Eligible",
		"6": "Repair Inprogress",
		"7": "Repair Completed",
		"8": "Pick UP",
		"9": "Data SC",
	}
	for kode, label := range harapan {
		require.Equalf(t, label, inboxservicecenter.RepairStatusLabel(kode), "kode %s", kode)
	}
	require.Equal(t, "", inboxservicecenter.RepairStatusLabel(""))
}

// TestKodeAsingDikembalikanApaAdanya — menyembunyikannya di balik satu label seragam akan
// membuat nilai yang belum terbaca modul ini tidak pernah ketahuan.
func TestKodeAsingDikembalikanApaAdanya(t *testing.T) {
	require.Equal(t, "99", inboxservicecenter.RepairStatusLabel("99"))
	require.Equal(t, "X", inboxservicecenter.ApprovalStatusLabel("X"))
}

// ---------------------------------------------------------------------------
// Keterbatasan
// ---------------------------------------------------------------------------

// TestKeterbatasanDisebutkanApaAdanya — keempatnya dikirim ke layar supaya pengguna tidak
// melaporkan hal yang sudah diketahui sebagai kerusakan modul.
func TestKeterbatasanDisebutkanApaAdanya(t *testing.T) {
	limitations := inboxservicecenter.Limitations()
	require.Len(t, limitations, 7)
	for _, kalimat := range limitations {
		require.NotEmpty(t, kalimat)
	}
}

func mustQuery(
	t *testing.T,
	tab, keyword string,
	caller inboxservicecenter.Caller,
) inboxservicecenter.Query {
	t.Helper()
	q, err := inboxservicecenter.NewQuery(
		inboxservicecenter.QueryInput{Tab: tab, Keyword: keyword},
		caller,
	)
	require.NoError(t, err)
	return q
}
