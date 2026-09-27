package inboxsalvage

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
)

// Judul kolom berkas "Upload Detail Salvage".
//
// # Dari mana nama-nama ini berasal
//
// Dari `Activity/UploadDetailSalvage-Act.xml` langkah 3, yang membaca hasil
// `pxUploadCSVResults` lewat properti bernama sama persis — `.Item`, `.Quantity`,
// `.Satuan`, `.REMARKS` — dan menyalinnya ke grid `DetailSalvage.Data`. Pega memetakan
// judul kolom CSV ke properti menurut namanya, sehingga judul di berkas HARUS sama dengan
// nama properti itu.
//
// Perhatikan kapitalisasinya TIDAK seragam di sistem lama: tiga kolom berhuruf kapital di
// depan, satu kolom seluruhnya kapital. Pencocokan di sini mengabaikan besar-kecil huruf —
// lihat headerIndex.
const (
	ColumnItem     = "item"
	ColumnQuantity = "quantity"
	ColumnUnit     = "satuan"
	ColumnRemarks  = "remarks"
)

// RequiredUploadColumn adalah judul kolom yang WAJIB ada di berkas.
//
// Hanya `Item` yang wajib. Ketiga kolom lain boleh tidak ada sama sekali, dan itu bukan
// kelonggaran yang dikarang: `UploadDetailSalvage` menyalin keempatnya tanpa memeriksa satu
// pun, sehingga berkas yang hanya berisi nama barang memang diterima di Pega.
var RequiredUploadColumn = []string{ColumnItem}

// OptionalUploadColumn adalah judul kolom yang boleh ada dan boleh tidak.
var OptionalUploadColumn = []string{ColumnQuantity, ColumnUnit, ColumnRemarks}

// Galat pembacaan berkas unggahan.
var (
	// ErrUploadEmpty berarti berkasnya tidak punya baris judul sama sekali.
	ErrUploadEmpty = errors.New("inboxsalvage: berkas unggahan kosong")

	// ErrUploadColumnMissing berarti kolom wajib tidak ada di baris judul.
	ErrUploadColumnMissing = errors.New(
		"inboxsalvage: kolom wajib tidak ada di berkas unggahan")

	// ErrUploadTooManyRows berarti berkasnya melebihi batas baris.
	ErrUploadTooManyRows = errors.New(
		"inboxsalvage: berkas unggahan melebihi batas baris")
)

// ParseUpload membaca berkas CSV "Upload Detail Salvage" menjadi baris grid.
//
// # Ia MENGISI FORM, bukan menyimpan
//
// Ini yang paling mudah disalahpahami tentang tombolnya, dan penelusuran ke
// `Flow Action/UploadDetailSalvage-FA.xml` beserta activity di baliknya membuktikannya:
// ketiga langkahnya hanya menyalin isi berkas ke halaman klipboard `DetailSalvage.Data`.
// Tidak ada satu pun `RDB-List`, `Obj-Save`, maupun `Commit`.
//
// Penyimpanan baru terjadi ketika pengguna menekan Submit, dan itu jalur yang berbeda
// (`SetStsSalvagePNC_act`). Akibatnya: mengunggah berkas lalu menutup layar TIDAK
// meninggalkan apa pun di basis data — sama seperti di Pega.
//
// # Pemisah kolom
//
// Koma, mengikuti `pxUploadCSVResults` yang memakai `LocaleCode="en"`. Berkas yang disunting
// di Excel dengan setelan Indonesia memakai TITIK KOMA, dan berkas seperti itu akan terbaca
// sebagai satu kolom — ditolak di sini dengan alasan kolom wajib tidak ada, bukan diterima
// diam-diam sebagai nol baris.
func ParseUpload(source io.Reader) ([]DetailItem, error) {
	reader := csv.NewReader(source)

	// Jumlah kolom per baris dibiarkan BERVARIASI. Berkas yang disunting tangan sering
	// kehilangan koma di ujung baris yang kolom terakhirnya kosong, dan menolak seluruh
	// berkas karenanya berarti menolak berkas yang isinya sebenarnya benar.
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err == io.EOF {
		return nil, ErrUploadEmpty
	}
	if err != nil {
		return nil, err
	}

	index := headerIndex(header)
	for _, required := range RequiredUploadColumn {
		if _, found := index[required]; !found {
			return nil, ErrUploadColumnMissing
		}
	}

	items := []DetailItem{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		item := DetailItem{
			Name:     valueAt(record, index, ColumnItem),
			Quantity: valueAt(record, index, ColumnQuantity),
			Unit:     valueAt(record, index, ColumnUnit),
			Remarks:  valueAt(record, index, ColumnRemarks),
		}
		if item.IsEmpty() {
			continue
		}

		items = append(items, item)
		if len(items) > maxItems {
			return nil, ErrUploadTooManyRows
		}
	}

	return items, nil
}

// headerIndex memetakan judul kolom ke posisinya.
//
// Judulnya dipangkas dan dikecilkan hurufnya lebih dulu. Dua alasannya nyata, bukan
// kehati-hatian kosong: sistem lama sendiri menuliskan `REMARKS` seluruhnya kapital
// sementara ketiga kolom lain tidak, dan berkas yang disimpan Excel sering membawa spasi di
// ujung judul.
//
// Kolom berjudul GANDA menang yang PERTAMA. Berkas seperti itu lahir dari salin-tempel, dan
// memilih yang terakhir berarti membaca kolom yang biasanya kosong.
func headerIndex(header []string) map[string]int {
	index := map[string]int{}
	for position, title := range header {
		key := strings.ToLower(strings.TrimSpace(stripBOM(title)))
		if key == "" {
			continue
		}
		if _, exists := index[key]; exists {
			continue
		}
		index[key] = position
	}
	return index
}

// valueAt membaca satu sel, atau teks kosong bila kolomnya tidak ada di berkas ini.
func valueAt(record []string, index map[string]int, column string) string {
	position, found := index[column]
	if !found || position >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[position])
}

// stripBOM membuang penanda urutan bita di awal berkas.
//
// Excel menuliskannya pada setiap berkas CSV yang disimpan sebagai UTF-8, dan tanpa
// pembuangan ini judul kolom pertama terbaca `"\ufeffItem"` — tidak pernah cocok dengan
// `item`, sehingga berkas yang benar ditolak dengan alasan kolom wajib tidak ada.
func stripBOM(value string) string {
	return strings.TrimPrefix(value, "\ufeff")
}
