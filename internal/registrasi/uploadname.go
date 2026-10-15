package registrasi

import (
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// UploadFileName adalah nama berkas yang dikirim ke penyimpanan dokumen dan dicatat sebagai
// ATTACHNAME, disalin dari keempat pemanggil `InsertDokumenPNC` di Pega
// (`SetAdjustmentAcceptation` langkah 57.4, `UploadDocumentToGoogleStorage`,
// `CNMUpdateMasterRekening_act`, `ASMSendsEmailAttachments_PDF`):
//
//	@CurrentDate("yyMdhm-sS","Asia/Jakarta") + "-" + <id jenis dokumen> + "-" + <nama berkas>
//
// dengan spasi dibuang dan karakter di luar [a-zA-Z0-9 .-] dihapus dari id dan nama.
//
// # Kenapa awalan ini wajib, bukan hiasan
//
// Trigger GENERAL.TBIU_STORAGE_IMAGE (basis data ASMD) menolak FILENAME yang sudah ada di
// SELURUH tabel — lintas klaim dan lintas aplikasi — dengan ORA-20009 "File name sudah ada".
// Tanpa awalan waktu, berkas bernama sama yang pernah diunggah ke klaim mana pun membuat
// unggahan berikutnya gagal SETELAH berkasnya terkirim ke penyimpanan.
//
// Penggantian png/jpg/jpeg menjadi "avif" pada nama (langkah 57.4) TIDAK dibawa: modul
// dokumen penunjang menentukan konversi dari ekstensi nama yang diterimanya, sehingga nama
// ber-"avif" untuk isi yang belum dikonversi akan tercatat dengan jenis yang salah.
func UploadFileName(at time.Time, kind, name string) string {
	return javaStamp(at.In(clock.ZoneWIB)) + "-" + uploadNamePart(kind) + "-" + uploadNamePart(name)
}

// javaStamp meniru pola SimpleDateFormat "yyMdhm-sS": tahun dua digit, bulan/hari/jam
// (1–12)/menit/detik TANPA nol di depan, lalu milidetik tanpa nol di depan.
func javaStamp(t time.Time) string {
	hour := t.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	return fmt.Sprintf("%02d%d%d%d%d-%d%d",
		t.Year()%100, int(t.Month()), t.Day(), hour, t.Minute(), t.Second(), t.Nanosecond()/int(time.Millisecond))
}

func uploadNamePart(s string) string {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(s, " ", "") {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}
