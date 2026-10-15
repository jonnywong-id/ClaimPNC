package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasal"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql.
//
// Nama kueri adalah konstanta di dalam kode, dan ketiadaannya baru terlihat saat kueri itu
// pertama dijalankan — yang bisa jadi berbulan-bulan kemudian pada jalur yang jarang
// dilewati. Uji ini memindahkan kegagalannya ke waktu build.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range []string{
		"clause_list",
		"clause_get",
		"clause_list_id_locked",
		"clause_insert",
		"clause_update",
		"clause_delete",
		"clause_check_table",
		"business_search",
		"business_get",
		"business_check_table",
	} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

// Kueri pasal menyentuh tabel pasal; kueri lini bisnis menyentuh master lini bisnis.
//
// Keduanya dipisah tegas karena kewenangannya berbeda: yang pertama DITULIS modul ini,
// yang kedua HANYA DIBACA (ADR-0004, penulis tunggal per tabel). Satu kueri yang keliru
// menyentuh POOLDATA.BUSINESS dengan pernyataan penulisan akan melanggar kewenangan itu
// tanpa satu pun galat basis data.
func TestQueriesTargetTheRightTable(t *testing.T) {
	for name, text := range query {
		upperCase := strings.ToUpper(text)

		switch {
		case strings.HasPrefix(name, "clause_"):
			require.Containsf(t, upperCase, "POOLDATA.V_M_DATA_PASAL",
				"kueri %q harus menyentuh tabel pasal", name)

		case strings.HasPrefix(name, "business_"):
			require.Containsf(t, upperCase, "POOLDATA.BUSINESS",
				"kueri %q harus menyentuh master lini bisnis", name)
			require.NotContainsf(t, upperCase, "V_M_DATA_PASAL",
				"kueri %q menyentuh tabel pasal — periksa ulang nama tabelnya", name)

		default:
			t.Fatalf("kueri %q tidak berawalan clause_ maupun business_", name)
		}
	}
}

// Master lini bisnis TIDAK PERNAH ditulis modul ini.
//
// Ia milik ruleset GISFW dan dimiliki sistem lain. Uji ini menjaga batas kewenangan itu
// tetap ada meski kelak ada yang menambahkan kueri baru ke berkas yang sama.
func TestBusinessQueriesNeverWrite(t *testing.T) {
	for name, text := range query {
		if !strings.HasPrefix(name, "business_") {
			continue
		}
		upperCase := strings.ToUpper(text)
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE "} {
			require.NotContainsf(t, upperCase, forbidden,
				"kueri %q menulis ke master lini bisnis; tabel itu HANYA DIBACA", name)
		}
	}
}

// DELETE hanya ada di SATU kueri, dan itu memang disengaja.
//
// `D-66` menetapkan soft delete menyeluruh. Modul ini menyupersedenya atas keputusan Work
// Owner 2026-09-19 — tetapi pengecualian itu berlaku untuk SATU pernyataan saja, yaitu
// tombol Hapus yang memang sudah ada di layar lama.
//
// Uji ini yang menjaga pengecualiannya tidak melebar. Sebuah DELETE yang kelak menyelinap
// ke kueri lain — misalnya sebagai "hapus lalu sisip ulang" seperti pola
// `PEGA_CONVERT_JSONKLAIM_PNC` yang `D-66` cabut — akan menggagalkannya.
func TestDeleteAppearsInExactlyOneQuery(t *testing.T) {
	var withDelete []string
	for name, text := range query {
		if strings.Contains(strings.ToUpper(text), "DELETE") {
			withDelete = append(withDelete, name)
		}
	}
	require.Equal(t, []string{"clause_delete"}, withDelete,
		"DELETE fisik hanya boleh ada pada clause_delete; lihat masterpasal.Repo")
}

