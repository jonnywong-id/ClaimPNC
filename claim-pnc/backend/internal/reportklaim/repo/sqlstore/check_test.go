package sqlstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tanpa koneksi kedua, pemeriksaan TIDAK menyentuh basis data sama sekali dan tidak
// panik. Ia harus dapat dijalankan pada rakitan yang memang tidak punya blok ANEKA_*,
// karena justru keadaan itulah yang ingin dilaporkannya.
func TestCheckSourceTanpaKoneksiKeduaTidakMenyentuhBasisData(t *testing.T) {
	repo := NewRepo(nil, nil)

	state := repo.CheckSource(context.Background())

	require.False(t, state.SecondConnection)
	require.False(t, state.HolidayReadable)
	require.NoError(t, state.HolidayError)
	require.Zero(t, state.HolidayDays)
	require.NotZero(t, state.HolidayYear, "tahun acuan tetap disebut supaya laporannya dapat dibaca")
}

// NotPorted sepakat dengan daftar yang dikunci uji kueri.
//
// Keduanya kini KOSONG — ketiga kueri Non-MBU sudah dipindahkan. Uji ini tetap berdiri
// karena yang dijaganya bukan angkanya melainkan kesepakatannya: dua sumber yang tidak
// sepakat berarti salah satunya usang, dan yang usang di sini adalah laporan kesiapan
// yang dibaca orang saat memasang aplikasi.
func TestNotPortedSepakatDenganDaftarYangDikunci(t *testing.T) {
	terlihat := map[string]bool{}
	for _, item := range NotPorted() {
		terlihat[item.Query] = true
		require.NotEmpty(t, item.Title, "judulnya ikut disebut supaya terbaca tanpa membuka katalog")
		require.False(t, hasQuery(item.Query), "yang disebut harus benar-benar tidak ada")
	}

	require.Len(t, terlihat, len(kueriBelumDipindahkan))
	for name := range kueriBelumDipindahkan {
		require.Truef(t, terlihat[name], "kueri %q tidak disebut NotPorted", name)
	}
}

// Seluruh kueri sudah dipindahkan, dan mode periksa harus mengatakannya — bukan diam.
func TestNotPortedKosongSaatSeluruhKueriSudahAda(t *testing.T) {
	require.Empty(t, NotPorted())
}
