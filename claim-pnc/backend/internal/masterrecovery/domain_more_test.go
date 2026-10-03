package masterrecovery_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
)

// failingReader selalu gagal dibaca.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("putus") }

func TestParseClaimLineRejectsUnreadableSource(t *testing.T) {
	_, _, err := masterrecovery.ParseClaimLine(failingReader{})
	require.ErrorIs(t, err, masterrecovery.ErrClaimLineUnreadable)
}

func TestParseClaimLineRejectsOversizedFile(t *testing.T) {
	big := strings.Repeat("a", masterrecovery.MaxDocumentBytes+1)
	_, _, err := masterrecovery.ParseClaimLine(strings.NewReader(big))
	require.ErrorIs(t, err, masterrecovery.ErrClaimLineTooMany)
}

func TestParseClaimLineRejectsMalformedCSV(t *testing.T) {
	// Kutip yang tidak ditutup membuat pembaca CSV gagal.
	_, _, err := masterrecovery.ParseClaimLine(strings.NewReader("\"POL-1,100\n"))
	require.ErrorIs(t, err, masterrecovery.ErrClaimLineUnreadable)
}

func TestParseClaimLineReportsLongPolicyAndNegativeAmount(t *testing.T) {
	long := strings.Repeat("P", masterrecovery.MaxPolicyNoLength+1)
	line, violation, err := masterrecovery.ParseClaimLine(strings.NewReader(
		"nomor polis,nilai klaim\n" + long + ",10\nPOL-2,-5\nPOL-3,7\n"))
	require.NoError(t, err)
	require.Equal(t, []masterrecovery.ClaimLine{{PolicyNo: "POL-3", ClaimAmount: 7}}, line)
	require.Equal(t, []masterrecovery.Violation{
		{Field: masterrecovery.FieldClaimLine, Message: "Baris 2: nomor polis melebihi 200 karakter."},
		{Field: masterrecovery.FieldClaimLine, Message: "Baris 3: nilai klaim tidak boleh negatif."},
	}, violation)
}

func TestParseClaimLineRejectsTooManyRows(t *testing.T) {
	var b strings.Builder
	for i := 0; i <= masterrecovery.MaxClaimLineRows; i++ {
		b.WriteString("P,1\n")
	}
	_, _, err := masterrecovery.ParseClaimLine(strings.NewReader(b.String()))
	require.ErrorIs(t, err, masterrecovery.ErrClaimLineTooMany)
}

func TestValidationErrorJoinsEveryViolation(t *testing.T) {
	require.NoError(t, masterrecovery.NewValidationError(nil))

	err := masterrecovery.NewValidationError([]masterrecovery.Violation{
		{Field: "a", Message: "satu"}, {Field: "b", Message: "dua"},
	})
	require.EqualError(t, err, "masterrecovery: validasi gagal — a: satu; b: dua")
}

func TestCheckRecoveryLengthLimitsAndClaimLines(t *testing.T) {
	violation := masterrecovery.CheckRecovery(masterrecovery.Recovery{
		PrincipalName:        strings.Repeat("N", masterrecovery.MaxPrincipalNameLength+1),
		Year:                 "2026",
		Remark:               strings.Repeat("R", masterrecovery.MaxRemarkLength+1),
		CasePosition:         strings.Repeat("C", masterrecovery.MaxCasePositionLength+1),
		ClientID:             strings.Repeat("I", masterrecovery.MaxClientIDLength+1),
		VirtualAccountNumber: strings.Repeat("V", masterrecovery.MaxVirtualAccountLength+1),
		PolicyNo:             strings.Repeat("P", masterrecovery.MaxPolicyNoLength+1),
		ClaimLine:            []masterrecovery.ClaimLine{{PolicyNo: " ", ClaimAmount: -1}},
	})

	fields := make([]string, 0, len(violation))
	for _, v := range violation {
		fields = append(fields, v.Field)
	}
	require.Equal(t, []string{
		masterrecovery.FieldPrincipalName, masterrecovery.FieldRemark, masterrecovery.FieldCasePosition,
		masterrecovery.FieldClientID, masterrecovery.FieldVirtualAccount, masterrecovery.FieldPolicyNo,
		masterrecovery.FieldClaimLine, masterrecovery.FieldClaimLine,
	}, fields)
	require.Equal(t, "Nama principal paling panjang 200 karakter.", violation[0].Message)
	require.Equal(t, "Baris 1 pada daftar klaim tidak menyebut nomor polis.", violation[6].Message)
	require.Equal(t, "Baris 1 pada daftar klaim bernilai negatif.", violation[7].Message)
}

func TestCheckDocumentLimits(t *testing.T) {
	violation := masterrecovery.CheckDocument(masterrecovery.Document{
		Name:     strings.Repeat("n", masterrecovery.MaxDocumentNameLength+1),
		Note:     strings.Repeat("k", masterrecovery.MaxDocumentNoteLength+1),
		MimeType: strings.Repeat("m", masterrecovery.MaxMimeTypeLength+1),
		Content:  make([]byte, masterrecovery.MaxDocumentBytes+1),
	})
	var message []string
	for _, v := range violation {
		require.Equal(t, masterrecovery.FieldDocument, v.Field)
		message = append(message, v.Message)
	}
	require.Equal(t, []string{
		"Berkas bukti bayar paling besar 5 MB.",
		"Nama berkas paling panjang 255 karakter.",
		"Keterangan berkas paling panjang 255 karakter.",
		"Jenis berkas tidak dikenali penyimpanan dokumen.",
	}, message)

	// Nama kosong dilaporkan terpisah.
	violation = masterrecovery.CheckDocument(masterrecovery.Document{Content: []byte("x")})
	require.Equal(t, []masterrecovery.Violation{{Field: masterrecovery.FieldDocument, Message: "Nama berkas bukti bayar tidak terbaca."}}, violation)
}

func TestCheckVirtualAccountRequestLimits(t *testing.T) {
	violation := masterrecovery.CheckVirtualAccountRequest(masterrecovery.VirtualAccountRequest{
		ClientID:      strings.Repeat("c", masterrecovery.MaxClientIDLength+1),
		PrincipalName: strings.Repeat("p", masterrecovery.MaxPrincipalNameLength+1),
		Email:         strings.Repeat("e", 100) + "@x.id",
	})
	require.Equal(t, []masterrecovery.Violation{
		{Field: masterrecovery.FieldClientID, Message: "Client ID paling panjang 100 karakter."},
		{Field: masterrecovery.FieldPrincipalName, Message: "Nama principal paling panjang 200 karakter."},
		{Field: masterrecovery.FieldEmail, Message: "Email inputor VA paling panjang 100 karakter."},
	}, violation)
}
