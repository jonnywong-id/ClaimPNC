package inboxmanager_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

// TestKodeTabMengikutiKontainerSection mengunci kode tab pada angka yang dipakai export.
//
// Ketiga belas kontainer `Section/InboxManager_Sec` bersyarat `FlagManager.AlasanKlaim==1`
// sampai `==13`, dan angka yang sama ditulis `Activity/CountDashbroardManager` ke isian
// `ALASAN` setiap pencacah. Menomori ulang di sini akan memutus penelusuran balik ke export
// tanpa satu pun galat.
func TestKodeTabMengikutiKontainerSection(t *testing.T) {
	tabs := inboxmanager.Tabs()
	require.Len(t, tabs, 13, "layar lama punya tepat tiga belas kontainer")

	for i, tab := range tabs {
		require.Equal(t, strconv.Itoa(i+1), tab.Code,
			"kode tab ke-%d harus %d, mengikuti FlagManager.AlasanKlaim", i+1, i+1)
	}
}

// TestSubTabApprovalProgressKlaimTidakDibawa menahan bagian yang MATI di Pega kembali.
//
// `Section/Sec_PaymentAkseptasiKlaimCase1-Section.xml:67064` memasang
// `pyContainerVisibleWhen = 1==2` pada sub-tab "Approval Progress Klaim" — kondisi yang tidak
// pernah benar. Bagian itu sudah dimatikan di sistem lama dan tidak pernah tampil bagi siapa
// pun.
func TestSubTabApprovalProgressKlaimTidakDibawa(t *testing.T) {
	for _, tab := range inboxmanager.Tabs() {
		require.NotContains(t, tab.Name, "Progress Klaim",
			"sub-tab yang bersyarat 1==2 di Pega tidak boleh dibawa")
	}
}

// TestHanyaSatuTabDibatasiLiniBisnis mengunci batas yang benar-benar ada di export.
//
// Hanya `Section/InboxKonfirmasiHE_Section` yang memasang syarat jabatan. Dua belas kontainer
// lain tidak punya satu pun — yang menjaga siapa boleh membuka layarnya adalah butir menunya.
func TestHanyaSatuTabDibatasiLiniBisnis(t *testing.T) {
	dibatasi := []inboxmanager.Tab{}
	for _, tab := range inboxmanager.Tabs() {
		if tab.LineBusiness != "" {
			dibatasi = append(dibatasi, tab)
		}
	}

	require.Len(t, dibatasi, 1)
	require.Equal(t, inboxmanager.TabNomorRangka, dibatasi[0].Code)
	require.Equal(t, inboxmanager.LineNonMBU, dibatasi[0].LineBusiness)
}

// TestTabNomorRangkaTersembunyiBagiLiniLain menegakkan batas itu di server.
func TestTabNomorRangkaTersembunyiBagiLiniLain(t *testing.T) {
	rangka, _ := inboxmanager.FindTab(inboxmanager.TabNomorRangka)

	nonMBU := inboxmanager.Caller{Login: "A", LineBusiness: inboxmanager.LineNonMBU}
	require.True(t, inboxmanager.CanSee(nonMBU, rangka))

	pa := inboxmanager.Caller{Login: "B", LineBusiness: inboxmanager.LinePA}
	require.False(t, inboxmanager.CanSee(pa, rangka))

	// Unit organisasi Development membukanya, persis seperti di Pega.
	dev := inboxmanager.Caller{Login: "C", OrgUnit: inboxmanager.DevelopmentOrgUnit}
	require.True(t, inboxmanager.CanSee(dev, rangka))

	// Petugas tanpa lini bisnis tetap melihat dua belas tab lainnya — batasnya hanya pada
	// satu tab, bukan pada layarnya.
	kosong := inboxmanager.Caller{Login: "D"}
	require.Len(t, inboxmanager.VisibleTabs(kosong), 12)
}

