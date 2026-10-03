package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/config"
)

// TestDocumentStorageActiveFlags membuktikan kedua penanda mengikuti alamatnya masing-masing.
func TestDocumentStorageActiveFlags(t *testing.T) {
	require.False(t, config.DocumentStorage{BaseURL: "  "}.Active())
	require.True(t, config.DocumentStorage{BaseURL: "http://app13"}.Active())
	require.False(t, config.DocumentStorage{}.ConverterActive())
	require.True(t, config.DocumentStorage{ConverterURL: "http://aiimage"}.ConverterActive())
}

// TestLoadCollectsEveryMalformedValue membuktikan seluruh isian cacat dilaporkan sekaligus.
func TestLoadCollectsEveryMalformedValue(t *testing.T) {
	cleanEnv(t)
	t.Setenv("APP_ENV", "uat")
	t.Setenv("IDENTITAS_ADAPTER", "ldap")
	t.Setenv("PENYIMPANAN", "berkas")
	t.Setenv("SESI_MASA_BERLAKU", "setengah-jam")
	t.Setenv("HCQ_LOGIN_BATAS_WAKTU", "-5s")
	t.Setenv("SMTP_PORT", "dua-lima")

	_, err := config.Load()
	require.Error(t, err)
	message := err.Error()
	require.Contains(t, message, `APP_ENV "uat" tidak dikenal`)
	require.Contains(t, message, `IDENTITAS_ADAPTER "ldap" tidak dikenal`)
	require.Contains(t, message, `PENYIMPANAN "berkas" tidak dikenal`)
	require.Contains(t, message, `SESI_MASA_BERLAKU harus berupa durasi`)
	require.Contains(t, message, `HCQ_LOGIN_BATAS_WAKTU harus lebih besar dari nol`)
	require.Contains(t, message, `SMTP_PORT harus berupa angka`)
}

// TestDatabaseMissingUsesPortalPrefixByDefault membuktikan awalan baku POOLDATA_.
func TestDatabaseMissingUsesPortalPrefixByDefault(t *testing.T) {
	missing := config.Database{Alias: "ASM", Host: "h", User: "u"}.Missing()
	require.Equal(t, []string{"POOLDATA_ASM_SANDI", "POOLDATA_ASM_SERVICE"}, missing)
}
