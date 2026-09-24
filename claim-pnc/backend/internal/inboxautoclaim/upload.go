package inboxautoclaim

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MaxUploadRow membatasi jumlah baris yang diterima satu unggahan.
//
// Batasnya ada supaya satu berkas keliru tidak menahan seluruh isi tabel di memori
// sekaligus. Angkanya bukan aturan bisnis — sistem lama tidak punya batas sama sekali —
// melainkan pagar operasional.
const MaxUploadRow = 5000

// Kolom berkas unggahan.
//
// # Dari mana nama-nama ini berasal
//
// Dari `Activity/InsertKlaimToTable_Other-Act.xml`, yang membaca hasil `pxUploadCSVResults`
// lewat properti bernama sama. Versi pertama modul ini MENGARANG judul kolomnya dari nama
// kolom tabel, karena flow action `PNCUploadClaimCSV` belum ada di export; ia tiba
// 2026-09-19 dan daftar di bawah menggantikannya.
//
// # Tiga kolom yang DULU saya wajibkan dan ternyata TIDAK ADA di berkas
//
//	inisialid  kode perusahaan  -> diturunkan dari polis, bukan diketik pengunggah
//	prodke     nomor produk     -> dicari dari polis
//	currency   mata uang        -> diambil dari snapshot polis
//
// Ketiganya hasil lookup, bukan isian. Meminta pengunggah mengisinya berarti meminta nilai
// yang tidak ia ketahui — dan yang lebih buruk, membiarkannya salah.
const (
	ColumnPolicyNo     = "policyno"
	ColumnClaimAmount  = "claimamount"
	ColumnDateOfLoss   = "dateofloss"
	ColumnReportDate   = "reportdate"
	ColumnCauseOfLoss  = "causeofloss"
	ColumnKeyword      = "keyword"
	ColumnReason       = "alasanklaim"
	ColumnObjectName   = "objectname"
	ColumnFlagNoPayout = "flagtidakbayar"
	ColumnContractNo   = "contractno"
)

// RequiredUploadColumn adalah judul kolom yang WAJIB ada di berkas.
//
// `causeofloss` sengaja TIDAK wajib: `Activity/CreateCasePNC_AutoClaim-Act.xml` menangani
// penyebab kerugian kosong sebagai KEGAGALAN BARIS ("Penyebab kerugian tidak ditemukan"),
// bukan sebagai penolakan berkas. Menolaknya di sini mengubah perilaku yang sudah ada.
var RequiredUploadColumn = []string{
	ColumnPolicyNo,
	ColumnClaimAmount,
	ColumnDateOfLoss,
	ColumnReportDate,
}

// OptionalUploadColumn adalah judul kolom yang boleh ada dan boleh tidak.
var OptionalUploadColumn = []string{
	ColumnCauseOfLoss,
	ColumnKeyword,
	ColumnReason,
	ColumnObjectName,
	ColumnFlagNoPayout,
	ColumnContractNo,
}

// Pesan hasil pemrosesan satu baris.
//
// Keenam teks di bawah DISALIN HARFIAH dari `Activity/InsertKlaimToTable_Other-Act.xml`.
// Ia bukan pesan karangan: teks inilah yang tersimpan di kolom TMP_MESSAGE, yang terbaca
// petugas di grid, dan yang keluar di berkas ekspor GAGAL. Mengubah satu huruf pun membuat
// baris lama dan baris baru tidak dapat dibandingkan.
const (
	// MessageReceiverNotFound: polis tidak menunjuk perusahaan rekanan mana pun yang aktif
	// di master. Satu-satunya kegagalan yang membuat baris TIDAK disisipkan — tanpa kode
	// perusahaan, barisnya tidak punya tempat di grid.
	MessageReceiverNotFound = "Sumber Bisnis Tidak ditemukan"

	// MessagePolicyNotFound: polisnya tidak ada di JSON_POLIS.
	MessagePolicyNotFound = "No Polis tidak di temukan"

	// MessageObjectNotFound: prodke tidak dapat diturunkan dari polis.
	MessageObjectNotFound = "Objek tidak di temukan"

	// MessageReportBeforeLoss: tanggal lapor mendahului tanggal kejadian.
	MessageReportBeforeLoss = "Tanggal lapor harus setelah tanggal kejadian"

	// MessageLossOutsidePolicy: DOL di luar periode polis.
	//
	// BELUM DAPAT DIPERIKSA modul ini — ia menuntut periode polis dari snapshot (B-1).
	// Konstantanya tetap ada supaya baris lama yang sudah memuat pesan ini tetap dikenali
	// sebagai gagal, dan supaya pemeriksaannya kelak memakai teks yang sama persis.
	MessageLossOutsidePolicy = "Tanggal kejadian tidak dalam range polis"

	// MessagePremiumUnpaid: premi belum lunas.
	//
	// BELUM DAPAT DIPERIKSA modul ini — `CekPremiAutoKlaim` tidak ada di export.
	MessagePremiumUnpaid = "Premi belum lunas"
)

