package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
)

func completeProfile() auth.Profile {
	return auth.Profile{
		Identity: "90000001",
		Name:     "Contoh User",
		Kind:     auth.Employee,
		Login:    "contoh",
		Email:    "contoh@example.invalid",
	}
}

func TestCompleteProfileAccepted(t *testing.T) {
	require.NoError(t, completeProfile().Check())
}

// Field yang kosong ditolak sebagai galat, bukan diteruskan diam-diam
// (TKT-F3-001, acceptance criteria terakhir).
func TestProfileWithEmptyFieldRejected(t *testing.T) {
	kasus := map[string]func(*auth.Profile){
		"identitas kosong":    func(p *auth.Profile) { p.Identity = "" },
		"nama kosong":         func(p *auth.Profile) { p.Name = "" },
		"jenis tidak dikenal": func(p *auth.Profile) { p.Kind = "" },
	}
	for name, clear := range kasus {
		t.Run(name, func(t *testing.T) {
			p := completeProfile()
			clear(&p)
			err := p.Check()
			require.Error(t, err)

			var issues *auth.IncompleteProfileError
			require.ErrorAs(t, err, &issues)
			require.Len(t, issues.EmptyFields, 1)
		})
	}
}

func TestEmptyCredentialIsIncomplete(t *testing.T) {
	require.False(t, auth.Credential{}.Complete())
	require.False(t, auth.Credential{Username: " ", Password: "x"}.Complete())
	require.False(t, auth.Credential{Username: "budi"}.Complete())
	require.True(t, auth.Credential{Username: "budi", Password: "x"}.Complete())
}

// Sidik token bersifat satu arah dan panjangnya tetap; token mentahnya tidak muncul
// di dalamnya.
func TestTokenDigestIsOneWay(t *testing.T) {
	token := auth.Token("token-contoh-yang-panjang")
	fingerprint := token.Digest()

	require.Len(t, fingerprint, 64)
	require.NotContains(t, fingerprint, string(token))
	require.Equal(t, fingerprint, token.Digest(), "sidik harus tetap untuk token yang sama")
	require.NotEqual(t, fingerprint, auth.Token("token-contoh-yang-panjan").Digest())
}

func TestIssueTokenProducesDistinctValues(t *testing.T) {
	pertama, err := auth.IssueToken(nil)
	require.NoError(t, err)
	kedua, err := auth.IssueToken(nil)
	require.NoError(t, err)
	require.NotEqual(t, pertama, kedua)
}

// Sesi yang dicabut terbaca sebagai dicabut walau kebetulan juga sudah kedaluwarsa —
// administrator yang mencabut harus melihat alasan yang benar.
func TestRevocationTakesPrecedenceOverExpiry(t *testing.T) {
	sekarang := time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)
	dicabut := sekarang.Add(-time.Hour)

	s := auth.Session{
		ExpiresAt: sekarang.Add(-time.Minute),
		RevokedAt: &dicabut,
	}
	require.ErrorIs(t, s.Check(sekarang), auth.ErrSessionRevoked)
	require.False(t, s.Active(sekarang))
	require.Zero(t, s.Remaining(sekarang))
}

func TestSessionActiveUntilExpiry(t *testing.T) {
	sekarang := time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)
	s := auth.Session{ExpiresAt: sekarang.Add(10 * time.Minute)}

	require.NoError(t, s.Check(sekarang))
	require.True(t, s.Active(sekarang))
	require.Equal(t, 10*time.Minute, s.Remaining(sekarang))

	require.ErrorIs(t, s.Check(s.ExpiresAt), auth.ErrSessionExpired,
		"tepat pada batas berlaku, sesi sudah tidak sah")
}
