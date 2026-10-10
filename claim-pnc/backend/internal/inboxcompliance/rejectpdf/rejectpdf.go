// Package rejectpdf membentuk PDF "Surat Penolakan dan Penarikan Dana Klaim" — tombol
// "Generate PDF" pada form Download Dokumen Reject.
//
// # Sumbernya templat Pega, bukan rancangan kami
//
// Susunan, urutan baris, dan seluruh teks tetapnya disalin dari
// `HTML/RejectRefundLetter-HTML.xml` (`pySourceStream`). Kalimat tetapnya dibawa APA
// ADANYA, termasuk tanda baca dan nomor rekeningnya — `D-13` menetapkan teks mengikuti
// Pega, dan surat ini keluar ke nasabah.
//
// Suratnya DUA HALAMAN. Itu bukan tafsir: templat memasang
// `page-break-before: always` pada posisi 5457, dan kedua halaman memang berbeda isi —
// halaman pertama ditujukan ke pemegang polis, halaman kedua ke pesertanya.
//
// # Suratnya memang SEBAGIAN BESAR kosong, dan itu bentuk yang benar
//
// Pada templat ini **sel nilainya kosong**. Baris a sampai e hanya memuat labelnya,
// alasan penolakan hanya memuat angka `1.`, `2.`, `3.` tanpa isi, dan `Up.` beserta
// `Jabatan.` adalah teks tetap — tidak ada satu pun `pega:reference` ke
// `RejectCompliance.*` di seluruh berkas. Yang terisi hanya ENAM merge field: tanggal
// surat, nomor surat, nama dan alamat tertanggung, serta nama objek pertanggungan.
//
// Artinya surat ini **tidak memuat satu pun isian yang baru saja diketik petugas** pada
// form Download Dokumen Reject. Itu terlihat seperti templat yang belum selesai
// diwiring — dan versi pertama paket ini memang mengisinya atas dugaan itu.
//
// **Work Owner menetapkan sebaliknya pada 2026-10-08:** templat Pega diikuti apa adanya,
// karena ia sudah merupakan format yang sesuai. Pengisian itu dicabut.
//
// Bacaan yang sejalan dengan ketetapan itu sudah terlihat sejak awal di templatnya
// sendiri: `Rp._____________` pada paragraf pengembalian dana adalah garis isian tangan.
// Suratnya memang **dicetak lalu dilengkapi tangan**, bukan dirakit penuh oleh sistem.
//
// # Yang digambar kosong, dan tetap digambar
//
//	baris a sampai e     label dan titik dua, tanpa nilai
//	daftar alasan        tepat tiga baris: "1." "2." "3."
//	Up. / Jabatan.       teks tetap berakhir titik
//	Rp._____________     garis bawah apa adanya
//	alamat tertanggung   satu baris kosong bila tidak terbaca
//
// Tidak satu pun dihilangkan. Baris yang hilang membuat surat terbaca seolah memang tidak
// punya isian itu, sedangkan baris kosong menyediakan tempat untuk diisi. Pola yang sama
// dipakai `inboxrclpucl/suratpdf`.
package rejectpdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"

	"claim-pnc/internal/inboxcompliance"
)

// Renderer memenuhi inboxcompliance.RejectLetterRenderer.
type Renderer struct{}

var _ inboxcompliance.RejectLetterRenderer = Renderer{}

const (
	fontSurat = "Times"
	body      = 10.0
	line      = 4.6
	kiri      = 19.0
	kanan     = 16.0
	atas      = 14.0

	// lebarLabel menyamakan letak titik dua pada tabel data a–e.
	//
	// Diambil dari label terpanjangnya, "Nilai Klaim yang Sudah Dibayarkan". Lebar yang
	// mengikuti isi akan membuat deret titik duanya patah — di templat ia tabel.
	lebarLabel = 62.0

	// lebarHuruf adalah kolom "a.", "b.", … di depan label.
	lebarHuruf = 6.0

	// lebarTandaTangan adalah satu kolom blok tanda tangan; templat memberinya 33%.
	lebarTandaTangan = 58.0
)

