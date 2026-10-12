package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastergroupingsparepart"
)

// sampleGrouping adalah baris yang dipakai menghitung jumlah argumen.
//
// Isinya tidak diperiksa satu pun uji di berkas ini — yang diuji adalah JUMLAH argumen yang
// dihasilkan insertArguments dan updateArguments, bukan nilainya. Ia sengaja hampir kosong
// supaya tidak ada yang tergoda menambahkan pemeriksaan nilai ke uji yang menjaga bentuk.
func sampleGrouping() mastergroupingsparepart.Grouping {
	return mastergroupingsparepart.Grouping{ID: "1", PartNumber: "SP-1001"}
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"grouping_list",
		"grouping_list_search",
		"grouping_get",
		"grouping_find_by_key",
		"grouping_lock_by_key",
		"grouping_insert",
		"grouping_update",
		"grouping_group_insert",
		"grouping_group_update",
		"grouping_set_status",
		"grouping_max_id",
		"grouping_max_id_mirror",
		"grouping_group_numbers",
		"grouping_group_by_chassis",
		"grouping_panel_list",
		"grouping_side_list",
		"grouping_vehicle_type_list",
		"grouping_part_find",
		"grouping_count_by_status",
		"grouping_count_all",
		"grouping_count_without_group",
		"grouping_count_chassis_mismatch",
		"grouping_count_child_name_as_panel",
		"grouping_count_child_name_as_location",
		"grouping_count_child_rows",
		"grouping_count_orphan_part",
		"grouping_count_orphan_panel",
		"grouping_check_table",
		"grouping_check_group_table",
		"grouping_check_vehicle_type_table",
		"grouping_check_child_name_column",
		"grouping_check_json_mirror",
		"grouping_count_json_mirror",
	}

	for _, name := range usedNames {
		require.NotEmptyf(t, getQuery(name), "kueri %q kosong", name)
	}
	require.Lenf(t, query, len(usedNames),
		"ada kueri di berkas .sql yang tidak dipakai kode, atau sebaliknya")
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { getQuery("grouping_tidak_ada") })
}

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
		"TO_NUMBER": "penguraian angka dilakukan di Go; lihat grouping_group_numbers",
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
// celah injeksi — dan modul ini justru yang paling terkena di sistem lama:
// `Activity/UpdateGroupingSparepartHE_act` merangkai keempat penyaring kunci alaminya sebagai
// teks, lengkap dengan tanda kutip tunggal yang diketik pengguna.
func TestQueriesUseParameterBinding(t *testing.T) {
	for _, name := range []string{
		"grouping_list",
		"grouping_list_search",
		"grouping_get",
		"grouping_find_by_key",
		"grouping_lock_by_key",
		"grouping_insert",
		"grouping_update",
		"grouping_group_insert",
		"grouping_group_update",
		"grouping_set_status",
		"grouping_group_by_chassis",
		"grouping_panel_list",
		"grouping_side_list",
		"grouping_vehicle_type_list",
		"grouping_part_find",
		"grouping_count_by_status",
	} {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	// Keempat kunci alami terikat sebagai empat parameter terpisah.
	for _, name := range []string{"grouping_find_by_key", "grouping_lock_by_key"} {
		text := getQuery(name)
		for _, marker := range []string{":1", ":2", ":3", ":4"} {
			require.Containsf(t, text, marker,
				"kueri %q harus mengikat keempat kunci alaminya; %s hilang", name, marker)
		}
	}

	// Pencarian mengikat kata kuncinya EMPAT KALI — satu untuk tiap kolom yang dicari. Lihat
	// catatan pada grouping_list_search.
	text := getQuery("grouping_list_search")
	for _, marker := range []string{":2", ":3", ":4", ":5"} {
		require.Containsf(t, text, marker,
			"kata kunci pencarian harus terikat pada %s, bukan dirangkai ke teks SQL", marker)
	}
}

// TIDAK ADA DELETE sama sekali di modul ini.
//
// `D-66` menetapkan soft delete menyeluruh. Berbeda dari Master Panel — yang tabel anaknya
// menuntut hapus-lalu-sisip-ulang karena baris anaknya tidak punya kunci sendiri — tabel
// pendamping modul ini berhubungan SATU-LAWAN-SATU dengan induknya, sehingga ia dapat
// di-UPDATE di tempat. Tidak ada alasan apa pun untuk sebuah DELETE di sini.
func TestNoDeleteAnywhere(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; D-66 melarang penghapusan fisik data bernilai bisnis", name)
	}
}

