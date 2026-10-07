// Package sqlfile memuat kueri bernama dari berkas .sql yang disematkan ke dalam binary.
//
// Setiap paket sqlstore menyimpan kuerinya di berkas .sql terpisah dari kode Go
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 5). Di dalam berkas itu setiap kueri diawali penanda
//
//	-- name: <nama>
//
// dan berlanjut sampai penanda berikutnya. Paket ini satu-satunya tempat aturan pemecahan
// itu hidup; sebelumnya aturan yang sama tersalin di 75 paket sqlstore.
//
// Pemakaian di sebuah paket sqlstore:
//
//	//go:embed *.sql
//	var queryFiles embed.FS
//
//	var query = sqlfile.MustLoad(queryFiles, "namamodul/sqlstore")
//
//	func getQuery(name string) string { return sqlfile.MustGet(query, "namamodul/sqlstore", name) }
package sqlfile

import (
	"fmt"
	"io/fs"
	"strings"
)

const marker = "-- name:"

// Option mengubah cara badan kueri dibentuk. Tanpa opsi apa pun, perilakunya adalah yang
// dipakai sebagian besar modul: baris komentar dibuang dan kueri berbadan kosong dilewati.
type Option func(*options)

type options struct {
	keepComments          bool
	stripTrailingComments bool
	keepEmpty             bool
}

// KeepComments mempertahankan baris komentar di dalam badan kueri, sehingga ikut terkirim
// ke basis data. Dipakai modul yang sejak awal mengirim kuerinya utuh.
func KeepComments() Option { return func(o *options) { o.keepComments = true } }

// StripTrailingComments membuang baris komentar dan baris kosong di UJUNG badan kueri saja;
// komentar di tengah tetap ada. Hanya bermakna bersama KeepComments.
func StripTrailingComments() Option { return func(o *options) { o.stripTrailingComments = true } }

// KeepEmpty tetap menyimpan nama kueri yang badannya kosong (sebagai teks kosong), alih-alih
// melewatinya.
func KeepEmpty() Option { return func(o *options) { o.keepEmpty = true } }

// Split memecah isi satu berkas .sql menjadi kueri bernama.
//
// Baris sebelum penanda pertama diabaikan. Komentar dibuang secara baku karena isinya
// panjang dan seluruhnya ikut terkirim setiap kali kueri dijalankan — dan ikut muncul di
// jejak basis data saat DBA menelusuri kueri yang lambat.
func Split(content string, opts ...Option) map[string]string {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		lines := body
		if !o.keepComments {
			lines = withoutComments(lines)
		} else if o.stripTrailingComments {
			lines = withoutTrailingComments(lines)
		}
		text := strings.TrimSpace(strings.Join(lines, "\n"))
		if text != "" || o.keepEmpty {
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

func withoutComments(lines []string) []string {
	var kept []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			kept = append(kept, line)
		}
	}
	return kept
}

func withoutTrailingComments(lines []string) []string {
	end := len(lines)
	for end > 0 {
		trimmed := strings.TrimSpace(lines[end-1])
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			break
		}
		end--
	}
	return lines[:end]
}

// MustLoad membaca seluruh berkas di akar files dan menggabungkan kuerinya.
//
// Ia panik bila berkas tidak terbaca atau satu nama dipakai dua kali. Panik di sini disengaja:
// berkasnya disematkan saat kompilasi, jadi kegagalan adalah cacat pemrograman yang harus
// terlihat saat aplikasi start, bukan galat yang dilihat pengguna. owner adalah awalan pesan
// panik, misalnya "registrasi/sqlstore".
func MustLoad(files fs.ReadDirFS, owner string, opts ...Option) map[string]string {
	result := map[string]string{}
	entries, err := files.ReadDir(".")
	if err != nil {
		panic(owner + ": tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, entry := range entries {
		content, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			panic(owner + ": tidak dapat membaca " + entry.Name() + ": " + err.Error())
		}
		for name, text := range Split(string(content), opts...) {
			if _, clash := result[name]; clash {
				panic(owner + ": nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// MustGet mengembalikan teks kueri bernama name dan panik bila namanya tidak ada.
//
// Nama kueri adalah konstanta di dalam kode, bukan masukan pengguna, sehingga ketiadaannya
// adalah cacat pemrograman yang harus terlihat saat pertama dijalankan.
func MustGet(queries map[string]string, owner, name string) string {
	text, exists := queries[name]
	if !exists {
		panic(fmt.Sprintf("%s: kueri %q tidak ditemukan di berkas .sql", owner, name))
	}
	return text
}