// Teks tetap yang dipakai kedua halaman, disalin apa adanya dari templat.
const (
	perihal = "Tolakan dan Penarikan Dana Klaim Asuransi Kecelakaan Diri a/n "

	pembuka = "Bersama surat ini, kami informasikan bahwa diantara dokumen klaim dengan " +
		"data sebagai berikut :"

	penutup = "Demikian kami sampaikan, atas perhatian dan kerjasama yang telah terjalin " +
		"baik selama ini kami ucapkan terima kasih."

	rekening = "Bank BCA Cabang Gajah Mada No. 0123.301.5859 a/n PT. Asuransi Sinar Mas "

	tenggat = "selambat-lambatnya 14 (empat belas) hari kalender sejak tanggal surat ini."
)

// Render membentuk PDF surat penolakan — dua halaman.
func (Renderer) Render(d inboxcompliance.RejectLetterDocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(kiri, atas, kanan)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetTitle("Surat Penolakan dan Penarikan Dana Klaim", true)

	w := &writer{pdf: pdf}
	w.halamanPertama(d)
	w.halamanKedua(d)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("rejectpdf: menulis PDF: %w", err)
	}
	return buf.Bytes(), nil
}

type writer struct{ pdf *fpdf.Fpdf }

// halamanPertama ditujukan ke pemegang polis.
//
// Lima baris data (a–e), alasan penolakan LEBIH DULU, baru paragraf pengembalian dana —
// urutan itu kebalikan dari halaman kedua, dan memang begitu di templat.
func (w *writer) halamanPertama(d inboxcompliance.RejectLetterDocument) {
	w.pdf.AddPage()

	w.kepala(d)
	w.alamat(d)
	// "Up." dan "Jabatan." adalah TEKS TETAP berakhir titik pada templat — keduanya
	// tanpa merge field, disediakan untuk diisi tangan.
	w.baris(body, false, "Up.")
	w.baris(body, false, "Jabatan.")
	w.jeda(line)

	w.perihal(d.SubjectName)
	w.jeda(line)
	w.baris(body, false, "Dengan Hormat,")
	w.jeda(line)
	w.alinea(false, pembuka)

	w.tabelData(true)
	w.jeda(line)

	w.alinea(false, "Bersama ini kami sampaikan bahwa klaim Asuransi dengan data-data "+
		"tersebut di atas ditolak berdasarkan hal-hal sebagai berikut :")
	w.alasan()

	w.alinea(false, "Atas hal ini kami mohon bantuan Bapak/Ibu untuk peserta melakukan "+
		"pengembalian klaim senilai Rp._____________ ke rekening kami di "+rekening+tenggat)
	w.alinea(false, penutup)

	w.baris(body, false, "Hormat Kami,")
	w.tandaTangan(
		"Rudy Widjaja", "Division Head",
		"dr. Margaretha Rosa G, AAK", "Accident & Health Ins. Claim Dept. Head",
	)

	w.jeda(line)
	w.baris(body, false, "Tembusan:")
	w.butir("Direktur PT. Asuransi Sinar Mas")
	w.butir(RecipientLine(d))

	w.jeda(line)
	w.baris(body, false, "Lampiran:")
	w.butir("Surat " + d.LetterNumber + ", surat tolakan klaim Asuransi Kecelakaan Diri a/n " +
		d.SubjectName)
}

// halamanKedua ditujukan ke pesertanya.
//
// Tiga hal membedakannya dari halaman pertama, dan ketiganya dari templat — bukan
// penyederhanaan: hanya EMPAT baris data (tanpa tanggal pembayaran), paragraf pengembalian
// dana mendahului alasan, dan penanda tangannya berbeda.
func (w *writer) halamanKedua(d inboxcompliance.RejectLetterDocument) {
	w.pdf.AddPage()

	w.kepala(d)
	w.alamat(d)

	// "Up. Nama Pasien" — pada templat ini TEKS TETAP, bukan merge field. Dibiarkan apa
	// adanya; menggantinya dengan nama pasien sebenarnya adalah perubahan yang tidak
	// diminta, dan halaman ini memang ditujukan ke peserta secara umum.
	w.baris(body, false, "Up. Nama Pasien")
	w.baris(body, false, "Jabatan.")
	w.jeda(line)

	w.perihal(d.SubjectName)
	w.jeda(line)
	w.baris(body, false, "Dengan Hormat,")
	w.jeda(line)
	w.alinea(false, pembuka)

	w.tabelData(false)
	w.jeda(line)

	w.alinea(false, "Maka kami mohon untuk dapat segera dilakukan pengembalian dana klaim "+
		"tersebut di atas melalui transfer ke rekening kami di "+rekening+tenggat)
	w.alinea(false, "Keputusan tersebut didasarkan atas temuan kami sebagai berikut :")
	w.alasan()

	w.alinea(false, penutup)

	w.baris(body, false, "Hormat Kami,")
	w.tandaTangan(
		"dr. Margaretha Rosa G, AAK", "Accident & Health Ins. Claim Dept. Head",
		"dr. Wahyu Kristanti", "Claim Section Head",
	)

	w.jeda(line)
	w.baris(body, false, "Tembusan:")
	w.butir(RecipientLine(d))
}

