package reportklaim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func petaContoh() *ProgressNames {
	return NewProgressNames(map[string]string{
		"1": "Survey",
		"2": "Analisa",
		"3": "Akseptasi",
	})
}

func TestPositionsMerangkaiNamaMenurutUrutanJSON(t *testing.T) {
	raw := `{"ObjectList":[{"BranchName":"3"},{"BranchName":"1"}]}`

	require.Equal(t, "Akseptasi, Survey", petaContoh().Positions(raw))
}

// Duplikat TIDAK dibuang — sistem lama pun merangkainya dua kali.
func TestPositionsTidakMembuangDuplikat(t *testing.T) {
	raw := `{"ObjectList":[{"BranchName":"1"},{"BranchName":"1"}]}`

	require.Equal(t, "Survey, Survey", petaContoh().Positions(raw))
}

// Kode yang tidak dikenal master dikeluarkan APA ADANYA. Sistem lama melempar
// NO_DATA_FOUND dan menjatuhkan seluruh laporan; satu baris rusak tidak boleh membuat
// puluhan ribu baris lain gagal terunduh.
func TestKodeTidakDikenalDikeluarkanApaAdanya(t *testing.T) {
	raw := `{"ObjectList":[{"BranchName":"1"},{"BranchName":"99"}]}`

	require.Equal(t, "Survey, 99", petaContoh().Positions(raw))
}

func TestJSONRusakMengosongkanSelnyaSaja(t *testing.T) {
	require.Equal(t, "", petaContoh().Positions("{bukan json"))
}

func TestJSONKosongMenghasilkanTeksKosong(t *testing.T) {
	require.Equal(t, "", petaContoh().Positions(""))
	require.Equal(t, "", petaContoh().Positions("   "))
	require.Equal(t, "", petaContoh().Positions(`{"ObjectList":[]}`))
}

// Objek tanpa BranchName dilewati, bukan menghasilkan koma menggantung.
func TestObjekTanpaKodeDilewati(t *testing.T) {
	raw := `{"ObjectList":[{"BranchName":"1"},{},{"BranchName":"2"}]}`

	require.Equal(t, "Survey, Analisa", petaContoh().Positions(raw))
}

// Peta yang sumbernya tidak terbaca TIDAK mengeluarkan kode mentah seolah itu nama.
func TestPetaTidakTersediaMenghasilkanTeksKosong(t *testing.T) {
	raw := `{"ObjectList":[{"BranchName":"1"}]}`

	require.Equal(t, "", UnavailableProgressNames().Positions(raw))
}

// Peta yang terbaca tetapi KOSONG berbeda dari peta yang tidak terbaca: yang pertama
// mengeluarkan kodenya, yang kedua tidak mengeluarkan apa pun.
func TestPetaKosongBerbedaDariPetaTidakTersedia(t *testing.T) {
	raw := `{"ObjectList":[{"BranchName":"1"}]}`

	require.Equal(t, "1", NewProgressNames(nil).Positions(raw))
	require.Equal(t, "", UnavailableProgressNames().Positions(raw))
}
