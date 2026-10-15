package mastersparepart

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Unggah CSV master sparepart — padanan `PNCUploadMasterSparepartCSV`.
//
// # Dari mana bentuk berkasnya diketahui
//
// Activity `PNCUploadMasterSparepart_Act` TIDAK menyebut satu pun nama kolom. Ia menyerahkan
// penguraian ke `pxUploadCSVResults` bawaan Pega, yang memetakan **header CSV langsung ke
// nama properti** kelas `ASM-FW-GCNMFW-Int-SPAREPART_HE`, lalu seluruh halaman diserialkan
// apa adanya oleh `@GCNM.GetPageJSONString()`.
//
// Karena itu header CSV = nama properti = nama kolom tabel. Itu bukan tebakan; ia mengikuti
// dari serialisasi menyeluruh tersebut.
//
// # Yang TIDAK diterima dari berkas
//
// `ID`, `APPROVAL`, `USER_UPDATE`, `TGL_UPDATE_HARGA`, dan `DOKUMENID` sengaja diabaikan
// meski kolomnya muncul di berkas. Kelimanya ditetapkan server:
//
//   - `ID` dari pencarian `NO_SPART`, atau diterbitkan baru
//   - `APPROVAL` selalu `"0"` — `PNCUploadMasterSparepart_Act` menyetelnya tanpa syarat
//   - `USER_UPDATE` dari pengunggah, `TGL_UPDATE_HARGA` dari jam sistem
//   - `DOKUMENID` milik jalur unggah dokumen, bukan jalur ini
//
// Menerimanya dari berkas akan membuat satu baris CSV dapat **menyetujui dirinya sendiri**.

// MaxCSVRows membatasi jumlah baris data satu unggahan.
//
// Pega tidak punya batas yang terbaca, dan ketiadaan batas itulah masalahnya: satu berkas
// 200.000 baris akan menahan satu transaksi basis data selama menit-menit sambil mengunci
// baris yang sedang dipakai pengguna lain. Angkanya dipilih agar satu unggahan tetap selesai
// dalam hitungan detik; berkas yang lebih besar dipecah.
// bomPrefix adalah Byte Order Mark UTF-8, ditulis sebagai escape — BUKAN karakternya
// sendiri. Menaruh karakternya langsung di source Go membuat kompilator menolaknya
// ("invalid BOM in the middle of the file"), dan itu sudah sekali terjadi di sini.
const bomPrefix = "\ufeff"

const MaxCSVRows = 5000

// ErrCSVEmpty: berkas tidak memuat satu baris data pun.
var ErrCSVEmpty = errors.New("mastersparepart: berkas CSV tidak memuat baris data")

// ErrCSVTooManyRows: baris melebihi MaxCSVRows.
var ErrCSVTooManyRows = errors.New("mastersparepart: baris CSV melebihi batas")

// ErrCSVHeaderMissing: header wajib tidak ada.
var ErrCSVHeaderMissing = errors.New("mastersparepart: header CSV tidak lengkap")

// csvField memetakan header berkas ke isian Input.
//
// Kuncinya DIHURUFBESARKAN dan dipangkas saat dibaca, sehingga `nama_spart`, `NAMA_SPART`,
// dan ` Nama_Spart ` sama-sama diterima. Pega sendiri tidak peka huruf besar-kecil di sini,
// dan menolak berkas hanya karena kapitalisasi akan menghasilkan kegagalan yang membingungkan.
var csvField = map[string]func(*Input, string){
	"NAMA_SPART":       func(i *Input, v string) { i.Name = v },
	"NO_SPART":         func(i *Input, v string) { i.Number = v },
	"KODE_SPART":       func(i *Input, v string) { i.Code = v },
	"HARGA_JUAL":       func(i *Input, v string) { i.SellingPrice = v },
	"KATEGORI_SPART":   func(i *Input, v string) { i.CategoryID = v },
	"TIPE_SPART":       func(i *Input, v string) { i.TypeID = v },
	"BERAT":            func(i *Input, v string) { i.Weight = v },
	"PANJANG":          func(i *Input, v string) { i.Length = v },
	"LEBAR":            func(i *Input, v string) { i.Width = v },
	"TINGGI":           func(i *Input, v string) { i.Height = v },
	"MIN_STOCK":        func(i *Input, v string) { i.MinStock = v },
	"MAX_STOCK":        func(i *Input, v string) { i.MaxStock = v },
	"QTY_PESAN":        func(i *Input, v string) { i.OrderQuantity = v },
	"PROD_DATE":        func(i *Input, v string) { i.ProductionDate = v },
	"SUBSTITUSI_SPART": func(i *Input, v string) { i.Substitute = v },
	"JENIS_SPART":      func(i *Input, v string) { i.Kind = v },
	"SATUAN":           func(i *Input, v string) { i.Unit = v },
	"STS_AKTIF":        func(i *Input, v string) { i.ActiveStatus = v },
	"STS_PART":         func(i *Input, v string) { i.PartStatus = v },
}

