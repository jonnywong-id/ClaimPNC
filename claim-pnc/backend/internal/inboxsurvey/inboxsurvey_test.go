package inboxsurvey_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsurvey"
)

func TestClosedWorkStatuses(t *testing.T) {
	require.Equal(t,
		[]string{inboxsurvey.StatusWorkCompleted, inboxsurvey.StatusWorkRejected},
		inboxsurvey.ClosedWorkStatuses())
}

func TestTabsReturnsACopyInOrder(t *testing.T) {
	tabs := inboxsurvey.Tabs()
	require.Equal(t, []inboxsurvey.Tab{
		inboxsurvey.TabOutstanding, inboxsurvey.TabInvoice, inboxsurvey.TabClose,
		inboxsurvey.TabAll, inboxsurvey.TabNotAnswered, inboxsurvey.TabNotReplied,
		inboxsurvey.TabReplied,
	}, tabs)

	tabs[0] = "rusak"
	require.Equal(t, inboxsurvey.TabOutstanding, inboxsurvey.Tabs()[0])
}

// siap adalah kesiapan portal yang kelima kolomnya sudah ada DAN terisi.
func siap() inboxsurvey.Readiness {
	return inboxsurvey.Readiness{
		AdjusterAccept: true, WorkStatus: true, Reference: true,
		AdjusterPIC: true, SurveyLocation: true,
	}
}

func TestUnavailableReasonAndAvailability(t *testing.T) {
	// Portal yang belum siap — keadaan produksi per 2026-10-07.
	var belum inboxsurvey.Readiness

	for _, tab := range []inboxsurvey.Tab{
		inboxsurvey.TabOutstanding, inboxsurvey.TabAll,
		inboxsurvey.TabInvoice, inboxsurvey.TabClose,
	} {
		require.Contains(t, belum.UnavailableReason(tab), "ADJUSTERACCEPT")
		require.Contains(t, belum.UnavailableReason(tab), "RESCHEDULE_LOCATION")
		require.False(t, belum.TabAvailable(tab))
	}

	// Ketiga tab komunikasi tidak pernah ditahan — ia tidak menyentuh satu pun kolom itu.
	for _, tab := range []inboxsurvey.Tab{
		inboxsurvey.TabNotAnswered, inboxsurvey.TabNotReplied, inboxsurvey.TabReplied,
	} {
		require.Empty(t, belum.UnavailableReason(tab))
		require.True(t, belum.TabAvailable(tab))
	}
}

// TestPortalSiapMembukaKetujuhTab adalah separuh yang hilang sampai 2026-10-07.
//
// Sebelum itu ketersediaan tab adalah KONSTANTA, sehingga keadaan "kolomnya sudah terisi" tidak
// dapat diuji sama sekali — dan ketika `pega_dev83` benar-benar terisi, layar tetap menyatakan
// seluruh barisnya kosong. Tidak ada uji yang gagal, karena tidak ada uji yang dapat gagal.
func TestPortalSiapMembukaKetujuhTab(t *testing.T) {
	ready := siap()

	for _, tab := range inboxsurvey.Tabs() {
		require.Truef(t, ready.TabAvailable(tab), "tab %s", tab)
		require.Emptyf(t, ready.UnavailableReason(tab), "tab %s", tab)
	}
	require.True(t, ready.Complete())
}

// TestSebagianSiapTetapMenahanSeluruhnya mengunci sifat biner Complete.
//
// Kueri varian penuh menyebut KELIMA kolom sekaligus. Satu saja yang belum ada menghasilkan
// ORA-00904 saat parse — sebelum satu baris pun dibaca. Jadi "sebagian siap" harus diperlakukan
// sama dengan "belum siap", bukan dihidupkan sebagian.
func TestSebagianSiapTetapMenahanSeluruhnya(t *testing.T) {
	ready := siap()
	ready.Reference = false

	require.False(t, ready.Complete())
	require.False(t, ready.TabAvailable(inboxsurvey.TabOutstanding))
	require.Contains(t, ready.UnavailableReason(inboxsurvey.TabOutstanding), "REFNO")
	require.NotContains(t, ready.UnavailableReason(inboxsurvey.TabOutstanding), "ADJUSTERACCEPT")
}