// TestDashboardOutstandingTidakPunyaPenyaringPeriode mengunci selisih yang mudah "diperbaiki"
// menjadi salah.
//
// Ketiga kueri Pega yang memasoknya tidak menyaring tanggal sama sekali: yang dihitungnya
// klaim yang SEDANG berjalan, bukan klaim pada suatu periode. Menambahkan penyaring periode di
// sana mengubah arti angkanya.
func TestDashboardOutstandingTidakPunyaPenyaringPeriode(t *testing.T) {
	outstanding, _ := inboxmanager.FindTab(inboxmanager.TabOutstanding)
	require.False(t, outstanding.HasPeriodFilter)

	produktivitas, _ := inboxmanager.FindTab(inboxmanager.TabProduktivitas)
	require.True(t, produktivitas.HasPeriodFilter)

	klaim, _ := inboxmanager.FindTab(inboxmanager.TabKlaim)
	require.True(t, klaim.HasPeriodFilter)
}

// TestPersetujuanPaymentAkseptasiDitahan mengunci keputusan yang menyangkut uang.
//
// Menyetujui di Pega ikut menjalankan `TransferToKasir_act_Leader`, dan rantai itu belum
// lengkap. Menuliskan STSAPP='1' tanpa langkah itu menandai pembayaran sudah disetujui padahal
// tidak pernah sampai ke kasir.
func TestPersetujuanPaymentAkseptasiDitahan(t *testing.T) {
	tab, _ := inboxmanager.FindTab(inboxmanager.TabPaymentAkseptasi)

	require.True(t, tab.Decision.Decidable)
	require.NotEmpty(t, tab.Decision.ApproveBlockedReason,
		"jalur setuju tab ini WAJIB ditahan sampai rantai transfer kasirnya lengkap")

	caller := inboxmanager.Caller{Login: "JONNY"}

	_, err := inboxmanager.NewDecision(
		tab.Code, inboxmanager.VerdictApprove, []string{"AKS-1"}, "", caller)
	require.ErrorIs(t, err, inboxmanager.ErrApproveBlocked)

	// Menolak TIDAK ditahan: jalur itu berhenti di UPDATE tabel checker.
	decision, err := inboxmanager.NewDecision(
		tab.Code, inboxmanager.VerdictReject, []string{"AKS-1"}, "nilai belum sesuai", caller)
	require.NoError(t, err)
	require.Equal(t, inboxmanager.StatusRejected, decision.Verdict.Value())
}

// TestAlasanWajibHanyaPadaAntreanYangPunyaKolomnya menegakkan aturan yang LEBIH KETAT daripada
// Pega.
//
// Pega menerima penolakan tanpa alasan apa pun. Di sini penolakan tanpa alasan ditolak pada
// antrean yang tabelnya punya kolom alasan — kolom itu satu-satunya hal yang memberi tahu
// pengaju kenapa barisnya ditolak.
func TestAlasanWajibHanyaPadaAntreanYangPunyaKolomnya(t *testing.T) {
	caller := inboxmanager.Caller{Login: "JONNY"}

	// Master Bengkel PUNYA kolom alasan.
	_, err := inboxmanager.NewDecision(
		inboxmanager.TabMasterBengkel, inboxmanager.VerdictReject,
		[]string{"BGK-1"}, "", caller)
	require.Error(t, err)

	var validation *inboxmanager.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxmanager.FieldReason, validation.Violations[0].Field)

	// Nomor Rangka TIDAK punya kolom alasan — penolakan tanpa alasan sah.
	decision, err := inboxmanager.NewDecision(
		inboxmanager.TabNomorRangka, inboxmanager.VerdictReject,
		[]string{"a|b|c|d"}, "",
		inboxmanager.Caller{Login: "JONNY", LineBusiness: inboxmanager.LineNonMBU})
	require.NoError(t, err)
	require.Empty(t, decision.Reason)
}

// TestAlasanDibuangPadaAntreanTanpaKolomAlasan menahan teks pengguna tersimpan di tempat yang
// tidak ada.
func TestAlasanDibuangPadaAntreanTanpaKolomAlasan(t *testing.T) {
	decision, err := inboxmanager.NewDecision(
		inboxmanager.TabKategoriSparepart, inboxmanager.VerdictReject,
		[]string{"KAT-1"}, "alasan yang tidak punya kolom",
		inboxmanager.Caller{Login: "JONNY"})

	require.NoError(t, err)
	require.Empty(t, decision.Reason)
}

