package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportklaim/repo/sqlstore"
)

// Mode kedua perkakas ini: MENJALANKAN setiap kueri, bukan hanya mengurainya.
//
// # Apa yang ditambahkan dibanding EXPLAIN PLAN
//
// EXPLAIN PLAN membuktikan nama tabel dan kolomnya ada dan sintaksnya sah. Ia TIDAK
// membuktikan:
//
//	nilai bind yang dikirim aplikasi cocok jumlah dan tipenya
//	DB Link benar-benar menjawab saat dipakai (bukan hanya sewaktu diurai)
//	penyaringnya mengembalikan baris sama sekali
//
// Yang terakhir paling halus: kueri yang menyaring ke kolom yang salah tetap SAH, tetap
// lulus EXPLAIN, dan mengembalikan nol baris selamanya. Berkas kosong bukan galat, dan
// pengguna yang menerimanya tidak punya cara tahu sebabnya.
//
// # Yang sengaja TIDAK dilakukan
//
// Perkakas ini TIDAK PERNAH mencetak isi baris. Basis data ini memuat data nasabah
// sungguhan — nomor polis, nama tertanggung, NPWP, nomor rekening — dan `D-69` melarang
// menuliskannya ke artefak mana pun. Yang dilaporkan hanya: berhasil atau tidak, berapa
// kolom, dan ada baris atau tidak.
//
// Hanya SATU baris yang diambil, lalu kursornya ditutup.

// bindUji adalah penyaring yang dipakai memanggil kueri. Rentangnya lebar supaya kueri
// yang benar punya kesempatan mengembalikan baris — "nol baris" baru bermakna bila
// rentangnya memang tidak membatasi.
func bindUji() reportklaim.Filter {
	return reportklaim.Filter{
		From: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		// Sengaja BUKAN Non-MBU: PlansForCheck memanggil tiap laporan dua kali — sekali
		// dengan lini ini, sekali dengan Non-MBU. Memasang Non-MBU di sini membuat
		// keduanya sama, dan tiga kueri lini-lain tidak pernah ikut diuji.
		BusinessLine:     reportklaim.BusinessLinePA,
		ComplianceStatus: "1",
		BusinessCode:     "",

		// FixedParam milik TOMBOL, bukan isian pengguna — dan tanpa ini `statusapprove`
		// terkirim sebagai teks kosong, sehingga panel Data Komite selalu nol baris.
		// Kosongnya tidak menimbulkan galat apa pun; itulah sebabnya ia perlu disebut.
		FixedParam: map[string]string{"statusapprove": "1"},
	}
}

type hasilProbe struct {
	nama   string
	kode   string
	kolom  int
	adaBar bool
	lama   time.Duration
	err    error
	lewati string
}

// kodeBisnisNyata mengambil satu kode bisnis yang BENAR-BENAR DIPAKAI KLAIM.
//
// Bukan dari daftar pilihan panelnya: daftar itu memuat seluruh isi master `business`,
// sementara hanya sebagian kecil kode yang pernah dipakai klaim. Mengambil yang pertama
// dari daftar pilihan menghasilkan nol baris — dan nol baris itu akan terbaca sebagai
// cacat kueri, padahal yang salah masukan ujinya.
//
// Ini kekeliruan yang benar-benar terjadi pada putaran pertama probe ini.
func kodeBisnisNyata(ctx context.Context, db *sql.DB) string {
	var kode string
	err := db.QueryRowContext(ctx, `
		SELECT p.businesscode
		  FROM pooldata.t_claim_pnc p
		 WHERE p.businesscode IS NOT NULL
		   AND EXISTS (SELECT 1 FROM pooldata.pega_dashboardpnc a WHERE a.noklaim = p.claimno)
		 FETCH NEXT 1 ROW ONLY`).Scan(&kode)
	if err != nil {
		return ""
	}
	return kode
}

func jalankanProbe(ctx context.Context, db *sql.DB) error {
	f := bindUji()
	f.BusinessCode = kodeBisnisNyata(ctx, db)
	rencana := sqlstore.PlansForCheck(f)
	sort.Slice(rencana, func(i, j int) bool { return rencana[i].Name < rencana[j].Name })

	fmt.Println("Menjalankan setiap kueri dengan bind yang BENAR-BENAR dipakai aplikasi.")
	fmt.Println("Satu baris diambil lalu kursornya ditutup. Isi baris tidak pernah dicetak.")
	fmt.Println()

	var hasil []hasilProbe
	for _, r := range rencana {
		if anekaOnly[r.Name] {
			hasil = append(hasil, hasilProbe{nama: r.Name, kode: r.Code,
				lewati: "koneksi kedua (ANEKA_*) belum terpasang"})
			continue
		}
		hasil = append(hasil, probeSatu(ctx, db, r))
	}

	var gagal, kosong int
	for _, h := range hasil {
		switch {
		case h.lewati != "":
			fmt.Printf("  %-32s DILEWATI  %s\n", h.nama, h.lewati)
		case h.err != nil:
			gagal++
			fmt.Printf("  %-32s GAGAL     %s\n", h.nama, ora.FindString(h.err.Error()))
		case !h.adaBar:
			kosong++
			fmt.Printf("  %-32s NOL BARIS (%d kolom, %s) — penyaringnya perlu diperiksa\n",
				h.nama, h.kolom, h.lama.Round(time.Millisecond))
		default:
			fmt.Printf("  %-32s ada baris (%d kolom, %s)\n",
				h.nama, h.kolom, h.lama.Round(time.Millisecond))
		}
	}

	fmt.Printf("\n%d dijalankan · %d GAGAL · %d nol baris\n", len(hasil), gagal, kosong)
	if gagal > 0 {
		return fmt.Errorf("%d kueri gagal dijalankan", gagal)
	}
	return nil
}

func probeSatu(ctx context.Context, db *sql.DB, r sqlstore.PlanForCheck) hasilProbe {
	h := hasilProbe{nama: r.Name, kode: r.Code}

	// Batas waktu per kueri. Basis data ini dipakai bersama Pega yang sedang melayani;
	// satu kueri yang menyapu seluruh tabel tidak boleh menahannya berlama-lama.
	c, batal := context.WithTimeout(ctx, 60*time.Second)
	defer batal()

	mulai := time.Now()
	rows, err := db.QueryContext(c, sqlstore.AllQueriesForCheck()[r.Name], r.Args...)
	if err != nil {
		h.err, h.lama = err, time.Since(mulai)
		return h
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		h.err, h.lama = err, time.Since(mulai)
		return h
	}
	h.kolom = len(cols)

	if rows.Next() {
		// Dipindai ke penampung buang supaya tipe kolomnya benar-benar diuji — tanpa
		// Scan, ketidakcocokan tipe tidak akan muncul. Nilainya tidak pernah dibaca.
		buang := make([]any, len(cols))
		for i := range buang {
			buang[i] = new(any)
		}
		if err := rows.Scan(buang...); err != nil {
			h.err, h.lama = err, time.Since(mulai)
			return h
		}
		h.adaBar = true
	}

	h.err = rows.Err()
	h.lama = time.Since(mulai)
	return h
}
