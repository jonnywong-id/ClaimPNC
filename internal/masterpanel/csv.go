package masterpanel

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Dua jalur unggah CSV — padanan `PNCUploadMasterPanelCSV` dan `PNCUploadLokasiPanelCSV`.
//
// # Dari mana bentuk berkasnya diketahui — dan apa yang TIDAK diketahui
//
// Kedua rule itu **tidak ada di export** (diperiksa 2026-10-06, snapshot 2.960 berkas XML);
// keduanya hanya dirujuk sebagai `<pyLocalAction>` pada `Section/BrowsePanelHE-Section.xml`.
// Jadi susunan kolomnya **tidak terbaca langsung**.
//
// Yang dipakai sebagai gantinya bukan tebakan bebas, melainkan **rule sekerabat yang ADA**:
// `Flow Action/PNCUploadMasterSparepartCSV-FA.xml` beserta
// `Activity/PNCUploadMasterSparepart_Act`, dan padanannya untuk Bengkel. Ketiganya satu
// keluarga — seluruhnya memakai `pxUploadCSVResults` bawaan Pega, yang memetakan **header
// CSV langsung ke nama properti** kelas integrasinya, lalu halamannya diserialkan apa adanya
// oleh `@GCNM.GetPageJSONString()`.
//
// Karena itu header CSV = nama properti = nama kolom tabel. Itu mengikuti dari mekanisme
// yang sama, bukan dari pola penamaan.
//
// # Yang tetap harus dikonfirmasi Tim Pega
//
// Satu hal tidak dapat disimpulkan dari rule sekerabat mana pun: **apakah baris hasil unggah
// masuk antrean persetujuan.** Di sini ia diperlakukan masuk — `APPROVAL := "0"` tanpa
// syarat, persis seperti `PNCUploadMasterSparepart_Act` dan persis seperti jalur Simpan
// modul ini sendiri.
//
// Itu pilihan yang **paling aman bila salah**: unggahan yang seharusnya langsung disetujui
// hanya tertahan satu langkah, sedangkan unggahan yang seharusnya tertahan dan langsung
// disetujui akan memintas seluruh kontrol persetujuan.

// bomPrefix adalah Byte Order Mark UTF-8, ditulis sebagai escape — BUKAN karakternya
// sendiri. Menaruh karakternya langsung di source Go membuat kompilator menolaknya.
const bomPrefix = "\ufeff"

// MaxCSVRows membatasi jumlah baris data satu unggahan.
//
// Angkanya sama dengan Master Sparepart — satu batas untuk seluruh aplikasi. Pega tidak
// punya batas yang terbaca, dan ketiadaan batas itulah masalahnya: berkas 200.000 baris
// akan menahan basis data selama menit-menit sambil mengunci baris yang sedang dipakai
// pengguna lain.
const MaxCSVRows = 5000

// ErrCSVEmpty: berkas tidak memuat satu baris data pun.
var ErrCSVEmpty = errors.New("masterpanel: berkas CSV tidak memuat baris data")

// ErrCSVTooManyRows: baris melebihi MaxCSVRows.
var ErrCSVTooManyRows = errors.New("masterpanel: baris CSV melebihi batas")

// ErrCSVHeaderMissing: header wajib tidak ada.
var ErrCSVHeaderMissing = errors.New("masterpanel: header CSV tidak lengkap")

// panelCSVField memetakan header berkas ke isian Input.
//
// Kuncinya dihurufbesarkan dan dipangkas saat dibaca, sehingga `name`, `NAME`, dan ` Name `
// sama-sama diterima. Menolak berkas hanya karena kapitalisasi menghasilkan kegagalan yang
// membingungkan, dan Pega sendiri tidak peka huruf besar-kecil di sini.
var panelCSVField = map[string]func(*Input, string){
	"NAME":               func(i *Input, v string) { i.Name = v },
	"STS_REPAIR":         func(i *Input, v string) { i.RepairStatus = v },
	"STS_EDIT_QTY":       func(i *Input, v string) { i.EditQuantityStatus = v },
	"STS_PREMIUM_REPAIR": func(i *Input, v string) { i.PremiumRepairStatus = v },
	"STS_PECAH":          func(i *Input, v string) { i.ShatterStatus = v },
	"STS_STICKER":        func(i *Input, v string) { i.StickerStatus = v },
	"STS_SISI":           func(i *Input, v string) { i.SideStatus = v },
	"STS_RUSAK_PARAH":    func(i *Input, v string) { i.SevereDamageStatus = v },
	"EXCLUSION_C":        func(i *Input, v string) { i.ExclusionC = v },
}

