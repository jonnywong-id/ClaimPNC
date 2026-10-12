package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/outstandingclaim"
)

func TestEmptyDocumentIsNotAnError(t *testing.T) {
	// Gabungan ke JSON_KLAIM adalah LEFT JOIN, sehingga klaim yang belum punya baris di
	// sana mengembalikan NULL. Klaim seperti itu TETAP dapat dibuka — hanya isinya kosong.
	doc, err := parseDocument("")
	require.NoError(t, err)
	require.Empty(t, doc)

	doc, err = parseDocument("   \n  ")
	require.NoError(t, err)
	require.Empty(t, doc)
}

func TestBrokenDocumentIsAnError(t *testing.T) {
	// Dokumen yang ADA tetapi rusak TIDAK diturunkan menjadi rincian kosong. Rincian kosong
	// terbaca sebagai data yang belum diisi, dan kerusakan data yang tersaji sebagai keadaan
	// normal tidak pernah dilaporkan siapa pun.
	_, err := parseDocument(`{"InsuredName": `)
	require.Error(t, err)
}

func TestNumbersKeepTheirExactText(t *testing.T) {
	// Hampir setiap angka di layar ini adalah NILAI UANG atau persentase share reasuransi.
	// Dibaca sebagai float64, nilai sebesar ini dibulatkan tanpa satu pun galat (`I-12`).
	doc, err := parseDocument(`{"TotalSumInsuredIDR": 1234567890123.45}`)
	require.NoError(t, err)

	value, found := doc.lookup("TotalSumInsuredIDR")
	require.True(t, found)
	require.Equal(t, "1234567890123.45", text(value))
}

func TestLookupWalksNestedPaths(t *testing.T) {
	doc, err := parseDocument(`{"PolicyData": {"PolicyNo": "99.001"}}`)
	require.NoError(t, err)

	value, found := doc.lookup("PolicyData.PolicyNo")
	require.True(t, found)
	require.Equal(t, "99.001", text(value))
}

func TestMissingIsDistinctFromEmpty(t *testing.T) {
	// Inilah alasan dokumennya diurai di Go alih-alih dipetik `JSON_VALUE`: "jalur tidak
	// ada" menunjuk jalur yang salah, "ada tetapi kosong" menunjuk data yang belum diisi,
	// dan keduanya menuntut tindakan yang berbeda.
	doc, err := parseDocument(`{"InsuredName": ""}`)
	require.NoError(t, err)

	_, found := doc.lookup("InsuredName")
	require.True(t, found, "isian kosong tetap ADA")

	_, found = doc.lookup("NamaYangTidakAda")
	require.False(t, found)
}

func TestLookupStopsAtNonObject(t *testing.T) {
	// Jalur yang menembus nilai skalar tidak boleh panik — bentuk dokumen ini belum pernah
	// diperiksa (`R-08`), sehingga jalur yang menabrak skalar adalah kemungkinan nyata.
	doc, err := parseDocument(`{"PolicyData": "bukan objek"}`)
	require.NoError(t, err)

	_, found := doc.lookup("PolicyData.PolicyNo")
	require.False(t, found)
}

func TestStructuredValueBecomesEmptyText(t *testing.T) {
	// Isian skalar yang ternyata berisi struktur adalah tanda jalurnya salah. Menumpahkan
	// JSON mentah ke sel tabel hanya memindahkan kebingungannya ke pengguna.
	require.Equal(t, "", text(map[string]any{"a": "b"}))
	require.Equal(t, "", text([]any{1, 2}))
	require.Equal(t, "", text(nil))
}

func TestSingleObjectCountsAsOneRow(t *testing.T) {
	// Pega menulis page list berisi satu baris kadang sebagai objek tunggal. Membuangnya
	// membuat grid tampak kosong padahal berisi.
	grid, found := outstandingclaim.FindGrid(outstandingclaim.GridInterestTotal)
	require.True(t, found)

	single := map[string]any{"Currency": "IDR", "Value": "15000000000"}
	rows := rows(single, grid.Columns)

	require.Len(t, rows, 1)
	require.Equal(t, "IDR", rows[0]["currency"])
	require.Equal(t, "15000000000", rows[0]["value"])
}

func TestNonListValueGivesNoRows(t *testing.T) {
	grid, _ := outstandingclaim.FindGrid(outstandingclaim.GridInterestTotal)
	require.Empty(t, rows("bukan senarai", grid.Columns))
	require.Empty(t, rows(nil, grid.Columns))
}