// UploadRow adalah satu baris berkas unggahan setelah dibaca dan dibersihkan.
type UploadRow struct {
	// LineNumber adalah nomor baris di dalam berkas, terhitung sejak baris judul.
	//
	// Ia dibawa sampai ke laporan hasil karena yang diperbaiki pengguna adalah berkasnya.
	LineNumber int

	PolicyNo    string
	ClaimAmount string
	DateOfLoss  string
	ReportDate  string
	CauseOfLoss string
	Keyword     string

	// Reason adalah kolom `alasanklaim`, disimpan ke NOTE.
	Reason string

	ObjectName   string
	FlagNoPayout string
	ContractNo   string
}

// Resolution adalah hasil pencarian polis untuk satu baris unggahan.
//
// Ketiganya TIDAK berasal dari berkas: perusahaan diturunkan dari
// `t_general.sourceofbusiness`, prodke dari `json_polis`, dan keduanya dicari per baris
// karena satu berkas dapat memuat polis dari beberapa perusahaan sekaligus.
type Resolution struct {
	CompanyCode string
	CompanyName string
	ProductSeq  string

	// Message berisi pesan kegagalan bila pencariannya tidak lengkap; kosong bila lolos.
	Message string
}

// UploadLine adalah satu baris yang siap disisipkan, beserta hasil pemeriksaannya.
type UploadLine struct {
	Row         UploadRow
	CompanyCode string
	CompanyName string
	ProductSeq  string

	// Message kosong berarti baris lolos dan akan diproses menjadi klaim. Terisi berarti
	// baris tetap disisipkan tetapi langsung terhitung GAGAL di grid.
	Message string
}

// Accepted menyatakan baris ini lolos seluruh pemeriksaan yang dapat dijalankan.
func (l UploadLine) Accepted() bool { return strings.TrimSpace(l.Message) == "" }

// BatchRef menyebut satu batch yang terbentuk dari sebuah unggahan.
type BatchRef struct {
	CompanyCode string
	CompanyName string
	BatchNumber string
	Rows        int

	// Succeeded dan Failed adalah hasil pemeriksaan SAAT UNGGAH, bukan hasil pemrosesan
	// menjadi klaim. Baris "Succeeded" di sini berarti lolos dan MENUNGGU diproses.
	Succeeded int
	Failed    int
}

// UploadResult adalah ringkasan satu unggahan.
//
// Satu berkas dapat menghasilkan beberapa batch: kode perusahaan diturunkan per baris dari
// polisnya, sehingga satu berkas berisi polis dari dua perusahaan menghasilkan dua batch.
type UploadResult struct {
	Batch []BatchRef
	Rows  int

	// Rejected adalah baris yang TIDAK disisipkan sama sekali, beserta alasannya.
	//
	// Hanya satu sebab yang sampai ke sini: perusahaannya tidak dapat diturunkan dari
	// polis. Tanpa kode perusahaan, baris itu tidak punya tempat di grid mana pun —
	// menyisipkannya berarti membuat baris yang tidak akan pernah terlihat siapa pun.
	Rejected []RejectedRow
}

// RejectedRow adalah satu baris berkas yang tidak dapat disisipkan.
type RejectedRow struct {
	LineNumber int
	PolicyNo   string
	Message    string
}