// Pola SQL yang dilarang §4.3 `08-TECHNICAL-STRATEGY.md` tidak boleh ada.
//
// Kelimanya membuat SQL tidak dapat berjalan sama di Oracle 19c dan PostgreSQL 17+ (D-20),
// dan `SELECT *` membuat kolom baru di basis data diam-diam mengubah perilaku aplikasi.
//
// `TO_CHAR` khususnya: kueri lama memakainya untuk memformat tampilan, dan pemformatan
// dilakukan di Go.
func TestNoForbiddenSQLPattern(t *testing.T) {
	forbidden := []string{"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "TO_CHAR(", "SELECT *"}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — dilarang §4.3 08-TECHNICAL-STRATEGY.md", name, pattern)
		}
	}
}

// Tidak ada pemanggilan stored procedure (D-02).
//
// `PEGA_D_PASAL_MASTER` logikanya naik ke Go; objeknya boleh ditinggalkan (D-68). Uji ini
// menjaga ia tidak dipanggil kembali sebagai jalan pintas.
func TestNoStoredProcedureCall(t *testing.T) {
	for name, text := range query {
		upperCase := strings.ToUpper(text)
		require.NotContainsf(t, upperCase, "PEGA_D_PASAL_MASTER",
			"kueri %q memanggil procedure; logikanya sudah naik ke Go (D-02)", name)
		require.NotContainsf(t, upperCase, "BEGIN ",
			"kueri %q memakai blok PL/SQL; ia tidak portabel (D-20)", name)
	}
}

// Nilai tidak pernah dirangkai ke dalam teks kueri.
//
// Kueri lama merangkai penyaringnya dari `{ASIS:TempSearchBisnis.DESCRIPTION}` — persis
// celah yang §4.3 tutup. Uji ini menjaga pola itu tidak kembali dalam bentuk apa pun.
func TestNoClipboardStringConcatenation(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, text, "{ASIS", "kueri %q merangkai SQL dari nilai klipboard", name)
		require.NotContainsf(t, text, "||", "kueri %q merangkai teks SQL", name)
	}
}

// Batas hasil pencarian di kueri dan di domain harus sama.
//
// Keduanya adalah angka yang sama yang tinggal di dua tempat, dan tidak ada apa pun selain
// uji ini yang menjaganya tetap sama. Bila keduanya berpisah, frontend akan menampilkan
// "50 teratas" untuk daftar yang sebenarnya dipotong di angka lain.
func TestSearchLimitMatchesDomain(t *testing.T) {
	want := fmt.Sprintf("FETCH NEXT %d ROWS ONLY", masterpasal.MaxLookupRows)
	require.Contains(t, strings.ToUpper(getQuery("business_search")), want)
}

// Kedua kueri pembaca pasal mengembalikan kolom pada URUTAN yang sama.
//
// `scanRow` membaca keduanya dengan SATU fungsi, berdasarkan POSISI kolom. Satu kolom yang
// bergeser di salah satunya akan menaruh dokumen JSON ke kolom No Pasal tanpa satu pun
// galat — dan yang terbaca pengguna adalah nomor pasal berisi seluruh dokumen.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	want := []string{"IDDATA", "IDPASAL", "JSONPASAL"}

	for _, name := range []string{"clause_list", "clause_get"} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, want, selectedColumns(t, getQuery(name)))
		})
	}
}

