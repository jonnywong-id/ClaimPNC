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

func TestUnavailableReasonAndAvailability(t *testing.T) {
	for _, tab := range []inboxsurvey.Tab{
		inboxsurvey.TabOutstanding, inboxsurvey.TabAll, inboxsurvey.TabInvoice,
	} {
		require.Contains(t, inboxsurvey.UnavailableReason(tab), "ADJUSTERACCEPT")
		require.False(t, tab.Available())
	}
	require.Contains(t, inboxsurvey.UnavailableReason(inboxsurvey.TabClose), "PYSTATUSWORK")
	require.False(t, inboxsurvey.TabClose.Available())

	for _, tab := range []inboxsurvey.Tab{
		inboxsurvey.TabNotAnswered, inboxsurvey.TabNotReplied, inboxsurvey.TabReplied,
	} {
		require.Empty(t, inboxsurvey.UnavailableReason(tab))
		require.True(t, tab.Available())
	}
}

func TestDefaultAvailableTabSkipsBlockedTabs(t *testing.T) {
	// Tab bawaan dan tiga tab sesudahnya terhalang; yang pertama tersedia adalah tab
	// komunikasi.
	require.Equal(t, inboxsurvey.TabNotAnswered, inboxsurvey.DefaultAvailableTab())
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

func TestKPIKindAndFilterNormalize(t *testing.T) {
	require.True(t, inboxsurvey.KPIFinal.Valid())
	require.True(t, inboxsurvey.KPIQuarterly.Valid())
	require.True(t, inboxsurvey.KPIOutstanding.Valid())
	require.False(t, inboxsurvey.KPIKind("x").Valid())

	require.Equal(t,
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIOutstanding, Category: "A", Year: "2026"},
		inboxsurvey.KPIFilter{Kind: "x", Category: " A ", Year: " 2026 "}.Normalize())
	require.Equal(t, inboxsurvey.KPIFinal,
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIFinal}.Normalize().Kind)
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
