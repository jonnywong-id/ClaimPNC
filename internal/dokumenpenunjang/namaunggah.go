package dokumenpenunjang

import (
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// NamaUnggah adalah nama berkas yang dikirim ke penyimpanan, disalin dari pemanggil
// `InsertDokumenPNC` di Pega (`SetAdjustmentAcceptation` 57.4, `UploadDocumentToGoogleStorage`,
// `CNMUpdateMasterRekening_act`, `ASMSendsEmailAttachments_PDF`):
//
//	@CurrentDate("yyMdhm-sS","Asia/Jakarta") + "-" + <ID jenis dokumen> + "-" + <nama berkas>
//
// dengan spasi dibuang dan karakter di luar [a-zA-Z0-9 .-] dihapus dari ID dan nama.
//
// Awalan waktu itu yang membuat FILENAME unik: trigger GENERAL.TBIU_STORAGE_IMAGE menolak
// nama yang sudah ada di seluruh T_STORAGE_IMAGE (ORA-20009 "File name sudah ada"), lintas
// klaim dan lintas aplikasi.
//
// Modul registrasi punya salinan aturan yang sama (`registrasi.UploadFileName`) untuk
// unggahan akseptasi dan Unggah Dokumen, supaya kedua domain tidak saling mengimpor.
func NamaUnggah(saat time.Time, jenis, nama string) string {
	t := saat.In(clock.ZoneWIB)
	jam := t.Hour() % 12
	if jam == 0 {
		jam = 12
	}
	// SimpleDateFormat "yyMdhm-sS": tanpa nol di depan kecuali tahun; S = milidetik.
	cap := fmt.Sprintf("%02d%d%d%d%d-%d%d",
		t.Year()%100, int(t.Month()), t.Day(), jam, t.Minute(), t.Second(), t.Nanosecond()/int(time.Millisecond))
	return cap + "-" + bagianNama(jenis) + "-" + bagianNama(nama)
}

func bagianNama(s string) string {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(s, " ", "") {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}
