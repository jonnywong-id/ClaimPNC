package reportklaim

import (
	"time"

	"claim-pnc/internal/platform/clock"
)

// HolidayCalendar adalah kumpulan hari libur perusahaan.
//
// # Kenapa ia ada di paket domain, bukan di adapter
//
// `D-50` menetapkan logika `DATAMINING.GET_WORKING_HOURS` **ditulis ulang di Go**, bukan
// dipanggil lewat API, karena perhitungan jam kerja dan kalender libur adalah **aturan
// bisnis** — bukan pengambilan data. Yang diambil dari luar hanyalah DAFTAR TANGGALNYA;
// aturan tentang apa yang dihitung sebagai hari kerja hidup di sini.
//
// Akibat langsung yang sudah diketahui: kalender libur menjadi master data milik aplikasi
// ini (`F-4`), dan dari mana daftar hari libur diperoleh setiap tahun belum ditetapkan.
// Sampai master itu ada, isinya dibaca dari `GENERAL.HRD_LBR` lewat koneksi kedua portal.
type HolidayCalendar struct {
	// day dikunci dengan tanggal WIB berbentuk "2006-01-02".
	//
	// Kuncinya teks, bukan time.Time, karena dua time.Time yang menunjuk hari yang sama
	// dapat berbeda sebagai nilai — zona, jam, dan monotonic clock membuatnya tidak
	// dapat dibandingkan dengan aman sebagai kunci peta.
	day map[string]struct{}

	// loaded menandai apakah kalendernya benar-benar terisi dari sumbernya.
	//
	// Ia DIBEDAKAN dari kalender kosong. Kalender yang tidak dapat dibaca dan kalender
	// yang memang tidak punya hari libur pada rentang itu menghasilkan angka yang
	// berbeda — dan tanpa pembedaan ini, kegagalan membaca sumbernya akan terbaca
	// sebagai "tahun ini tidak ada hari libur".
	loaded bool
}

// NewHolidayCalendar membentuk kalender dari daftar tanggal.
//
// Daftar kosong menghasilkan kalender yang TERISI tetapi tanpa hari libur — berbeda dari
// kalender yang tidak tersedia, lihat UnavailableCalendar.
func NewHolidayCalendar(days []time.Time) *HolidayCalendar {
	c := &HolidayCalendar{day: make(map[string]struct{}, len(days)), loaded: true}
	for _, d := range days {
		c.day[dayKey(d)] = struct{}{}
	}
	return c
}

// UnavailableCalendar adalah kalender yang sumbernya tidak dapat dibaca.
//
// Perhitungan hari kerja atasnya mengembalikan WorkingDaysUnknown, bukan angka yang
// kebetulan masuk akal.
func UnavailableCalendar() *HolidayCalendar { return &HolidayCalendar{} }

// Available menyatakan apakah kalender ini benar-benar terisi dari sumbernya.
func (c *HolidayCalendar) Available() bool { return c != nil && c.loaded }

// Count mengembalikan banyaknya hari libur yang tercatat.
func (c *HolidayCalendar) Count() int {
	if c == nil {
		return 0
	}
	return len(c.day)
}

// IsHoliday menyatakan apakah sebuah tanggal WIB adalah hari libur.
func (c *HolidayCalendar) IsHoliday(d time.Time) bool {
	if c == nil || !c.loaded {
		return false
	}
	_, ada := c.day[dayKey(d)]
	return ada
}

func dayKey(d time.Time) string {
	return clock.DateWIB(d).Format("2006-01-02")
}

// WorkingDaysUnknown adalah hasil perhitungan yang tidak dapat dilakukan.
//
// Nilainya -1 dan itu BUKAN pilihan sembarang: sistem lama menghasilkan `-1` pada keadaan
// yang sama, dan tiga dari empat kolom "Lama proses" memetakan `-1` menjadi `"0"` lewat
// `@If(Param.CountBusiness=="-1","0",Param.CountBusiness)`. Memakai angka yang sama
// membuat pemetaan itu dapat ditiru apa adanya.
const WorkingDaysUnknown = -1

// WorkingDaysBetween menghitung jumlah HARI KERJA dari satu tanggal ke tanggal lain.
//
// # Aturannya, dibaca dari GCNMTimeDifferenceWorkCalender_Act
//
// Sistem lama menghitungnya begini:
//
//	selisih  = jumlah hari kalender antara kedua tanggal
//	hasil    = selisih − (hari Sabtu/Minggu di dalam rentang)
//	                   − (hari libur di dalam rentang yang BUKAN Sabtu/Minggu)
//
// Pengecualian akhir pekan pada penghitungan hari libur ada di kuerinya sendiri
// (`RDB List/CheckHoliday_SQL-SQL.xml`): ia menyaring
// `TRIM(TO_CHAR(tanggal,'DAY')) NOT IN ('SABTU','MINGGU','SATURDAY','SUNDAY')`. Tanpa itu
// hari libur yang jatuh pada Sabtu akan dikurangkan DUA KALI.
//
// Di sini keduanya menjadi satu perulangan: sebuah hari dihitung bila ia bukan akhir
// pekan DAN bukan hari libur. Hasilnya sama, dan tidak ada yang dapat terkurang dua kali.
//
// # Rentangnya setengah terbuka
//
// Hari awal TIDAK dihitung, hari akhir dihitung — itulah arti "selisih" pada rumus
// aslinya. Klaim yang diregistrasi dan ditransfer pada hari yang sama menghasilkan 0,
// bukan 1.
//
// # Yang dikembalikan saat tidak dapat dihitung
//
// WorkingDaysUnknown, pada tiga keadaan: salah satu tanggal kosong, tanggal akhir
// mendahului tanggal awal, atau kalender liburnya tidak tersedia. Ketiganya dibedakan
// dari hasil 0 — nol berarti "tidak ada hari kerja di antaranya", bukan "tidak diketahui".
func WorkingDaysBetween(from, to time.Time, calendar *HolidayCalendar) int {
	if from.IsZero() || to.IsZero() {
		return WorkingDaysUnknown
	}
	if !calendar.Available() {
		return WorkingDaysUnknown
	}

	start := clock.DateWIB(from)
	end := clock.DateWIB(to)
	if end.Before(start) {
		return WorkingDaysUnknown
	}

	count := 0
	for day := start.AddDate(0, 0, 1); !day.After(end); day = day.AddDate(0, 0, 1) {
		switch day.Weekday() {
		case time.Saturday, time.Sunday:
			continue
		}
		if calendar.IsHoliday(day) {
			continue
		}
		count++
	}
	return count
}
