package sqlstore

import (
	"regexp"
	"strings"
	"testing"
)

// Pengiriman ULANG ke RCL/PUCL wajib mencabut kedua penanda siklus sebelumnya.
//
// # Kenapa uji ini ada
//
// `PUCL_APPROVE` dan `TGL_CETAK_DOKUMEN_PUCL` menyatakan POSISI klaim di dalam satu siklus
// RCL/PUCL, bukan catatan milik petugas. Keduanya sempat masuk daftar "tidak disentuh saat
// memperbarui" bersama `KOMENTAR_PUCL`, `MSIG`, dan `TGL_TERIMA_DOKUMEN_PUCL` — alasan yang
// benar untuk ketiga kolom itu dan salah untuk kedua kolom ini.
//
// Akibat membawanya ke siklus baru: klaim gugur dari KETIGA tab Inbox RCL/PUCL sekaligus,
// tanpa satu pun galat.
//
//	tab 1 Cetak Surat          TGL_CETAK_DOKUMEN_PUCL IS NULL   -> terisi, gugur
//	tab 2 Kelengkapan Dokumen  PUCL_APPROVE <> '1'              -> '1',    gugur
//	tab 3 Klaim MSIG           MSIG = 'MSIG'                    -> NULL,   gugur
//
// Terukur pada `PNCN.26.31` (2026-10-05): "Kirim Ke Analyst" 06:36 menyetel
// `PUCL_APPROVE = '1'`, lalu pengiriman ulang 08:18 memperbarui suratnya tanpa mencabutnya.
// Klaimnya tidak terlihat di tab mana pun sementara tugasnya berjalan normal.
//
// Kegagalannya SENYAP — tidak ada galat, tidak ada baris log, dan layarnya tampak wajar
// karena klaim lain tetap tergambar. Itu sebabnya penjagaannya dipasang di sini, bukan
// diserahkan kepada pembacaan ulang berkas `.sql`.
func TestPengirimanUlangRCLPUCLMencabutPenandaSiklusSebelumnya(t *testing.T) {
	body := stripComments(loadQuery("surat_rclpucl_perbarui"))

	for _, kolom := range []string{"PUCL_APPROVE", "TGL_CETAK_DOKUMEN_PUCL"} {
		if !strings.Contains(body, kolom+" ") && !strings.Contains(body, kolom+"=") {
			t.Errorf("surat_rclpucl_perbarui tidak menyebut %s sama sekali.\n"+
				"Kolom ini WAJIB dikosongkan saat pengiriman ulang; membiarkannya "+
				"membuat klaim gugur dari ketiga tab Inbox RCL/PUCL tanpa galat.", kolom)
			continue
		}
		if !mengosongkan(body, kolom) {
			t.Errorf("surat_rclpucl_perbarui menyebut %s tetapi tidak mengosongkannya ke NULL.\n"+
				"Siklus baru harus bermula dari keadaan yang SAMA dengan pengiriman pertama, "+
				"dan surat_rclpucl_sisip tidak menulis kolom ini sama sekali.", kolom)
		}
	}

	// Ketiga kolom berikut memang milik petugas RCL/PUCL. Menyentuhnya dari sini menghapus
	// pekerjaan orang lain — kebalikan dari cacat di atas, dan sama merusaknya.
	for _, kolom := range []string{"KOMENTAR_PUCL", "MSIG", "TGL_TERIMA_DOKUMEN_PUCL"} {
		if strings.Contains(body, kolom) {
			t.Errorf("surat_rclpucl_perbarui menyentuh %s; kolom itu milik petugas RCL/PUCL "+
				"dan menimpanya dari jalur pengiriman akan menghapus pekerjaannya", kolom)
		}
	}

	// UPDATE dan INSERT wajib SEPADAN jumlah bind-nya.
	//
	// `rclpucl.go` memakai SATU daftar argumen untuk keduanya — `update` menutup dengan
	// CLAIMID, `insert` membukanya — dan go-ora mengikat menurut urutan KEMUNCULAN penanda,
	// bukan menurut nomornya. Satu bind yang bertambah di satu kueri saja menggeser seluruh
	// sisanya **tanpa galat sintaks**: nilainya masuk ke kolom yang salah.
	//
	// Angkanya sengaja TIDAK dipatok konstanta. Patokan 25 pernah menahan penambahan kolom
	// yang sah (`NAMA_DOKTER_RCL`, 2026-10-07) dan memaksa uji ini disunting bersama setiap
	// kolom baru — padahal yang benar-benar dijaga adalah kesepadanannya, bukan angkanya.
	// Dihitung dengan pola `:angka`, bukan dengan mencacah aksara `:` — prosa di kepala
	// kueri pun memuat titik dua, dan mencacahnya menghasilkan angka yang tidak berarti.
	penanda := regexp.MustCompile(`:\d+`)
	perbarui := len(penanda.FindAllString(body, -1))
	sisip := len(penanda.FindAllString(loadQuery("surat_rclpucl_sisip"), -1))
	if perbarui != sisip {
		t.Errorf("surat_rclpucl_perbarui memuat %d penanda bind, surat_rclpucl_sisip %d.\n"+
			"Keduanya memakai satu daftar argumen yang sama; selisih ini menggeser nilai ke "+
			"kolom yang salah tanpa galat sintaks.", perbarui, sisip)
	}

	// Dan pengosongan kedua kolom siklus tetap LITERAL, bukan bind — itu yang menjaga
	// keduanya sepadan tanpa menambah argumen di sisi Go.
	for _, kolom := range []string{"PUCL_APPROVE", "TGL_CETAK_DOKUMEN_PUCL"} {
		if strings.Contains(body, kolom+"           = :") || strings.Contains(body, kolom+" = :") {
			t.Errorf("%s dikosongkan lewat penanda bind; tulis NULL sebagai literal supaya "+
				"daftar argumen UPDATE dan INSERT tetap satu", kolom)
		}
	}
}

// mengosongkan menyatakan kolom disetel ke NULL di dalam klausa SET.
func mengosongkan(body, kolom string) bool {
	for _, line := range strings.Split(body, "\n") {
		teks := strings.TrimSpace(line)
		if !strings.HasPrefix(teks, kolom) {
			continue
		}
		sisa := strings.TrimSpace(strings.TrimPrefix(teks, kolom))
		if !strings.HasPrefix(sisa, "=") {
			continue
		}
		nilai := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(sisa, "=")), ","))
		return strings.EqualFold(nilai, "NULL")
	}
	return false
}
