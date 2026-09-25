package reportkpi

import "time"

// Bentuk data mentah tab KPI PIC Teknik, beserta perhitungan kalender hari kerjanya.

// PICProfile adalah satu petugas pada satu lini bisnis.
type PICProfile struct {
	// OperatorID adalah `OPERATOR_ID` pada `MST_USER_TEKNIK`.
	OperatorID string

	// Leader menyatakan `STS_LEADER = 'TRUE'`.
	Leader bool
}

// PICQuery adalah penyaring yang dipakai keempat kelompok kueri tab ini.
type PICQuery struct {
	Line BusinessLine
	From time.Time
	To   time.Time
}

// ProgressCount adalah cacah pembaruan progres satu PIC.
type ProgressCount struct {
	PIC string

	// Total adalah seluruh pembaruan progres — kolom `NOAKSEP` kueri lama.
	Total float64

	// OnTime adalah yang TIDAK terlambat — kolom `STSKLAIM` kueri lama.
	//
	// Namanya di sana `STSKLAIM` dan tidak menyiratkan apa pun tentang isinya; yang
	// menjelaskannya adalah nama variabel penerimanya, `local.tdkterlambat`.
	OnTime float64
}

// DateSpan adalah sepasang tanggal yang selisih hari kerjanya dinilai.
type DateSpan struct {
	PIC   string
	Start time.Time
	End   time.Time
}

// AcceptanceSpan adalah satu klaim pada penilaian Akseptasi Klaim.
//
// Ia punya DUA pasang tanggal, dan mana yang dipakai ditentukan `Team` — bukan oleh
// ketersediaan tanggalnya. Lihat AcceptanceSpan.Span.
type AcceptanceSpan struct {
	PIC string

	// Team adalah kolom `LEADER_MEMBER` klaimnya: LEADER, MEMBER, atau FAC IN.
	Team string

	ReceiveLOD     time.Time
	CommitteeDate  time.Time
	AcceptanceDate time.Time
}

// Span memilih pasangan tanggal yang berlaku, meniru percabangan activity lama.
//
//	.CommentLOD == "LEADER"                      → ReceiveDateLOD  → AcceptedDate
//	.CommentLOD == "MEMBER" || == "FAC IN"       → AcceptedDateKomite → AcceptedDate
//
// Klaim yang `LEADER_MEMBER`-nya bukan salah satu dari ketiganya **tidak dinilai sama
// sekali** di sistem lama — tidak masuk pembilang maupun tetap di pembagi. Itu direplikasi
// dengan mengembalikan `ok=false`.
func (a AcceptanceSpan) Span() (start, end time.Time, ok bool) {
	switch a.Team {
	case TeamLeader:
		return a.ReceiveLOD, a.AcceptanceDate, true
	case TeamMember, "FAC IN":
		return a.CommitteeDate, a.AcceptanceDate, true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// ClosureSpan adalah satu klaim pada penilaian SLA Klaim.
type ClosureSpan struct {
	PIC string

	// Team adalah kolom `LEADER_MEMBER`, yang memilih pita dan ambang hari SLA.
	Team string

	RegisterDate time.Time
	CloseDate    time.Time
}

// TeamOf menormalkan `LEADER_MEMBER` menjadi LEADER atau MEMBER.
//
// Meniru `@If(.RW == "LEADER", "LEADER", "MEMBER")` — apa pun selain persis "LEADER"
// diperlakukan MEMBER, termasuk nilai kosong. Itu bukan penyederhanaan saya; itu yang
// tertulis di activity lama.
func TeamOf(value string) string {
	if value == TeamLeader {
		return TeamLeader
	}
	return TeamMember
}

// WorkingDaysBetween menghitung selisih HARI KERJA antara dua tanggal.
//
// # Kenapa dihitung di sini dan bukan di basis data
//
// `D-50` menetapkan perhitungan jam kerja dan kalender libur **ditulis ulang di Go**, karena
// ia aturan bisnis — bukan pengambilan data. Yang diambil dari basis data hanyalah daftar
// tanggal liburnya.
//
// # Cara sistem lama menghitungnya
//
// `GCNMTimeDifferenceWorkCalender_Act` mengurangi akhir pekan, lalu mengurangi hari libur
// yang dihitung `CheckHoliday_SQL`. Kueri itu sudah MENGECUALIKAN libur yang jatuh pada
// Sabtu dan Minggu:
//
//	AND TRIM(TO_CHAR(tanggal,'DAY')) NOT IN ('SABTU','MINGGU','SATURDAY','SUNDAY')
//
// sehingga libur yang jatuh di akhir pekan tidak terpotong dua kali. Daftar yang diterima
// fungsi ini karenanya sudah bersih dari akhir pekan, dan di sini ia dikurangkan apa adanya.
//
// Rentangnya SETENGAH TERBUKA pada ujung awal: hari mulai tidak dihitung, hari selesai
// dihitung — sehingga dua peristiwa pada hari yang sama berselisih nol hari, seperti di
// sistem lama.
func WorkingDaysBetween(start, end time.Time, holidays []time.Time) int {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return 0
	}

	from := dayOf(start)
	to := dayOf(end)

	days := 0
	for cursor := from.AddDate(0, 0, 1); !cursor.After(to); cursor = cursor.AddDate(0, 0, 1) {
		if cursor.Weekday() == time.Saturday || cursor.Weekday() == time.Sunday {
			continue
		}
		days++
	}

	for _, holiday := range holidays {
		if holiday.IsZero() {
			continue
		}
		day := dayOf(holiday)
		if day.After(from) && !day.After(to) {
			days--
		}
	}

	if days < 0 {
		return 0
	}
	return days
}

// dayOf membuang bagian jamnya, menyisakan tanggal saja.
//
// Sepadan dengan `TRUNC(tanggal)` pada kueri lama, dan ia diperlukan: tanpa itu, dua
// peristiwa pada hari yang sama tetapi berbeda jam akan terhitung sebagai selisih.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
