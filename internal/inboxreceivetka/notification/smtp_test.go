package notification_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/inboxreceivetka/notification"
)

func sampleNotice() inboxreceivetka.Notice {
	dateOfLoss := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	return inboxreceivetka.Notice{
		ClaimNumber:     "PNC-200118",
		PolicyNumber:    "00.000.2026.00118",
		InsuredName:     "PT Contoh Karya Mandiri",
		ParticipantName: "Peserta Contoh Dua",
		DateOfLoss:      &dateOfLoss,
		CompletedAt:     time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
	}
}

// Subjeknya ditiru dari activity lama apa adanya, supaya penerima yang sudah menyaring
// kotak masuknya berdasarkan subjek tidak perlu mengubah apa pun.
func TestSubjectFollowsTheOldActivity(t *testing.T) {
	require.Equal(t,
		"Notifikasi Kelengkapan Dokumen Klaim TKA / PNC-200118",
		notification.Subject(sampleNotice()))
}

// Keenam baris badan surel dari `HTML/NotificationKelengkapanTKA-HTML.xml`, pada urutan
// aslinya.
//
// Urutannya diuji, bukan hanya keberadaannya: label yang benar berpasangan dengan nilai yang
// salah adalah kelas cacat yang paling mungkin terjadi di sini, karena rule aslinya memang
// menampung nama orang di properti bernama `District` dan tanggal di properti bernama
// `Country`.
func TestHTMLBodyKeepsTheSixRowsInOrder(t *testing.T) {
	body := notification.HTMLBody(sampleNotice())

	expected := []string{
		"No Klaim", "PNC-200118",
		"No Polis", "00.000.2026.00118",
		"Nama Tertanggung", "PT Contoh Karya Mandiri",
		"Nama Peserta", "Peserta Contoh Dua",
		"Date Of Loss", "14/08/2026",
		"Tanggal Kelengkapan Dokumen", "24/09/2026",
	}

	at := 0
	for _, fragment := range expected {
		index := strings.Index(body[at:], fragment)
		require.GreaterOrEqualf(t, index, 0,
			"badan surel tidak memuat %q sesudah bagian sebelumnya; badan: %s", fragment, body)
		at += index + len(fragment)
	}
}

// Tanggalnya dd/mm/yyyy, bentuk yang dibaca penerima di Indonesia — dan bentuk yang sama
// dengan yang dirangkai activity lama lewat pemotongan teks `yyyymmdd`.
//
// Ia sengaja BERBEDA dari ISO 8601 pada kontrak API, yang pembacanya mesin.
func TestHTMLBodyWritesDatesInIndonesianForm(t *testing.T) {
	body := notification.HTMLBody(sampleNotice())
	require.Contains(t, body, "14/08/2026")
	require.NotContains(t, body, "2026-08-14")
}

// Tanggal yang kosong ditulis sebagai tanda hubung, bukan sel kosong maupun "01/01/0001".
func TestHTMLBodyWritesDashForMissingDate(t *testing.T) {
	notice := sampleNotice()
	notice.DateOfLoss = nil
	notice.ParticipantName = ""

	body := notification.HTMLBody(notice)
	require.NotContains(t, body, "0001")
	require.Equal(t, 2, strings.Count(body, "<td>-</td>"),
		"Date Of Loss dan Nama Peserta yang kosong keduanya ditulis sebagai tanda hubung")
}

// Nama tertanggung dan nama peserta adalah teks yang dimasukkan orang lain. Menempelkannya
// mentah ke dalam HTML berarti isi basis data dapat menyuntikkan markup ke dalam kotak masuk
// penerimanya.
func TestHTMLBodyEscapesValues(t *testing.T) {
	notice := sampleNotice()
	notice.InsuredName = `PT <script>alert("x")</script> & Rekan`

	body := notification.HTMLBody(notice)
	require.NotContains(t, body, "<script>")
	require.Contains(t, body, "&lt;script&gt;")
	require.Contains(t, body, "&amp; Rekan")
}

// Konfigurasi yang belum lengkap DITOLAK sebelum satu sambungan pun dibuka.
//
// Ia bukan kegagalan yang membatalkan pekerjaan — usecase memperlakukannya sebagai
// pemberitahuan yang gagal, dan tanggalnya tetap tersimpan.
func TestSenderRejectsIncompleteConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  notification.Config
	}{
		{name: "tanpa host", cfg: notification.Config{Port: 25, From: "a@b", To: []string{"c@d"}}},
		{name: "tanpa port", cfg: notification.Config{Host: "surel", From: "a@b", To: []string{"c@d"}}},
		{name: "tanpa pengirim", cfg: notification.Config{Host: "surel", Port: 25, To: []string{"c@d"}}},
		{name: "tanpa penerima", cfg: notification.Config{Host: "surel", Port: 25, From: "a@b"}},
		{
			name: "penerima hanya spasi",
			cfg:  notification.Config{Host: "surel", Port: 25, From: "a@b", To: []string{"  "}},
		},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			require.False(t, one.cfg.Complete())

			err := notification.NewSender(one.cfg).
				NotifyDocumentCompleted(t.Context(), sampleNotice())
			require.Error(t, err)
		})
	}
}

func TestRecorderCountsAttemptsEvenWhenItFails(t *testing.T) {
	recorder := notification.NewRecorder()
	require.NoError(t, recorder.NotifyDocumentCompleted(t.Context(), sampleNotice()))
	require.Len(t, recorder.Notices(), 1)
	require.Equal(t, 1, recorder.Attempts())

	recorder.SetError(errors.New("server surel tidak dapat dihubungi"))
	require.Error(t, recorder.NotifyDocumentCompleted(t.Context(), sampleNotice()))

	require.Len(t, recorder.Notices(), 1, "yang gagal tidak ikut direkam")
	require.Equal(t, 2, recorder.Attempts(), "tetapi percobaannya tetap tercacah")
}