func TestDefaultAvailableTabSkipsBlockedTabs(t *testing.T) {
	var belum inboxsurvey.Readiness

	// Tab bawaan dan tiga tab sesudahnya terhalang; yang pertama tersedia adalah tab
	// komunikasi.
	require.Equal(t, inboxsurvey.TabNotAnswered, belum.DefaultAvailableTab())

	// Begitu portalnya siap, tab bawaan kembali menjadi Outstanding — tanpa satu baris pun
	// disunting, yang sebelum 2026-10-07 tidak benar.
	require.Equal(t, inboxsurvey.DefaultTab, siap().DefaultAvailableTab())
}

func TestTabValid(t *testing.T) {
	require.True(t, inboxsurvey.TabReplied.Valid())
	require.False(t, inboxsurvey.Tab("lain").Valid())
}

func TestAgingDays(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	now := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC) // 08:00 WIB tanggal 30

	require.Nil(t, inboxsurvey.SurveyTask{}.AgingDays(now, wib))

	// 2026-09-28 18:00 UTC = 2026-09-29 01:00 WIB — satu hari, bukan dua.
	task := inboxsurvey.SurveyTask{CreatedAt: time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC)}
	require.Equal(t, 1, *task.AgingDays(now, wib))

	// Tanpa lokasi, perhitungannya UTC: 28 → 30 = dua hari.
	require.Equal(t, 2, *task.AgingDays(now, nil))

	// Tanggal masuk di masa depan dibulatkan menjadi nol, bukan negatif.
	future := inboxsurvey.SurveyTask{CreatedAt: now.Add(72 * time.Hour)}
	require.Equal(t, 0, *future.AgingDays(now, wib))
}

func TestFilterNormalize(t *testing.T) {
	require.Equal(t, inboxsurvey.Filter{
		Tab: inboxsurvey.DefaultTab, Search: "PNC", Limit: inboxsurvey.DefaultLimit,
	}, inboxsurvey.Filter{Tab: "x", Search: "  PNC ", Limit: -1, Offset: -5}.Normalize())

	require.Equal(t, inboxsurvey.Filter{
		Tab: inboxsurvey.TabReplied, Limit: inboxsurvey.MaxLimit, Offset: 30,
	}, inboxsurvey.Filter{Tab: inboxsurvey.TabReplied, Limit: 999, Offset: 30}.Normalize())
}

// TestPilihanPanelKPISamaDenganPega mengunci isi kedua dropdown.
//
// Nilainya diambil `Activity/GetFilterKPI-Act.xml` apa adanya — `TipeData` dan `TipeExport` —
// dan dikirim ke layar tanpa diterjemahkan, karena nilai yang sama itulah yang dibandingkan
// kueri terhadap kolom `tipe`.
func TestPilihanPanelKPISamaDenganPega(t *testing.T) {
	require.Equal(t, []inboxsurvey.SurveyStatus{
		inboxsurvey.SurveyStatusAll,
		inboxsurvey.SurveyStatusOutstanding,
		inboxsurvey.SurveyStatusFinal,
	}, inboxsurvey.SurveyStatuses())

	require.Equal(t, []inboxsurvey.ReportType{
		inboxsurvey.ReportSummary,
		inboxsurvey.ReportDetail,
	}, inboxsurvey.ReportTypes())

	require.Equal(t, []string{"1", "2", "3", "4"}, inboxsurvey.Quarters())
}

// TestKeduaIsianPanelKPIWajib menjaga keduanya tidak pernah dipilihkan untuk pengguna.
//
// Layar lama menandai Status Survey dan Tipe Report dengan bintang merah. Memilihkan salah
// satunya berarti menjalankan laporan yang tidak diminta siapa pun, lalu menampilkan angkanya
// seolah itu yang dicari — dan angka yang masuk akal tidak pernah dilaporkan sebagai salah.
func TestKeduaIsianPanelKPIWajib(t *testing.T) {
	err := inboxsurvey.KPIFilter{}.Check()
	require.ErrorIs(t, err, inboxsurvey.ErrKPIFilterIncomplete)
	require.Contains(t, err.Error(), "Status Survey")
	require.Contains(t, err.Error(), "Tipe Report")

	// Kesalahan dikumpulkan SELURUHNYA, bukan satu per satu — pengguna tidak perlu menekan
	// Cari dua kali untuk mengetahui dua hal.
	err = inboxsurvey.KPIFilter{Status: inboxsurvey.SurveyStatusAll}.Check()
	require.ErrorIs(t, err, inboxsurvey.ErrKPIFilterIncomplete)
	require.NotContains(t, err.Error(), "Status Survey")

	require.NoError(t, inboxsurvey.KPIFilter{
		Status: inboxsurvey.SurveyStatusAll,
		Report: inboxsurvey.ReportSummary,
	}.Check())
}

