package sqlstore

// Pohon klaim dijodohkan lewat OBJECTID dan OBJECTCOVERAGEID, bukan lewat URUTAN.
//
// Kedua uji di berkas ini menjaga keadaan nyata yang pernah melumpuhkan hampir seluruh layar
// klaim, dan keduanya tidak akan terlihat dari uji jalur bahagia — di sana URUTAN selalu
// terisi, padahal di basis data ia hampir selalu kosong.

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// be4StepByName mengganti baris satu langkah tanpa bergantung pada urutannya.
//
// Memakai indeks akan membuat uji ini pecah setiap kali ada langkah baru disisipkan di
// tengah — padahal yang diujinya tidak berubah sama sekali.
func be4StepByName(steps []be4Step, name string, cols int, rows [][]driver.Value) {
	for i := range steps {
		if steps[i].name == name {
			steps[i].cols = cols
			steps[i].rows = rows
			return
		}
	}
	panic("langkah tidak ada: " + name)
}

// TestClaimGetWorksWhenUrutanIsNull membuktikan klaim warisan tetap dapat dibuka.
//
// URUTAN dan URUTAN_OBJEK adalah kolom yang ditambahkan proyek ini; baris yang ditulis Pega
// tidak pernah mengisinya. Diukur pada 2026-10-07: 2.611 dari 2.726 baris objek aktif dan
// 2.598 dari 2.635 baris coverage KOSONG, sehingga 1.665 dari 1.686 klaim gagal dibuka
// dengan `converting NULL to int is unsupported`.
//
// Yang diuji di sini bukan sekadar "tidak galat": pohonnya harus benar-benar tersusun —
// coverage menempel ke objeknya, dan spreading menempel ke coverage-nya.
func TestClaimGetWorksWhenUrutanIsNull(t *testing.T) {
	steps := be4GetSteps()
	be4StepByName(steps, "objek_daftar", 6, [][]driver.Value{
		{nil, "OBJ1 ", "Gudang", "Jakarta", nil, nil},
	})
	be4StepByName(steps, "coverage_daftar", 9, [][]driver.Value{
		{"OBJ1 ", "1", nil, nil, "COV1", "C1", int64(1000), "Kebakaran", nil},
	})
	be4StepByName(steps, "spreading_daftar", 6, [][]driver.Value{
		{"OBJ1", "1", int64(1), "10001", "OR", int64(400000)},
	})

	db, mock := be4DB(t)
	be4ExpectAll(mock, steps)

	k, err := NewClaimStore(db).Get(context.Background(), "K1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, k.InsuredItem, 1)
	require.Equal(t, "Gudang", k.InsuredItem[0].Name)
	require.Len(t, k.InsuredItem[0].Coverage, 1, "coverage harus menempel meski URUTAN kosong")
	require.Equal(t, "Kebakaran", k.InsuredItem[0].Coverage[0].Name)
	require.Len(t, k.InsuredItem[0].Coverage[0].Spreading, 1,
		"spreading harus menempel lewat OBJECTID dan OBJECTCOVERAGEID")
	require.Equal(t, registrasi.Percent(400000), k.InsuredItem[0].Coverage[0].Spreading[0].Share)
}

// TestClaimGetPrefersFirstRowOnDuplicateObjectID membuktikan OBJECTID kembar diselesaikan
// secara deterministik.
//
// Pada 4 klaim terdapat dua baris objek ber-OBJECTID sama, dan pada 4 dari 6 pasangnya nama
// objeknya pun berbeda — jadi ia memang dua objek, bukan satu baris ganda. Tanpa aturan
// "baris pertama menang", coverage-nya menempel ke baris mana pun yang kebetulan terbaca
// belakangan, dan hasilnya berubah-ubah antar pembacaan.
func TestClaimGetPrefersFirstRowOnDuplicateObjectID(t *testing.T) {
	steps := be4GetSteps()
	be4StepByName(steps, "objek_daftar", 6, [][]driver.Value{
		{nil, "OBJ1", "Gudang Depan", "Jakarta", nil, nil},
		{nil, "OBJ1", "Gudang Belakang", "Bandung", nil, nil},
	})
	be4StepByName(steps, "coverage_daftar", 9, [][]driver.Value{
		{"OBJ1", "1", nil, nil, "COV1", "C1", int64(1000), "Kebakaran", nil},
	})
	be4StepByName(steps, "spreading_daftar", 6, [][]driver.Value{})

	db, mock := be4DB(t)
	be4ExpectAll(mock, steps)

	k, err := NewClaimStore(db).Get(context.Background(), "K1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	// KEDUA objeknya tetap ada — yang ambigu hanya ke mana coverage-nya menempel.
	require.Len(t, k.InsuredItem, 2)
	require.Len(t, k.InsuredItem[0].Coverage, 1, "coverage menempel ke baris PERTAMA")
	require.Empty(t, k.InsuredItem[1].Coverage)
}
