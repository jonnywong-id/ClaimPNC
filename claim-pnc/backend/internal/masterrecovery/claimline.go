package masterrecovery

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
)

// Pembacaan berkas CSV "Upload Data Klaim".
//
// # Kenapa ia di lapisan domain
//
// Yang dikerjakan di sini bukan urusan HTTP maupun SQL, melainkan ATURAN: kolom apa yang
// diharapkan, baris mana yang sah, dan bagaimana angka dibaca. Menaruhnya di transport
// akan membuat aturan yang sama ditulis ulang oleh setiap pemanggil baru — impor batch,
// misalnya — dan ditulis sedikit berbeda setiap kali.
//
// `encoding/csv` dan `io` bukan HTTP, bukan SQL, dan bukan driver basis data, sehingga
// aturan ketergantungan lapisan (`08-TECHNICAL-STRATEGY.md` §2) tidak dilanggar.
//
// # Bentuk berkas, dan dari mana bentuknya diketahui
//
// Grid unggahan di `Section/OutstandingMasterRecovery-Section.xml` memuat TEPAT DUA
// kolom, keduanya read-only:
//
//	:3783  "No Polis"     → .PolicyNo
//	:3872  "Nilai Klaims" → .ClaimAmount   (pxNumber)
//
// Rule `DownloadFileCSVFormaatter` yang menyediakan berkas contohnya ADA di export
// (`Activity/DownloadFileCSVFormaatter-Act.xml`, Applies To `Data-Portal`), dan isinya
// mengungkap ketidakcocokan yang perlu diketahui:
//
//	langkah 2  TempsFormaater.pxResults(<APPEND>).PolicyNo := "TESTINg…"   ← SATU kolom
//	langkah 3  Call pxConvertResultsToCSV, FileName "Fromat Recovery Klaim"
//
// **Berkas contohnya hanya satu kolom — `PolicyNo` — sedangkan grid unggahannya dua**
// (`:3783` "No Polis" dan `:3872` "Nilai Klaims"). Judulnya pun dibentuk
// `pxConvertResultsToCSV` dari nama properti, bukan ditulis sebagai teks yang dibaca
// manusia.
//
// Keputusan Work Owner 2026-09-29: berkas contoh mengikuti Pega — SATU kolom. Yang
// menyesuaikan adalah pembacanya, yang memperlakukan kolom nilai klaim sebagai OPSIONAL
// dan mengisinya nol untuk dilengkapi petugas di layar. Kolom kedua yang terisi tetapi
// bukan angka tetap ditolak: itu salah ketik, bukan kolom yang tidak ada.
//
// Pembacaannya dibuat LAPANG dengan sengaja:
//
//   - Pemisah titik koma maupun koma sama-sama diterima. Excel berbahasa Indonesia
//     menulis titik koma; menolaknya akan membuat berkas yang diekspor petugas sendiri
//     ditolak aplikasi.
//   - Baris judul dikenali dan dilewati, tetapi tidak diwajibkan.
//   - Baris kosong dilewati — berkas Excel hampir selalu berakhir dengan beberapa.
//
// Yang TIDAK lapang adalah isinya: nomor polis kosong dan nilai yang bukan angka
// DITOLAK, dengan menyebut nomor barisnya. Menerimanya diam-diam akan menyimpan batch
// yang jumlahnya tidak pernah cocok dengan berkas asalnya.
const MaxClaimLineRows = 5000

// byteOrderMark adalah tiga bita yang Excel tuliskan di awal berkas CSV yang disimpannya
// sebagai UTF-8.
//
// Ia ditulis sebagai bita, bukan sebagai hurufnya sendiri: huruf itu tidak terlihat di
// penyunting mana pun, dan berkas sumber Go yang memuatnya di tengah baris ditolak
// kompilator. Membuangnya wajib — tanpa itu, nomor polis pada baris pertama terbaca
// membawa tiga bita tak terlihat di depannya dan tidak akan pernah cocok dengan polis mana
// pun.
const byteOrderMark = "\xef\xbb\xbf"

// ErrClaimLineEmpty dikembalikan bila berkas tidak memuat satu pun baris yang dapat
// dibaca. Dibedakan dari galat bentuk supaya layar dapat mengatakan "berkasnya kosong"
// alih-alih "berkasnya cacat".
var (
	ErrClaimLineEmpty      = errors.New("masterrecovery: berkas daftar klaim tidak memuat baris")
	ErrClaimLineTooMany    = errors.New("masterrecovery: berkas daftar klaim melebihi batas baris")
	ErrClaimLineUnreadable = errors.New("masterrecovery: berkas daftar klaim tidak dapat dibaca")
)

