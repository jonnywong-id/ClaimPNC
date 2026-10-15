package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"supplier_list",
		"supplier_list_search",
		"supplier_get",
		"supplier_find_by_name",
		"supplier_lock_by_name",
		"supplier_read_document",
		"supplier_insert",
		"supplier_update",
		"supplier_code_distinct",
		"supplier_count_all",
		"supplier_check_table",
		"supplier_check_document",
		"supplier_site",
		"supplier_next_sequence",
		"supplier_branch_list",
		"supplier_city_search",
		"supplier_country_list",
		"supplier_bank_list",
		"approval_insert",
		"approval_check_table",
		"approval_count_pending",
	}
	for _, name := range usedNames {
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
//
// Satu pola yang paling mudah menyelinap masuk di modul ini: notasi titik Oracle
// (`A.JSONDATA.NAMA`) yang dipakai kueri lamanya. Ia TIDAK ada padanannya di PostgreSQL,
// dan penggantinya JSON_VALUE — lihat TestNoOracleDotNotation.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *": "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":     "pakai COALESCE",
		"SYSDATE":  "pakai CURRENT_TIMESTAMP, atau kirim waktunya sebagai parameter dari seam Clock",
		"DECODE(":  "pakai CASE WHEN",
		"ROWNUM":   "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":   "pakai POSITION",
		"LISTAGG(": "pakai STRING_AGG",
		"TO_CHAR(": "pemformatan tanggal dan angka dilakukan di Go",
		"VARCHAR2": "VARCHAR sudah diterima Oracle maupun PostgreSQL",
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

// Notasi titik Oracle atas kolom JSON dilarang.
//
// Kueri lamanya memakainya — `A.JSONDATA.NAMA AS "NAMA"` pada
// `RDB List/GetDataEditMasterSupller-SQL.xml` — dan bentuk itu TIDAK ADA di PostgreSQL.
// Menyalinnya apa adanya akan membuat seluruh modul ini berhenti bekerja pada hari
// perpindahan basis data, dan berhentinya tidak akan terlihat sebelum itu.
//
// Ini uji yang khas modul ini: modul master lain tidak membaca kolom JSON sama sekali.
func TestNoOracleDotNotation(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "JSONDATA.",
			"kueri %q memakai notasi titik Oracle atas JSONDATA; pakai JSON_VALUE(JSONDATA, '$.KUNCI')",
			name)
	}
}

// FROM DUAL dilarang di mana pun KECUALI satu kueri.
//
// Pengecualian ini disengaja dan terbatas: NEXTVAL menuntutnya, dan memakai sequence yang
// sama dengan procedure lama adalah syarat agar ID yang diterbitkan aplikasi ini tidak
// pernah bertabrakan dengan ID yang pernah diterbitkan Pega.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "supplier_next_sequence"

	for name, text := range query {
		if name == exempted {
			require.Contains(t, strings.ToUpper(text), "FROM DUAL",
				"kueri urutan memang harus memakainya; bila tidak lagi, hapus pengecualiannya")
			continue
		}
		require.NotContainsf(t, strings.ToUpper(text), "FROM DUAL",
			"kueri %q memakai FROM DUAL; hanya %q yang dibenarkan", name, exempted)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, 538 kemunculan.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"supplier_list_search",
		"supplier_get",
		"supplier_find_by_name",
		"supplier_lock_by_name",
		"supplier_read_document",
		"supplier_insert",
		"supplier_update",
		"supplier_city_search",
		"approval_insert",
		"approval_count_pending",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	require.Containsf(t, getQuery("supplier_city_search"), ":2",
		"pencarian kota mencocokkan nama DAN ID; keduanya harus terikat")
}

// Tidak ada DELETE terhadap tabel mana pun. Sistem lama tidak punya satu pun — layarnya
// bahkan tidak punya tombolnya — dan D-66 melarang penghapusan fisik data bernilai bisnis.
func TestNoDeleteStatement(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; supplier yang tidak dipakai ditandai lewat STS_AKTIF, bukan dibuang (D-66)",
			name)
	}
}

