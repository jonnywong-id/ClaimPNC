// Command cekobjek menjawab satu pertanyaan: objek ini ADA di mana, dan apakah akun
// aplikasi dapat melihatnya?
//
// # Kenapa ia ada
//
// `-periksa` melaporkan `ORA-00942: table or view does not exist` untuk CPNC_PENGGUNA dan
// CPNC_SESI_AKTIF. Galat itu AMBIGU, dan ambiguitasnya mahal: ia berbunyi sama persis
// untuk tiga keadaan yang tindak lanjutnya sama sekali berbeda —
//
//	objek memang belum dibuat          → jalankan migrasi
//	objek ada di skema lain            → butuh sinonim
//	objek ada, tetapi tanpa hak SELECT → butuh GRANT
//
// Kueri aplikasi menyebut tabelnya TANPA nama skema, sehingga ia dicari hanya di skema
// milik akun koneksi. Perkakas ini menanyakan ALL_OBJECTS — yang memuat objek di SELURUH
// skema yang dapat dilihat akun itu — sehingga ketiganya dapat dibedakan.
//
// Ia MEMBACA saja: tidak ada satu pun pernyataan yang menulis.
package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "github.com/sijms/go-ora/v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func run() error {
	env, err := bacaEnv(".env")
	if err != nil {
		return err
	}

	user := env["POOLDATA_ASM_PENGGUNA"]
	dsn := fmt.Sprintf("oracle://%s:%s@%s:%s/%s",
		url.QueryEscape(user), url.QueryEscape(env["POOLDATA_ASM_SANDI"]),
		env["POOLDATA_ASM_HOST"], env["POOLDATA_ASM_PORT"], env["POOLDATA_ASM_SERVICE"])

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	var current string
	if err := db.QueryRow(
		"SELECT SYS_CONTEXT('USERENV','CURRENT_SCHEMA') FROM DUAL").Scan(&current); err != nil {
		return err
	}
	fmt.Printf("akun koneksi : %s\nskema aktif  : %s\n\n", user, current)

	cekTautan(db)
	cekHakDDL(db)
	cekKolomPenyaringLini(db)

	for _, nama := range os.Args[1:] {
		cari(db, strings.ToUpper(nama))
	}
	return nil
}

// cari melaporkan setiap objek bernama itu yang DAPAT DILIHAT akun ini, di skema mana pun.
func cari(db *sql.DB, nama string) {
	rows, err := db.Query(
		"SELECT OWNER, OBJECT_TYPE, STATUS FROM ALL_OBJECTS WHERE OBJECT_NAME = :1", nama)
	if err != nil {
		fmt.Printf("%-20s GAGAL dicari: %v\n", nama, err)
		return
	}
	defer func() { _ = rows.Close() }()

	found := false
	for rows.Next() {
		var owner, jenis, status string
		if err := rows.Scan(&owner, &jenis, &status); err != nil {
			fmt.Printf("%-20s GAGAL dibaca: %v\n", nama, err)
			return
		}
		found = true
		fmt.Printf("%-20s ADA  owner=%-12s jenis=%-10s status=%s\n", nama, owner, jenis, status)
	}
	if !found {
		fmt.Printf("%-20s TIDAK TERLIHAT oleh akun ini di skema mana pun\n", nama)
	}
}