// csvRequiredHeader adalah header yang HARUS ada.
//
// Keempatnya sama dengan isian wajib pada form — berkas tanpa salah satunya tidak akan
// menghasilkan satu baris sah pun, dan menolaknya di muka jauh lebih baik daripada
// melaporkan ribuan baris gagal satu per satu.
var csvRequiredHeader = []string{"NAMA_SPART", "NO_SPART", "KODE_SPART", "HARGA_JUAL"}

// CSVRow adalah satu baris berkas beserta nomor barisnya.
//
// Nomornya disimpan karena laporan hasil unggah TIDAK berguna tanpa itu: pengguna yang
// diberi tahu "12 baris gagal" tanpa nomor baris harus menebak yang mana.
type CSVRow struct {
	// Line adalah nomor baris di dalam BERKAS, dengan header terhitung sebagai baris 1 —
	// supaya angkanya cocok dengan yang dilihat pengguna di penyunting teksnya.
	Line int

	Input Input
}

// ParseCSV membaca berkas CSV menjadi daftar baris.
//
// Pemisahnya koma, dan itu mengikuti `pxUploadCSVResults`. Berkas bertitik-koma — yang lahir
// dari Excel berlokal Indonesia — akan terbaca sebagai satu kolom, dan kegagalannya muncul
// sebagai header tidak lengkap, bukan sebagai baris kosong yang membingungkan.
//
// BOM UTF-8 di depan header dibuang: Excel menuliskannya, dan tanpa pembuangan itu header
// pertama terbaca `<BOM>NAMA_SPART` dan dianggap tidak dikenal.
func ParseCSV(r io.Reader) ([]CSVRow, error) {
	reader := csv.NewReader(r)
	// Jumlah kolom per baris dibiarkan bebas: berkas nyata kerap punya baris yang kehilangan
	// kolom terakhir karena nilainya kosong. Yang kurang diisi kosong, bukan ditolak.
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, ErrCSVEmpty
	}
	if err != nil {
		return nil, fmt.Errorf("mastersparepart: membaca header CSV: %w", err)
	}

	index := map[int]func(*Input, string){}
	seen := map[string]bool{}
	for position, raw := range header {
		name := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(raw, bomPrefix)))
		if setter, known := csvField[name]; known {
			index[position] = setter
			seen[name] = true
		}
	}

	var missing []string
	for _, need := range csvRequiredHeader {
		if !seen[need] {
			missing = append(missing, need)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrCSVHeaderMissing, strings.Join(missing, ", "))
	}

	var rows []CSVRow
	line := 1
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("mastersparepart: membaca baris CSV ke-%d: %w", line+1, err)
		}
		line++

		if blankRecord(record) {
			// Baris kosong di ujung berkas lazim, dan melaporkannya sebagai kegagalan akan
			// membuat setiap berkas yang diakhiri enter terlihat bermasalah.
			continue
		}

		var input Input
		for position, setter := range index {
			if position < len(record) {
				setter(&input, record[position])
			}
		}

		// NAMA_SPART DIHURUFBESARKAN — dan hanya di jalur ini.
		//
		// `Activity/PNCUploadMasterSparepart_Act-Act.xml:91020` menyimpan hasil
		// `@toUpperCase(TempSparepart.NAMA_SPART)` KEMBALI ke properti yang sama, sehingga
		// yang tersimpan adalah bentuk huruf besarnya.
		//
		// Jalur form TIDAK melakukannya: `UpdateSparepartHE_act` dan `SetValueSparepartHE`
		// nol `toUpperCase`, dan yang ada di `ValidateMasterSparepart` hanya menghurufbesarkan
		// ke variabel LOKAL untuk membandingkan kunci ganda — nilainya tidak pernah
		// dikembalikan ke properti yang disimpan.
		//
		// Perbedaan itu janggal, tetapi ia perilaku yang berjalan hari ini, dan `P-5`
		// menuntutnya dipertahankan sampai ada keputusan yang menyatakan sebaliknya.
		// Menaruhnya di `Input.Clean()` akan menghurufbesarkan jalur form juga — perubahan
		// perilaku pada jalur yang sudah selesai, bukan penyalinan.
		input.Name = strings.ToUpper(strings.TrimSpace(input.Name))

		rows = append(rows, CSVRow{Line: line, Input: input})

		if len(rows) > MaxCSVRows {
			return nil, fmt.Errorf("%w: lebih dari %d baris", ErrCSVTooManyRows, MaxCSVRows)
		}
	}

	if len(rows) == 0 {
		return nil, ErrCSVEmpty
	}
	return rows, nil
}

// blankRecord menyatakan seluruh selnya kosong.
func blankRecord(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