// TestBentukHasilMengikutiKombinasiIsian mengunci percabangan `GetReportKPIAdjuster`.
//
// Kelima bentuk berasal dari rule yang berbeda, dan setiap baris artinya berbeda pula. Satu
// cabang yang salah menghasilkan tabel yang terlihat benar dan menjawab pertanyaan lain.
func TestBentukHasilMengikutiKombinasiIsian(t *testing.T) {
	bentuk := func(status inboxsurvey.SurveyStatus, tipe inboxsurvey.ReportType, kuartal string) inboxsurvey.KPIShape {
		return inboxsurvey.KPIFilter{Status: status, Report: tipe, Quarter: kuartal}.Shape()
	}

	require.Equal(t, inboxsurvey.ShapePerAdjuster,
		bentuk(inboxsurvey.SurveyStatusFinal, inboxsurvey.ReportSummary, ""))

	// Status ALL BUKAN "tanpa penyaring": Pega menggabungkan dua blok dan memberi label
	// kategorinya, sehingga satu adjuster muncul DUA kali dengan angka masing-masing.
	require.Equal(t, inboxsurvey.ShapePerAdjusterStatus,
		bentuk(inboxsurvey.SurveyStatusAll, inboxsurvey.ReportSummary, ""))

	require.Equal(t, inboxsurvey.ShapePerYear,
		bentuk(inboxsurvey.SurveyStatusFinal, inboxsurvey.ReportSummary, "3"))
	require.Equal(t, inboxsurvey.ShapePerQuarterYear,
		bentuk(inboxsurvey.SurveyStatusFinal, inboxsurvey.ReportSummary, inboxsurvey.QuarterAll))

	// DATA DETAIL mengalahkan seluruh kombinasi lain — ia rule tersendiri.
	for _, kuartal := range []string{"", "2", inboxsurvey.QuarterAll} {
		require.Equal(t, inboxsurvey.ShapeDetail,
			bentuk(inboxsurvey.SurveyStatusAll, inboxsurvey.ReportDetail, kuartal))
	}
}

// TestKuartalALLBerbedaDariKuartalKosong menutup salah paham yang mudah terjadi.
//
// Keduanya sama-sama "bukan satu kuartal", tetapi menempuh rule yang BERBEDA: kosong berarti
// kendalinya tidak dipakai sama sekali, "ALL" berarti keempat kuartal diminta sekaligus.
func TestKuartalALLBerbedaDariKuartalKosong(t *testing.T) {
	require.False(t, inboxsurvey.KPIFilter{Quarter: inboxsurvey.QuarterAll}.QuarterChosen())
	require.False(t, inboxsurvey.KPIFilter{}.QuarterChosen())
	require.True(t, inboxsurvey.KPIFilter{Quarter: "4"}.QuarterChosen())
}

// TestKendaliKuartalMengikutiStatusSurvey meniru `pyVisibleWhen` panel KPI apa adanya.
func TestKendaliKuartalMengikutiStatusSurvey(t *testing.T) {
	require.True(t, inboxsurvey.SurveyStatusAll.QuarterApplies())
	require.True(t, inboxsurvey.SurveyStatusFinal.QuarterApplies())
	require.False(t, inboxsurvey.SurveyStatusOutstanding.QuarterApplies())

	// Nilai yang tertinggal dari pilihan sebelumnya dibuang bersama kendalinya. Tanpa ini,
	// hasilnya tersaring kuartal yang tidak terlihat di mana pun pada layar.
	clean := inboxsurvey.KPIFilter{
		Status:  inboxsurvey.SurveyStatusOutstanding,
		Report:  inboxsurvey.ReportSummary,
		Quarter: "2",
		Year:    "2026",
	}.Normalize()
	require.Empty(t, clean.Quarter)
	require.Empty(t, clean.Year)
	require.Equal(t, inboxsurvey.ShapePerAdjuster, clean.Shape())
}

