package sqlstore

import (
	"strings"
	"testing"
)

// Pernyataan baru benar-benar terdaftar dan berbentuk UPDATE pada tabel daftar kerja.
// Tanpa uji ini, salah ketik nama kueri baru terlihat saat tombolnya ditekan di produksi.
func TestMirrorDaftarKerjaTerdaftar(t *testing.T) {
	sql := query("mirror_daftar_kerja")
	for _, want := range []string{
		"UPDATE POOLDATA.T_CLAIMLIST_ADMIN",
		"PXASSIGNEDOPERATORID = :1",
		"PXTASKLABEL          = :2",
		"WHERE PYID = :3",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("mirror_daftar_kerja tidak memuat %q:\n%s", want, sql)
		}
	}
}

// technical_pic TIDAK BOLEH lagi digerakkan tabel kerja Pega: klaim PNCN tidak punya
// baris di sana, dan kueri yang menggerakkannya dari sana menolak setiap klaim PNCN.
func TestTechnicalPICTidakBergantungTabelPega(t *testing.T) {
	sql := query("technical_pic")
	if strings.Contains(sql, "FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK") {
		t.Fatalf("technical_pic masih digerakkan tabel kerja Pega:\n%s", sql)
	}
	for _, want := range []string{"c.CLAIMID", "c.PICTEKNIK", "FROM POOLDATA.T_CLAIM_PNC"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("technical_pic tidak memuat %q:\n%s", want, sql)
		}
	}
}

// work_object_key TIDAK BOLEH digerakkan tabel kerja Pega, dengan alasan yang sama persis
// seperti technical_pic di atas. Yang mati karenanya bukan satu tombol melainkan TIGA:
// penerbitan surat RCL/PUCL, unggah dokumen, dan penulisan riwayat klaim — seluruhnya untuk
// klaim PNCN, dan dua di antaranya gagal TANPA galat yang terbaca petugas.
func TestWorkObjectKeyTidakBergantungTabelPega(t *testing.T) {
	sql := query("work_object_key")
	if strings.Contains(sql, "FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK") {
		t.Fatalf("work_object_key masih digerakkan tabel kerja Pega:\n%s", sql)
	}
	for _, want := range []string{"c.CLAIMID", "FROM POOLDATA.T_CLAIM_PNC"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("work_object_key tidak memuat %q:\n%s", want, sql)
		}
	}
}

// Kedua kueri dokumen mengakui kepemilikan lewat TIGA bentuk kunci. Mengurangi salah satunya
// membuat dokumen klaim PNCN hilang dari daftar, atau — lebih buruk — tergambar di daftar
// tetapi menolak diunduh. Keduanya diuji bersama supaya tidak dapat menyimpang.
func TestKueriDokumenMengakuiKlaimPNCN(t *testing.T) {
	for _, nama := range []string{"documents", "document_content"} {
		sql := query(nama)
		for _, want := range []string{
			"FROM POOLDATA.T_CLAIM_PNC c",
			"c.CLAIMID",
			"'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)",
			"OR a.IDPEGA = 'ASM-FW-GCNMFW-WORK ' || TRIM(:",
		} {
			if !strings.Contains(sql, want) {
				t.Fatalf("kueri %s tidak memuat %q:\n%s", nama, want, sql)
			}
		}
		// Sejak 2026-10-08 kunci ketiga dirangkai dari nomor case; tabel objek kerja Pega
		// tidak dipakai lagi.
		if strings.Contains(sql, "PC_ASM_FW_GCNMFW_WORK") {
			t.Fatalf("kueri %s masih membaca tabel objek kerja Pega:\n%s", nama, sql)
		}
	}
}

// letters_missing memakai penyaring yang SAMA PERSIS dengan migrasi pemulihnya
// (`migrations/0015_pucl_cetak_tanpa_surat.up.sql`). Keduanya hidup di berkas yang berbeda
// dan akan menyimpang bila tidak dijaga — dan penyimpangannya menghasilkan angka yang
// melaporkan klaim bermasalah sementara migrasinya tidak menyentuhnya, atau sebaliknya.
func TestLettersMissingMemakaiPenyaringYangSama(t *testing.T) {
	sql := query("letters_missing")
	for _, want := range []string{
		"p.TGL_CETAK_DOKUMEN_PUCL IS NOT NULL",
		"TRIM(p.PUCL_APPROVE) <> '1'",
		"p.MSIG IS NULL",
		"TRIM(a.CATEGORY) = 'Notification'",
		"'PUCL.pdf', 'RCL.pdf', 'Notification.pdf'",
		"'ASM-FW-GCNMFW-WORK ' || TRIM(c.CLAIMNO)",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("letters_missing tidak memuat %q:\n%s", want, sql)
		}
	}
}

// Penjagaan "sudah di Analyst" bertumpu pada kueri ini; salah ketik namanya baru terlihat
// saat tombolnya ditekan di produksi.
func TestTahapTugasTerbukaTerdaftar(t *testing.T) {
	sql := query("tahap_tugas_terbuka")
	for _, want := range []string{"SELECT TAHAP FROM CPNC_TUGAS", "SELESAI_PADA IS NULL"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("tahap_tugas_terbuka tidak memuat %q:\n%s", want, sql)
		}
	}
}
