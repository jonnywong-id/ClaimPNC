package masterlogin_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterlogin"
)

func violationsOf(t *testing.T, in masterlogin.Input) map[string]string {
	t.Helper()
	err := in.Check()
	if err == nil {
		return nil
	}
	var validation *masterlogin.ValidationError
	require.True(t, errors.As(err, &validation))
	out := map[string]string{}
	for _, v := range validation.Violation {
		out[v.Field] = v.Message
	}
	return out
}

func TestCheckReportsEveryLengthLimit(t *testing.T) {
	got := violationsOf(t, masterlogin.Input{
		Name:    strings.Repeat("a", masterlogin.MaxNameLength+1),
		Email:   strings.Repeat("e", masterlogin.MaxEmailLength+1),
		Phone:   strings.Repeat("1", masterlogin.MaxPhoneLength+1),
		Address: strings.Repeat("x", masterlogin.MaxAddressLength+1),
	})
	require.Contains(t, got["nama"], "paling panjang")
	require.Contains(t, got["email"], "paling panjang")
	require.Contains(t, got["telp"], "paling panjang")
	require.Contains(t, got["alamat"], "paling panjang")
}

func TestCheckReportsMissingAndUnusableValues(t *testing.T) {
	got := violationsOf(t, masterlogin.Input{})
	require.Equal(t, "Nama wajib diisi.", got["nama"])
	require.Equal(t, "Email wajib diisi.", got["email"])
	require.Equal(t, "Telp wajib diisi.", got["telp"])

	got = violationsOf(t, masterlogin.Input{Name: "- . -", Email: "a@b.c", Phone: "1"})
	require.Contains(t, got["nama"], "setidaknya satu huruf atau angka")
	require.Len(t, got, 1)

	require.Nil(t, violationsOf(t, masterlogin.Input{Name: "Budi", Email: "a@b.c", Phone: "1"}))
}

func TestValidationErrorListsEveryViolation(t *testing.T) {
	err := masterlogin.Input{}.Check()
	require.Equal(t,
		"masterlogin: isian tidak sah (nama: Nama wajib diisi.; email: Email wajib diisi.; telp: Telp wajib diisi.)",
		err.Error())

	one := masterlogin.OneViolation("login", "sudah dipakai")
	require.Equal(t, "masterlogin: isian tidak sah (login: sudah dipakai)", one.Error())
}