// TestStatusALLMenempuhKueriTersendiri menutup salah paham yang sempat saya bangun sendiri.
//
// "ALL" adalah pilihan di layar, BUKAN nilai yang pernah ada di kolom `tipe` — dan ia juga
// BUKAN "tanpa penyaring". Pega menjalankan kueri lain yang menggabungkan dua blok dan memberi
// label kategorinya masing-masing.
//
// Bedanya nyata: "tanpa penyaring" merata-ratakan kedua kategori menjadi SATU angka per
// adjuster; Pega memberi DUA baris dengan angka masing-masing. Angka pertama tidak pernah ada
// di layar lama, dan ia terlihat sangat masuk akal — itu yang membuatnya berbahaya.
func TestStatusALLMenempuhKueriTersendiri(t *testing.T) {
	require.Equal(t, inboxsurvey.ShapePerAdjusterStatus, inboxsurvey.KPIFilter{
		Status: inboxsurvey.SurveyStatusAll, Report: inboxsurvey.ReportSummary,
	}.Shape())

	require.Empty(t, inboxsurvey.KPIFilter{Status: inboxsurvey.SurveyStatusAll}.CategoryValue())
	require.Equal(t, "FINAL",
		inboxsurvey.KPIFilter{Status: inboxsurvey.SurveyStatusFinal}.CategoryValue())
	require.Equal(t, "OUTSTANDING",
		inboxsurvey.KPIFilter{Status: inboxsurvey.SurveyStatusOutstanding}.CategoryValue())
}

// TestAppointmentNoMemotongPrefixKelasPega mengunci kolom "Appointment No".
//
// # Kenapa uji ini ada
//
// Kolom ini sempat dinyatakan menunggu `ADJUSTER_PIC` dari Tim Pega selama berhari-hari, atas
// dugaan bahwa ia menggambar nama adjuster. Yang mematahkannya adalah Work Owner yang melihat
// layar Pega berjalan: isinya `SRV-xxxxx`.
//
// Nomor itu ternyata sudah di tangan sejak awal — `CASEID` dikurangi prefix kelas Pega — dan
// Pega sendiri menghitungnya persis begitu (`SetTempLostAdjuster-Act.xml:6197`). Uji ini
// mengunci kesetaraan itu supaya kolomnya tidak pernah lagi dinyatakan menunggu siapa pun.
func TestAppointmentNoMemotongPrefixKelasPega(t *testing.T) {
	cases := []struct {
		nama     string
		surveyID string
		mau      string
	}{
		{"kunci utuh berprefix", "ASM-FW-GCNMFW-WORK SRV-12345", "SRV-12345"},
		{"sudah tanpa prefix", "SRV-12345", "SRV-12345"},
		{"berspasi di ujung", "  ASM-FW-GCNMFW-WORK SRV-7  ", "SRV-7"},
		{"kosong", "", ""},

		// Nilai yang TIDAK berbentuk seperti dugaan dikembalikan apa adanya, bukan dipotong
		// membabi buta pada posisi 19. Di sinilah TrimPrefix berbeda dari @substring(.,19,30),
		// dan perbedaannya justru muncul pada data yang menyimpang.
		{"bentuk lain tidak dipotong", "SRV-INI-PANJANG-SEKALI-SEKALI", "SRV-INI-PANJANG-SEKALI-SEKALI"},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			task := inboxsurvey.SurveyTask{SurveyID: c.surveyID}
			require.Equal(t, c.mau, task.AppointmentNo())
		})
	}
}

// TestAppointmentNoSepadanDenganPemotonganPega menjaga satu angka yang mudah dikira sepele.
//
// `SetTempLostAdjuster-Act.xml:6197` memotong pada posisi 19. Angka itu hanya benar selama
// prefiksnya tepat 19 karakter — dan bila keduanya berbeda, nomor survei terpotong di tengah
// tanpa galat apa pun.
//
// Diuji lewat PERILAKU, bukan lewat panjang konstantanya: yang harus sepadan adalah hasilnya,
// dan itu yang dilihat pengguna.
func TestAppointmentNoSepadanDenganPemotonganPega(t *testing.T) {
	const kunci = "ASM-FW-GCNMFW-WORK SRV-98765"

	task := inboxsurvey.SurveyTask{SurveyID: kunci}
	require.Equal(t, kunci[19:], task.AppointmentNo(),
		"Pega memotong @substring(.CaseID,19,30); hasil di sini harus sama")
}