// Hanya dua objek yang boleh DITULIS modul ini.
//
// Keenam objek acuan lainnya hanya dibaca (ADR-0004, penulis tunggal per tabel). Yang
// paling penting dijaga di sini adalah M_BRANCH: ia juga tabel berkolom JSONDATA, sehingga
// satu salah ketik nama tabel pada pernyataan tulis akan menimpa master cabang milik
// seluruh aplikasi — dan salahnya tidak akan tertangkap kompilator mana pun.
func TestOnlyOwnedTablesAreWritten(t *testing.T) {
	readOnlyObjects := []string{
		"M_BRANCH",
		"COUNTRY",
		"GENERAL.LST_BANK_GROUP",
		"POOLDATA.M_SITE_DATABASE",
	}
	writableObjects := []string{"M_SUPPLIER", "POOLDATA.PROTEKSI_KLAIMMBU"}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.HasPrefix(upperCase, "INSERT") && !strings.HasPrefix(upperCase, "UPDATE") {
			continue
		}
		for _, object := range readOnlyObjects {
			require.NotContainsf(t, upperCase, object,
				"kueri %q menulis ke %s, padahal objek itu hanya boleh dibaca (ADR-0004)", name, object)
		}

		touched := false
		for _, object := range writableObjects {
			if strings.Contains(upperCase, object) {
				touched = true
			}
		}
		require.Truef(t, touched,
			"kueri tulis %q menyentuh tabel yang bukan milik modul ini", name)
	}
}

// Keempat kueri pembaca master harus menyebut kunci pada URUTAN YANG SAMA.
//
// scanRow membaca keempatnya dengan satu fungsi, berdasarkan POSISI kolom. Satu kunci yang
// bergeser di salah satu kueri akan menaruh nomor rekening ke kolom alamat tanpa satu pun
// galat — dan pada modul ini kolomnya dua puluh sembilan, sehingga pergeseran itu tidak
// mungkin terlihat dengan membaca.
//
// Nama kuncinya DIPERIKSA satu per satu, bukan hanya jumlahnya. Kedua puluh lima kunci
// pertama dibaca langsung dari `RDB List/GetDataEditMasterSupller-SQL.xml:90-118`, dan
// salah satu huruf yang berbeda membuat layar menampilkan kolom kosong tanpa galat —
// JSON_VALUE menjawab NULL alih-alih gagal.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	expected := []string{
		"ID", "OLDID",
		"NAMA", "ALAMAT", "KOTA", "NAMA_CABANG", "KODE_POS", "NEGARA",
		"TELEPON", "FAX", "EMAIL", "NPWP", "CONTACT_PERSON",
		"STS_REKANAN", "JENIS_STATUS", "SUPPLIER_HE",
		"TOP", "TOD", "KETERANGAN",
		"BANK", "ACCOUNT_NO", "ACCOUNT_NAME", "BANK_BRANCH",
		"JENIS_SUPPLIER", "STS_AKTIF_PROMLIST", "STS_AKTIF", "STS_AUTOPAYMENT",
		"USERKLAIMID", "TGL_INSERT",
	}

	for _, name := range []string{
		"supplier_list", "supplier_list_search", "supplier_get", "supplier_find_by_name",
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, selectedKeys(getQuery(name)))
		})
	}
}

// Kunci yang dibaca kueri harus sama persis dengan kunci yang ditulis documentValue.
//
// Tanpa uji ini, sebuah kunci dapat ditulis tetapi tidak pernah dibaca kembali — dan
// akibatnya hanya terlihat sebagai isian yang diam-diam kosong setelah disimpan.
//
// Tiga kunci sengaja TIDAK dibaca kueri mana pun dan karena itu dikecualikan: ID dan OLDID
// adalah KOLOM tabel, bukan kunci di dalam dokumen, sehingga keduanya dibaca sebagai kolom
// biasa. Kunci ID tetap DITULIS ke dalam dokumen supaya salinan yang tersimpan di baris
// permintaan persetujuan dapat menyebut supplier-nya sendiri.
func TestWrittenKeysMatchReadKeys(t *testing.T) {
	written := documentValue(mastersupplier.Supplier{})

	read := map[string]bool{}
	for _, key := range selectedKeys(getQuery("supplier_get")) {
		read[key] = true
	}

	for key := range written {
		if key == keyID {
			continue // kolom, bukan kunci yang dibaca dari dokumen
		}
		require.Truef(t, read[key],
			"kunci %q ditulis documentValue tetapi tidak dibaca kueri mana pun", key)
	}

	for key := range read {
		if key == "ID" || key == "OLDID" {
			continue // keduanya kolom tabel
		}
		require.Containsf(t, written, key,
			"kunci %q dibaca kueri tetapi tidak pernah ditulis documentValue", key)
	}
}

