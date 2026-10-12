package inboxxol

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Unggahan "Upload MBU Salvage" pada layar rincian XOL.
//
// # Dari mana aturannya
//
// Seluruhnya dari rule sistem lama, bukan dari tampilan:
//
//	Section/DetailValueClaimXOL-Section.xml:6343   tombolnya, local action
//	Flow Action/PNCUploadClaimCSV_MBUSalvage-FA.xml  pxUploadCSVResults lalu activity
//	Activity/ConvertDataCsvSalvageMBUToPage-Act.xml  pemeriksaan dan pemetaan kolom
//	RDB List/InsertDataSalvageMBU-SQL.xml            sisipan akhirnya
//	RDB List/BrowseCurrency-SQL.xml                  pencarian ID mata uang
//
// Judul kolom berkasnya sama dengan berkas contoh yang diturunkan tombol "Format MBU
// Salvage" — lihat uploadTemplates.

// maxUploadBytes membatasi berkas yang dibaca.
//
// Sistem lama tidak punya batas sama sekali; batas di sini bukan peniruan melainkan
// syarat agar satu berkas tidak dapat menghabiskan memori satu instans.
const maxUploadBytes = 8 << 20

// bom adalah tanda urutan byte yang dituliskan Excel di depan judul kolom pertama.
//
// Tanpa pembuangannya, judul kolom pertama tidak pernah cocok dengan judul yang dicari,
// dan pesan galatnya berbunyi "kolom claimno belum ada" pada berkas yang justru memuatnya.
var bom = string(rune(0xFEFF))

// salvageUploadColumns adalah judul kolom yang WAJIB ada di berkas, sesuai berkas contoh.
//
// Urutannya tidak mengikat — kolom dicari menurut judulnya, bukan menurut posisinya,
// sehingga berkas yang kolomnya tersusun ulang tetap terbaca.
var salvageUploadColumns = []string{
	"claimno", "dateofloss", "currency", "salvageos", "salvageaksep", "causeofloss",
}

// SalvageUploadRow adalah satu baris berkas unggahan MBU Salvage.
//
// Nilainya masih TEKS apa adanya dari berkas. Pengubahan ke angka terjadi belakangan dan
// kegagalannya dilaporkan per baris, bukan menggugurkan seluruh berkas.
type SalvageUploadRow struct {
	// LineNumber adalah nomor baris di dalam berkas, terhitung sejak baris judul.
	//
	// Ia dibawa sampai ke laporan hasil karena yang diperbaiki pengguna adalah berkasnya,
	// dan pada berkas ratusan baris "ada yang gagal" saja tidak dapat ditindaklanjuti.
	LineNumber int

	ClaimNo      string
	DateOfLoss   string
	Currency     string
	SalvageOS    string
	SalvageAksep string
	CauseOfLoss  string
}

// SalvageUploadResult adalah hasil satu unggahan.
type SalvageUploadResult struct {
	// Rows adalah jumlah baris berisi di dalam berkas, termasuk yang ditolak.
	Rows int

	// Inserted adalah jumlah baris yang benar-benar tersimpan.
	Inserted int

	// Rejected adalah baris yang TIDAK disisipkan, beserta sebabnya.
	Rejected []RejectedUploadRow
}

// RejectedUploadRow adalah satu baris berkas yang tidak dapat disisipkan.
type RejectedUploadRow struct {
	LineNumber int
	ClaimNo    string
	Message    string
}

