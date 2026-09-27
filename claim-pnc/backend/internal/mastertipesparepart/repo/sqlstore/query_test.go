package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql, dan sebaliknya.
// Tanpa uji ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya —
// dan kueri yang tidak lagi dipakai akan menumpuk tanpa ada yang membuangnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"type_list",
		"type_list_search",
		"type_get",
		"type_find_by_name",
		"type_lock_table",
		"type_next_id",
		"type_insert",
		"type_update",
		"type_set_status",
		"type_category_list",
		"type_count_by_status",
		"type_count_all",
		"type_check_table",
		"type_count_unknown_status",
		"type_count_duplicate_name",
		"type_count_orphan_category",
		"type_count_orphan_sparepart",
	}

	for _, name := range usedNames {
		require.NotEmptyf(t, getQuery(name), "kueri %q kosong", name)
	}
	require.Lenf(t, query, len(usedNames),
		"ada kueri di berkas .sql yang tidak dipakai kode, atau sebaliknya")
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { getQuery("type_tidak_ada") })
}

// Disiplin SQL portabel (D-20). Satu set kueri harus berjalan sama di Oracle 19c dan
// PostgreSQL 17+.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":  "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan tanggal dan angka dilakukan di Go",
		"TO_NUMBER": "penguraian angka dilakukan di Go; lihat tidyNumber",
		"VARCHAR2":  "VARCHAR sudah diterima Oracle maupun PostgreSQL",
		"FROM DUAL": "modul ini tidak menerbitkan nomor lewat sequence",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, upperCase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL adalah
// celah injeksi — persis pola `{ASIS:...}` yang 03-CURRENT-ARCHITECTURE.md §4.5 catat.
func TestQueriesUseParameterBinding(t *testing.T) {
	for _, name := range []string{
		"type_list",
		"type_list_search",
		"type_get",
		"type_find_by_name",
		"type_insert",
		"type_update",
		"type_set_status",
		"type_category_list",
		"type_count_by_status",
	} {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	// Pencarian mengikat kata kuncinya pada `:2`, dan MENYEBUTNYA DUA KALI — satu untuk nama
	// tipe, satu untuk nama kategori. Penyebutan ulang satu parameter posisional diterima
	// kedua driver; pola yang sama sudah dipakai `bengkel_list_search`.
	//
	// Yang dijaga uji ini adalah bahwa kata kuncinya benar-benar TERIKAT pada kedua sisi OR,
	// bukan dirangkai ke teks SQL pada salah satunya.
	search := getQuery("type_list_search")
	require.Equalf(t, 2, strings.Count(search, ":2"),
		"kata kunci harus terikat pada KEDUA kolom yang dicari; ditemukan %d penyebutan",
		strings.Count(search, ":2"))
	require.NotContains(t, search, ":3",
		"kata kunci yang sama tidak boleh dikirim dua kali sebagai parameter berbeda")

	// Penyimpanan mengikat keempat nilainya terpisah: nama, kategori, status, dan kunci.
	update := getQuery("type_update")
	for _, marker := range []string{":1", ":2", ":3", ":4"} {
		require.Containsf(t, update, marker,
			"type_update harus mengikat keempat nilainya; %s hilang", marker)
	}
}

// TIDAK ADA DELETE sama sekali di modul ini.
//
// `D-66` menetapkan soft delete menyeluruh, dan sistem lama pun tidak punya satu pun DELETE
// terhadap tabel ini — kesembilan rule yang menyentuhnya tidak memuat satu pun pernyataan
// hapus. Tipe yang tidak lagi dipakai DITOLAK, bukan dibuang.
func TestNoDeleteAnywhere(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; D-66 melarang penghapusan fisik data bernilai bisnis", name)
	}
}