// Kelima kelompok sandi harus dibaca kueri distinct, dan namanya harus sama dengan yang
// dicocokkan ListCodes.
//
// Nama kelompoknya adalah literal di dalam SQL dan konstanta di dalam Go. Keduanya harus
// sama persis; kalau tidak, dropdown-nya diam-diam kosong.
func TestCodeGroupNamesMatch(t *testing.T) {
	text := getQuery("supplier_code_distinct")
	for _, group := range []string{
		keyPartnerStatus, keySupplyType, keySupplierType, keyActiveRequested, keyAutoPayment,
	} {
		require.Containsf(t, text, "'"+group+"'",
			"kelompok sandi %q tidak dibaca kueri distinct", group)
	}
}

// Lebar nomor urut ID supplier adalah SEBELAS, bukan sepuluh seperti Master Bengkel.
//
// Ia dibaca dari `Database/PEGA_M_SUPPLIER.prc:21`. Menyalin lebar dari modul tetangga
// akan menerbitkan kunci yang berbeda bentuk dari seluruh baris yang sudah ada — dan
// bedanya satu karakter, yang tidak mungkin terlihat tanpa membandingkannya.
func TestSequenceWidthFollowsProcedure(t *testing.T) {
	require.Equal(t, 11, mastersupplier.SequenceWidth)
	require.Equal(t, "0100000000001",
		mastersupplier.ComposeID("01", 1, mastersupplier.SequenceWidth))
}

// selectedItem mencocokkan satu butir pada klausa SELECT: sebuah ekspresi JSON_VALUE atas
// JSONDATA, atau sebuah nama kolom biasa.
//
// Urutan alternatifnya MENENTUKAN. Cabang JSON_VALUE harus lebih dulu supaya ia melahap
// seluruh ekspresinya; bila dibalik, nama kolom biasa akan mencocoki potongan
// "JSON_VALUE" dan "JSONDATA" di dalamnya.
//
// Memecah klausa SELECT dengan memisahkan koma TIDAK dapat dipakai di modul ini — setiap
// ekspresi JSON_VALUE memuat koma di dalam tanda kurungnya sendiri.
var selectedItem = regexp.MustCompile(
	`(?i)JSON_VALUE\s*\(\s*JSONDATA\s*,\s*'\$\.([A-Za-z0-9_]+)'\s*\)|([A-Za-z_][A-Za-z0-9_]*)`)

// selectedKeys mengambil nama kolom dan kunci JSON dari klausa SELECT, pada urutannya.
//
// Kolom biasa diambil apa adanya; ekspresi JSON_VALUE diambil nama kuncinya. Hanya klausa
// SELECT pertama yang dibaca — klausa sesudah FROM diabaikan, termasuk penyaring yang
// kebetulan memuat JSON_VALUE juga.
func selectedKeys(text string) []string {
	upperCase := strings.ToUpper(text)
	start := strings.Index(upperCase, "SELECT ")
	stop := strings.Index(upperCase, "\n  FROM ")
	if start < 0 || stop < 0 {
		return nil
	}

	body := text[start+len("SELECT ") : stop]

	var result []string
	for _, match := range selectedItem.FindAllStringSubmatch(body, -1) {
		switch {
		case match[1] != "":
			result = append(result, strings.ToUpper(match[1]))
		case match[2] != "":
			result = append(result, strings.ToUpper(match[2]))
		}
	}
	return result
}
