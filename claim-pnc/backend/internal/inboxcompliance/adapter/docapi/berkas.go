package docapi

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// gcsPrefix adalah awalan bucket yang dipangkas sebelum jalur objek dikirim.
//
// Dari `DeleteAttachDoc` langkah 11: `@replaceAll(..., "gs://<bucket>/", "")`. Nama
// bucket-nya SENGAJA tidak ditulis — ia alamat penyimpanan produksi (`D-69`). Yang
// dipangkas adalah pola `gs://<apa pun>/`, sehingga paket ini tidak perlu mengetahuinya.
const gcsPrefix = "gs://"

// waktuSekarang dapat diganti uji. Satu-satunya pemakaian jam di paket ini, dan ia hanya
// membentuk nama folder — bukan nilai bisnis (`F-5` tidak menuntut seam Clock di sini).
var waktuSekarang = time.Now

// bukanAlfanumerik meniru `@pxReplaceAllViaRegex(param.Filename, "[^a-zA-Z0-9]", "")`.
//
// Perhatikan apa yang ikut terbuang: **titik dan ekstensinya**. `Surat-Ket.pdf` menjadi
// `SuratKetpdf`. Itu terlihat seperti cacat, tetapi tipe berkasnya tetap terbawa lewat
// field MimeType yang terpisah — jadi ekstensi pada nama memang tidak dipakai layanan.
var bukanAlfanumerik = regexp.MustCompile(`[^a-zA-Z0-9]`)

// mimeByExtension meniru rantai `@If` bertingkat lima belas pada
// `Activity/InsertDokumenPNC-Act.xml` langkah 16, apa adanya.
//
// Kuncinya HURUF BESAR, karena Pega membandingkannya lewat `@toUpperCase`.
var mimeByExtension = map[string]string{
	"PNG":  "image/png",
	"JPG":  "image/jpeg",
	"JPEG": "image/jpeg",
	"AVIF": "image/avif",
	"TXT":  "text/plain",
	"DOC":  "application/msword",
	"DOCX": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"PDF":  "application/pdf",
	"EML":  "message/rfc822",
	"RAR":  "application/vnd.rar",
	"ZIP":  "application/zip",
	"CSV":  "text/csv",
	"XLS":  "application/vnd.ms-excel",
	"XLSX": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"PPT":  "application/vnd.ms-powerpoint",
	"PPTX": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

// mimeDari memetakan ekstensi menjadi tipe MIME.
//
// Yang tidak dikenali menjadi `"application/" + ekstensi`, persis cabang terakhir rantai
// `@If` Pega. Itu menghasilkan tipe yang kadang tidak sah — `application/xyz` — dan itu
// memang keluaran sistem lama.
func mimeDari(ekstensi string) string {
	bersih := strings.TrimPrefix(strings.TrimSpace(ekstensi), ".")
	if mime, ada := mimeByExtension[strings.ToUpper(bersih)]; ada {
		return mime
	}
	return "application/" + bersih
}

// folderUnggah membentuk `Doc/YYYY/MM/`.
//
// Dari langkah 16: `"Doc/" + substring(CurrentDateTime,0,4) + "/" +
// substring(CurrentDateTime,4,6) + "/"`. Format waktu Pega `YYYYMMDDThhmmss`, sehingga
// potongan 0–4 adalah tahun dan 4–6 bulan.
func folderUnggah(pada time.Time) string {
	return fmt.Sprintf("Doc/%04d/%02d/", pada.Year(), int(pada.Month()))
}

// pangkasBucket membuang awalan `gs://<bucket>/` dari jalur objek.
func pangkasBucket(jalur string) string {
	if !strings.HasPrefix(jalur, gcsPrefix) {
		return jalur
	}
	sisa := strings.TrimPrefix(jalur, gcsPrefix)
	if potong := strings.Index(sisa, "/"); potong >= 0 {
		return sisa[potong+1:]
	}
	return sisa
}
