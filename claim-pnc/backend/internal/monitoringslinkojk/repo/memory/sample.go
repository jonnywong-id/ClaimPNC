package memory

import (
	"time"

	"claim-pnc/internal/monitoringslinkojk"
)

// NewSampleRepo membentuk penyimpanan berisi data contoh.
//
// # Apa yang dijamin data ini
//
// Ia mencakup SELURUH jalur layar, termasuk empat yang paling mudah terlewat:
//
//   - **Satu klaim dengan DUA fasilitas kredit.** Kunci barisnya harus tetap berbeda;
//     bila kunci dirangkai dari nomor klaim saja, tabel di layar akan menampilkan salah
//     satunya dua kali dan yang satunya hilang.
//   - **Baris berlini SURETY BOND.** Ia hanya muncul ketika penyaring Business Name
//     DIBALIK, dan itulah satu-satunya cara membuktikan "SURETY BOND" meniadakan
//     Asuransi Kredit alih-alih memilih Surety Bond.
//   - **Baris tepat di batas rentang tanggal**, diregistrasi pukul 23.40. Ia hilang
//     seketika bila batas atas diperlakukan `<=` terhadap tengah malam.
//   - **Kolom bernilai kosong.** Kolom laporan regulator memang bisa kosong, dan layar
//     harus tetap terbaca ketika itu terjadi.
//
// Seluruh nilainya KARANGAN. Tidak ada nomor polis, nama tertanggung, maupun nomor
// rekening nyata di sini (`D-69`).
func NewSampleRepo() *Repo {
	return NewRepo(
		WithRows(monitoringslinkojk.SegmentD01, sampleD01()),
		WithRows(monitoringslinkojk.SegmentF06, sampleF06()),
	)
}