// ParseClaimLine membaca daftar polis dan nilai klaim dari sebuah berkas CSV.
//
// Nilai kedua memuat pelanggaran per baris. Baris yang melanggar TIDAK ikut dikembalikan,
// sehingga pemanggil tidak pernah menyimpan separuh berkas tanpa menyadarinya.
func ParseClaimLine(source io.Reader) ([]ClaimLine, []Violation, error) {
	content, err := io.ReadAll(io.LimitReader(source, MaxDocumentBytes+1))
	if err != nil {
		return nil, nil, ErrClaimLineUnreadable
	}
	if len(content) > MaxDocumentBytes {
		return nil, nil, ErrClaimLineTooMany
	}

	text := strings.TrimPrefix(string(content), byteOrderMark)
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = detectSeparator(text)
	// Jumlah kolom per baris tidak dipaksa seragam: berkas Excel kerap membawa kolom
	// kosong di kanan, dan menolak seluruh berkas karenanya tidak menolong siapa pun.
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	record, err := reader.ReadAll()
	if err != nil {
		return nil, nil, ErrClaimLineUnreadable
	}

	var (
		line      []ClaimLine
		violation []Violation
	)
	for index, row := range record {
		number := strconv.Itoa(index + 1)

		if blankRow(row) {
			continue
		}
		if index == 0 && headerRow(row) {
			continue
		}

		policyNo := strings.TrimSpace(row[0])
		if policyNo == "" {
			violation = append(violation, Violation{
				Field:   FieldClaimLine,
				Message: "Baris " + number + " tidak menyebut nomor polis.",
			})
			continue
		}
		if tooLong(policyNo, MaxPolicyNoLength) {
			violation = append(violation, Violation{
				Field:   FieldClaimLine,
				Message: "Baris " + number + ": nomor polis melebihi " + strconv.Itoa(MaxPolicyNoLength) + " karakter.",
			})
			continue
		}

		// Kolom kedua OPSIONAL, dan itu mengikuti berkas contoh Pega.
		//
		// `Activity/DownloadFileCSVFormaatter-Act.xml` hanya menuliskan satu kolom —
		// `PolicyNo` — sementara grid unggahannya memuat dua. Menuntut dua kolom di sini
		// akan membuat berkas contoh yang kita bagikan sendiri DITOLAK saat diunggah
		// kembali.
		//
		// Baris tanpa kolom kedua karena itu diterima dengan nilai nol, dan petugas
		// melengkapinya di layar. Yang tetap ditolak hanyalah kolom kedua yang TERISI
		// tetapi bukan angka — itu salah ketik, bukan kolom yang memang tidak ada.
		var amount Amount
		if len(row) >= 2 && strings.TrimSpace(row[1]) != "" {
			parsed, err := ParseAmount(row[1])
			if err != nil {
				violation = append(violation, Violation{
					Field:   FieldClaimLine,
					Message: "Baris " + number + ": nilai klaim bukan angka rupiah utuh.",
				})
				continue
			}
			amount = parsed
		}
		if amount < 0 {
			violation = append(violation, Violation{
				Field:   FieldClaimLine,
				Message: "Baris " + number + ": nilai klaim tidak boleh negatif.",
			})
			continue
		}

		line = append(line, ClaimLine{PolicyNo: policyNo, ClaimAmount: amount})
		if len(line) > MaxClaimLineRows {
			return nil, nil, ErrClaimLineTooMany
		}
	}

	if len(line) == 0 && len(violation) == 0 {
		return nil, nil, ErrClaimLineEmpty
	}
	return line, violation, nil
}

// TotalClaimAmount menjumlahkan nilai klaim seluruh baris.
//
// Dipakai layar untuk memperlihatkan jumlah berkas yang baru diunggah, sehingga petugas
// dapat membandingkannya dengan angka yang diketiknya sendiri sebelum menyimpan. Sistem
// lama tidak punya penjumlahan ini — ia perbaikan kecil yang tidak mengubah apa pun yang
// tersimpan.
func TotalClaimAmount(line []ClaimLine) Amount {
	var total Amount
	for _, l := range line {
		total += l.ClaimAmount
	}
	return total
}

// detectSeparator memilih pemisah kolom dari isi berkasnya sendiri.
//
// Baris pertama yang tidak kosong yang diperiksa, dan titik koma menang bila jumlahnya
// lebih banyak — itulah keluaran Excel pada lokal Indonesia. Menebaknya dari isi jauh
// lebih andal daripada memaksa satu bentuk: berkas yang sama dapat diekspor dua petugas
// dengan dua lokal berbeda.
func detectSeparator(text string) rune {
	for _, row := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(row)
		if trimmed == "" {
			continue
		}
		if strings.Count(trimmed, ";") > strings.Count(trimmed, ",") {
			return ';'
		}
		return ','
	}
	return ','
}