func TestReadDocumentCountsMissingPaths(t *testing.T) {
	// Hitungan inilah satu-satunya cara jalur yang salah terlihat: layar berisi 97 isian
	// kosong terbaca sama persis, entah karena klaimnya memang belum diisi atau karena
	// seluruh jalurnya salah.
	doc, err := parseDocument(`{"InsuredName": "PT Contoh", "IDMaster": "TRP-1"}`)
	require.NoError(t, err)

	values, gridRows, stats := readDocument(doc)

	// Angkanya DITURUNKAN dari section.go, bukan ditulis tetap di sini: menambah satu isian
	// di sana tidak boleh menggagalkan uji ini, tetapi isian yang berhenti terbaca harus.
	readable := 0
	for _, field := range outstandingclaim.Fields() {
		if !field.Blocked {
			readable++
		}
	}

	require.Equal(t, 2, stats.FieldsFound)
	require.Equal(t, readable-2, stats.FieldsMissing)
	require.Equal(t, 0, stats.GridsFound)
	require.Equal(t, "PT Contoh", values["insured_name"])
	require.Empty(t, gridRows)
}

func TestReadDocumentNeverReadsBlockedPaths(t *testing.T) {
	// Kedelapan isian `.TreatyInMaster.*` tidak punya jalur, dan tidak boleh terisi dari
	// jalur mana pun — termasuk dari kunci di dokumen yang kebetulan bernama sama.
	doc, err := parseDocument(`{"CEDING": "Asuransi Contoh", "ceding_name": "Asuransi X"}`)
	require.NoError(t, err)

	values, _, _ := readDocument(doc)
	require.NotContains(t, values, "ceding_name")
	require.NotContains(t, values, "ri_type")
}

func TestReadDocumentNeverReadsBlockedGrids(t *testing.T) {
	// Grid lampiran diisi Report Definition yang tidak ada di export. Ia tidak boleh
	// terisi dari dokumen klaim, sekalipun dokumennya kebetulan punya kunci bernama sama.
	doc, err := parseDocument(`{"attachment": [{"NOTE": "x"}]}`)
	require.NoError(t, err)

	_, gridRows, _ := readDocument(doc)
	require.NotContains(t, gridRows, outstandingclaim.GridAttachment)
}

func TestReadDocumentFillsGridsFromTheirOwnPaths(t *testing.T) {
	doc, err := parseDocument(`{
		"InterestList": [
			{"ObjectName": "Gudang", "CurrencyID": "IDR",
			 "KursObjectItem": "1", "TSIPerObject": "10000000000"}
		],
		"SpreadingClaim": [
			{"TreatyName": "Quota Share", "SharePercentage": "25",
			 "Currency": "IDR", "ClaimSpreaded": "487500000"}
		]
	}`)
	require.NoError(t, err)

	_, gridRows, stats := readDocument(doc)

	require.Equal(t, 2, stats.GridsFound)

	interest := gridRows[outstandingclaim.GridInterest]
	require.Len(t, interest, 1)
	require.Equal(t, "Gudang", interest[0]["object_name"])

	// Pasangan yang TAMPAK tertukar di Pega dibawa apa adanya — lihat catatan pada
	// section.go. Uji ini menjaga agar tidak ada yang menukarnya tanpa keputusan.
	require.Equal(t, "1", interest[0]["kurs"])
	require.Equal(t, "10000000000", interest[0]["tsi_per_object"])

	spreading := gridRows[outstandingclaim.GridSpreadingClaim]
	require.Len(t, spreading, 1)
	require.Equal(t, "Quota Share", spreading[0]["treaty_type"])
}

func TestGridRowsDropNonObjects(t *testing.T) {
	doc, err := parseDocument(`{"InterestList": ["bukan objek", {"ObjectName": "Gudang"}]}`)
	require.NoError(t, err)

	_, gridRows, _ := readDocument(doc)
	require.Len(t, gridRows[outstandingclaim.GridInterest], 1)
}

func TestEmptyListIsFoundButHasNoRows(t *testing.T) {
	// Grid yang senarainya ADA tetapi kosong BERBEDA dari grid yang jalurnya tidak ada:
	// yang pertama berarti "tidak ada isinya", yang kedua berarti jalurnya salah.
	doc, err := parseDocument(`{"InterestList": []}`)
	require.NoError(t, err)

	_, gridRows, stats := readDocument(doc)
	require.Equal(t, 1, stats.GridsFound)
	require.Contains(t, gridRows, outstandingclaim.GridInterest)
	require.Empty(t, gridRows[outstandingclaim.GridInterest])
}