// sampleDate menyusun waktu registrasi contoh.
//
// UTC, sejalan dengan `08-TECHNICAL-STRATEGY.md` §4.4: seluruh waktu disimpan UTC dan
// dikonversi ke WIB hanya di satu tempat.
func sampleDate(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func sampleD01() []Record {
	const kredit = monitoringslinkojk.CreditInsuranceBusinessType

	return []Record{
		{
			RegisteredAt: sampleDate(2026, time.March, 4, 9, 15),
			BusinessType: kredit,
			Row: monitoringslinkojk.Row{
				monitoringslinkojk.RowKeyColumn: "PNCN.26.0101|KTR-2026-0101",

				"no_klaim":                 "PNCN.26.0101",
				"contract_no":              "KTR-2026-0101",
				"nomor_rekening_fasilitas": "9900000101",
				"no_cif_debitur":           "CIF0000101",
				"kode_jenis_fasilitas":     "10",
				"sumber_dana":              "01",
				"start_polis":              "01/01/2026",
				"end_polis":                "31/12/2026",
				"suku_bunga":               "12.5",
				"kode_valuta":              "IDR",
				"nilai_mata_uang_asal":     "IDR 150000000",

				// Kedua kolom tanggal ini SENGAJA bernilai sama. Di produksi keduanya
				// memang terisi dari kolom yang sama — lihat d01Columns.
				"tanggal_pembayaran": "04/03/2026",
				"tanggal_kondisi":    "04/03/2026",

				"kode_kolektibilitas": "4",
				"tanggal_macet":       "04/03/2026",
				"kode_sebab_macet":    "02",
				"tunggakan":           "45000000",
				"jumlah_kewajiban":    "150000000",
				"kode_kondisi":        "01",
				"keterangan":          "Klaim kredit macet, debitur perorangan",
				"kode_kantor_cabang":  "001",
				"operasi_data":        "C",
				"no_ktp":              "",
				"npwp_perusahaan":     "",
				"no_polis":            "POL-CONTOH-0101",
			},
		},
		{
			// Klaim yang SAMA dengan baris di atas, fasilitas kredit KEDUA.
			RegisteredAt: sampleDate(2026, time.March, 4, 9, 15),
			BusinessType: kredit,
			Row: monitoringslinkojk.Row{
				monitoringslinkojk.RowKeyColumn: "PNCN.26.0101|KTR-2026-0102",

				"no_klaim":                 "PNCN.26.0101",
				"contract_no":              "KTR-2026-0102",
				"nomor_rekening_fasilitas": "9900000102",
				"no_cif_debitur":           "CIF0000101",
				"kode_jenis_fasilitas":     "10",
				"sumber_dana":              "01",
				"start_polis":              "01/01/2026",
				"end_polis":                "31/12/2026",
				"suku_bunga":               "11.75",
				"kode_valuta":              "IDR",
				"nilai_mata_uang_asal":     "IDR 75000000",
				"tanggal_pembayaran":       "04/03/2026",
				"tanggal_kondisi":          "04/03/2026",
				"kode_kolektibilitas":      "3",
				"tanggal_macet":            "04/03/2026",
				"kode_sebab_macet":         "02",
				"tunggakan":                "12000000",
				"jumlah_kewajiban":         "75000000",
				"kode_kondisi":             "01",

				// Keterangan KOSONG — kolom laporan memang bisa kosong.
				"keterangan": "",

				"kode_kantor_cabang": "001",
				"operasi_data":       "C",
				"no_ktp":             "",
				"npwp_perusahaan":    "",
				"no_polis":           "POL-CONTOH-0101",
			},
		},
		{
			// Lini SELAIN Asuransi Kredit — hanya muncul pada pilihan "SURETY BOND".
			RegisteredAt: sampleDate(2026, time.March, 31, 23, 40),
			BusinessType: "Bonding",
			Row: monitoringslinkojk.Row{
				monitoringslinkojk.RowKeyColumn: "PNCN.26.0207|KTR-2026-0207",

				"no_klaim":                 "PNCN.26.0207",
				"contract_no":              "KTR-2026-0207",
				"nomor_rekening_fasilitas": "9900000207",
				"no_cif_debitur":           "CIF0000207",
				"kode_jenis_fasilitas":     "20",
				"sumber_dana":              "02",
				"start_polis":              "15/02/2026",
				"end_polis":                "14/02/2027",
				"suku_bunga":               "0",
				"kode_valuta":              "IDR",
				"nilai_mata_uang_asal":     "IDR 500000000",
				"tanggal_pembayaran":       "31/03/2026",
				"tanggal_kondisi":          "31/03/2026",
				"kode_kolektibilitas":      "1",
				"tanggal_macet":            "31/03/2026",
				"kode_sebab_macet":         "",
				"tunggakan":                "0",
				"jumlah_kewajiban":         "500000000",
				"kode_kondisi":             "00",
				"keterangan":               "Surety bond, belum ada tunggakan",
				"kode_kantor_cabang":       "001",
				"operasi_data":             "U",
				"no_ktp":                   "",
				"npwp_perusahaan":          "",
				"no_polis":                 "POL-CONTOH-0207",
			},
		},
		{
			// Di LUAR rentang contoh Maret — memastikan penyaring tanggal benar-benar
			// menyaring, bukan meloloskan seluruhnya.
			RegisteredAt: sampleDate(2026, time.January, 12, 8, 0),
			BusinessType: kredit,
			Row: monitoringslinkojk.Row{
				monitoringslinkojk.RowKeyColumn: "PNCN.26.0033|KTR-2026-0033",

				"no_klaim":                 "PNCN.26.0033",
				"contract_no":              "KTR-2026-0033",
				"nomor_rekening_fasilitas": "9900000033",
				"no_cif_debitur":           "CIF0000033",
				"kode_jenis_fasilitas":     "10",
				"sumber_dana":              "01",
				"start_polis":              "01/01/2026",
				"end_polis":                "31/12/2026",
				"suku_bunga":               "13",
				"kode_valuta":              "IDR",
				"nilai_mata_uang_asal":     "IDR 20000000",
				"tanggal_pembayaran":       "12/01/2026",
				"tanggal_kondisi":          "12/01/2026",
				"kode_kolektibilitas":      "5",
				"tanggal_macet":            "12/01/2026",
				"kode_sebab_macet":         "03",
				"tunggakan":                "20000000",
				"jumlah_kewajiban":         "20000000",
				"kode_kondisi":             "02",
				"keterangan":               "Klaim lunas",
				"kode_kantor_cabang":       "001",
				"operasi_data":             "C",
				"no_ktp":                   "",
				"npwp_perusahaan":          "",
				"no_polis":                 "POL-CONTOH-0033",
			},
		},
	}
}

// sampleF06 menyusun baris contoh segmen F06.
//
// Hanya kedelapan kolom bersumber yang diisi — sama persis dengan yang dihasilkan
// penyimpanan SQL. Mengisi ketiga puluh kolom lainnya di sini akan membuat layar tampak
// lengkap saat dikembangkan lalu kosong saat dijalankan terhadap Oracle, dan perbedaan
// itu baru ketahuan di produksi.
// sampleF06 menyusun baris contoh segmen F06.
//
// Ia MENURUNKAN barisnya dari sampleD01, dan itu bukan kemalasan: grid kedua segmen
// memakai daftar kolom yang sama (lihat f06Columns), sehingga dua salinan yang berbeda
// hanya akan membuat salah satunya usang tanpa ada yang menyadarinya.
//
// Yang ditambahkan adalah kunci khusus BERKAS EKSPOR F06 — `nomor_cif_debitur`,
// `jenis_kelamin`, `tanggal_lahir`, `alamat`, `kode_pos`, `telepon` — yang tidak tampil
// di grid tetapi dipakai ekspornya.
//
// Di produksi keduanya berbeda asal: D01 membaca tabel SLIK yang SUDAH tersusun, F06
// membaca data klaim SUMBERNYA. Perbedaan itu tidak dapat ditiru repo dalam memori, dan
// memang bukan yang diujinya.
func sampleF06() []Record {
	source := sampleD01()
	out := make([]Record, 0, len(source))

	for _, record := range source {
		row := make(monitoringslinkojk.Row, len(record.Row)+6)
		for key, value := range record.Row {
			row[key] = value
		}

		row["nomor_cif_debitur"] = record.Row.Get("no_cif_debitur")
		row["jenis_kelamin"] = "L"
		row["tanggal_lahir"] = "17/08/1985"
		row["alamat"] = "Jalan Contoh Nomor 1"
		row["kode_pos"] = "12345"
		row["telepon"] = "0210000000"

		record.Row = row
		out = append(out, record)
	}
	return out
}
