package main

import (
	"context"
	"database/sql"
	"fmt"
)

// cariObjekKoneksiKedua menjawab pertanyaan yang menentukan bentuk permintaan ke Infra:
//
//	apakah kedua objek koneksi kedua benar-benar ada di basis data LAIN,
//	ataukah ia sebenarnya terjangkau dari koneksi yang sudah terpasang?
//
// Bedanya besar. Bila terjangkau, yang dibutuhkan hanya HAK BACA atau SINONIM — satu
// permintaan kecil ke DBA. Bila tidak, dibutuhkan parameter koneksi ke basis data lain,
// yang menempuh pihak berbeda dan waktu tunggu berbeda.
//
// Di Pega keduanya dibaca lewat DB Link `@asmd.sinarmas.co.id`, tetapi itu TIDAK
// membuktikan basis datanya berbeda hari ini: DB Link dapat saja menunjuk instans yang
// sama, dan lingkungan DEV sering menyatukan apa yang di produksi terpisah.
func cariObjekKoneksiKedua(ctx context.Context, db *sql.DB) {
	fmt.Println("Objek koneksi kedua — apakah terjangkau dari koneksi yang sudah ada?")

	objek := []string{"LST_MITRA", "HRD_LBR"}

	for _, nama := range objek {
		fmt.Printf("  %s\n", nama)

		// 1. Terbaca langsung dengan nama berkualifikasi skema?
		var n int64
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM general.`+nama).Scan(&n)
		if err == nil {
			fmt.Printf("    GENERAL.%-10s TERBACA — %d baris. Tidak perlu koneksi kedua.\n", nama, n)
			continue
		}
		fmt.Printf("    GENERAL.%-10s tidak terbaca: %s\n", nama, ora.FindString(err.Error()))

		// 2. Ada di katalog tetapi tanpa hak baca? Itu membedakan "tidak ada" dari
		//    "ada tetapi belum di-grant" — dua perkara dengan jalan keluar berbeda.
		katalog(ctx, db,
			`SELECT COUNT(*) FROM all_objects WHERE object_name = :1`,
			nama, "      terlihat di ALL_OBJECTS")

		// 3. Ada sinonim yang menunjuknya?
		katalog(ctx, db,
			`SELECT COUNT(*) FROM all_synonyms WHERE synonym_name = :1 OR table_name = :1`,
			nama, "      punya sinonim")

		// 4. Ada DB Link terdaftar yang dapat menjangkaunya?
		katalog(ctx, db,
			`SELECT COUNT(*) FROM all_db_links WHERE UPPER(db_link) LIKE '%ASMD%'`,
			nama, "      DB Link ber-ASMD terdaftar")
	}
	fmt.Println()
}

func katalog(ctx context.Context, db *sql.DB, q, arg, label string) {
	var n int64
	var err error
	if countBind(q) == 0 {
		err = db.QueryRowContext(ctx, q).Scan(&n)
	} else {
		err = db.QueryRowContext(ctx, q, arg).Scan(&n)
	}
	if err != nil {
		fmt.Printf("%s: GAGAL %s\n", label, ora.FindString(err.Error()))
		return
	}
	fmt.Printf("%s: %d\n", label, n)
}

// countBind menghitung KEMUNCULAN placeholder — bukan nomor uniknya, karena driver
// mengikat menurut kemunculan (lihat ujiBind).
func countBind(q string) int {
	n := 0
	for i := 0; i+1 < len(q); i++ {
		if q[i] == ':' && q[i+1] >= '0' && q[i+1] <= '9' {
			n++
		}
	}
	return n
}

// ujiLewatDBLink menjalankan KEDUA kueri koneksi kedua apa adanya, hanya dengan akhiran
// DB Link ditambahkan — supaya pilihan "pakai DB Link" terbukti, bukan diduga.
//
// Hasilnya 2026-10-09: keduanya TERBACA dari koneksi yang sudah ada di `.env`.
// `general.lst_mitra` 2.159 login mitra, `general.hrd_lbr` 230 hari libur 2015..2026.
//
// Artinya tidak ada tabel yang perlu dibuat, dan tidak ada data yang perlu dipindahkan —
// yang kurang hanyalah JALAN menuju tabel yang sudah ada.
func ujiLewatDBLink(ctx context.Context, db *sql.DB) {
	// Nama DB Link ini sudah tertulis di Steering Lampiran F sebagai `@ASMD`; ia nama
	// tautan, bukan kredensial.
	const link = "@ASMD.SINARMAS.CO.ID"

	fmt.Println("Kedua kueri koneksi kedua, dijalankan lewat DB Link yang sudah ada:")

	var n int64
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
		  SELECT login_aplikasi
		    FROM general.lst_mitra`+link+`
		   WHERE login_aplikasi IS NOT NULL)`).Scan(&n)
	if err != nil {
		fmt.Println("  report_mitra_logins     GAGAL", ora.FindString(err.Error()))
	} else {
		fmt.Printf("  report_mitra_logins     %d login mitra\n", n)
	}

	f := bindUji()
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
		  SELECT tanggal
		    FROM general.hrd_lbr`+link+`
		   WHERE tanggal >= :1 AND tanggal <= :2)`, f.From, f.To).Scan(&n)
	if err != nil {
		fmt.Println("  report_holiday_calendar GAGAL", ora.FindString(err.Error()))
	} else {
		fmt.Printf("  report_holiday_calendar %d hari libur pada rentang uji\n", n)
	}
	fmt.Println()
}

// ujiPenyaringMitra menjawab pertanyaan terakhir: sesudah penyaringnya bekerja, apakah
// masih ada baris yang tersisa?
//
// Kueri Mitra yang mengembalikan baris dan daftar login yang terbaca belum cukup — bila
// irisan keduanya kosong, berkasnya tetap kosong dan pengguna tetap tidak mendapat apa
// pun. Perbedaan "penyaringnya salah" dan "memang tidak ada penugasan mitra pada periode
// itu" hanya terlihat dari angka ini.
func ujiPenyaringMitra(ctx context.Context, db *sql.DB) {
	var total, cocok int64

	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COUNT(CASE WHEN UPPER(TRIM(a.userassign)) IN (
		                      SELECT UPPER(TRIM(b.login_aplikasi))
		                        FROM general.lst_mitra@ASMD.SINARMAS.CO.ID b
		                       WHERE b.login_aplikasi IS NOT NULL)
		                  THEN 1 END)
		  FROM pooldata.pnc_chronologytat a`).Scan(&total, &cocok)
	if err != nil {
		fmt.Println("Penyaring Mitra: GAGAL", ora.FindString(err.Error()))
		fmt.Println()
		return
	}

	fmt.Printf("Penyaring Mitra: %d baris penugasan, %d di antaranya milik petugas mitra\n",
		total, cocok)
	fmt.Println()
}
