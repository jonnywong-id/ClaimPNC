package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	used := []string{
		"dominant_factor_list",
		"dominant_factor_get",
		"dominant_factor_lock_ids",
		"dominant_factor_insert",
		"dominant_factor_update",
		"dominant_factor_check_table",
	}
	for _, name := range used {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
//
// TO_NUMBER ikut dilarang di sini, dan itu bukan tambahan hiasan: godaan memakainya nyata
// di modul ini, karena ID harus diurutkan secara numerik. Pengurutannya dikerjakan di Go
// justru supaya kuerinya tetap portabel.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":    "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":        "pakai COALESCE",
		"SYSDATE":     "pakai CURRENT_TIMESTAMP",
		"DECODE(":     "pakai CASE WHEN",
		"ROWNUM":      "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":      "pakai POSITION",
		"LISTAGG(":    "pakai STRING_AGG",
		"TO_CHAR(":    "pemformatan tanggal dan angka dilakukan di Go",
		"TO_NUMBER(":  "pembacaan ID sebagai bilangan dilakukan di Go — lihat masterdominanfactor.NumericID",
		"LPAD(":       "pemformatan angka dilakukan di Go",
		"FROM DUAL":   "modul ini tidak memakai sequence, jadi tidak ada alasan memerlukannya",
		"CONNECT BY":  "tidak portabel",
		"KEEP (DENSE": "tidak portabel",
	}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, uppercase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, uppercase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat `{ASIS:...}`, 538
// kemunculan (`docs/Steering/11-SECURITY.md` §5).
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterized := []string{
		"dominant_factor_get",
		"dominant_factor_insert",
		"dominant_factor_update",
	}
	for _, name := range parameterized {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Tidak ada satu pun jalur yang boleh MENGHAPUS baris master.
//
// Alasannya bukan selera: `Database/PEGA_M_DOMINAN_FACTOR.prc` hanya mengenal INSERT dan
// UPDATE, layar Pega tidak punya tombol hapus, dan menghapus satu baris akan membuat
// setiap baris `T_CLAIM_DOMINANFACTOR` yang menyimpan ID itu kehilangan artinya —
// sehingga laporan Outstanding per Cabang menampilkan faktor yang hilang tanpa penjelasan
// (`ADR-0012`).
func TestNoQueryDeletes(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		require.NotContainsf(t, uppercase, "DELETE", "kueri %q menghapus baris", name)
		require.NotContainsf(t, uppercase, "TRUNCATE", "kueri %q mengosongkan tabel", name)
		require.NotContainsf(t, uppercase, "DROP ", "kueri %q membuang objek basis data", name)
	}
}

// Kueri tulis hanya boleh menyentuh M_DOMINAN_FACTOR. Menyentuh T_CLAIM_DOMINANFACTOR
// berarti modul master ikut mengubah data KLAIM — pelanggaran batas modul sekaligus
// pelanggaran `P-1`, karena tabel itu dimiliki jalur klaim dan bukan layar ini.
func TestWriteQueriesTouchOnlyMasterTable(t *testing.T) {
	for _, name := range []string{"dominant_factor_insert", "dominant_factor_update"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "POOLDATA.M_DOMINAN_FACTOR",
			"kueri %q harus menulis ke tabel masternya", name)
		require.NotContains(t, text, "T_CLAIM_DOMINANFACTOR",
			"kueri %q tidak boleh menyentuh data klaim", name)
	}
}

// Penyisipan bergantung pada kunci baris untuk menutup cacat balapan `max+1` yang
// diwarisi `Database/PEGA_M_DOMINAN_FACTOR.prc:11`. Tanpa FOR UPDATE, dua penyimpanan
// yang tiba bersamaan sama-sama membaca nomor tertinggi yang sama.
func TestInsertPathLocksRows(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("dominant_factor_lock_ids")), "FOR UPDATE",
		"tanpa kunci ini, dua penyimpanan bersamaan dapat menghasilkan ID kembar")
}

// Kueri daftar SENGAJA tanpa ORDER BY — pengurutannya numerik dan dikerjakan di Go.
// Menambahkan ORDER BY di sini akan mengurutkan sebagai TEKS, sehingga `10` mendahului
// `9` dan daftarnya tampak melompat-lompat.
func TestListQueryHasNoTextualOrdering(t *testing.T) {
	require.NotContains(t, strings.ToUpper(getQuery("dominant_factor_list")), "ORDER BY",
		"urutan dikerjakan masterdominanfactor.SortByID supaya numerik dan portabel")
}