// headerRow mengenali baris judul supaya ia tidak terbaca sebagai data.
//
// # Kenapa dua cara, bukan satu
//
// Versi sebelumnya mengenalinya HANYA dari kolom kedua yang bukan angka. Itu runtuh
// begitu berkas satu kolom diterima (mengikuti berkas contoh Pega): berkas
// `POL-1\nPOL-2` akan membuat baris PERTAMA dikira judul, dan satu polis hilang tanpa
// satu pun pesan.
//
// Karena itu nama judulnya kini dicocokkan lebih dulu — termasuk `PolicyNo`, nama yang
// benar-benar dihasilkan `pxConvertResultsToCSV` pada berkas contoh Pega.
//
// # Satu keadaan yang memang tidak dapat dibedakan
//
// Berkas TANPA judul yang baris pertamanya bernilai klaim cacat — `POL-1;seribu` —
// tetap terbaca sebagai judul. Itu tidak terhindarkan tanpa menuntut judul, dan
// akibatnya terbatas: satu baris terlewat, bukan salah nilai. Berkas contoh yang
// dibagikan selalu berjudul, sehingga keadaan ini praktis hanya muncul pada berkas yang
// disusun tangan.
func headerRow(row []string) bool {
	if len(row) == 0 {
		return true
	}
	if headerName[normalizeHeader(row[0])] {
		return true
	}
	// Berkas satu kolom yang namanya tidak dikenali diperlakukan sebagai DATA, bukan
	// judul — inilah yang menyelamatkan baris pertama berkas tanpa judul.
	if len(row) < 2 || strings.TrimSpace(row[1]) == "" {
		return false
	}
	_, err := ParseAmount(row[1])
	return err != nil
}

// headerName memuat nama judul kolom pertama yang dikenali.
//
// `policyno` adalah yang dihasilkan `pxConvertResultsToCSV` dari nama properti; sisanya
// bentuk yang wajar diketik orang. Dicocokkan setelah dinormalkan, sehingga "No Polis",
// "no_polis", dan "NOPOLIS" sama-sama dikenali.
var headerName = map[string]bool{
	"policyno":   true,
	"nopolis":    true,
	"nomorpolis": true,
	"nomerpolis": true,
}

func normalizeHeader(cell string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(cell)) {
		if r == ' ' || r == '_' || r == '-' || r == '.' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func blankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// ClaimLineTemplate adalah isi berkas contoh yang diunduh tombol "Format File".
//
// Ia dibangun di sini, bukan disimpan sebagai berkas statis, supaya bentuk yang
// DIUNDUH dan bentuk yang DIBACA tidak dapat berbeda pendapat — keduanya berasal dari
// satu berkas sumber ini.
//
// Menggantikan rule `DownloadFileCSVFormaatter`, dan mengikutinya apa adanya.
//
// # Satu kolom, karena begitulah contoh Pega (keputusan Work Owner 2026-09-29)
//
// `Activity/DownloadFileCSVFormaatter-Act.xml` hanya berisi tiga langkah nyata:
//
//	Page-New       TempsFormaater
//	Property-Set   TempsFormaater.pxResults(<APPEND>).PolicyNo := "TESTINg…"
//	Call           pxConvertResultsToCSV   FileName "Fromat Recovery Klaim"
//
// Satu kolom, satu baris contoh. Versi sebelumnya di sini menambahkan kolom kedua "Nilai
// Klaim" karena grid unggahannya memang dua kolom — itu menyimpang dari Pega, dan
// penyimpangan itu ditarik.
//
// Yang membuat penarikan ini aman: pembacanya di atas kini menerima berkas satu kolom,
// dengan nilai klaim nol yang dilengkapi petugas di layar. Jadi berkas contoh ini tetap
// dapat diunggah kembali — hal yang, di sistem lama, tidak berlaku.
//
// Judul kolomnya memakai nama properti `PolicyNo`, sama seperti keluaran
// `pxConvertResultsToCSV` yang membentuk judul dari nama properti — bukan teks yang
// dibaca manusia.
func ClaimLineTemplate() string {
	var builder strings.Builder
	builder.WriteString("PolicyNo\n")
	// Satu baris contoh, sama seperti Pega. Nilainya diganti nomor yang jelas contoh —
	// nilai asli pada rule lama adalah nomor polis nyata, dan `D-69` melarang menuliskan
	// data nasabah ke berkas yang di-commit.
	builder.WriteString("CONTOH-POLIS-0001\n")
	return builder.String()
}
