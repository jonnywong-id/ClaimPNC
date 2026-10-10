package db

import (
	go_ora "github.com/sijms/go-ora/v2"
)

// BlobValue membungkus isi berkas supaya diikat sebagai BLOB.
//
// Tanpa pembungkus, go-ora mengikat []byte sebagai RAW, yang dibatasi 32.767 byte —
// dokumen coverage polis berobjek banyak melampauinya, dan Oracle menolaknya dengan
// galat yang tidak menyebut sebabnya. Pembungkusnya tinggal di paket ini supaya modul
// tidak mengimpor driver (lihat IsMissingObject).
//
// nil diikat sebagai NULL.
func BlobValue(data []byte) any {
	if data == nil {
		return nil
	}
	return go_ora.Blob{Data: data, Valid: true}
}

// ClobValue membungkus teks supaya diikat sebagai CLOB, bukan VARCHAR2 (batas 32.767).
func ClobValue(text string) any {
	return go_ora.Clob{String: text, Valid: true}
}