// kepala menggambar tanggal dan nomor surat.
func (w *writer) kepala(d inboxcompliance.RejectLetterDocument) {
	w.baris(body, false, "Jakarta, "+d.LetterDate)
	w.baris(body, false, "No. "+d.LetterNumber)
	w.jeda(line)
}

// alamat menggambar blok "Kepada Yth.".
//
// Badan usaha dan perorangan digambar pada baris yang sama seperti templat — salah satunya
// kosong — dan baris alamat selalu digambar meski kosong (lihat catatan paket).
func (w *writer) alamat(d inboxcompliance.RejectLetterDocument) {
	w.baris(body, false, "Kepada Yth.")
	if nama := strings.TrimSpace(d.RecipientCompany + " " + d.RecipientPerson); nama != "" {
		w.baris(body, true, nama)
	}
	w.terbungkus(body, false, d.RecipientAddress)
}

// RecipientLine memilih kalimat tembusan persis seperti kedua `pega:when` templat.
//
// Urutannya penting: templat menguji "perorangan kosong" LEBIH DULU, sehingga ketika
// keduanya terisi yang menang adalah bentuk badan usaha. Ditiru apa adanya.
//
// Diekspor karena inilah satu-satunya percabangan isi di paket ini, dan isi PDF yang sudah
// terbentuk tidak dapat dibaca balik dengan murah — fpdf memampatkan alirannya. Menguji
// fungsinya langsung jauh lebih berguna daripada mencocokkan bita.
func RecipientLine(d inboxcompliance.RejectLetterDocument) string {
	perusahaan := strings.TrimSpace(d.RecipientCompany)
	perorangan := strings.TrimSpace(d.RecipientPerson)

	if perorangan == "" {
		return strings.TrimSpace("Direktur " + perusahaan)
	}
	if perusahaan == "" {
		return strings.TrimSpace("Pihak Sdr. " + perorangan)
	}
	return strings.TrimSpace("Direktur " + perusahaan)
}

// tabelData menggambar baris a sampai e — LABEL SAJA, tanpa nilai.
//
// Templat memuat `<td></td>` kosong pada kolom nilainya; tidak ada merge field di sana.
// Barisnya karena itu disediakan untuk diisi tangan setelah dicetak.
//
// Baris e — tanggal pembayaran — hanya ada di halaman pertama. Halaman kedua memang
// berhenti di d, dan itu bukan penyederhanaan di sini melainkan isi templatnya.
func (w *writer) tabelData(denganTanggalBayar bool) {
	label := []string{
		"Nama Pasien / No.Reg.",
		"Tempat Kejadian/Perawatan",
		"Tanggal Kejadian",
		"Nilai Klaim yang Sudah Dibayarkan",
	}
	if denganTanggalBayar {
		label = append(label, "Tanggal pembayaran klaim")
	}

	for i, l := range label {
		w.barisData(string(rune('a'+i))+".", l)
	}
}

