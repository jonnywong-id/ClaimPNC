package auth_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
)

var (
	createdAt = time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	laterAt   = time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
)

func fullProfile() auth.Profile {
	p := completeProfile()
	p.Company = "ASM"
	p.Branch = "Jakarta"
	p.BranchCode = "01"
	p.Position = "Staf"
	p.DetailBranchCode = "0101"
	return p
}

func TestFromProfileCopiesProfileAndStartsActive(t *testing.T) {
	user := auth.FromProfile(fullProfile(), createdAt)
	require.Equal(t, auth.User{
		Identity: "90000001", Kind: auth.Employee, Name: "Contoh User", Login: "contoh",
		Email: "contoh@example.invalid", Company: "ASM", Branch: "Jakarta", BranchCode: "01",
		Position: "Staf", DetailBranchCode: "0101",
		Active: true, CreatedAt: createdAt, UpdatedAt: createdAt,
	}, user)
}

func TestRefreshFromUpdatesProfileButKeepsLocalFields(t *testing.T) {
	user := auth.User{Identity: "90000001", OperatorID: "OP1", Active: false, CreatedAt: createdAt}
	p := fullProfile()
	p.Name = "Nama Baru"
	user.RefreshFrom(p, laterAt)

	require.Equal(t, "Nama Baru", user.Name)
	require.Equal(t, "0101", user.DetailBranchCode)
	require.Equal(t, "OP1", user.OperatorID)
	require.False(t, user.Active)
	require.Equal(t, createdAt, user.CreatedAt)
	require.Equal(t, laterAt, user.UpdatedAt)
}

func TestRefreshFromFillsMissingCreatedAt(t *testing.T) {
	user := auth.User{}
	user.RefreshFrom(fullProfile(), laterAt)
	require.Equal(t, laterAt, user.CreatedAt)
}

func TestIncompleteProfileErrorListsFieldsSorted(t *testing.T) {
	err := &auth.IncompleteProfileError{EmptyFields: []string{"nama", "identitas", "jenis"}}
	require.EqualError(t, err, "auth: profil tidak lengkap, field kosong: identitas, jenis, nama")
	// Urutan asli tidak ikut diubah.
	require.Equal(t, []string{"nama", "identitas", "jenis"}, err.EmptyFields)
}

func TestNewSessionIDIsHexOfSixteenBytes(t *testing.T) {
	id, err := auth.NewSessionID(bytes.NewReader(bytes.Repeat([]byte{0xab}, 16)))
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("ab", 16), id)

	// Tanpa sumber acak: memakai crypto/rand.
	random, err := auth.NewSessionID(nil)
	require.NoError(t, err)
	require.Len(t, random, 32)

	_, err = auth.NewSessionID(bytes.NewReader([]byte{1, 2}))
	require.Error(t, err)
}

func TestIssueTokenFailsOnShortRandomSource(t *testing.T) {
	_, err := auth.IssueToken(bytes.NewReader([]byte{1}))
	require.Error(t, err)

	_, err = auth.IssueToken(errorReader{})
	require.Error(t, err)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("sumber acak rusak") }