// bacaEnv membaca berkas .env sederhana: KUNCI=nilai, baris komentar diabaikan.
func bacaEnv(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

// cekTautan menguji apakah DB Link warisan Pega masih dapat dipakai akun ini.
//
// Ia ada karena pertanyaannya praktis: bila `general.hrd_lbr@ASMD` ternyata MASIH terbaca
// dari akun POOLDATA, maka kalender hari libur dapat diambil hari ini juga — tanpa
// menunggu kredensial ASMD yang terpisah. `D-25` tetap menghendaki DB Link diganti
// koneksi langsung, tetapi mengetahui ada-tidaknya jalan sementara mengubah urutan kerja.
func cekTautan(db *sql.DB) {
	fmt.Println()
	rows, err := db.Query("SELECT DB_LINK FROM ALL_DB_LINKS")
	if err != nil {
		fmt.Printf("daftar DB Link  : tidak dapat dibaca: %v\n", err)
	} else {
		defer func() { _ = rows.Close() }()
		ada := false
		for rows.Next() {
			var nama string
			if err := rows.Scan(&nama); err == nil {
				fmt.Printf("DB Link terlihat: %s\n", nama)
				ada = true
			}
		}
		if !ada {
			fmt.Println("DB Link terlihat: (tidak ada satu pun)")
		}
	}

	var satu int
	err = db.QueryRow(
		"SELECT 1 FROM general.hrd_lbr@ASMD.SINARMAS.CO.ID WHERE 1 = 0 UNION ALL SELECT 1 FROM DUAL").Scan(&satu)
	if err != nil {
		fmt.Printf("hrd_lbr@ASMD    : TIDAK terbaca: %v\n", err)
		return
	}
	fmt.Println("hrd_lbr@ASMD    : TERBACA lewat DB Link")
}


// cekHakDDL melaporkan apakah akun aplikasi dapat membuat tabel sendiri.
//
// Ia menentukan SIAPA yang menjalankan migrasi, bukan apakah migrasinya perlu. `D-63`
// tetap menuntut persetujuan Work Owner apa pun jawabannya — yang berubah hanya apakah
// DBA harus turun tangan atau cukup menyetujui.
func cekHakDDL(db *sql.DB) {
	var jumlah int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM SESSION_PRIVS WHERE PRIVILEGE = 'CREATE TABLE'").Scan(&jumlah)
	if err != nil {
		fmt.Printf("hak CREATE TABLE: tidak dapat diperiksa: %v\n", err)
		return
	}
	if jumlah == 0 {
		fmt.Println("hak CREATE TABLE: TIDAK ADA — migrasi harus dijalankan DBA")
		return
	}
	fmt.Println("hak CREATE TABLE: ADA pada akun aplikasi")
}
// cekKolomPenyaringLini memeriksa kolom yang dipakai PENYARING LINI BISNIS.
//
// # Kenapa ia terpisah dari probe biasa
//
// Penyaring lini bisnis disisipkan ke dalam kueri lewat penanda /*LINE_FILTER*/, sehingga
// ia TIDAK ikut terbaca oleh probe yang menjalankan kueri dasarnya. Dua kolom yang hanya
// muncul di penyaring itu — GROUP_PANEL dan GROUPBISNISID — karena itu lolos dari
// pemeriksaan, dan ketiadaannya baru muncul sebagai ORA-00904 saat pengguna menekan Cari.
//
// Yang membuatnya mudah terlewat: penyaring yang SAMA ditempelkan ke DUA tabel berbeda —
// PEGA_DASHBOARDPNC pada komponen Progress, dan T_CLAIM_PNC pada tiga komponen lainnya.
// Satu kolom yang ada di salah satunya dan tidak di yang lain akan membuat sebagian tab
// bekerja dan sebagian gagal, yang jauh lebih membingungkan daripada gagal seluruhnya.
func cekKolomPenyaringLini(db *sql.DB) {
	fmt.Println()
	pasangan := []struct{ tabel, kolom string }{
		{"PEGA_DASHBOARDPNC", "GROUP_PANEL"},
		{"PEGA_DASHBOARDPNC", "GROUPBISNISID"},
		{"T_CLAIM_PNC", "GROUP_PANEL"},
		{"T_CLAIM_PNC", "GROUPBISNISID"},
	}
	for _, p := range pasangan {
		var jenis string
		err := db.QueryRow(
			"SELECT DATA_TYPE FROM ALL_TAB_COLUMNS WHERE OWNER = 'POOLDATA' "+
				"AND TABLE_NAME = :1 AND COLUMN_NAME = :2", p.tabel, p.kolom).Scan(&jenis)
		switch {
		case err == sql.ErrNoRows:
			fmt.Printf("kolom penyaring : %s.%-14s TIDAK ADA\n", p.tabel, p.kolom)
		case err != nil:
			fmt.Printf("kolom penyaring : %s.%-14s gagal diperiksa: %v\n", p.tabel, p.kolom, err)
		default:
			fmt.Printf("kolom penyaring : %s.%-14s ada (%s)\n", p.tabel, p.kolom, jenis)
		}
	}

	// Bila kolomnya tidak ada, yang dibutuhkan berikutnya adalah nama PENGGANTINYA —
	// bukan sekadar kepastian bahwa ia tidak ada.
	cariMirip(db, "T_CLAIM_PNC", "%PANEL%")
	cariMirip(db, "T_CLAIM_PNC", "%BISNIS%")
	cariMirip(db, "T_CLAIM_PNC", "%BUSINESS%")
}

// cariMirip menyebut kolom satu tabel yang namanya cocok dengan sebuah pola.
func cariMirip(db *sql.DB, tabel, pola string) {
	rows, err := db.Query(
		"SELECT COLUMN_NAME, DATA_TYPE FROM ALL_TAB_COLUMNS WHERE OWNER = 'POOLDATA' "+
			"AND TABLE_NAME = :1 AND COLUMN_NAME LIKE :2 ORDER BY COLUMN_NAME", tabel, pola)
	if err != nil {
		fmt.Printf("kolom mirip %-12s gagal dicari: %v\n", pola, err)
		return
	}
	defer func() { _ = rows.Close() }()

	ada := false
	for rows.Next() {
		var nama, jenis string
		if err := rows.Scan(&nama, &jenis); err == nil {
			fmt.Printf("kolom mirip %-12s %s.%s (%s)\n", pola, tabel, nama, jenis)
			ada = true
		}
	}
	if !ada {
		fmt.Printf("kolom mirip %-12s %s: tidak ada yang cocok\n", pola, tabel)
	}
}