// selectedColumns mengambil daftar kolom pada klausa SELECT sebuah kueri.
func selectedColumns(t *testing.T, text string) []string {
	t.Helper()

	upperCase := strings.ToUpper(text)
	start := strings.Index(upperCase, "SELECT ")
	require.GreaterOrEqual(t, start, 0, "kueri tidak punya klausa SELECT")
	end := strings.Index(upperCase, "FROM ")
	require.Greater(t, end, start, "kueri tidak punya klausa FROM")

	var result []string
	for _, column := range strings.Split(upperCase[start+len("SELECT "):end], ",") {
		clean := strings.TrimSpace(column)
		if clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

// Dokumen yang disusun dapat dibaca kembali utuh.
//
// Ini uji terpenting kedua di berkas ini: seluruh isi pasal selain No Pasal hidup di dalam
// satu kolom CLOB, sehingga satu nama kunci yang salah ketik berarti isian itu HILANG
// tanpa satu pun galat — dan baru terlihat saat pengguna membuka kembali pasalnya.
func TestDocumentRoundTrip(t *testing.T) {
	original := masterpasal.Clause{
		ID:            "7",
		Number:        "PSL-007",
		Text:          "Penanggung menjamin kerugian akibat kebakaran.",
		Description:   "Jaminan dasar",
		Category:      masterpasal.CategoryPolicyCoverage,
		CategoryLabel: masterpasal.CategoryLabel(masterpasal.CategoryPolicyCoverage),
		Business: []masterpasal.Business{
			{ID: "2004", Name: "Fire / Property"},
			{ID: "", Name: "Diketik bebas"},
		},
	}

	payload, err := encode(original)
	require.NoError(t, err)

	back, err := scanRow(fakeRow{id: original.ID, number: original.Number, payload: payload})
	require.NoError(t, err)
	require.Equal(t, original, back)
}

// Nama kunci dokumen harus SAMA PERSIS dengan yang dibaca Pega.
//
// Keempatnya terbukti dari `RDB List/GetDataCOLByPasalBisnis_Sql-SQL.xml`, yang membacanya
// dengan `json_value(JSONPASAL, '$.<kunci>')`. Selama masa paralel, sistem lama masih
// membaca dokumen yang ditulis aplikasi ini — satu kunci yang berbeda membuat isian itu
// terbaca kosong di Pega, dan tidak ada apa pun yang memberi tahu.
func TestDocumentKeysMatchPega(t *testing.T) {
	payload, err := encode(masterpasal.Clause{
		ID: "7", Number: "PSL-007", Text: "isi", Description: "keterangan",
		Category: "1", CategoryLabel: "Jaminan Polis",
		Business: []masterpasal.Business{{ID: "2004", Name: "Fire / Property"}},
	})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload), &raw))

	for _, key := range []string{"M_COL_ID", "OLD_M_COL_ID", "DESCRIPTION", "OLD_D_COL_ID", "pyCountry", "LOSS_CODE", "BISNISID"} {
		require.Containsf(t, raw, key, "kunci %q dibaca Pega dan wajib ada", key)
	}

	business, ok := raw["BISNISID"].([]any)
	require.True(t, ok)
	require.Len(t, business, 1)
	require.Contains(t, business[0], "ID")
	require.Contains(t, business[0], "Note")
}

// Dokumen yang ditulis Pega dapat dibaca meski tipenya bukan teks.
//
// `pyCountry` dibandingkan sebagai ANGKA di Pega (`JaminanPengecualianApproval==1`),
// sehingga baris lama dapat memuat `"pyCountry": 1`. Tanpa penerima yang memaafkan, satu
// baris semacam itu membuat SELURUH daftar gagal dibaca.
func TestDocumentToleratesNonTextValues(t *testing.T) {
	raw := `{"M_COL_ID":"PSL-009","DESCRIPTION":"isi","OLD_D_COL_ID":null,` +
		`"pyCountry":1,"LOSS_CODE":"Jaminan Polis",` +
		`"BISNISID":[{"ID":2004,"Note":"Fire / Property"}]}`

	clause, err := scanRow(fakeRow{id: "9", number: "PSL-009", payload: raw})
	require.NoError(t, err)

	require.Equal(t, "1", clause.Category, "angka JSON terbaca sebagai teks yang sama")
	require.Equal(t, "Jaminan Polis", clause.CategoryLabel)
	require.Empty(t, clause.Description, "null menjadi teks kosong, bukan galat")
	require.Equal(t, "2004", clause.Business[0].ID)
}

