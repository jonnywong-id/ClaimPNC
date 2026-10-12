package main

import (
	"context"
	"database/sql"
	"fmt"
)

// diagnosaNolBaris menjawab satu pertanyaan untuk kueri yang mengembalikan nol baris:
// apakah penyaringnya salah, atau datanya memang tidak ada.
//
// Caranya menambahkan syarat SATU DEMI SATU dan menghitung barisnya. Syarat mana yang
// menjatuhkan hitungannya ke nol — itulah yang perlu diperiksa.
//
// Seluruhnya HITUNGAN dan KODE STATUS. Tidak ada satu pun nilai milik nasabah yang dibaca
// maupun dicetak (`D-69`).
func diagnosaNolBaris(ctx context.Context, db *sql.DB) {
	const rantaiKomite = `
	  FROM pooldata.t_claim_adjustment a
	  JOIN pooldata.t_claim_pnc b ON a.claimid = b.claimid
	  JOIN datapega.pc_asm_fw_gcnmfw_work d ON b.claimid = d.pzinskey
	  JOIN pooldata.t_claim_komite_list k ON k.no_klaim = d.pyid AND k.komite_id = a.caseidkomite
	 WHERE b.grouppanel IN ('003','004','006','009')
	   AND b.branchname <> 'ASNET'`

	fmt.Println("Kenapa sebagian kueri mengembalikan nol baris — syarat ditambah satu demi satu:")

	kelompok := []struct {
		judul string
		uji   [][2]string
	}{
		{"report_klaim_pegipegi", [][2]string{
			{"klaim Travel seluruhnya",
				`SELECT COUNT(*) FROM pooldata.t_claim_pnc WHERE grouppanel = '005'`},
			{"+ berawalan polis mitra", // nilai awalannya tidak dicetak
				`SELECT COUNT(*) FROM pooldata.t_claim_pnc WHERE grouppanel = '005' AND nopolis LIKE '122N%'`},
		}},

		{"report_komite_nonmbu", [][2]string{
			{"rantai join lengkap + Non-MBU",
				`SELECT COUNT(*)` + rantaiKomite},
			{"+ punya tanggal tutup klaim",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND d.closeclaimdate_1 IS NOT NULL`},
			{"+ businesscode tidak dikecualikan",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND d.closeclaimdate_1 IS NOT NULL
				   AND b.businesscode NOT IN ('10145','10168','10165','10164','10053',
				                              '10075','10126','10011','10077','10007')`},
			{"+ komite berstatus approve 1",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND d.closeclaimdate_1 IS NOT NULL
				   AND k.komite_id IN (SELECT f.komite_id
				                         FROM pooldata.t_claim_komite_list f
				                        WHERE f.statusapprove = '1'
				                          AND f.no_klaim = d.pyid)`},
			{"+ tanggal tutup di dalam 2015..2026",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND CAST(d.closeclaimdate_1 AS DATE) >= DATE '2015-01-01'
				   AND CAST(d.closeclaimdate_1 AS DATE) <= DATE '2026-12-31'`},

			// Satu-satunya syarat yang belum diuji di atas, dan satu-satunya yang tersisa
			// sebagai penyebab. Ia ADA di sumber Pega — `pooldata.pega_dashboardpnc c`
			// dengan `b.claimno = c.noklaim` — jadi bila ini yang menjatuhkannya, nol
			// barisnya adalah perilaku sistem lama, bukan cacat pemindahan.
			{"+ JOIN pega_dashboardpnc (ada juga di Pega)",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND EXISTS (SELECT 1 FROM pooldata.pega_dashboardpnc c WHERE c.noklaim = b.claimno)`},

			// Tiap syarat SENDIRI meninggalkan puluhan ribu baris, tetapi kuerinya nol.
			// Berarti yang menjatuhkannya adalah GABUNGAN dua syarat, bukan salah satunya.
			// Pasangan yang paling mungkin: klaim yang sudah ditutup versus klaim yang
			// punya baris dashboard.
			{"+ tanggal tutup DAN dashboard sekaligus",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND d.closeclaimdate_1 IS NOT NULL
				   AND EXISTS (SELECT 1 FROM pooldata.pega_dashboardpnc c WHERE c.noklaim = b.claimno)`},
			{"    lalu + businesscode tidak dikecualikan",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND d.closeclaimdate_1 IS NOT NULL
				   AND EXISTS (SELECT 1 FROM pooldata.pega_dashboardpnc c WHERE c.noklaim = b.claimno)
				   AND b.businesscode NOT IN ('10145','10168','10165','10164','10053',
				                              '10075','10126','10011','10077','10007')`},
			{"    lalu + rentang tanggal 2015..2026",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND EXISTS (SELECT 1 FROM pooldata.pega_dashboardpnc c WHERE c.noklaim = b.claimno)
				   AND CAST(d.closeclaimdate_1 AS DATE) >= DATE '2015-01-01'
				   AND CAST(d.closeclaimdate_1 AS DATE) <= DATE '2026-12-31'`},
			{"    lalu + komite berstatus approve 1",
				`SELECT COUNT(*)` + rantaiKomite + `
				   AND d.closeclaimdate_1 IS NOT NULL
				   AND EXISTS (SELECT 1 FROM pooldata.pega_dashboardpnc c WHERE c.noklaim = b.claimno)
				   AND k.komite_id IN (SELECT f.komite_id
				                         FROM pooldata.t_claim_komite_list f
				                        WHERE f.statusapprove = '1'
				                          AND f.no_klaim = d.pyid)`},
		}},

		{"report_klaim_per_bisnis", [][2]string{
			{"baris dashboard seluruhnya",
				`SELECT COUNT(*) FROM pooldata.pega_dashboardpnc`},
			{"+ punya klaim yang cocok nomornya",
				`SELECT COUNT(*) FROM pooldata.pega_dashboardpnc a
				  WHERE EXISTS (SELECT 1 FROM pooldata.t_claim_pnc p WHERE p.claimno = a.noklaim)`},
			{"kode bisnis yang BENAR-BENAR dipakai klaim",
				`SELECT COUNT(DISTINCT p.businesscode) FROM pooldata.t_claim_pnc p`},
			{"  di antaranya ada di daftar pilihan panel",
				`SELECT COUNT(DISTINCT p.businesscode) FROM pooldata.t_claim_pnc p
				  WHERE p.businesscode IN (SELECT id FROM pooldata.business WHERE note IS NOT NULL)`},
		}},
	}

	for _, g := range kelompok {
		fmt.Println("  " + g.judul)
		for _, u := range g.uji {
			var n int64
			if err := db.QueryRowContext(ctx, u[1]).Scan(&n); err != nil {
				fmt.Printf("    %-46s GAGAL %s\n", u[0], ora.FindString(err.Error()))
				continue
			}
			fmt.Printf("    %-46s %d\n", u[0], n)
		}
	}
	fmt.Println()
}

// sebaranStatusApprove mencetak KODE status beserta jumlahnya — bukan data nasabah, dan
// inilah yang menjawab apakah nilai yang dipakai penyaring memang ada.
func sebaranStatusApprove(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx,
		`SELECT statusapprove, COUNT(*) FROM pooldata.t_claim_komite_list GROUP BY statusapprove ORDER BY 1`)
	if err != nil {
		fmt.Println("  sebaran statusapprove: GAGAL", ora.FindString(err.Error()))
		return
	}
	defer rows.Close()

	fmt.Print("  sebaran statusapprove:")
	for rows.Next() {
		var kode sql.NullString
		var n int64
		if err := rows.Scan(&kode, &n); err != nil {
			break
		}
		label := "NULL"
		if kode.Valid {
			label = kode.String
		}
		fmt.Printf("  %s=%d", label, n)
	}
	fmt.Println()
	fmt.Println()
}
