package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Label mengikuti GetDataTertartanggungFromASMTelfFax: TelfaxType 6 atau nomor ber-@ adalah Email.
func TestInsuredTypeNames(t *testing.T) {
	require.Equal(t, "Alamat Rumah", registrasi.AddressTypeName(" 1 "))
	require.Equal(t, "Alamat Korespondensi", registrasi.AddressTypeName("8"))
	require.Equal(t, "", registrasi.AddressTypeName("9"))

	require.Equal(t, "Telepon Biasa", registrasi.PhoneTypeName("1", "0811"))
	require.Equal(t, "Telp Kantor", registrasi.PhoneTypeName("5", "021"))
	require.Equal(t, "Email", registrasi.PhoneTypeName("6", ""))
	require.Equal(t, "Email", registrasi.PhoneTypeName("1", "a@contoh.example"))
	require.Equal(t, "", registrasi.PhoneTypeName("", "0811"))
}
