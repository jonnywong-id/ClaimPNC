package notification_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/notification"
)

func notice() inboxsalvage.SubmissionNotice {
	return inboxsalvage.SubmissionNotice{
		SalvageID:    "77",
		ClaimNo:      "PNC-2044",
		InsuredName:  "PT Contoh Sejahtera",
		SalvageType:  "Barang Sisa",
		Location:     "Gudang Cakung",
		Currency:     "IDR",
		MinimumValue: "15000000",
		OfferValue:   "17500000",
		Remark:       "Terbakar sebagian",
		Submitter:    "ADMINPNC",
		ItemCount:    3,
	}
}

func TestHTMLBodyCarriesEveryFieldTheRecipientNeeds(t *testing.T) {
	body := notification.HTMLBody(notice())

	for _, want := range []string{
		"PNC-2044", "77", "PT Contoh Sejahtera", "Barang Sisa", "Gudang Cakung",
		"IDR 15000000", "IDR 17500000", "3 item", "Terbakar sebagian", "ADMINPNC",
	} {
		require.Contains(t, body, want)
	}
}

// Isian yang kosong menjadi TANDA HUBUNG, bukan sel kosong.
//
// Sel kosong terbaca seperti surel yang rusak; tanda hubung menyatakan datanya memang
// tidak ada.
func TestHTMLBodyMarksEmptyFieldsWithADash(t *testing.T) {
	body := notification.HTMLBody(inboxsalvage.SubmissionNotice{ClaimNo: "PNC-1"})
	require.Contains(t, body, "<td>-</td>")
}

// Pengajuan tanpa barang DIBEDAKAN dari jumlah yang tidak diketahui.
//
// Grid Detail Item Salvage boleh kosong, sehingga "0 item" adalah jawaban yang benar —
// bukan data yang hilang.
func TestHTMLBodyWritesZeroItemsAsANumber(t *testing.T) {
	body := notification.HTMLBody(inboxsalvage.SubmissionNotice{ClaimNo: "PNC-1"})
	require.Contains(t, body, "0 item")
}

// Nilai uang TANPA mata uang tetap ditulis, hanya tanpa awalannya.
func TestHTMLBodyWritesTheAmountWithoutACurrencyWhenThereIsNone(t *testing.T) {
	one := notice()
	one.Currency = ""

	body := notification.HTMLBody(one)
	require.Contains(t, body, "<td>15000000</td>")
}

// Seluruh nilai di-escape.
//
// Lokasi dan remark diketik orang lain; menempelkannya mentah berarti isi basis data dapat
// menyuntikkan markup ke dalam kotak masuk penerimanya.
func TestHTMLBodyEscapesWhatPeopleTyped(t *testing.T) {
	one := notice()
	one.Remark = `<script>alert("x")</script>`

	body := notification.HTMLBody(one)
	require.NotContains(t, body, "<script>")
	require.Contains(t, body, "&lt;script&gt;")
}

// Konfigurasi tanpa penerima BUKAN setengah aktif — ia tidak aktif.
//
// Menyatakannya aktif akan menyembunyikan konfigurasi yang belum selesai di balik
// pengiriman yang tidak pernah sampai ke siapa pun.
func TestConfigWithoutRecipientsIsNotComplete(t *testing.T) {
	cfg := notification.Config{Host: "smtp.internal", Port: 25, From: "noreply@contoh.id"}
	require.False(t, cfg.Complete())

	cfg.To = []string{"salvage@contoh.id"}
	require.True(t, cfg.Complete())

	// Alamat yang hanya berisi spasi tidak dihitung sebagai penerima.
	cfg.To = []string{"   "}
	require.False(t, cfg.Complete())
}

// Subjeknya dari rule lama, lewat domain — bukan disusun ulang di sini.
func TestSubjectComesFromTheDomain(t *testing.T) {
	require.Equal(t, "Pengajuan Salvage an PT Contoh Sejahtera", notice().Subject())
}

// Perekam adalah adapter KEDUA yang membuat seam Notifier nyata.
func TestRecorderKeepsEveryNoticeAndCountsEveryAttempt(t *testing.T) {
	recorder := notification.NewRecorder()

	require.NoError(t, recorder.NotifySalvageSubmitted(t.Context(), notice()))
	require.Len(t, recorder.Notices(), 1)
	require.Equal(t, 1, recorder.Attempts())

	// Percobaan yang GAGAL tetap dicacah: uji perlu dapat membedakan "dicoba lalu gagal"
	// dari "seam memang tidak dipasang", dan yang kedua tidak pernah sampai ke sini.
	recorder.SetError(errFailed)
	require.Error(t, recorder.NotifySalvageSubmitted(t.Context(), notice()))
	require.Len(t, recorder.Notices(), 1, "yang gagal tidak ikut terekam")
	require.Equal(t, 2, recorder.Attempts())
}

var errFailed = errSentinel("server surel mati")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