// Hanya DUA tabel yang ditulis: POOLDATA.SPAREPART_HE_VIN_KEY dan _VIN_GROUP.
//
// Keempat sumber acuan dan penyimpanan JSON milik Pega HANYA DIBACA (ADR-0004, penulis
// tunggal per tabel). Satu INSERT atau UPDATE yang menyentuh salah satunya berarti aplikasi
// ini menulis ke tabel yang pemiliknya modul atau sistem lain.
func TestOnlyOwnedTablesAreWritten(t *testing.T) {
	owned := []string{
		"POOLDATA.SPAREPART_HE_VIN_KEY",
		"POOLDATA.SPAREPART_HE_VIN_GROUP",
	}
	readOnly := []string{
		"POOLDATA.PANEL_HE",
		"POOLDATA.LOKASI_PANEL_HE",
		"POOLDATA.SPAREPART_HE",
		"BRANDDETAIL",
		"POOLDATA.M_SPAREPART_HE_VIN_KEY",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.Contains(upperCase, "INSERT INTO") && !strings.HasPrefix(upperCase, "UPDATE") {
			continue
		}

		tokens := tokenise(upperCase)
		touchesOwned := false
		for _, table := range owned {
			if strings.Contains(tokens, " "+table+" ") {
				touchesOwned = true
			}
		}
		require.Truef(t, touchesOwned,
			"kueri tulis %q tidak menyentuh satu pun tabel yang dimiliki modul ini", name)

		for _, table := range readOnly {
			require.NotContainsf(t, tokens, " "+table+" ",
				"kueri tulis %q menyentuh %s yang HANYA BOLEH DIBACA", name, table)
		}
	}
}

// Keempat kueri pembaca menyebut kolom pada urutan yang SAMA.
//
// Satu fungsi scanRow membaca keempatnya berdasarkan POSISI, dan satu kolom yang bergeser
// akan menaruh nama panel ke kolom nomor rangka tanpa satu pun galat.
//
// `grouping_check_table` sengaja TIDAK ikut dibandingkan: ia membaca tabel induk saja — lima
// belas kolom tanpa kedua kolom milik tabel pendamping — dan memaksanya sebentuk akan menuntut
// kueri pemeriksaan ikut menggabungkan kedua tabel, sehingga ia tidak lagi membuktikan bahwa
// tabel INDUKNYA dapat dibaca.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	reference := selectedColumns(t, getQuery("grouping_list"))
	require.Len(t, reference, 16, "keenam belas kolom harus dibaca")

	for _, name := range []string{
		"grouping_list_search",
		"grouping_get",
		"grouping_find_by_key",
	} {
		require.Equalf(t, reference, selectedColumns(t, getQuery(name)),
			"kueri %q membaca kolom pada urutan yang berbeda dari grouping_list", name)
	}
}

