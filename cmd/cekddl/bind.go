package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ujiBind membuktikan BAGAIMANA go-ora mengikat `:n`, alih-alih mengandalkan ingatan.
//
// Pertanyaannya nyata dan jawabannya menentukan: bila pengikatannya menurut NOMOR, maka
// `:3` boleh muncul sebelum `:1` dan boleh berulang. Bila menurut KEMUNCULAN, keduanya
// cacat — dan cacatnya tidak selalu berupa galat, melainkan bisa berupa nilai yang tertukar
// diam-diam.
//
// Catatan proyek sudah menyebut "driver mengikat menurut kemunculan"
// (`catatan-pengembangan.md:2239`). Ia diuji ulang di sini karena konsekuensinya mahal dan
// biayanya satu detik.
func ujiBind(ctx context.Context, db *sql.DB) {
	fmt.Println("Perilaku pengikatan `:n` pada go-ora:")

	uji := []struct {
		nama string
		sql  string
		args []any
	}{
		{"tanpa ulang      :1,:2", "SELECT :1, :2 FROM DUAL", []any{"a", "b"}},
		{"ulang di awal    :1,:2,:1", "SELECT :1, :2, :1 FROM DUAL", []any{"a", "b"}},
		{"ulang di akhir   :1,:2,:3,:3", "SELECT :1, :2, :3, :3 FROM DUAL", []any{"a", "b", "c"}},
		{"ulang + 4 arg    :1,:2,:3,:3", "SELECT :1, :2, :3, :3 FROM DUAL", []any{"a", "b", "c", "d"}},
		{"tidak berurutan  :3,:1,:2", "SELECT :3, :1, :2 FROM DUAL", []any{"a", "b", "c"}},
		{"persis report_tat bentuknya", "SELECT :1, :2, :3, :3, :3, :3, :3 FROM DUAL", []any{"a", "b", "c"}},
	}

	for _, u := range uji {
		// Jumlah kolom hasil = jumlah KEMUNCULAN placeholder, bukan jumlah argumen.
		kolom := strings.Count(u.sql, ":")
		tujuan := make([]any, kolom)
		for i := range tujuan {
			tujuan[i] = new(any)
		}

		err := db.QueryRowContext(ctx, u.sql, u.args...).Scan(tujuan...)
		if err != nil {
			fmt.Printf("  %-32s DITOLAK  %s\n", u.nama, ora.FindString(err.Error()))
			continue
		}

		var urut []string
		for _, t := range tujuan {
			urut = append(urut, fmt.Sprintf("%v", *(t.(*any))))
		}
		fmt.Printf("  %-32s diterima, hasil %v\n", u.nama, urut)
	}
	fmt.Println()
}