// panelCSVRequiredHeader adalah header yang HARUS ada.
//
// Kesepuluhnya, bukan sebagian — dan itu mengikuti validasi form modul ini, yang menandai
// kesepuluh isian induk wajib (`pyRequired` pada `Section/BrowsePanelHEApproval`). Berkas
// yang kehilangan salah satunya tidak akan menghasilkan satu baris sah pun, dan menolaknya
// di muka jauh lebih baik daripada melaporkan ribuan baris gagal satu per satu.
var panelCSVRequiredHeader = []string{
	"NAME", "STS_REPAIR", "STS_EDIT_QTY", "STS_PREMIUM_REPAIR", "STS_PECAH",
	"STS_STICKER", "STS_SISI", "STS_RUSAK_PARAH", "EXCLUSION_C",
}

// PanelCSVRow adalah satu baris berkas master panel beserta nomor barisnya.
type PanelCSVRow struct {
	// Line adalah nomor baris di dalam BERKAS, header terhitung sebagai baris 1 — supaya
	// angkanya cocok dengan yang dilihat pengguna di penyunting teksnya.
	Line int

	Input Input
}

// ParsePanelCSV membaca berkas master panel.
//
// Pemisahnya koma, mengikuti `pxUploadCSVResults`. Berkas bertitik-koma — yang lahir dari
// Excel berlokal Indonesia — akan terbaca sebagai satu kolom, dan kegagalannya muncul
// sebagai header tidak lengkap, bukan sebagai baris kosong yang membingungkan.
//
// BOM UTF-8 di depan header dibuang: Excel menuliskannya, dan tanpa pembuangan itu header
// pertama terbaca `<BOM>NAME` lalu dianggap tidak dikenal.
func ParsePanelCSV(r io.Reader) ([]PanelCSVRow, error) {
	index, reader, err := readHeader(r, panelCSVField, panelCSVRequiredHeader)
	if err != nil {
		return nil, err
	}

	var rows []PanelCSVRow
	line := 1
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("masterpanel: membaca baris CSV ke-%d: %w", line+1, readErr)
		}
		line++

		if blankRecord(record) {
			// Baris kosong di ujung berkas lazim; melaporkannya sebagai kegagalan akan
			// membuat setiap berkas yang diakhiri enter terlihat bermasalah.
			continue
		}

		var input Input
		for position, setter := range index {
			if position < len(record) {
				setter(&input, record[position])
			}
		}
		rows = append(rows, PanelCSVRow{Line: line, Input: input})

		if len(rows) > MaxCSVRows {
			return nil, fmt.Errorf("%w: lebih dari %d baris", ErrCSVTooManyRows, MaxCSVRows)
		}
	}

	if len(rows) == 0 {
		return nil, ErrCSVEmpty
	}
	return rows, nil
}

// LocationCSVRow adalah satu baris berkas lokasi panel.
//
// Ia TIDAK memakai Input: berkas ini tidak menyentuh satu pun isian induk. Ia hanya
// menyebutkan panel mana dan lokasi apa.
type LocationCSVRow struct {
	Line int

	// PanelName adalah panel tujuannya, dirujuk lewat NAMA-nya.
	//
	// Nama, bukan ID_PANEL, dan itu pilihan yang disengaja. ID_PANEL diterbitkan server
	// (`PANEL_HE_SEQ`) dan tidak diketahui penyusun berkas sebelum panelnya ada; nama
	// justru yang ia ketik sendiri. Kolom `ID_PANEL` tetap DITERIMA bila ada — lihat
	// locationCSVField — supaya berkas hasil ekspor pun terbaca.
	PanelName string
	PanelID   string

	Location PanelLocation
}

// locationCSVTarget menampung satu baris lokasi sebelum dipetakan.
type locationCSVTarget struct {
	PanelName string
	PanelID   string
	Name      string
	Side      string
}