// Nomor rangka yang DITAMPILKAN berasal dari tabel pendamping, sedangkan yang DIPERIKSA
// keunikannya berasal dari tabel induk.
//
// Itu bukan kelalaian melainkan peniruan sistem lama: `GetDataMasterGrouping` memilih
// `B.NO_RANGKA` sementara pencarian duplikat dan pencarian nomor grup menyaring `A.NO_RANGKA`.
// Uji ini mengunci pembagian itu supaya perbedaannya tidak hilang diam-diam — dan supaya
// siapa pun yang menyeragamkannya tahu ia sedang mengubah perilaku.
func TestChassisReadFromCompanionButCheckedOnParent(t *testing.T) {
	list := tokenise(strings.ToUpper(getQuery("grouping_list")))
	require.Contains(t, list, " B.NO_RANGKA, ",
		"daftar menampilkan nomor rangka tabel pendamping")

	for _, name := range []string{"grouping_find_by_key", "grouping_lock_by_key"} {
		text := strings.ToUpper(getQuery(name))
		require.Containsf(t, text, "NO_RANGKA", "kueri %q harus menyaring nomor rangka", name)
		require.NotContainsf(t, tokenise(text), " B.NO_RANGKA ",
			"kueri %q harus menyaring NO_RANGKA tabel INDUK, bukan pendampingnya", name)
	}
}

func TestInsertColumnCountMatchesArguments(t *testing.T) {
	text := getQuery("grouping_insert")

	open := strings.Index(text, "(")
	closing := strings.Index(text, ")")
	require.Greater(t, closing, open)

	columns := strings.Split(text[open+1:closing], ",")
	require.Len(t, insertArguments(sampleGrouping()), len(columns),
		"jumlah argumen tidak sama dengan jumlah kolom pada grouping_insert")
	require.Len(t, columns, 15)
}

func TestGroupInsertColumnCountMatchesArguments(t *testing.T) {
	text := getQuery("grouping_group_insert")

	open := strings.Index(text, "(")
	closing := strings.Index(text, ")")
	require.Greater(t, closing, open)

	columns := strings.Split(text[open+1:closing], ",")
	require.Len(t, groupInsertArguments(sampleGrouping()), len(columns))
	require.Len(t, columns, 3)
}

func TestUpdateArgumentCountMatchesQuery(t *testing.T) {
	text := getQuery("grouping_update")

	// Lima belas argumen: empat belas kolom yang ditulis, ditambah ID sebagai penyaring WHERE.
	require.Len(t, updateArguments(sampleGrouping()), 15)
	require.Contains(t, text, ":15", "ID harus menjadi parameter terakhir")
}

