package notification

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

func TestComposeEmail(t *testing.T) {
	notice := inboxmanager.PaymentNotice{
		To: "pic@contoh.test", ClaimNo: "C-1", InsuredName: "PT <Contoh>", AcceptedNo: "A-9",
		Status: inboxmanager.PaymentNotAccepted, Remark: "<script>x</script>", DecidedBy: "MGR",
	}
	message := string(composeEmail("pengirim@contoh.test", "pic@contoh.test\r\nBcc: x@contoh.test", notice))

	require.Contains(t, message, "To: pic@contoh.test\r\n")
	require.NotContains(t, message, "Bcc:", "baris baru di alamat tidak boleh menyisipkan header")
	require.Contains(t, message, "Subject: Pengecekan akseptasi CLAIM C-1 | PT <Contoh> | No Aksep A-9\r\n")

	body := message[strings.Index(message, "\r\n\r\n"):]
	require.Contains(t, body, "Tidak Setuju")
	require.Contains(t, body, "&lt;script&gt;x&lt;/script&gt;")
	require.NotContains(t, body, "<script>")
	require.Contains(t, body, "PT &lt;Contoh&gt;")
}

func TestNotifyTanpaPenerima(t *testing.T) {
	sender := NewSender(Config{Host: "smtp.contoh.test", Port: 25, From: "a@contoh.test"})
	err := sender.NotifyPaymentChecked(context.Background(), inboxmanager.PaymentNotice{To: " "})
	require.ErrorIs(t, err, ErrNoRecipient)
}

func TestConfigComplete(t *testing.T) {
	require.False(t, Config{}.Complete())
	require.True(t, Config{Host: "smtp.contoh.test", Port: 25, From: "a@contoh.test"}.Complete())
}
