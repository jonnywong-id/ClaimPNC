package inboxmanager_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

// Pilihan Status Approval mengikuti Prompt List `AcceptanceStatus` dari layar rule Pega.
func TestPaymentStatusOptionsSepertiPega(t *testing.T) {
	require.Equal(t, []inboxmanager.FilterOption{
		{Value: "0", Label: "Sedang Proses Komite"},
		{Value: "1", Label: "Setuju"},
		{Value: "2", Label: "Tidak Setuju"},
	}, inboxmanager.PaymentStatusOptions)

	require.Equal(t, "Tidak Setuju", inboxmanager.PaymentStatusLabel(" 2 "))
	require.Equal(t, "9", inboxmanager.PaymentStatusLabel("9"), "kode tak dikenal ditulis apa adanya")
}

func TestNewPaymentSave(t *testing.T) {
	caller := inboxmanager.Caller{Login: " mgr "}

	_, err := inboxmanager.NewPaymentSave("AKS-1", "2", "x", inboxmanager.Caller{})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	// Sedang Proses Komite boleh tanpa catatan.
	save, err := inboxmanager.NewPaymentSave("AKS-1", "0", "", caller)
	require.NoError(t, err)
	require.Equal(t, "mgr", save.Caller.Login)
	require.Equal(t, "1", save.AdjustmentFlag(), "FlagAtasan = \"1\" bila bukan Setuju")

	// Tidak Setuju menuntut Remark Approval; status tak dikenal dan nomor kosong ditandai
	// sekaligus.
	_, err = inboxmanager.NewPaymentSave(" ", "2", " ", caller)
	require.Equal(t, []inboxmanager.Violation{
		{Field: inboxmanager.FieldKeys, Message: "Nomor akseptasi wajib diisi."},
		{Field: inboxmanager.FieldReason, Message: "Alasan penolakan wajib diisi."},
	}, violationsOf(t, err))

	_, err = inboxmanager.NewPaymentSave("AKS-1", "setuju", "", caller)
	require.Equal(t, []inboxmanager.Violation{
		{Field: inboxmanager.FieldVerdict, Message: "Status Approval tidak dikenal."},
	}, violationsOf(t, err))
}

func TestPaymentAdjustmentFlagSetuju(t *testing.T) {
	// `@if(Param.stsapprove==1,"","1")` — SIMPAN Setuju sendiri masih ditahan, tetapi aturan
	// nilainya dikunci di sini supaya tidak berubah diam-diam saat penahanannya dicabut.
	require.Equal(t, "", inboxmanager.PaymentSave{Status: inboxmanager.PaymentAccepted}.AdjustmentFlag())
	require.Equal(t, "1", inboxmanager.PaymentSave{Status: inboxmanager.PaymentNotAccepted}.AdjustmentFlag())
}

func TestPaymentDocumentKey(t *testing.T) {
	require.Equal(t, "3", inboxmanager.PaymentDocumentKey(3))

	n, ok := inboxmanager.PaymentDocumentIndex(" 3 ")
	require.True(t, ok)
	require.Equal(t, 3, n)

	for _, bad := range []string{"", "0", "-1", "abc", "1.5"} {
		_, ok := inboxmanager.PaymentDocumentIndex(bad)
		require.Falsef(t, ok, "kunci %q", bad)
	}

	require.True(t, inboxmanager.PaymentDocument{ImageID: "IMG"}.Openable())
	require.True(t, inboxmanager.PaymentDocument{HasFile: true}.Openable())
	require.False(t, inboxmanager.PaymentDocument{ImageID: " "}.Openable())
}

// Subjek surel disalin dari activity (:2806).
func TestPaymentNoticeSubject(t *testing.T) {
	notice := inboxmanager.PaymentNotice{ClaimNo: "C-1", InsuredName: "PT Contoh", AcceptedNo: "A-9"}
	require.Equal(t, "Pengecekan akseptasi CLAIM C-1 | PT Contoh | No Aksep A-9", notice.Subject())
}
