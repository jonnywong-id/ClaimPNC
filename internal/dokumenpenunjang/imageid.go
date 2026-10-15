package dokumenpenunjang

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// NewImageID menerbitkan kunci dokumen seperti RDB `GenerateImageID`
// (`RDB List/GenerateImageID-SQL.xml`), yang dijalankan `InsertDokumenPNC` sesudah berkasnya
// terunggah (`Activity/InsertDokumenPNC-Act.xml:4216`, `:4448`):
//
//	SELECT STANDARD_HASH('ASMPP' || TO_CHAR(SYSTIMESTAMP, 'DD/MM/YYYY HH24:MI:SS.FF3'), 'MD5')
//
// Hasilnya RAW 16 bait, tersimpan sebagai 32 karakter heksadesimal huruf besar — bentuk
// seluruh IMAGEID yang sudah ada di GENERAL.T_STORAGE_IMAGE (diperiksa 2026-09-30).
//
// Kunci ini BUKAN dari layanan penyimpanan: Pega tidak membaca ImageID dari responsnya.
// Seperti aslinya, dua unggahan pada milidetik yang sama menghasilkan kunci yang sama.
func NewImageID(at time.Time) string {
	sum := md5.Sum([]byte("ASMPP" + at.In(clock.ZoneWIB).Format("02/01/2006 15:04:05.000")))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
