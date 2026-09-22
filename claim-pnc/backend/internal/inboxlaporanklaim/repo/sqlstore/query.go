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

// sourceName adalah nama fragmen WITH yang dipakai bersama seluruh kueri pembaca.
const sourceName = "claim_report_source"

// sourced menyambung fragmen sumber dengan salah satu badan kueri.
//
// # Kenapa disambung, bukan ditulis utuh sembilan kali
//
// Fragmen sumbernya lima puluh baris dan memuat SELURUH aturan gabungan kedua tabel —
// termasuk cara Position, accepted, dan rejected diturunkan. Menuliskannya ulang di
// setiap badan berarti enam salinan yang harus berubah bersamaan, dan satu yang
// tertinggal akan membuat satu tab menyaring atas dasar yang berbeda dari tab lain.
// Cacat seperti itu tidak menghasilkan galat; ia menghasilkan daftar yang tampak wajar.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang disambung adalah dua teks dari berkas `.sql` milik kita sendiri, keduanya
// konstanta saat kompilasi. Tidak ada satu pun nilai dari pengguna yang menyentuhnya —
// seluruh nilai tetap lewat parameter binding. Yang dilarang `08-TECHNICAL-STRATEGY.md`
// §4.3 adalah merangkai NILAI ke dalam teks SQL, persis pola `{ASIS:…}` warisan.
func sourced(bodyName string) string {
	return getQuery(sourceName) + "\n" + getQuery(bodyName)
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func getQuery(name string) string {
	text, existing := query[name]
	if !existing {
		panic(fmt.Sprintf("inboxlaporanklaim/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxlaporanklaim/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("inboxlaporanklaim/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxlaporanklaim/sqlstore: nama kueri ganda: " + name)
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
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, line := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, marker) {
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
