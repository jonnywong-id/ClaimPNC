package sqlstore

import (
	"context"
	"sort"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
)

// SourceState adalah keadaan sumber data modul Report Klaim pada satu portal.
//
// Ia hanya memuat hal-hal yang TIDAK dapat diketahui dari kode. Yang dapat dibaca dari
// kode — daftar panel, kolomnya, penyaringnya — sengaja tidak diulang di sini.
type SourceState struct {
	// SecondConnection: koneksi kedua portal (`ANEKA_<PORTAL_ALIAS>_*`) terpasang.
	//
	// Ketiadaannya BUKAN galat. Ia menentukan apakah kolom hari kerja pada Report TAT
	// terisi atau ditandai tidak diketahui — dan perbedaan itu tidak terlihat sebagai
	// kegagalan di layar mana pun.
	SecondConnection bool

	// HolidayReadable: kalender libur benar-benar dapat dibaca lewat koneksi kedua.
	//
	// Terpasang dan dapat dibaca adalah dua hal berbeda: koneksinya boleh hidup
	// sementara akun aplikasi tidak punya hak SELECT atas tabelnya.
	HolidayReadable bool
	HolidayError    error

	// HolidayDays: jumlah hari libur yang tercatat untuk tahun berjalan.
	//
	// Nol bukan kegagalan, tetapi ia perlu disebut: kalender kosong menghasilkan
	// perhitungan hari kerja yang hanya memotong akhir pekan, dan hasilnya terlihat
	// wajar.
	HolidayDays int
	HolidayYear int
}

// CheckSource memeriksa kesiapan sumber data modul ini.
//
// Ia tidak menyentuh satu pun kueri laporan. Menjalankan 25 kueri laporan atas data
// produksi hanya untuk memeriksa kesiapan akan memakan waktu yang tidak sepadan, dan
// kegagalannya pun tidak dapat dibedakan dari penyaring yang kebetulan kosong.
func (r *Repo) CheckSource(ctx context.Context) SourceState {
	state := SourceState{
		SecondConnection: r.aneka != nil,
		HolidayYear:      time.Now().In(clock.ZoneWIB).Year(),
	}
	if !state.SecondConnection {
		return state
	}

	from := time.Date(state.HolidayYear, time.January, 1, 0, 0, 0, 0, clock.ZoneWIB)
	to := time.Date(state.HolidayYear, time.December, 31, 0, 0, 0, 0, clock.ZoneWIB)

	rows, err := r.aneka.QueryContext(ctx, getQuery("report_holiday_calendar"), from, to)
	if err != nil {
		state.HolidayError = err
		return state
	}
	defer rows.Close()

	for rows.Next() {
		var day any
		if err := rows.Scan(&day); err != nil {
			state.HolidayError = err
			return state
		}
		state.HolidayDays++
	}
	if err := rows.Err(); err != nil {
		state.HolidayError = err
		return state
	}

	state.HolidayReadable = true
	return state
}

// NotPortedReport adalah satu pasangan laporan dan lini bisnis yang kuerinya belum ada.
type NotPortedReport struct {
	Report       reportklaim.Code
	Title        string
	BusinessLine reportklaim.BusinessLine
	Query        string
}

// NotPorted menyebutkan laporan yang kuerinya belum selesai dipindahkan dari export.
//
// Ia DIHITUNG dari rencana dan berkas .sql yang ada, bukan dari daftar yang ditulis
// tangan. Daftar tulisan tangan akan tetap menyebut nama yang sudah selesai dipindahkan,
// dan menyembunyikan kemajuan adalah kekeliruan yang sama buruknya dengan menyembunyikan
// kekurangan.
func NotPorted() []NotPortedReport {
	var result []NotPortedReport

	for _, report := range reportklaim.CatalogInPegaOrder() {
		p, known := plans[report.Code]
		if !known {
			continue
		}
		for _, line := range reportklaim.BusinessLineOptions() {
			name := p.query(reportklaim.Filter{BusinessLine: line.Value})
			if hasQuery(name) {
				continue
			}
			result = append(result, NotPortedReport{
				Report:       report.Code,
				Title:        report.Title,
				BusinessLine: line.Value,
				Query:        name,
			})
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Report != result[j].Report {
			return result[i].Report < result[j].Report
		}
		return result[i].BusinessLine < result[j].BusinessLine
	})
	return result
}