// TestKunciGandaDibuang menahan laporan selisih yang keliru.
//
// Kunci yang dikirim dua kali akan berubah sekali, dan selisihnya akan terbaca sebagai "sudah
// diputuskan orang lain" — laporan yang salah tentang hal yang justru paling perlu dipercaya.
func TestKunciGandaDibuang(t *testing.T) {
	decision, err := inboxmanager.NewDecision(
		inboxmanager.TabKategoriSparepart, inboxmanager.VerdictApprove,
		[]string{"KAT-1", " KAT-1 ", "", "KAT-2"}, "",
		inboxmanager.Caller{Login: "JONNY"})

	require.NoError(t, err)
	require.Equal(t, []string{"KAT-1", "KAT-2"}, decision.Keys)
}

// TestKeputusanMenolakTabYangBukanAntrean menahan rute tulis dipanggil atas dashboard.
func TestKeputusanMenolakTabYangBukanAntrean(t *testing.T) {
	for _, code := range []string{
		inboxmanager.TabOutstanding, inboxmanager.TabProduktivitas,
		inboxmanager.TabKlaim, inboxmanager.TabApprovalMaster,
	} {
		_, err := inboxmanager.NewDecision(
			code, inboxmanager.VerdictApprove, []string{"X"}, "",
			inboxmanager.Caller{Login: "JONNY"})
		require.ErrorIs(t, err, inboxmanager.ErrQueueNotDecidable, "tab %s", code)
	}
}

// TestStaleMenghitungSelisihYangTidakBerubah mengunci angka yang dikirim ke layar.
func TestStaleMenghitungSelisihYangTidakBerubah(t *testing.T) {
	require.Equal(t, 0, inboxmanager.DecisionResult{Requested: 3, Changed: 3}.Stale())
	require.Equal(t, 2, inboxmanager.DecisionResult{Requested: 3, Changed: 1}.Stale())

	// Changed yang lebih besar daripada Requested tidak mungkin terjadi, dan bila terjadi ia
	// tidak boleh menghasilkan angka negatif di layar.
	require.Equal(t, 0, inboxmanager.DecisionResult{Requested: 1, Changed: 5}.Stale())
}

// TestPeriodeRentangMemakaiSelangSetengahTerbuka mengunci pembetulan batas atas.
//
// `TGLKLAIM` bertipe TIMESTAMP. Batas atas yang inklusif akan membuang seluruh baris pada hari
// terakhir yang berjam selain tengah malam — cacat yang tidak menghasilkan satu pun galat.
func TestPeriodeRentangMemakaiSelangSetengahTerbuka(t *testing.T) {
	query, err := inboxmanager.NewQuery(inboxmanager.QueryInput{
		Tab: inboxmanager.TabKlaim,
		Period: inboxmanager.PeriodInput{
			Mode:  inboxmanager.PeriodRange,
			From:  "2026-09-01",
			Until: "2026-09-30",
		},
	}, inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	require.Equal(t,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), query.Period.From)
	require.Equal(t,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), query.Period.Until,
		"batas atas digeser satu hari supaya eksklusif")

	require.Equal(t,
		time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), query.Period.PriorFrom)
	require.Equal(t,
		time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), query.Period.PriorUntil)
}

// TestPeriodeBulanMenjadiSatuBulanTakwim mengunci bentuk kedua penyaring periode.
func TestPeriodeBulanMenjadiSatuBulanTakwim(t *testing.T) {
	query, err := inboxmanager.NewQuery(inboxmanager.QueryInput{
		Tab:    inboxmanager.TabProduktivitas,
		Period: inboxmanager.PeriodInput{Mode: inboxmanager.PeriodMonth, Month: "2026-02"},
	}, inboxmanager.Caller{Login: "JONNY"})
	require.NoError(t, err)

	require.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), query.Period.From)
	require.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), query.Period.Until)

	// Pengurangan setahun memakai takwim, bukan 365 hari — 2024 kabisat.
	require.Equal(t, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), query.Period.PriorFrom)
}