// Dokumen KOSONG dibaca sebagai pasal tanpa isi; dokumen RUSAK menjadi galat bernama.
//
// Keduanya sengaja diperlakukan berbeda — alasannya ada pada doc comment scanRow.
func TestDocumentEmptyVersusMalformed(t *testing.T) {
	t.Run("kosong sah", func(t *testing.T) {
		clause, err := scanRow(fakeRow{id: "3", number: "PSL-003", payload: "   "})
		require.NoError(t, err)
		require.Equal(t, "3", clause.ID)
		require.Equal(t, "PSL-003", clause.Number)
		require.Empty(t, clause.Text)
		require.Equal(t, "Notifikasi", clause.CategoryLabel)
	})

	t.Run("rusak menjadi galat yang menyebut IDDATA", func(t *testing.T) {
		_, err := scanRow(fakeRow{id: "4", number: "PSL-004", payload: `{"DESCRIPTION":`})
		require.Error(t, err)
		require.Contains(t, err.Error(), `"4"`, "galat harus menyebut baris mana yang rusak")
	})
}

// Sebutan kategori DITURUNKAN ulang dari kodenya, tidak dipakai apa adanya dari dokumen.
//
// Keduanya tersimpan berdampingan dan tidak ada apa pun yang menjaganya sejalan. Yang
// menjadi sumber kebenaran adalah kodenya, karena itulah yang dipakai layar memilih ulang.
func TestCategoryLabelIsDerivedNotTrusted(t *testing.T) {
	raw := `{"pyCountry":"2","LOSS_CODE":"Jaminan Polis"}`

	clause, err := scanRow(fakeRow{id: "5", number: "PSL-005", payload: raw})
	require.NoError(t, err)
	require.Equal(t, "2", clause.Category)
	require.Equal(t, "Pengecualian", clause.CategoryLabel,
		"sebutan menyusul KODE, bukan sebaliknya")
}

// Kata kunci pencarian diloloskan sebelum masuk ke LIKE.
//
// Tanpa itu, tanda persen yang diketik pengguna tetap berlaku sebagai wildcard dan hasil
// pencariannya tidak dapat dijelaskan kepada yang mengetiknya.
func TestLookupArgumentsEscapeWildcard(t *testing.T) {
	pattern, exact := lookupArguments("  fire%_x  ")
	require.Equal(t, `%FIRE\%\_X%`, pattern)
	require.Equal(t, "FIRE%_X", exact)
}

// fakeRow meniru satu baris hasil kueri, supaya scanRow dapat diuji tanpa basis data.
//
// Ia MENUNTUT ketiga penampungnya bertipe *sql.NullString. Tuntutan itu disengaja: bila
// scanRow kelak membaca salah satu kolom langsung ke `string`, baris NULL akan menjadi
// galat pemindaian di produksi — dan uji ini yang menangkapnya lebih dulu.
type fakeRow struct {
	id      string
	number  string
	payload string
}

func (f fakeRow) Scan(target ...any) error {
	values := []string{f.id, f.number, f.payload}
	if len(target) != len(values) {
		return fmt.Errorf("scanRow meminta %d kolom, baris tiruan menyediakan %d",
			len(target), len(values))
	}

	for position, value := range values {
		holder, ok := target[position].(*sql.NullString)
		if !ok {
			return fmt.Errorf("penampung kolom ke-%d bertipe %T, bukan *sql.NullString",
				position+1, target[position])
		}
		*holder = sql.NullString{String: value, Valid: true}
	}
	return nil
}

// Nomor urut yang dihasilkan repo dan yang dihasilkan domain harus sama bentuknya.
//
// Keduanya dipakai berdampingan — domain menurunkan nomornya, repo menuliskannya — dan
// bentuk yang berbeda berarti baris yang baru disisipkan tidak dapat ditemukan kembali
// oleh kueri yang memakai TRIM.
func TestGeneratedIDHasNoPadding(t *testing.T) {
	id := masterpasal.NextSequence([]string{"9"})
	require.Equal(t, "10", id)

	number, err := strconv.Atoi(id)
	require.NoError(t, err)
	require.Equal(t, 10, number)
}