// locationCSVField memetakan header berkas lokasi.
//
// Dua jalan menunjuk panel diterima — `NAME` dan `ID_PANEL` — dan salah satunya cukup.
// Keduanya ada karena berkas yang disusun tangan memakai nama, sedangkan berkas hasil
// ekspor membawa ID.
var locationCSVField = map[string]func(*locationCSVTarget, string){
	"NAME":     func(t *locationCSVTarget, v string) { t.PanelName = v },
	"PYLABEL":  func(t *locationCSVTarget, v string) { t.Name = v },
	"STS_SISI": func(t *locationCSVTarget, v string) { t.Side = v },

	// Dua kolom di bawah TIDAK ada pada berkas Pega. Keduanya diterima sebagai alias
	// supaya berkas yang disusun tangan — yang wajar memakai kata "lokasi" — tidak
	// ditolak hanya karena headernya lebih terbaca manusia daripada `pyLabel`.
	//
	// `pyLabel` tetap yang kanonikal, karena itulah yang dibaca
	// `Activity/PNCUploadLokasiSisiPanel_Act`.
	"LOKASI":       func(t *locationCSVTarget, v string) { t.Name = v },
	"LOKASI_PANEL": func(t *locationCSVTarget, v string) { t.Name = v },
	"ID_PANEL":     func(t *locationCSVTarget, v string) { t.PanelID = v },
}

// locationCSVRequiredHeader adalah header yang HARUS ada.
//
// `NAME` dan `ID_PANEL` TIDAK ada di sini meski salah satunya wajib: keduanya saling
// menggantikan, dan menuntut keduanya akan menolak berkas yang sah. Ketiadaan keduanya
// ditangkap per baris, bukan di header.
var locationCSVRequiredHeader = []string{"STS_SISI"}

// ParseLocationCSV membaca berkas lokasi panel.
func ParseLocationCSV(r io.Reader) ([]LocationCSVRow, error) {
	index, reader, err := readHeader(r, locationCSVField, locationCSVRequiredHeader)
	if err != nil {
		return nil, err
	}

	var rows []LocationCSVRow
	line := 1
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("masterpanel: membaca baris CSV ke-%d: %w", line+1, readErr)
		}
		line++

		if blankRecord(record) {
			continue
		}

		var target locationCSVTarget
		for position, setter := range index {
			if position < len(record) {
				setter(&target, record[position])
			}
		}
		rows = append(rows, LocationCSVRow{
			Line:      line,
			PanelName: target.PanelName,
			PanelID:   target.PanelID,
			Location:  PanelLocation{Name: target.Name, Side: csvWordToSide(target.Side)},
		})

		if len(rows) > MaxCSVRows {
			return nil, fmt.Errorf("%w: lebih dari %d baris", ErrCSVTooManyRows, MaxCSVRows)
		}
	}

	if len(rows) == 0 {
		return nil, ErrCSVEmpty
	}
	return rows, nil
}

// readHeader membaca baris header dan menyusun peta posisi kolom.
//
// Digeneralkan atas tipe sasarannya supaya kedua berkas memakai aturan header yang sama
// persis — BOM, kapitalisasi, pemangkasan, dan pelaporan kolom yang kurang. Menyalinnya
// dua kali akan membuat kedua berkas menerima hal yang sedikit berbeda, dan selisih itu
// hanya terlihat sebagai berkas yang ditolak tanpa sebab yang jelas.
func readHeader[T any](
	r io.Reader, field map[string]func(*T, string), required []string,
) (map[int]func(*T, string), *csv.Reader, error) {
	reader := csv.NewReader(r)
	// Jumlah kolom per baris dibiarkan bebas: berkas nyata kerap punya baris yang
	// kehilangan kolom terakhir karena nilainya kosong. Yang kurang diisi kosong, bukan
	// ditolak.
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, ErrCSVEmpty
	}
	if err != nil {
		return nil, nil, fmt.Errorf("masterpanel: membaca header CSV: %w", err)
	}

	index := map[int]func(*T, string){}
	seen := map[string]bool{}
	for position, raw := range header {
		name := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(raw, bomPrefix)))
		if setter, known := field[name]; known {
			index[position] = setter
			seen[name] = true
		}
	}

	var missing []string
	for _, need := range required {
		if !seen[need] {
			missing = append(missing, need)
		}
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("%w: %s", ErrCSVHeaderMissing, strings.Join(missing, ", "))
	}
	return index, reader, nil
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

// ImportLocationResult menandai hasil satu baris lokasi di dalam satu kelompok panel.
//
// Ia tinggal di paket domain, bukan di lapisan aplikasi, karena ia bagian dari kosakata
// berkasnya — "baris ini menambah" versus "baris ini sudah ada" — bukan detail orkestrasi.
type ImportLocationResult struct {
	Line int

	// Existing menyatakan lokasinya SUDAH ada pada panel itu sebelum berkas dibaca.
	Existing bool
}