// ParseSalvageUpload membaca berkas CSV menjadi baris unggahan.
//
// Ia HANYA membaca bentuknya. Pemeriksaan yang menyentuh basis data — pencarian ID mata
// uang — ada di lapisan aplikasi, supaya galat bentuk berkas dapat dijawab tanpa satu pun
// kueri.
func ParseSalvageUpload(source io.Reader) ([]SalvageUploadRow, error) {
	content, err := io.ReadAll(io.LimitReader(source, maxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("inboxxol: berkas tidak dapat dibaca: %w", err)
	}
	if len(content) > maxUploadBytes {
		return nil, NewValidationError([]Violation{{
			Field:   "berkas",
			Message: fmt.Sprintf("Berkas terlalu besar. Batasnya %d MB.", maxUploadBytes/(1<<20)),
		}})
	}

	// Tanda urutan byte dibuang. Excel menuliskannya di depan judul kolom pertama, dan
	// tanpa pembuangan ini "ClaimNo" tidak pernah cocok dengan judul yang dicari.
	text := strings.TrimPrefix(string(content), bom)

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = detectSeparator(text)
	// Jumlah kolom tidak dipaksa sama: baris yang kolomnya kurang dilaporkan sebagai
	// baris yang ditolak beserta nomornya, bukan sebagai galat penguraian yang menyebut
	// posisi byte dan tidak berarti apa pun bagi pengguna.
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	record, err := reader.ReadAll()
	if err != nil {
		return nil, NewValidationError([]Violation{{
			Field:   "berkas",
			Message: "Berkas tidak dapat dibaca sebagai CSV. Pastikan disimpan sebagai CSV, bukan XLSX.",
		}})
	}
	if len(record) == 0 {
		return nil, NewValidationError([]Violation{{
			Field:   "berkas",
			Message: "Berkas kosong.",
		}})
	}

	index, err := readUploadHeader(record[0])
	if err != nil {
		return nil, err
	}

	rows := make([]SalvageUploadRow, 0, len(record)-1)
	for offset, line := range record[1:] {
		if blankUploadLine(line) {
			// Baris kosong di akhir berkas adalah hal biasa pada berkas hasil Excel.
			continue
		}
		at := func(column string) string {
			position, known := index[column]
			if !known || position >= len(line) {
				return ""
			}
			return strings.TrimSpace(line[position])
		}
		rows = append(rows, SalvageUploadRow{
			LineNumber:   offset + 2,
			ClaimNo:      at("claimno"),
			DateOfLoss:   at("dateofloss"),
			Currency:     at("currency"),
			SalvageOS:    at("salvageos"),
			SalvageAksep: at("salvageaksep"),
			CauseOfLoss:  at("causeofloss"),
		})
	}
	if len(rows) == 0 {
		return nil, NewValidationError([]Violation{{
			Field:   "berkas",
			Message: "Berkas tidak memuat satu pun baris data.",
		}})
	}
	return rows, nil
}

// Reject menyatakan baris ini tidak dapat disisipkan, beserta sebabnya.
func (r SalvageUploadRow) Reject(message string) RejectedUploadRow {
	return RejectedUploadRow{LineNumber: r.LineNumber, ClaimNo: r.ClaimNo, Message: message}
}

// ShapeViolation memeriksa isian satu baris dan mengembalikan sebab penolakannya.
//
// # Kenapa justru Cause Of Loss dan Date Of Loss
//
// Karena itulah yang diperiksa sistem lama, dan hanya itu:
// `Activity/ConvertDataCsvSalvageMBUToPage-Act.xml:905` melewati baris yang
// `.CauseOfLoss ==""||.DateOfLoss ==""`. Keduanya kunci baris pada grid rincian — baris
// tanpa keduanya tidak akan pernah terlihat di layar mana pun.
//
// Nomor klaim TIDAK ikut diperiksa, dan itu menuruti sistem lama apa adanya (`P-5`).
func (r SalvageUploadRow) ShapeViolation() string {
	switch {
	case r.CauseOfLoss == "" && r.DateOfLoss == "":
		return "Cause Of Loss dan Date Of Loss kosong."
	case r.CauseOfLoss == "":
		return "Cause Of Loss kosong."
	case r.DateOfLoss == "":
		return "Date Of Loss kosong."
	}
	return ""
}

// Amounts mengubah kedua nilai salvage menjadi angka.
//
// # Kenapa koma diganti titik lebih dulu
//
// Karena sistem lama melakukannya: `@toDecimal(@replaceAll(.SALVAGEOS,",","."))`. Berkas
// yang disusun di Excel berlokal Indonesia memakai KOMA sebagai pemisah desimal, dan
// tanpa penggantian itu `1,5` terbaca gagal — atau, lebih buruk, terbaca `15`.
//
// Pemisah ribuan TIDAK ditangani, juga menuruti sistem lama: pada `1.234,56` penggantian
// menghasilkan `1.234.56` yang memang tidak dapat diubah menjadi angka, dan barisnya
// ditolak beserta sebabnya — bukan disimpan dengan nilai yang salah.
func (r SalvageUploadRow) Amounts() (outstanding float64, accepted float64, err error) {
	parse := func(label, raw string) (float64, error) {
		clean := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
		if clean == "" {
			return 0, nil
		}
		value, convErr := strconv.ParseFloat(clean, 64)
		if convErr != nil {
			return 0, fmt.Errorf("%s bukan angka: %q", label, raw)
		}
		return value, nil
	}

	outstanding, err = parse("Salvage OS", r.SalvageOS)
	if err != nil {
		return 0, 0, err
	}
	accepted, err = parse("Salvage Aksep", r.SalvageAksep)
	if err != nil {
		return 0, 0, err
	}
	return outstanding, accepted, nil
}

// SalvageInsert adalah satu baris siap sisip ke POOLDATA.T_SALVAGE_MBU.
//
// Nama fieldnya mengikuti ARTI kolomnya, bukan nama kolom tabelnya. Sistem lama mengisi
// kolom `DATEOFLOSS` lewat properti bernama `District`
// (`Activity/ConvertDataCsvSalvageMBUToPage-Act.xml:1377`) — alias menyesatkan seperti
// yang `D-19` larang dibawa.
type SalvageInsert struct {
	ClaimNo      string
	DateOfLoss   string
	CurrencyID   string
	SalvageOS    float64
	SalvageAksep float64
	CauseOfLoss  string

	// BusinessGroupID selalu MBU. Sistem lama pun menuliskannya sebagai literal `"10004"`
	// (`:1464`), bukan membacanya dari berkas — unggahan ini memang khusus MBU.
	BusinessGroupID string
}

// readUploadHeader memetakan judul kolom ke posisinya.
//
// Spasi, garis bawah, dan tanda hubung di dalam judul diabaikan: "Claim No", "claim_no",
// dan "ClaimNo" adalah kolom yang sama bagi pengguna.
func readUploadHeader(line []string) (map[string]int, error) {
	index := map[string]int{}
	for position, title := range line {
		clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(title, bom)))
		clean = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(clean)
		if clean == "" {
			continue
		}
		if _, duplicate := index[clean]; duplicate {
			return nil, NewValidationError([]Violation{{
				Field:   "berkas",
				Message: fmt.Sprintf("Judul kolom %q muncul dua kali.", clean),
			}})
		}
		index[clean] = position
	}

	// Seluruh kolom yang hilang disebut sekaligus, bukan satu per satu. Pengguna yang
	// berkasnya kurang tiga kolom tidak perlu mengunggah tiga kali untuk mengetahuinya.
	var missing []string
	for _, column := range salvageUploadColumns {
		if _, known := index[column]; !known {
			missing = append(missing, column)
		}
	}
	if len(missing) > 0 {
		return nil, NewValidationError([]Violation{{
			Field: "berkas",
			Message: fmt.Sprintf(
				"Judul kolom belum lengkap. Yang belum ada: %s. Unduh berkas contoh lewat Format MBU Salvage.",
				strings.Join(missing, ", ")),
		}})
	}
	return index, nil
}

// blankUploadLine menyatakan seluruh sel baris ini kosong.
func blankUploadLine(line []string) bool {
	for _, cell := range line {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// detectSeparator memilih pemisah kolom dari baris judul.
//
// Excel berlokal Indonesia menyimpan CSV dengan TITIK KOMA. Memaksa koma berarti seluruh
// baris judul terbaca sebagai satu kolom, dan pesan galatnya menyesatkan sepenuhnya.
func detectSeparator(text string) rune {
	head := text
	if cut := strings.IndexAny(head, "\r\n"); cut >= 0 {
		head = head[:cut]
	}
	if strings.Count(head, ";") > strings.Count(head, ",") {
		return ';'
	}
	return ','
}