// Kunci baris TIDAK PERNAH berpindah: grouping_update tidak boleh menyebut ID sebagai kolom
// yang ditulis.
func TestUpdateNeverMovesTheKey(t *testing.T) {
	text := strings.ToUpper(getQuery("grouping_update"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	// Dicari sebagai "ID =" dengan spasi di depannya, supaya ID_PANEL dan KATEGORI_SPART tidak
	// ikut tertangkap hanya karena namanya memuat huruf yang sama.
	require.NotContains(t, tokenise(setClause), " ID = ",
		"grouping_update tidak boleh menulis ID")
}

// Keputusan borongan hanya menyentuh APPROVAL.
//
// Tanpa uji ini, satu kolom yang ikut tertulis pada jalur keputusan akan menimpa isian yang
// diketik petugas lain — tanpa satu pun tanda di layar. Justru itulah yang dilakukan sistem
// lama, yang menyetujui dengan menjalankan ulang seluruh langkah penyimpanan; lihat
// usecase.Service.Decide.
func TestDecisionQueryTouchesOnlyApproval(t *testing.T) {
	text := strings.ToUpper(getQuery("grouping_set_status"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.Contains(t, setClause, "APPROVAL")
	for _, column := range []string{
		"NO_PART", "NAMA_PANEL", "NO_RANGKA", "SISI_PANEL", "NO_GROUP_RANGKA", "CATATAN",
	} {
		require.NotContainsf(t, setClause, column,
			"jalur keputusan tidak boleh menulis %s", column)
	}

	require.Contains(t, text, "<>",
		"baris yang sudah berstatus itu harus dikecualikan supaya jumlah berubah bermakna")
}

// Daftar panel menyaring status persetujuan lewat PARAMETER, bukan literal.
//
// Nilainya '1' — meniru parameter report definition pada autocomplete Nama Panel — tetapi ia
// datang dari konstanta Go, bukan dirangkai ke teks SQL. Itu yang membuatnya dapat diubah di
// satu tempat bila kelak Work Owner memutuskan lain.
func TestPanelLookupFiltersApprovalByParameter(t *testing.T) {
	text := strings.ToUpper(getQuery("grouping_panel_list"))
	require.Contains(t, text, "APPROVAL")
	require.Contains(t, text, ":1")
	require.NotContains(t, text, "= '1'",
		"kueri panel tidak boleh merangkai nilai penyaringnya ke teks SQL")

	require.Equal(t, string(mastergroupingsparepart.StatusApproved), approvedLookup)
	require.Equal(t, "1", approvedLookup)
}

// Lini bisnis pada daftar tipe kendaraan terikat sebagai parameter, dan nilainya "ANEKA".
//
// `RDB List/BrowseTypeHE_Sql-SQL.xml` menanamnya di dalam teks SQL; di sini ia konstanta
// bernama supaya terbaca sebagai keputusan alih-alih tersembunyi.
func TestVehicleTypeLookupBindsBusinessLine(t *testing.T) {
	text := strings.ToUpper(getQuery("grouping_vehicle_type_list"))
	require.Contains(t, text, ":1")
	require.NotContains(t, text, "'ANEKA'",
		"lini bisnis tidak boleh dirangkai ke teks SQL")
	require.Equal(t, "ANEKA", vehicleTypeGroup)

	// Nama tabelnya sengaja TANPA skema, persis seperti kueri aslinya; melengkapinya dengan
	// POOLDATA akan menjadi tebakan.
	require.NotContains(t, text, "POOLDATA.BRANDDETAIL",
		"branddetail disebut tanpa skema, mengikuti kueri aslinya")
}

// Pencarian sparepart TIDAK menyaring status persetujuan.
//
// `RDB List/GetDataSparepart-SQL.xml` tidak menyaringnya, dan menambahkannya akan menolak
// nomor sparepart yang hari ini diterima — termasuk sparepart yang sedang menunggu
// persetujuan.
func TestPartLookupDoesNotFilterApproval(t *testing.T) {
	text := strings.ToUpper(getQuery("grouping_part_find"))
	require.NotContains(t, text, "APPROVAL",
		"pencarian sparepart tidak boleh menyaring APPROVAL")
	require.Contains(t, text, "FETCH FIRST 1 ROWS ONLY",
		"hasilnya satu baris, dan urutannya harus pasti")
}

func TestLikeQueriesDeclareEscape(t *testing.T) {
	text := strings.ToUpper(getQuery("grouping_list_search"))
	require.Contains(t, text, "LIKE")
	require.Equal(t, 4, strings.Count(text, "ESCAPE '\\'"),
		"keempat LIKE harus menyebut karakter pelolosnya; Oracle tidak punya bawaan")
}

// Urutan daftar DITAMBAHKAN terhadap sistem lama, yang tidak menyebut satu pun kolom pengurut.
func TestListOrderIsDeterministic(t *testing.T) {
	for _, name := range []string{"grouping_list", "grouping_list_search"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "ORDER BY A.ID",
			"kueri %q harus punya urutan yang pasti", name)
	}

	// Pencarian nomor grup pun harus pasti: pemanggilnya hanya memakai satu baris, dan kueri
	// lama menyerahkan baris mana yang menang pada urutan yang dikembalikan Oracle.
	require.Contains(t, strings.ToUpper(getQuery("grouping_group_by_chassis")), "ORDER BY")
}

// Kueri pemeriksaan tidak mengambil satu baris pun, sehingga aman dijalankan terhadap
// produksi.
func TestCheckQueriesReadNoRows(t *testing.T) {
	for _, name := range []string{
		"grouping_check_table",
		"grouping_check_group_table",
		"grouping_check_vehicle_type_table",
		"grouping_check_child_name_column",
		"grouping_check_json_mirror",
	} {
		require.Containsf(t, getQuery(name), "1 = 0",
			"kueri pemeriksaan %q tidak boleh mengambil satu baris pun", name)
	}

	// Kolom NAMA ikut diperiksa: ia satu-satunya kolom di luar modul ini yang keberadaannya
	// menentukan apakah daftar Sisi dapat terisi sama sekali.
	require.Contains(t, strings.ToUpper(getQuery("grouping_check_child_name_column")), "NAMA")
}

func TestLikePattern(t *testing.T) {
	require.Equal(t, "%FILTER%", likePattern("filter"))
	require.Equal(t, "%FILTER%", likePattern("  Filter  "))

	// Tanda khusus LIKE diloloskan, supaya yang mengetik "%" tidak menarik seluruh tabel.
	require.Equal(t, `%100\%%`, likePattern("100%"))
	require.Equal(t, `%A\_B%`, likePattern("a_b"))
	require.Equal(t, `%A\\B%`, likePattern(`a\b`))
}

// upperKey menyiapkan keempat nilai kunci sepasang dengan `UPPER(TRIM(...))` pada kuerinya.
//
// Bila keduanya berbeda, pencarian kunci akan meleset tanpa satu pun galat — dan barisnya
// tersimpan ganda.
func TestUpperKeyMatchesQueryNormalisation(t *testing.T) {
	number, panel, chassis, side := upperKey(mastergroupingsparepart.NaturalKey{
		PartNumber:    "  sp-1001 ",
		PanelName:     " Kabin",
		ChassisNumber: "mhfxw1234k5678901 ",
		PanelSide:     " 1 ",
	})

	require.Equal(t, "SP-1001", number)
	require.Equal(t, "KABIN", panel)
	require.Equal(t, "MHFXW1234K5678901", chassis)
	require.Equal(t, "1", side)

	for _, name := range []string{"grouping_find_by_key", "grouping_lock_by_key"} {
		require.Equalf(t, 4, strings.Count(strings.ToUpper(getQuery(name)), "UPPER(TRIM("),
			"keempat kolom kunci pada %q harus dinormalkan sama seperti upperKey", name)
	}
}

// selectedColumns membaca nama kolom yang disebut sebuah SELECT, pada urutannya.
func selectedColumns(t *testing.T, text string) []string {
	t.Helper()

	upperCase := strings.ToUpper(text)
	start := strings.Index(upperCase, "SELECT ")
	require.GreaterOrEqual(t, start, 0)
	end := strings.Index(upperCase, "FROM ")
	require.Greater(t, end, start)

	var result []string
	for _, one := range strings.Split(upperCase[start+len("SELECT "):end], ",") {
		clean := strings.TrimSpace(one)
		// Alias tabel dibuang: yang dibandingkan adalah urutan kolomnya, bukan cara
		// menyebutnya.
		if dot := strings.LastIndex(clean, "."); dot >= 0 {
			clean = clean[dot+1:]
		}
		result = append(result, clean)
	}
	return result
}

// tokenise meratakan seluruh spasi menjadi satu dan memberi bantalan di kedua ujungnya.
//
// Dengan begitu `strings.Contains(tokenise(text), " NAMA ")` benar-benar mencocoki KATA
// `NAMA` — bukan potongan dari `NAMA_PANEL`. Pada modul ini pembedaan itu menentukan: nama
// tabel `POOLDATA.SPAREPART_HE` adalah awalan dari `POOLDATA.SPAREPART_HE_VIN_KEY`, sehingga
// pemeriksaan berbasis substring biasa akan selalu menyala.
func tokenise(text string) string {
	return " " + strings.Join(strings.Fields(text), " ") + " "
}