// Hanya SATU tabel yang ditulis: POOLDATA.GCNM_M_SPAREPART_TYPE.
//
// Tabel kategori dan tabel sparepart HANYA DIBACA (ADR-0004, penulis tunggal per tabel).
// Satu INSERT, UPDATE, atau LOCK yang menyentuh salah satunya berarti aplikasi ini menulis
// ke tabel yang pemiliknya modul lain — dan pada tabel kategori, pemiliknya adalah
// masterkategorisparepart yang sudah selesai.
func TestOnlyOwnedTableIsWritten(t *testing.T) {
	const owned = "POOLDATA.GCNM_M_SPAREPART_TYPE"
	readOnly := []string{
		"POOLDATA.GCNM_M_SPAREPART_CATEGORY",
		"POOLDATA.SPAREPART_HE",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)

		isWrite := strings.HasPrefix(strings.TrimSpace(upperCase), "INSERT") ||
			strings.HasPrefix(strings.TrimSpace(upperCase), "UPDATE") ||
			strings.HasPrefix(strings.TrimSpace(upperCase), "LOCK")
		if !isWrite {
			continue
		}

		require.Containsf(t, upperCase, owned,
			"kueri tulis %q tidak menyentuh tabel milik modul ini", name)
		for _, table := range readOnly {
			require.NotContainsf(t, upperCase, table,
				"kueri tulis %q menyentuh %s yang HANYA BOLEH DIBACA (P-1)", name, table)
		}
	}
}

// Daftar dan pembacaan satu baris memakai LEFT JOIN, bukan inner join.
//
// Ini SELISIH PERILAKU yang disengaja terhadap Pega: `BrowseMasterSparepartTypeClaimHE_sql`
// memakai inner join gaya koma, sehingga tipe yang menunjuk kategori yang tidak ada HILANG
// dari layar dan tidak dapat diperbaiki siapa pun.
//
// Uji ini menjaga keputusan itu tetap berlaku. Bila seseorang "merapikannya" menjadi inner
// join demi menyamai Pega, baris yatim akan lenyap lagi — dan uji ini yang menghentikannya
// sebelum sampai ke pengguna.
func TestJoinToCategoryIsLeftJoin(t *testing.T) {
	for _, name := range []string{"type_list", "type_list_search", "type_get"} {
		upperCase := strings.ToUpper(getQuery(name))
		require.Containsf(t, upperCase, "LEFT JOIN",
			"kueri %q harus memakai LEFT JOIN supaya baris tanpa kategori tetap terlihat", name)
		require.NotContainsf(t, upperCase, "INNER JOIN",
			"kueri %q tidak boleh membuang baris yang kategorinya hilang", name)
	}
}

// Penyaring status memakai TRIM di kedua sisi.
//
// Rule lama membandingkannya langsung. Bila kolomnya CHAR dan bukan VARCHAR2, Oracle
// memadatkan pembandingnya dengan spasi sehingga perbandingan itu tetap benar — tetapi
// PostgreSQL tidak melakukannya, dan baris yang sama akan hilang setelah pindah basis data
// (D-24).
func TestStatusFilterIsTrimmed(t *testing.T) {
	for _, name := range []string{
		"type_list",
		"type_list_search",
		"type_category_list",
		"type_count_by_status",
	} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "TRIM(",
			"kueri %q harus memangkas APPROVAL supaya Oracle dan PostgreSQL menjawab sama", name)
	}
}

// Pemeriksaan keunikan nama TIDAK menyaring APPROVAL maupun kategori.
//
// Ia meniru `RDB List/ValidationSparepartType-SQL.xml` apa adanya (`P-5`, keputusan Work
// Owner 2026-09-21): nama tipe unik di SELURUH tabel, dan nama yang pernah ditolak tetap
// memblokir.
//
// Uji ini menjaga cakupan itu tetap terlihat. Menambahkan penyaring APPROVAL atau kategori
// di sini akan MELONGGARKAN aturan diam-diam — perubahan perilaku yang hanya boleh diambil
// sebagai keputusan tertulis, bukan sebagai kerapian.
//
// Yang diperiksa hanyalah klausa WHERE-nya. Kedua kolom itu tetap ikut di-SELECT — pemanggil
// membutuhkan kuncinya untuk mengecualikan baris yang sedang disunting, dan scanNameRow
// membaca keempat kolomnya — sehingga memeriksa seluruh teks kueri akan menuduh kolom yang
// sekadar dibaca sebagai penyaring.
func TestNameUniquenessCheckHasNoExtraFilter(t *testing.T) {
	upperCase := strings.ToUpper(getQuery("type_find_by_name"))

	position := strings.Index(upperCase, "WHERE")
	require.GreaterOrEqual(t, position, 0, "type_find_by_name harus punya klausa WHERE")
	where := upperCase[position:]

	require.NotContains(t, where, "APPROVAL",
		"pemeriksaan nama tidak menyaring APPROVAL; nama yang ditolak pun tetap memblokir")
	require.NotContains(t, where, "PART_CATEGORY_ID",
		"pemeriksaan nama tidak menyaring kategori; keunikannya berlaku di seluruh tabel")
}
