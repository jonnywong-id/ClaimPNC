package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/money"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestSeluruhKueriYangDipakaiAda(t *testing.T) {
	dipakai := []string{
		"ambang_komite_daftar",
		"ambang_komite_periksa_tabel",
	}
	for _, nama := range dipakai {
		t.Run(nama, func(t *testing.T) {
			require.NotPanics(t, func() { _ = query(nama) })
			require.NotEmpty(t, strings.TrimSpace(query(nama)))
		})
	}
}

func TestKueriYangTidakAdaMenimbulkanPanik(t *testing.T) {
	require.Panics(t, func() { _ = query("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestKueriMematuhiDisiplinSQLPortabel(t *testing.T) {
	terlarang := []struct {
		pola   string
		alasan string
	}{
		{"SELECT *", "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku"},
		{"NVL(", "pakai COALESCE"},
		{"SYSDATE", "pakai CURRENT_TIMESTAMP"},
		{"DECODE(", "pakai CASE WHEN"},
		{"ROWNUM", "pakai OFFSET ... FETCH NEXT ... ROWS ONLY"},
		{"INSTR(", "pakai POSITION"},
		{"LISTAGG(", "pakai STRING_AGG"},
		{"TO_CHAR(", "pemformatan tanggal dan angka dilakukan di Go"},
		{"LPAD(", "pemformatan angka dilakukan di Go"},
		{"FROM DUAL", "tidak ada urutan maupun ekspresi tanpa tabel di modul ini"},
		// Pengacakan pada jalur Simasnet DIBAWA — Work Owner menegaskan 2026-09-18 bahwa
		// ia disengaja: ia menyebar beban di antara beberapa orang yang berwenang pada
		// tingkat yang sama, sekaligus mengecualikan orang yang mengajukan.
		//
		// Yang dilarang di sini bukan perilakunya, melainkan TEMPATNYA. Diacak di dalam
		// SQL membuat aturannya tidak dapat diuji sama sekali — hasil yang berbeda tiap
		// kali dijalankan tidak dapat dibandingkan dengan apa pun. Pengacakannya karena
		// itu pindah ke Go, di balik seam `komite.Randomizer`, sehingga pengujian memakai
		// pemilih tetap sementara produksi tetap mengacak.
		{"DBMS_RANDOM", "pengacakan ada di Go di balik seam, bukan di dalam SQL"},
	}

	for nama, teks := range kueri {
		hurufBesar := strings.ToUpper(teks)
		for _, larangan := range terlarang {
			require.NotContainsf(t, hurufBesar, larangan.pola,
				"kueri %q memakai %q — %s", nama, larangan.pola, larangan.alasan)
		}
		require.NotContainsf(t, hurufBesar, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", nama)
	}
}

// Modul ini MEMBACA SAJA. Tidak satu pun kueri boleh mengubah isi tabel.
//
// Uji ini adalah penegak keputusan Work Owner 2026-09-17 dan `P-1`: POOLDATA.EMAILKOMITE
// masih ditulis Pega dan dibaca 17 kueri di sana. Memindahkan kepemilikannya menuntut
// prosedur `D-63`, dan sampai itu ditempuh, satu pernyataan tulis yang lolos ke sini akan
// membuat dua sistem menulis tabel yang sama — kelas kerusakan data yang hampir mustahil
// dilacak.
func TestTidakAdaKueriYangMenulis(t *testing.T) {
	menulis := []string{"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE", "DROP ", "ALTER "}

	for nama, teks := range kueri {
		hurufBesar := strings.ToUpper(teks)
		for _, pola := range menulis {
			require.NotContainsf(t, hurufBesar, pola,
				"kueri %q memakai %q — master ambang komite dibaca saja", nama, pola)
		}
	}
}

// Seluruh kueri hanya menyentuh EMAILKOMITE. Menyentuh tabel lain berarti modul ini
// mengambil lingkup yang bukan miliknya.
func TestKueriHanyaMenyentuhTabelAmbang(t *testing.T) {
	for nama, teks := range kueri {
		require.Containsf(t, strings.ToUpper(teks), "POOLDATA.EMAILKOMITE",
			"kueri %q tidak menyentuh tabel ambang", nama)
	}
}

// Kolom EMAIL dan CC TIDAK boleh ikut dibaca.
//
// `D-67` menetapkan alamat pribadi pada master lama — sekurang-kurangnya enam akun Gmail
// di jalur produksi — tidak dibawa ke sistem baru, dan `D-69` mewajibkan alamat surel
// disamarkan di seluruh artefak. Tidak membacanya sejak kueri membuat alamat itu tidak
// pernah sampai ke peramban, alih-alih mengandalkan setiap lapisan sesudahnya ingat
// membuangnya.
func TestKueriTidakMembacaAlamatSurel(t *testing.T) {
	for nama, teks := range kueri {
		hurufBesar := strings.ToUpper(teks)
		require.NotContainsf(t, hurufBesar, "EMAIL,",
			"kueri %q membaca kolom EMAIL", nama)
		require.NotRegexpf(t, `(?m)^\s*CC\s*,?\s*$`, hurufBesar,
			"kueri %q membaca kolom CC", nama)
	}
}

// Kolom yang dibaca harus tepat sama dengan yang dipindai kode. Selisih satu kolom
// membuat Scan gagal dengan pesan yang tidak menyebut kolom mana yang salah.
func TestJumlahKolomSesuaiDenganYangDipindai(t *testing.T) {
	const jumlahDipindai = 13 // lihat scanRow

	for _, nama := range []string{"ambang_komite_daftar", "ambang_komite_periksa_tabel"} {
		t.Run(nama, func(t *testing.T) {
			teks := query(nama)
			bagian := teks[strings.Index(strings.ToUpper(teks), "SELECT")+len("SELECT"):]
			bagian = bagian[:strings.Index(strings.ToUpper(bagian), "FROM")]
			require.Len(t, strings.Split(bagian, ","), jumlahDipindai,
				"jumlah kolom di kueri harus sama dengan yang dipindai scanRow")
		})
	}
}

// Penafsiran nilai kolom diuji tersendiri, karena tipe kolom yang sebenarnya BELUM
// diketahui — DDL tabel ini tidak pernah kita lihat (`R-08` masih terbuka).
//
// Modul ini karena itu harus tahan terhadap kedua kemungkinan: angka maupun teks.
func TestPenafsiranNilaiKolom(t *testing.T) {
	t.Run("penanda menyala hanya untuk 1 dan Y", func(t *testing.T) {
		for _, menyala := range []any{"1", " 1 ", "Y", "y", int64(1), float64(1), true} {
			require.Truef(t, toFlag(menyala), "%v seharusnya menyala", menyala)
		}
		// Kosong, NULL, dan "0" padam. Ini bukan tempat bermurah hati: STS_ADJ yang
		// salah dibaca menyala akan memasukkan baris pemberitahuan registrasi ke dalam
		// daftar penyetuju uang.
		for _, padam := range []any{nil, "", "0", " ", "N", int64(0), float64(0), false} {
			require.Falsef(t, toFlag(padam), "%v seharusnya padam", padam)
		}
	})

	t.Run("teks dirapikan dan tidak pernah bernotasi ilmiah", func(t *testing.T) {
		require.Equal(t, "", toText(nil))
		require.Equal(t, "4", toText(" 4 "))
		require.Equal(t, "MARTENPETRUSLALAMENTIK_1", toText("MARTENPETRUSLALAMENTIK_1\n"))
		require.Equal(t, "7", toText(int64(7)))
		// 'g' akan menghasilkan "1e+08" di sini; itulah sebabnya toText memakai 'f'.
		require.Equal(t, "100000000", toText(float64(100000000)))
	})

	t.Run("bilangan terbaca dari angka maupun teks", func(t *testing.T) {
		for _, masuk := range []any{int64(3), float64(3), "3", " 3 "} {
			hasil, err := toInt(masuk)
			require.NoError(t, err)
			require.Equal(t, 3, hasil)
		}

		// DEGREE kosong menjadi nol, bukan galat: baris seperti itu memang ada di
		// master dan toh bukan jenjang persetujuan.
		hasil, err := toInt(nil)
		require.NoError(t, err)
		require.Zero(t, hasil)
	})

	t.Run("nilai uang terbaca dari angka maupun teks", func(t *testing.T) {
		for _, masuk := range []any{int64(50_000_001), float64(50_000_001), "50000001"} {
			hasil, err := money.FromSQLValue(masuk)
			require.NoError(t, err)
			require.Equal(t, money.FromRupiah(50_000_001), hasil)
		}
	})
}
