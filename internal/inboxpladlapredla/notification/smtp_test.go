package notification_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla/notification"
)

// Host, port, dan alamat pengirim CUKUP untuk mengaktifkan pengiriman.
//
// Uji ini menjaga satu kekeliruan yang sudah pernah terjadi: gerbang aktifnya sempat
// memakai `cfg.SMTP.Active()`, yang mensyaratkan daftar penerima peringatan Tim IT.
// Modul ini tidak memakai daftar itu — penerimanya adalah reasuradur pada baris dokumen —
// sehingga tombol "SEND" tetap menolak meski konfigurasinya sudah lengkap.
//
// Penolakan seperti itu tidak dapat ditemukan sebabnya dari membaca pesannya.
func TestHostPortAndSenderAreEnoughToSend(t *testing.T) {
	cfg := notification.Config{
		Host: "relay.contoh.example",
		Port: 587,
		From: "auto@contoh.example",
	}

	require.True(t, cfg.Complete())
	require.Empty(t, cfg.Missing())
}

// Konfigurasi yang belum lengkap MENYEBUT isian mana yang kurang.
//
// Pesan "belum tersedia" tanpa keterangan membuat administrator menebak; pesan yang
// menyebut namanya dapat langsung ditindaklanjuti.
func TestIncompleteConfigNamesWhatIsMissing(t *testing.T) {
	kosong := notification.Config{}

	require.False(t, kosong.Complete())
	require.ElementsMatch(t,
		[]string{"host SMTP", "port SMTP", "alamat pengirim"}, kosong.Missing())

	sebagian := notification.Config{Host: "relay.contoh.example", From: "a@b.example"}
	require.False(t, sebagian.Complete())
	require.Equal(t, []string{"port SMTP"}, sebagian.Missing())
}

// Kredensial TIDAK menjadi syarat aktif.
//
// Relay internal sering menerima pengirim dari jaringan tepercaya tanpa autentikasi.
// Mensyaratkannya akan menolak konfigurasi yang sah.
func TestCredentialsAreNotRequired(t *testing.T) {
	cfg := notification.Config{
		Host: "relay.contoh.example",
		Port: 25,
		From: "auto@contoh.example",
	}

	require.True(t, cfg.Complete())
}
