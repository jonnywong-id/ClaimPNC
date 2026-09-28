package sqlstore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func getQuery(name string) string {
	text, existing := query[name]
	if !existing {
		panic(fmt.Sprintf("reportklaim/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

// hasQuery menyatakan apakah sebuah kueri sudah ada di berkas .sql.
//
// # Kenapa ada, padahal getQuery sudah panik bila tidak ada
//
// Karena tidak semua ketiadaan adalah cacat pemrograman. Sebuah laporan boleh berada di
// katalog sementara kuerinya belum ditulis — dan keadaan itu harus terbaca sebagai
// PENOLAKAN YANG MENYEBUTKAN SEBABNYA, bukan sebagai panik yang menjatuhkan permintaan,
// dan bukan pula sebagai hasil kosong yang terbaca "tidak ada data".
//
// **Sejak 2026-09-25 seluruh 26 kueri sudah ada**, sehingga jalur penolakan itu tidak lagi
// terpakai. Ia tidak dihapus: yang dijaganya bukan keadaan hari ini melainkan perilaku
// ketika laporan berikutnya ditambahkan ke katalog sebelum kuerinya ditulis. Lihat
// Repo.Stream, NotPorted, dan TestKueriBelumDipindahkanDitolakDenganSebabnya.
func hasQuery(name string) bool {
	_, existing := query[name]
	return existing
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("reportklaim/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("reportklaim/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("reportklaim/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, line := range body {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			statement = append(statement, line)
		}
		result[name] = strings.TrimSpace(strings.Join(statement, "\n"))
	}

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, line)
	}
	save()

	return result
}
