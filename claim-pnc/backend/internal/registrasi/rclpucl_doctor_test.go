package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/registrasi"
)

// Daftar grup akses dropdown "Nama Dokter" WAJIB sama persis dengan daftar yang dipakai
// Inbox RCL mencari identitas lama pemanggilnya.
//
// Keduanya adalah salinan karena modul tidak saling mengimpor di repositori ini, dan
// salinan yang menyimpang gagal DIAM-DIAM — bukan dengan galat:
//
//	daftar di sini lebih LONGGAR  -> analis memilih dokter yang antreannya tidak pernah
//	                                 menerima klaim itu; klaimnya hilang dari semua inbox
//	daftar di sini lebih KETAT    -> dokter yang sah tidak dapat dipilih sama sekali
//
// Uji ini satu-satunya yang menangkap penyimpangan itu sebelum produksi.
func TestGrupAksesDokterRCLSamaDenganInboxRCL(t *testing.T) {
	require.Equal(t, inboxrcl.LegacyAccessGroups, registrasi.RCLDoctorAccessGroups,
		"grup akses dropdown Nama Dokter menyimpang dari penyaring Inbox RCL")
	require.Equal(t, inboxrcl.ExcludedAccessGroup, registrasi.RCLDoctorExcludedAccessGroup,
		"grup akses yang dikecualikan menyimpang dari penyaring Inbox RCL")
}
