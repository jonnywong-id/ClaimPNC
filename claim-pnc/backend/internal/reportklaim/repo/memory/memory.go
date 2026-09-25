// Package memory memenuhi seam reportklaim.Repo tanpa basis data.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya bukan produksi:
//
//  1. Menjalankan aplikasi dengan `PENYIMPANAN=memori`, supaya layar Report Klaim dapat
//     dibuka dan diperiksa bentuknya tanpa kredensial Oracle sama sekali.
//  2. Menguji aturan modul tanpa database, tanpa jaringan, dan tanpa berkas
//     (`14-TESTING-STRATEGY.md` §3).
//
// # Barisnya SENGAJA tidak menyerupai data nyata
//
// Nilai contoh di bawah dibuat jelas-jelas buatan — nomor klaim `PNCN.26.0001`, nama
// "Contoh Tertanggung". Membuatnya menyerupai data sungguhan berarti mengundang orang
// menyimpulkan sesuatu dari berkas yang isinya karangan, dan `D-64` sudah menetapkan
// pengujian yang bermakna memakai salinan data produksi di staging — bukan data buatan.
package memory

import (
	"context"
	"fmt"

	"claim-pnc/internal/reportklaim"
)

// Repo menjawab dari memori.
type Repo struct {
	// rows adalah banyaknya baris contoh yang dihasilkan setiap laporan.
	rows int
}

// NewRepo membentuk repo memori.
func NewRepo() *Repo { return &Repo{rows: 3} }

// Stream menghasilkan beberapa baris contoh berisi SELURUH kolom laporan yang diminta.
//
// Kolomnya diambil dari katalog, bukan dikarang: dengan begitu berkas yang keluar punya
// susunan kolom yang SAMA dengan produksi, dan bentuk berkasnya dapat diperiksa tanpa
// basis data.
func (r *Repo) Stream(
	_ context.Context,
	report reportklaim.Report,
	filter reportklaim.Filter,
	emit func(reportklaim.Row) error,
) error {
	columns := report.Columns(filter)

	for i := 1; i <= r.rows; i++ {
		row := make(reportklaim.Row, len(columns))
		for _, c := range columns {
			row[c.Field] = fmt.Sprintf("contoh-%s-%d", c.Field, i)
		}
		// Beberapa kolom diberi bentuk yang dikenali supaya layar dan berkasnya terbaca
		// masuk akal saat diperiksa manusia.
		if _, ada := row["CaseID"]; ada {
			row["CaseID"] = fmt.Sprintf("PNCN.26.%04d", i)
		}
		if _, ada := row["ClaimNo"]; ada {
			row["ClaimNo"] = fmt.Sprintf("POLIS-CONTOH-%04d", i)
		}
		if err := emit(row); err != nil {
			return err
		}
	}
	return nil
}

// ListBusinessOptions mengembalikan beberapa pilihan bisnis contoh.
func (r *Repo) ListBusinessOptions(context.Context) ([]reportklaim.BusinessOption, error) {
	return []reportklaim.BusinessOption{
		{Code: "10076", Name: "Contoh Bisnis Aneka"},
		{Code: "10043", Name: "Contoh Bisnis Heavy Equipment"},
		{Code: "10083", Name: "Contoh Bisnis Bonding"},
	}, nil
}

// Selector mengembalikan RepoSelector yang melayani SATU alias portal saja.
//
// Portal lain ditolak, persis seperti adapter SQL menolaknya. Melayani seluruh alias dari
// satu penyimpanan memori akan membuat perpindahan portal tampak berhasil di
// pengembangan dan gagal di produksi — dan yang gagal di produksi adalah pemisahan data
// antar badan hukum (`R-20`).
func Selector(alias string) reportklaim.RepoSelector {
	repo := NewRepo()
	return func(requested string) (reportklaim.Repo, error) {
		if requested != alias {
			return nil, fmt.Errorf("reportklaim/memory: portal %q tidak tersedia; yang tersedia: %s", requested, alias)
		}
		return repo, nil
	}
}
