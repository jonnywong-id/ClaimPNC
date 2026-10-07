package sqlvalue

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// TimeOrNil mengubah kolom waktu yang dapat NULL menjadi pointer; NULL menjadi nil.
func TimeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	at := value.Time
	return &at
}

// NilIfBlank mengirim NULL untuk teks yang kosong atau hanya berisi spasi.
func NilIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// NilIfEmpty mengirim NULL hanya untuk teks yang benar-benar kosong.
func NilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// Like menyusun pola LIKE huruf besar yang memuat keyword di posisi mana pun; karakter
// khusus LIKE di-escape dengan backslash.
func Like(keyword string) string {
	escaped := strings.ToUpper(strings.TrimSpace(keyword))
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%"
}

// LikeOrAll seperti Like, tetapi keyword kosong menghasilkan pola yang cocok dengan semua.
func LikeOrAll(keyword string) string {
	clean := strings.TrimSpace(keyword)
	if clean == "" {
		return "%"
	}
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + strings.ToUpper(replacer.Replace(clean)) + "%"
}

// TidyNumber merapikan bentuk teks sebuah angka bulat.
//
// Yang dirapikan hanyalah bentuk yang PASTI mewakili bilangan bulat yang sama: "8.0" menjadi
// "8", "08" menjadi "8". Teks yang tidak dapat dibaca sebagai bilangan bulat dikembalikan APA
// ADANYA — termasuk notasi ilmiah dan kunci yang ternyata bukan angka. Mengubah yang tidak
// dikenali lebih berbahaya daripada membiarkannya: kunci yang dikarang tidak akan menemukan
// barisnya, sedangkan kunci ganjil yang dibiarkan masih menunjuk baris yang benar.
func TidyNumber(text string) string {
	if text == "" {
		return ""
	}
	number, err := strconv.ParseInt(strings.TrimSuffix(text, ".0"), 10, 64)
	if err != nil {
		return text
	}
	return strconv.FormatInt(number, 10)
}

// EscapeLike menetralkan karakter pola LIKE pada kata kunci pencarian: backslash, "%", dan
// "_" diloloskan dengan backslash — pelolos bawaan LIKE di Oracle maupun PostgreSQL. Tanpa
// ini "%" mencocokkan SELURUH baris dan "_" sembarang satu karakter, diam-diam.
func EscapeLike(keyword string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
}