// TestPeriodeSalahBentukDitolakDenganPesanYangTerbaca memastikan pengguna tahu yang harus
// diperbaiki.
func TestPeriodeSalahBentukDitolakDenganPesanYangTerbaca(t *testing.T) {
	_, err := inboxmanager.NewQuery(inboxmanager.QueryInput{
		Tab:    inboxmanager.TabKlaim,
		Period: inboxmanager.PeriodInput{Mode: inboxmanager.PeriodMonth, Month: "September"},
	}, inboxmanager.Caller{Login: "JONNY"})

	var validation *inboxmanager.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxmanager.FieldPeriod, validation.Violations[0].Field)
	require.Contains(t, validation.Violations[0].Message, "2026-09")
}

// TestPeriodeDiabaikanPadaTabTanpaPenyaringPeriode memastikan permintaan tidak gagal hanya
// karena membawa isian yang tidak dipakai.
func TestPeriodeDiabaikanPadaTabTanpaPenyaringPeriode(t *testing.T) {
	query, err := inboxmanager.NewQuery(inboxmanager.QueryInput{
		Tab:    inboxmanager.TabOutstanding,
		Period: inboxmanager.PeriodInput{Mode: inboxmanager.PeriodMonth, Month: "salah"},
	}, inboxmanager.Caller{Login: "JONNY"})

	require.NoError(t, err)
	require.True(t, query.Period.Empty())
}

// TestPencacahDiurutkanMengikutiUrutanKontainer menahan urutan di layar bergantung pada urutan
// selesainya kueri.
//
// Kode tab adalah ANGKA DALAM TEKS, sehingga pengurutan teks akan menaruh "10" sebelum "2".
func TestPencacahDiurutkanMengikutiUrutanKontainer(t *testing.T) {
	sorted := inboxmanager.SortCounters([]inboxmanager.Counter{
		{TabCode: inboxmanager.TabTipeSparepart},
		{TabCode: inboxmanager.TabOutstanding},
		{TabCode: inboxmanager.TabPenolakanKlaim},
		{TabCode: inboxmanager.TabMasterPanel},
	})

	require.Equal(t, []string{
		inboxmanager.TabOutstanding,
		inboxmanager.TabMasterPanel,
		inboxmanager.TabTipeSparepart,
		inboxmanager.TabPenolakanKlaim,
	}, []string{
		sorted[0].TabCode, sorted[1].TabCode, sorted[2].TabCode, sorted[3].TabCode,
	})
}

// TestPaginasiDibetulkanBukanDitolak memastikan salah ketik parameter tidak menggagalkan layar.
func TestPaginasiDibetulkanBukanDitolak(t *testing.T) {
	clean := inboxmanager.Pagination{Page: 0, Size: 0}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, inboxmanager.DefaultPageSize, clean.Size)

	capped := inboxmanager.Pagination{Page: 2, Size: 5000}.Normalize()
	require.Equal(t, inboxmanager.MaxPageSize, capped.Size)
}

// TestSliceQueueTidakPernahMengembalikanNil memastikan layar tidak perlu menangani dua bentuk
// "kosong".
func TestSliceQueueTidakPernahMengembalikanNil(t *testing.T) {
	page := inboxmanager.SliceQueue(nil, inboxmanager.Pagination{Page: 9, Size: 10})

	require.NotNil(t, page.Rows)
	require.Empty(t, page.Rows)
	require.Equal(t, 1, page.TotalPages(), "halaman minimal satu, bukan nol")
}

// TestSelisihTerencanaTidakKosong menahan daftar yang dikirim ke layar menjadi kosong.
//
// `D-54` menuntut setiap selisih pada uji kesetaraan gerbang 1 dapat dipetakan ke butir yang
// sudah diputuskan. Daftar kosong berarti tidak ada satu pun yang dapat dipetakan.
func TestSelisihTerencanaTidakKosong(t *testing.T) {
	require.NotEmpty(t, inboxmanager.PlannedDifferences)
	for i, difference := range inboxmanager.PlannedDifferences {
		require.NotEmptyf(t, difference, "selisih terencana ke-%d kosong", i)
	}
}