// barisData menggambar satu baris "a. Label :" dengan lebar label tetap.
//
// Lebarnya TETAP, bukan mengikuti isi: di templat ia tabel, dan tanda titik duanya
// berderet lurus. Ruang sesudah titik dua dibiarkan kosong untuk diisi tangan.
func (w *writer) barisData(huruf, label string) {
	w.set(body, false)
	w.pdf.CellFormat(lebarHuruf, line, teksAman(huruf), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(lebarLabel, line, teksAman(label), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(5, line, ":", "", 1, "L", false, 0, "")
}

// jumlahBarisAlasan adalah tiga, persis seperti templat.
//
// Templat menuliskan `1.` `2.` `3.` sebagai teks tetap di dalam tabel — bukan
// `pega:forEach`, dan bukan jumlah yang mengikuti data. Tiga adalah jumlah baris yang
// disediakan untuk ditulis tangan.
const jumlahBarisAlasan = 3

// alasan menggambar ketiga baris bernomor, tanpa isi.
func (w *writer) alasan() {
	for nomor := 1; nomor <= jumlahBarisAlasan; nomor++ {
		w.set(body, false)
		w.pdf.CellFormat(lebarHuruf, line, strconv.Itoa(nomor)+".", "", 1, "L", false, 0, "")

		// Satu baris kosong di bawah tiap nomor — templat memberi tabelnya
		// `cellpadding="16"`, sehingga barisnya memang renggang dan ada ruang menulis.
		w.jeda(line)
	}
	w.jeda(line)
}

// tandaTangan menggambar blok dua kolom.
func (w *writer) tandaTangan(namaKiri, jabatanKiri, namaKanan, jabatanKanan string) {
	// Ruang untuk tanda tangan basah atau cap.
	w.jeda(line * 5)

	w.set(body, true)
	w.pdf.CellFormat(lebarTandaTangan, line, teksAman(namaKiri), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(14, line, "", "", 0, "L", false, 0, "")
	w.pdf.CellFormat(lebarTandaTangan, line, teksAman(namaKanan), "", 1, "L", false, 0, "")

	w.set(body, false)
	w.pdf.CellFormat(lebarTandaTangan, line, teksAman(jabatanKiri), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(14, line, "", "", 0, "L", false, 0, "")
	w.pdf.CellFormat(lebarTandaTangan, line, teksAman(jabatanKanan), "", 1, "L", false, 0, "")
}

func (w *writer) set(size float64, tebal bool) {
	gaya := ""
	if tebal {
		gaya = "B"
	}
	w.pdf.SetFont(fontSurat, gaya, size)
}

func (w *writer) jeda(h float64) { w.pdf.Ln(h) }

func (w *writer) baris(size float64, tebal bool, s string) {
	w.set(size, tebal)
	w.pdf.MultiCell(0, line, teksAman(s), "", "L", false)
}

// terbungkus menggambar teks yang boleh memuat pergantian baris sendiri — alamat, misalnya.
//
// Teks kosong tetap menghasilkan SATU baris kosong, supaya tempatnya terlihat.
func (w *writer) terbungkus(size float64, tebal bool, s string) {
	for _, b := range strings.Split(teksAman(s), "\n") {
		w.baris(size, tebal, b)
	}
}

// perihal menggambar baris "Perihal", yang di templat tebal.
func (w *writer) perihal(nama string) {
	w.set(body, true)
	w.pdf.MultiCell(0, line, teksAman("Perihal: "+perihal+nama), "", "L", false)
}

// alinea menggambar satu paragraf beserta jarak sesudahnya.
func (w *writer) alinea(tebal bool, s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	w.baris(body, tebal, s)
	w.jeda(line)
}

// butir menggambar satu butir daftar bertanda hubung — `<ol>` pada templat.
func (w *writer) butir(s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	w.set(body, false)
	w.pdf.CellFormat(lebarHuruf, line, "-", "", 0, "L", false, 0, "")

	lebar, _ := w.pdf.GetPageSize()
	w.pdf.MultiCell(lebar-kiri-kanan-lebarHuruf, line, teksAman(s), "", "L", false)
}

// teksAman menyiapkan teks untuk fpdf, yang bekerja pada pengkodean satu bita.
//
// Isian surat datang dari ketikan petugas dan dapat memuat karakter di luar Latin-1 —
// tanda kutip miring yang disalin dari Word adalah contoh yang paling sering. Tanpa
// penyiapan ini karakter itu tergambar sebagai sampah, dan yang membacanya adalah nasabah.
func teksAman(s string) string {
	bersih := strings.NewReplacer(
		"‘", "'", "’", "'",
		"“", `"`, "”", `"`,
		"–", "-", "—", "-",
		"…", "...",
		" ", " ",
		"\r\n", "\n", "\r", "\n",
	).Replace(s)

	var b strings.Builder
	for _, r := range bersih {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 32:
			// Kendali lain dibuang — ia tidak tergambar, dan sebagiannya merusak tata letak.
		case r <= 255:
			b.WriteRune(r)
		default:
			b.WriteRune('?')
		}
	}
	return b.String()
}
