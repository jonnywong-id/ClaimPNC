package sqlstore

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"claim-pnc/internal/inboxlaporanklaim"
)

// oracleTableMissing adalah teks galat yang BENAR-BENAR dikembalikan Oracle produksi
// pada 2026-09-23 ketika POOLDATA.CPNC_LAPORAN_KLAIM belum dibuat, disalin apa adanya
// dari log — termasuk baris kedua yang ditambahkan driver.
//
// Disalin dari galat nyata, bukan dikarang, karena yang diuji di sini justru apakah
// bentuk nyata itu dikenali. Galat karangan yang lebih rapi daripada aslinya akan lulus
// tanpa membuktikan apa pun.
const oracleTableMissing = "ORA-00942: table or view does not exist\n error occur at position: 240"

// Kegagalan pada tabel milik aplikasi harus berubah menjadi galat yang menyebut
// perbaikannya, bukan diteruskan sebagai galat basis data biasa.
func TestTabelSendiriYangBelumAdaMenjadiGalatBerpenjelasan(t *testing.T) {
	asli := fmt.Errorf("menyisipkan %q: %w", "RCVN.26.1", errors.New(oracleTableMissing))

	hasil := ownTable(asli)

	if !errors.Is(hasil, inboxlaporanklaim.ErrStorageNotReady) {
		t.Fatalf("seharusnya menjadi ErrStorageNotReady; hasil: %v", hasil)
	}
	// Galat aslinya tetap terbawa. Log operator membutuhkan kode ORA-nya; menggantinya
	// dengan kalimat sendiri akan menghapus satu-satunya petunjuk yang dapat dicari.
	if !strings.Contains(hasil.Error(), "ORA-00942") {
		t.Errorf("kode ORA aslinya hilang dari galat: %v", hasil)
	}
}

// Kegagalan lain diteruskan APA ADANYA.
//
// Membungkus semuanya sebagai "tabel belum dibuat" akan menyembunyikan hak akses yang
// kurang dan bind yang salah di balik saran yang keliru — dan saran yang keliru lebih
// mahal daripada tidak ada saran sama sekali.
func TestKegagalanLainDiteruskanApaAdanya(t *testing.T) {
	asli := errors.New("ORA-01031: insufficient privileges")

	if hasil := ownTable(asli); !errors.Is(hasil, asli) {
		t.Errorf("galat lain tidak boleh diubah; hasil: %v", hasil)
	}
	if errors.Is(ownTable(asli), inboxlaporanklaim.ErrStorageNotReady) {
		t.Error("hak akses yang kurang tidak boleh terbaca sebagai tabel yang belum dibuat")
	}
}
