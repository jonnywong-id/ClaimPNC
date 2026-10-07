package reportklaim

import "strings"

// parseColumns membaca tabel kolom yang ditulis satu kolom per baris: judul, TAB, lalu nama
// field. Baris kosong diabaikan.
//
// Judul TIDAK dipangkas: spasinya bagian dari judul (lihat Column.Header). Tabel yang rusak
// adalah cacat pemrograman, bukan masukan pengguna, sehingga ia panik saat aplikasi start.
func parseColumns(table string) []Column {
	var out []Column
	for _, line := range strings.Split(table, "\n") {
		if line == "" {
			continue
		}
		header, field, ok := strings.Cut(line, "\t")
		if !ok || field == "" {
			panic("reportklaim: baris tabel kolom tidak sah: " + line)
		}
		out = append(out, Column{Header: header, Field: field})
	}
	return out
}
