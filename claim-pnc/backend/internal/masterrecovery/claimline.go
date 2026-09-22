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
// Rule `DownloadFileCSVFormaatter` yang menyediakan berkas contohnya TIDAK ADA di export
// — ia satu dari ±242 rule yang hilang (`R-16`), sehingga pemisah dan judul kolom
// sebenarnya tidak dapat dibaca. Yang ditegakkan di sini karena itu adalah bentuk yang
// dapat disimpulkan dari grid-nya, dan pembacaannya dibuat LAPANG dengan sengaja:
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
		if len(row) < 2 {
			violation = append(violation, Violation{
				Field:   FieldClaimLine,
				Message: "Baris " + number + " hanya memuat satu kolom; harus nomor polis dan nilai klaim.",
			})
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

		amount, err := ParseAmount(row[1])
		if err != nil {
			message := "Baris " + number + ": nilai klaim bukan angka rupiah utuh."
			if errors.Is(err, ErrAmountEmpty) {
				message = "Baris " + number + " tidak menyebut nilai klaim."
			}
			violation = append(violation, Violation{Field: FieldClaimLine, Message: message})
			continue
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
// Dikenali dari kolom KEDUA yang bukan angka: baris data selalu memuat nilai klaim di
// sana. Mencocokkan teks judulnya akan menuntut mengetahui judul yang sebenarnya — dan
// rule yang menyediakan berkas contohnya tidak ada di export.
func headerRow(row []string) bool {
	if len(row) < 2 {
		return true
	}
	_, err := ParseAmount(row[1])
	return err != nil
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
// Menggantikan rule `DownloadFileCSVFormaatter`, yang tidak ada di export sehingga isi
// contohnya tidak dapat ditiru persis. Yang ditiru adalah KOLOMNYA, yang terbaca dari
// grid unggahan.
func ClaimLineTemplate() string {
	var builder strings.Builder
	builder.WriteString("No Polis;Nilai Klaim\n")
	// Dua baris contoh, bukan nol: berkas berisi judul saja membuat petugas menebak
	// apakah angkanya boleh berpemisah ribuan. Contoh menjawabnya tanpa perlu bertanya.
	builder.WriteString("POL-0000001;1000000\n")
	builder.WriteString("POL-0000002;2500000\n")
	return builder.String()
}
