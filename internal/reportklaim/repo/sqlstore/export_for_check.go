package sqlstore

import "claim-pnc/internal/reportklaim"

// Berkas ini membuka isi paket bagi perkakas pemeriksa `cmd/cekddl`. Ia TIDAK dipakai
// aplikasi, dan tidak boleh dipakai aplikasi — yang dibukanya adalah bahan mentah kueri,
// bukan jalan pintas melewati Repo.

// AllQueriesForCheck membuka daftar kueri untuk perkakas pemeriksa.
//
// Keberadaannya memungkinkan satu perkakas mem-parse seluruh kueri terhadap basis data
// nyata tanpa menjalankannya — cara menemukan nama kolom yang keliru dibaca dari export
// sebelum pengguna menekan tombol Export.
func AllQueriesForCheck() map[string]string {
	out := make(map[string]string, len(query))
	for k, v := range query {
		out[k] = v
	}
	return out
}

// PlanForCheck adalah satu kueri beserta parameter yang BENAR-BENAR dikirim aplikasi.
type PlanForCheck struct {
	// Code adalah laporan pemiliknya, atau kosong untuk kueri master.
	Code string

	// Name adalah nama kueri di berkas .sql.
	Name string

	// Args adalah nilai bind, disusun oleh penyusun argumen yang sama dengan yang dipakai
	// Repo — bukan ditebak perkakasnya.
	Args []any
}

// PlansForCheck menyusun seluruh kueri laporan untuk satu penyaring.
//
// Penting bahwa ia memakai `plans` yang sama dengan Repo: kalau perkakasnya menyusun bind
// sendiri, yang diujinya bukan yang dijalankan aplikasi — dan ketidakcocokan urutan bind,
// kelas cacat yang paling mudah terjadi di sini, justru tidak akan pernah ketahuan.
//
// Tiga laporan memakai kueri BERBEDA untuk Non-MBU, sehingga penyaring dipanggil dua kali:
// sekali untuk lini yang diminta, sekali untuk Non-MBU. Yang berulang dibuang.
func PlansForCheck(f reportklaim.Filter) []PlanForCheck {
	seen := map[string]bool{}
	var out []PlanForCheck

	lines := []reportklaim.BusinessLine{f.BusinessLine, reportklaim.BusinessLineNonMBU}

	for code, p := range plans {
		for _, line := range lines {
			g := f
			g.BusinessLine = line

			name := p.query(g)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, PlanForCheck{Code: string(code), Name: name, Args: p.args(g)})
		}
	}
	return out
}