// ParseUpload membaca berkas CSV menjadi baris unggahan.
//
// Ia HANYA membaca dan memeriksa BENTUKNYA; pemeriksaan terhadap polis ada di lapisan
// aplikasi, karena ia menyentuh basis data. Keduanya dipisah supaya galat bentuk berkas
// ("judul kolom policyno tidak ada") tidak tercampur dengan galat isi — pengguna
// memperbaiki keduanya dengan cara yang berbeda.
//
// Pemisah kolom dikenali sendiri: koma dan titik koma sama-sama diterima, karena Excel
// berlokal Indonesia menyimpan CSV dengan titik koma dan itulah perkakas yang benar-benar
// dipakai petugas.
func ParseUpload(source io.Reader) ([]UploadRow, error) {
	content, err := io.ReadAll(io.LimitReader(source, maxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("inboxautoclaim: berkas tidak dapat dibaca: %w", err)
	}
	if len(content) > maxUploadBytes {
		return nil, &ValidationError{Violation: []Violation{{
			Field:   "berkas",
			Message: fmt.Sprintf("Berkas terlalu besar. Batasnya %d MB.", maxUploadBytes/(1<<20)),
		}}}
	}

	text := strings.TrimPrefix(string(content), "\ufeff")

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = detectSeparator(text)
	// Jumlah kolom tidak dipaksa sama: baris yang kolomnya kurang dilaporkan sebagai
	// pelanggaran berisi nomor barisnya, bukan sebagai galat penguraian yang menyebut
	// posisi byte dan tidak berarti apa pun bagi pengguna.
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	record, err := reader.ReadAll()
	if err != nil {
		return nil, &ValidationError{Violation: []Violation{{
			Field:   "berkas",
			Message: "Berkas tidak dapat dibaca sebagai CSV. Pastikan berkas disimpan sebagai CSV, bukan XLSX.",
		}}}
	}
	if len(record) == 0 {
		return nil, ErrEmptyUpload
	}

	index, err := readHeader(record[0])
	if err != nil {
		return nil, err
	}

	row := make([]UploadRow, 0, len(record)-1)
	for number, line := range record[1:] {
		if blankLine(line) {
			// Baris kosong di akhir berkas adalah hal biasa pada berkas hasil Excel.
			continue
		}
		row = append(row, UploadRow{
			// +2: satu untuk baris judul, satu karena manusia menghitung dari 1.
			LineNumber: number + 2,
			// Titik dibuang dari nomor polis, mengikuti
			// `InputParam.PolicyNo = @replaceAll(.PolicyNo,".","")`.
			PolicyNo:     strings.ToUpper(strings.ReplaceAll(pick(line, index, ColumnPolicyNo), ".", "")),
			ClaimAmount:  pick(line, index, ColumnClaimAmount),
			DateOfLoss:   pick(line, index, ColumnDateOfLoss),
			ReportDate:   pick(line, index, ColumnReportDate),
			CauseOfLoss:  pick(line, index, ColumnCauseOfLoss),
			Keyword:      pick(line, index, ColumnKeyword),
			Reason:       pick(line, index, ColumnReason),
			ObjectName:   pick(line, index, ColumnObjectName),
			FlagNoPayout: pick(line, index, ColumnFlagNoPayout),
			ContractNo:   pick(line, index, ColumnContractNo),
		})
	}

	if len(row) == 0 {
		return nil, ErrEmptyUpload
	}
	if len(row) > MaxUploadRow {
		return nil, &ValidationError{Violation: []Violation{{
			Field: "berkas",
			Message: fmt.Sprintf(
				"Berkas memuat %d baris; paling banyak %d baris sekali unggah. Pecah berkasnya.",
				len(row), MaxUploadRow),
		}}}
	}
	return row, nil
}

// maxUploadBytes membatasi besar berkas yang dibaca ke memori.
const maxUploadBytes = 8 << 20

// CheckUploadShape memeriksa bentuk isian setiap baris dan mengembalikan SEMUA
// pelanggarannya.
//
// # Apa yang diperiksa di sini, dan apa yang TIDAK
//
// Di sini hanya yang dapat dinilai tanpa menyentuh basis data: isian wajib terisi, tanggal
// berbentuk `dd/mm/yyyy`, nilai klaim berupa angka. Berkas yang gagal di sini DITOLAK
// SELURUHNYA — ia belum berbentuk berkas klaim sama sekali.
//
// Yang menyangkut polis — perusahaan, prodke, periode, premi — TIDAK diperiksa di sini.
// Kegagalannya tidak menolak berkas; barisnya tetap disisipkan dengan pesan pada
// TMP_MESSAGE, persis seperti `InsertKlaimToTable_Other`.
func CheckUploadShape(row []UploadRow) error {
	var violation []Violation

	for _, r := range row {
		at := func(column string) string {
			return fmt.Sprintf("baris %d · %s", r.LineNumber, column)
		}

		if r.PolicyNo == "" {
			violation = append(violation, Violation{
				Field:   at(ColumnPolicyNo),
				Message: "Nomor polis wajib diisi.",
			})
		}

		violation = append(violation, checkDate(at(ColumnDateOfLoss), "Tanggal kejadian", r.DateOfLoss)...)
		violation = append(violation, checkDate(at(ColumnReportDate), "Tanggal lapor", r.ReportDate)...)

		switch {
		case r.ClaimAmount == "":
			violation = append(violation, Violation{
				Field:   at(ColumnClaimAmount),
				Message: "Nilai klaim wajib diisi.",
			})
		case !decimalText(r.ClaimAmount):
			violation = append(violation, Violation{
				Field:   at(ColumnClaimAmount),
				Message: "Nilai klaim harus berupa angka tanpa pemisah ribuan. Pakai titik untuk desimal.",
			})
		}
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// CheckDateOrder memeriksa tanggal lapor tidak mendahului tanggal kejadian.
//
// Ia TERPISAH dari CheckUploadShape karena akibatnya berbeda: pelanggarannya tidak menolak
// berkas melainkan menandai barisnya gagal, mengikuti
// `Activity/InsertKlaimToTable_Other-Act.xml` langkah "tgl lapor harus setelah tgl
// kejadian".
//
// Perbandingannya dilakukan pada TEKS yang sudah dibalik menjadi `yyyymmdd`, bukan dengan
// mengubahnya menjadi waktu. Dua sebab: kolomnya memang teks, dan mengubahnya menjadi waktu
// menuntut zona waktu yang justru tidak boleh ikut menentukan (F-5).
//
// Nilai kembalinya pesan kegagalan, atau teks kosong bila lolos.
func CheckDateOrder(dateOfLoss, reportDate string) string {
	loss, lossOK := sortableDate(dateOfLoss)
	report, reportOK := sortableDate(reportDate)
	if !lossOK || !reportOK {
		// Bentuknya sudah dijamin CheckUploadShape; bila sampai di sini tetap tidak
		// terbaca, membiarkannya lolos lebih aman daripada menandainya gagal atas dasar
		// yang tidak dapat dipastikan.
		return ""
	}
	if report < loss {
		return MessageReportBeforeLoss
	}
	return ""
}

// sortableDate mengubah `dd/mm/yyyy` menjadi `yyyymmdd` yang dapat dibandingkan sebagai teks.
func sortableDate(value string) (string, bool) {
	if !dateText(value) {
		return "", false
	}
	return value[6:10] + value[3:5] + value[0:2], true
}

// checkDate memeriksa bentuk tanggal dd/mm/yyyy.
//
// # Kenapa bentuknya diperiksa, padahal nilainya disimpan apa adanya sebagai teks
//
// Karena `Activity/CreateCasePNC_AutoClaim-Act.xml` memotongnya dengan posisi karakter
// tetap lalu menempelkan "T000000.000 GMT". Teks yang bentuknya lain tidak menghasilkan
// galat — ia menghasilkan TANGGAL LAIN, diam-diam. "2026-09-19" terbaca sebagai tanggal
// "09-20" tahun "26-0", dan tidak ada apa pun yang memberitahukannya.
//
// DDL yang diterima 2026-09-19 membenarkan premisnya: TGLKEJADIAN dan TGLLAPOR keduanya
// VARCHAR2(50).
func checkDate(field, label, value string) []Violation {
	if value == "" {
		return []Violation{{Field: field, Message: label + " wajib diisi."}}
	}
	if !dateText(value) {
		return []Violation{{
			Field:   field,
			Message: label + " harus berformat dd/mm/yyyy, misalnya 17/09/2026.",
		}}
	}
	return nil
}

// dateText memeriksa bentuk dd/mm/yyyy beserta kewajaran angkanya.
//
// Tanggalnya TIDAK diubah menjadi time.Time dan tidak diperiksa terhadap kalender penuh
// (29 Februari, misalnya). Alasannya sengaja: pemeriksaan kalender menuntut penafsiran
// zona waktu, dan modul ini tidak boleh menafsirkan tanggal sama sekali.
func dateText(value string) bool {
	if len(value) != 10 || value[2] != '/' || value[5] != '/' {
		return false
	}
	day, errDay := strconv.Atoi(value[0:2])
	month, errMonth := strconv.Atoi(value[3:5])
	year, errYear := strconv.Atoi(value[6:10])
	if errDay != nil || errMonth != nil || errYear != nil {
		return false
	}
	return day >= 1 && day <= 31 && month >= 1 && month <= 12 && year >= 1900
}

// decimalText memeriksa teks berupa bilangan desimal tanpa pemisah ribuan.
//
// Nilainya TIDAK diubah menjadi float: I-12 menuntut nilai uang disimpan presisi penuh,
// dan float64 tidak dapat mewakili setiap nilai desimal dengan tepat.
func decimalText(value string) bool {
	digitSeen := false
	dotSeen := false

	body := value
	if strings.HasPrefix(body, "-") {
		body = body[1:]
	}

	for _, character := range body {
		switch {
		case character >= '0' && character <= '9':
			digitSeen = true
		case character == '.':
			if dotSeen {
				return false
			}
			dotSeen = true
		default:
			return false
		}
	}
	return digitSeen
}

// readHeader memetakan judul kolom ke posisinya.
func readHeader(line []string) (map[string]int, error) {
	index := map[string]int{}
	for position, title := range line {
		// Spasi, garis bawah, dan tanda hubung di dalam judul diabaikan: "Policy No",
		// "policy_no", dan "PolicyNo" adalah kolom yang sama bagi pengguna, dan menolak
		// salah satunya hanya karena spasi tidak menolong siapa pun.
		clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(title, "\ufeff")))
		clean = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(clean)
		if clean == "" {
			continue
		}
		if _, duplicate := index[clean]; duplicate {
			return nil, &ValidationError{Violation: []Violation{{
				Field:   "berkas",
				Message: fmt.Sprintf("Judul kolom %q muncul dua kali.", clean),
			}}}
		}
		index[clean] = position
	}

	var missing []string
	for _, column := range RequiredUploadColumn {
		if _, exists := index[column]; !exists {
			missing = append(missing, column)
		}
	}
	if len(missing) > 0 {
		return nil, &ValidationError{Violation: []Violation{{
			Field: "berkas",
			Message: fmt.Sprintf("Judul kolom yang wajib belum ada: %s.",
				strings.Join(missing, ", ")),
		}}}
	}
	return index, nil
}

// pick mengambil satu nilai kolom, atau teks kosong bila kolomnya tidak ada.
func pick(line []string, index map[string]int, column string) string {
	position, exists := index[column]
	if !exists || position >= len(line) {
		return ""
	}
	return strings.TrimSpace(line[position])
}

// blankLine menyatakan seluruh sel pada baris ini kosong.
func blankLine(line []string) bool {
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

// GroupUploadLine mengelompokkan baris yang sudah dicari perusahaannya.
//
// Urutan perusahaan mengikuti kemunculan pertamanya di berkas, bukan urutan abjad, supaya
// ringkasan yang dilihat pengguna sesudah mengunggah terbaca searah dengan berkasnya.
func GroupUploadLine(line []UploadLine) ([]string, map[string][]UploadLine) {
	var order []string
	group := map[string][]UploadLine{}

	for _, l := range line {
		code := strings.ToUpper(strings.TrimSpace(l.CompanyCode))
		if _, seen := group[code]; !seen {
			order = append(order, code)
		}
		group[code] = append(group[code], l)
	}
	return order, group
}
