package idformat

import (
	"strconv"
	"strings"
)

// Compose menyusun kunci berawalan site dengan nomor urut yang diberi nol di depan sampai
// selebar width digit.
func Compose(site string, sequence int64, width int) string {
	number := strconv.FormatInt(sequence, 10)
	if pad := width - len(number); pad > 0 {
		number = strings.Repeat("0", pad) + number
	}
	return strings.TrimSpace(site) + number
}
