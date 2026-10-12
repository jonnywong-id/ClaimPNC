package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji
// ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	dipakai := []string{
		"user_get_by_identity",
		"local_login_find_active",
		"service_address",
		"user_update",
		"user_insert",
		"session_insert",
		"session_get_by_fingerprint",
		"session_revoke",
		"session_extend",
	}
	for _, name := range dipakai {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	terlarang := map[string]string{
		"SELECT *":  "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan tanggal dan angka dilakukan di Go",
		"FROM DUAL": "tidak ada padanannya di PostgreSQL",
	}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for pola, alasan := range terlarang {
			require.NotContainsf(t, uppercase, pola,
				"kueri %q memakai %q — %s", name, pola, alasan)
		}
		require.NotContainsf(t, uppercase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}.
func TestQueriesUseParameterBinding(t *testing.T) {
	berparameter := []string{
		"user_get_by_identity",
		"local_login_find_active",
		"service_address",
		"user_update",
		"user_insert",
		"session_insert",
		"session_get_by_fingerprint",
		"session_revoke",
		"session_extend",
	}
	for _, name := range berparameter {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}
