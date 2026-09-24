package inboxrclpucl

import (
	"strings"
	"time"
)

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
//
// # Kenapa tidak ada satu pun penyaring di sini
//
// Karena grid layar lama tidak punya satu pun. Ketiga Report Definition-nya menyaring
// dengan nilai yang TETAP — antrean bersama, status kerja, tanggal cetak, penanda
// persetujuan, penanda MSIG — dan tidak satu pun dapat diubah pengguna.
//
// Kedua isian tanggal di tab "Cetak Surat" TIDAK termasuk di sini, dan itu bukan kelalaian:
// keduanya tidak menyaring grid sama sekali. Yang memakainya adalah laporan harian di balik
// tombol ekspor, dan ia punya permintaannya sendiri — lihat ReportRequest. Menaruhnya di
// sini akan membuat orang mengira grid-nya ikut tersaring.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan penyaringnya.
	Tab Tab

	// Caller adalah identitas pemanggil.
	//
	// Tidak satu pun kueri menyaring menurut nilai ini — antreannya bersama. Ia dibawa
	// untuk jejak log; lihat Caller.
	Caller Caller
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Tab tidak dikenal.",
		}})
	}

	// Tab terhalang DITOLAK di sini, bukan dibiarkan sampai ke penyimpanan.
	//
	// Tidak ada tab terhalang di layar ini. Pemeriksaannya tetap ada: begitu sebuah tab
	// ditandai terhalang kelak, permintaannya akan sampai ke repo yang tidak punya kueri
	// untuknya dan gagal sebagai galat internal 500 — jawaban yang tidak menyebut sebabnya
	// kepada siapa pun. Ditolak di sini, alasannya sampai ke layar apa adanya.
	if tab.Blocked {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: tab.BlockedReason,
		}})
	}

	return Query{Tab: tab, Caller: cleanCaller}, nil
}

// ReportInput adalah isian mentah permintaan laporan harian.
type ReportInput struct {
	// Tab adalah kode tab yang meminta laporan. Kosong berarti DefaultTab.
	//
	// Ia ikut diminta meski hanya satu tab yang punya laporan, supaya penolakannya dapat
	// menyebut tab mana yang salah — bukan sekadar "tidak tersedia".
	Tab string

	// From dan To adalah batas rentang tanggal, berbentuk `YYYY-MM-DD`.
	From string
	To   string
}

// ReportRequest adalah permintaan laporan harian yang sudah tervalidasi.
type ReportRequest struct {
	// Tab adalah tab yang meminta laporan. Ia selalu tab yang HasDateRangeReport.
	Tab Tab

	// Range adalah rentang tanggal yang sudah dipastikan lengkap dan berurutan.
	Range DateRange

	// Caller adalah identitas pemanggil, dipakai jejak log.
	Caller Caller
}

// reportDateLayout adalah bentuk tanggal yang diterima kontrak API.
//
// Bentuk ISO, bukan `dd/mm/yyyy` seperti di layar lama. Alasannya: yang mengirimnya adalah
// kontrak API, bukan layar Pega, dan `dd/mm/yyyy` tidak dapat dibedakan dari `mm/dd/yyyy`
// oleh pembacanya — satu kekeliruan yang menghasilkan rentang yang sah tetapi salah, tanpa
// satu pun galat. Penerjemahannya ke bentuk yang dimengerti basis data dikerjakan
// penyimpanan.
const reportDateLayout = "2006-01-02"

// NewReportRequest membentuk permintaan laporan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa kedua tanggalnya WAJIB
//
// Karena kueri lama memakai keduanya sebagai `>=` dan `<=` tanpa penjaga apa pun
// (`RDB List/GetDataPUCLRCLForDailyReport-SQL.xml`). Di Pega, mengosongkan salah satunya
// menyisipkan teks kosong ke dalam `to_date(…, 'dd/mm/yyyy')` dan menghasilkan galat basis
// data yang sampai ke pengguna sebagai kegagalan mentah.
//
// Di sini keduanya diperiksa lebih dulu, dan yang sampai ke pengguna adalah kalimat yang
// menyebut isian mana yang kurang. Itu selisih yang MEMPERBAIKI cara galat disampaikan,
// bukan mengubah hasil — rentang yang sah menghasilkan baris yang sama.
//
// # Kenapa SELURUH pelanggaran dikumpulkan
//
// Mengikuti `P-5` dan `11-CROSSCUTTING.md` §1.1: pengguna yang mengosongkan kedua tanggal
// diberi tahu keduanya sekaligus, bukan satu lalu satu lagi.
func NewReportRequest(input ReportInput, caller Caller) (ReportRequest, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return ReportRequest{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		return ReportRequest{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Tab tidak dikenal.",
		}})
	}
	if !tab.HasDateRangeReport {
		return ReportRequest{}, ErrReportNotAvailable
	}

	from := strings.TrimSpace(input.From)
	to := strings.TrimSpace(input.To)

	violations := []Violation{}

	fromDate, fromValid := parseReportDate(from)
	if !fromValid {
		violations = append(violations, Violation{
			Field:   FieldDateFrom,
			Message: messageForDate(from, "FROM RCL/PUCL"),
		})
	}

	toDate, toValid := parseReportDate(to)
	if !toValid {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: messageForDate(to, "TO RCL/PUCL"),
		})
	}

	// Urutan diperiksa hanya bila KEDUANYA terbaca. Memeriksanya lebih awal akan
	// menghasilkan pelanggaran ketiga yang membingungkan — pengguna diberi tahu urutannya
	// salah padahal yang salah adalah bentuk tanggalnya.
	if fromValid && toValid && fromDate.After(toDate) {
		violations = append(violations, Violation{
			Field: FieldDateTo,
			Message: "Tanggal \"TO RCL/PUCL\" tidak boleh lebih awal daripada " +
				"\"FROM RCL/PUCL\".",
		})
	}

	if len(violations) > 0 {
		return ReportRequest{}, NewValidationError(violations)
	}

	return ReportRequest{
		Tab:    tab,
		Range:  DateRange{From: from, To: to},
		Caller: cleanCaller,
	}, nil
}

// parseReportDate membaca satu batas tanggal.
//
// Ia memakai `time.Parse` yang MENOLAK tanggal yang tidak ada — `2026-02-30` gagal di sini,
// sementara pemeriksaan berbasis pola akan meloloskannya lalu menyerahkannya ke basis data.
func parseReportDate(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(reportDateLayout, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// messageForDate menyusun pesan yang membedakan "belum diisi" dari "salah bentuk".
//
// Keduanya dipisah karena tindakan pengguna berbeda: yang pertama menuntut ia mengisi, yang
// kedua menuntut ia memperbaiki. Pesan yang sama untuk keduanya membuat pengguna yang sudah
// mengisi mengira isiannya tidak terkirim.
func messageForDate(raw, label string) string {
	if raw == "" {
		return "Tanggal \"" + label + "\" wajib diisi untuk mengunduh laporan."
	}
	return "Tanggal \"" + label + "\" tidak terbaca. Gunakan bentuk tahun-bulan-tanggal, " +
		"misalnya 2026-09-23."
}
