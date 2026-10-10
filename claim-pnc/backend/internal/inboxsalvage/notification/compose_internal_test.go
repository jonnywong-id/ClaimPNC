package notification

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Penerima ganda disatukan, dan yang kosong dibuang.
//
// Isian "Email" pada form Tambah sering diisi alamat yang SUDAH ada di daftar tetap.
// Mengirimnya dua kali membuat penerima menerima surel ganda untuk satu pengajuan, dan itu
// terbaca seperti sistem yang mengirim berulang — persis hal yang membuat orang berhenti
// membaca surel otomatis.
func TestCleanAddressesDropsBlanksAndDuplicates(t *testing.T) {
	got := cleanAddresses([]string{
		"salvage@contoh.id",
		"  ",
		"SALVAGE@contoh.id",
		" pic@contoh.id ",
		"",
	})

	require.Equal(t, []string{"salvage@contoh.id", "pic@contoh.id"}, got)
}

// Kepala surel tidak boleh dapat disuntik baris baru.
//
// Tanpa penyaringan ini, satu nilai yang memuat CR atau LF dapat menambahkan penerima
// tersembunyi atau mengganti subjeknya — dan di modul ini nilai itu berasal dari nama
// tertanggung dan dari isian "Email" yang diketik petugas.
//
// # Yang diperiksa adalah JUMLAH BARIS KEPALA, bukan ada-tidaknya kata "Bcc:"
//
// Teks `Bcc:` yang diketik orang memang tetap muncul — sebagai bagian dari NILAI header
// To, bukan sebagai header baru. Memeriksa kata itu akan lulus karena alasan yang salah,
// dan gagal pada nilai sah yang kebetulan memuatnya. Yang menentukan adalah apakah ada
// CRLF yang menyelinap ke dalam kepala surel.
func TestComposeEmailCannotBeInjectedWithANewHeader(t *testing.T) {
	message := string(composeEmail(
		"noreply@contoh.id",
		[]string{"salvage@contoh.id\r\nBcc: diam-diam@contoh.id"},
		inboxsalvage.SubmissionNotice{
			ClaimNo:     "PNC-1",
			InsuredName: "PT Contoh\r\nSubject: diganti",
		},
	))

	head, _, found := strings.Cut(message, "\r\n\r\n")
	require.True(t, found, "kepala dan badan surel dipisahkan baris kosong")

	lines := strings.Split(head, "\r\n")
	require.Len(t, lines, 5, "From, To, Subject, MIME-Version, Content-Type — tidak lebih")

	require.True(t, strings.HasPrefix(lines[0], "From: "))
	require.True(t, strings.HasPrefix(lines[1], "To: "))
	require.True(t, strings.HasPrefix(lines[2], "Subject: "))
	require.Equal(t, "MIME-Version: 1.0", lines[3])
	require.Equal(t, "Content-Type: text/html; charset=UTF-8", lines[4])
}

// CC dan BCC TIDAK ditulis, mengikuti pemanggilan lamanya yang mengosongkan keduanya
// (`SetStsSalvagePNC_act:10200`, `:10215`).
func TestComposeEmailWritesNoCCOrBCC(t *testing.T) {
	message := string(composeEmail(
		"noreply@contoh.id",
		[]string{"salvage@contoh.id"},
		inboxsalvage.SubmissionNotice{ClaimNo: "PNC-1"},
	))

	head, _, _ := strings.Cut(message, "\r\n\r\n")
	require.NotContains(t, head, "Cc:")
	require.NotContains(t, head, "Bcc:")
	require.Contains(t, head, "Content-Type: text/html; charset=UTF-8")
}
