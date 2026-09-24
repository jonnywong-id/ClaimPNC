package komite

import (
	"time"

	"claim-pnc/internal/platform/money"
)

// ActorSystem adalah nama pelaku yang dicatat pada persetujuan yang tidak dilakukan
// manusia.
//
// `TKT-B07-003` mewajibkannya: persetujuan otomatis tetap harus dapat
// dipertanggungjawabkan, dan pelaku yang DIBIARKAN KOSONG membuat jejak audit tidak dapat
// membedakan "disetujui sistem" dari "tidak diketahui siapa".
//
// Ini sekaligus memperbaiki sistem lama, yang menyimpan jejaknya hanya sebagai teks
// catatan bebas — `"Auto Accept by PEGA Claim Non MBU"` di kolom KomiteComment — sehingga
// satu-satunya cara mengetahui persetujuan itu otomatis adalah mencocokkan kalimat.
const ActorSystem = "SISTEM"

// DefaultAutoAgeDays adalah lama menganggur sebelum sebuah jenjang komite memenuhi
// syarat disetujui otomatis.
//
// Angkanya berasal dari syarat di `Activity/AutoAcceptKomite-Act.xml`:
//
//	@DateTimeDifference(.Komite.DateOfComitee, @CurrentDateTime(), "D") > 2
//
// `> 2` pada ekspresi itu sempat ambigu — ia dapat berarti lewat 48 jam bila fungsinya
// mengembalikan pecahan, atau genap 72 jam bila ia mengembalikan bilangan bulat
// terpotong. **Work Owner menegaskan 2026-09-18: 72 jam.** Ambiguitasnya tertutup.
//
// Ia tetap parameter, bukan konstanta yang tertanam di aturannya, karena `D-15` melarang
// nilai bisnis di-hardcode — dan aturan yang menyetujui uang adalah yang paling tidak
// pantas tertanam di dalam kode.
const DefaultAutoAgeDays = 3

// AutoPolicy menetapkan kapan sebuah jenjang komite boleh disetujui tanpa manusia.
//
// Seluruh medannya dapat diubah tanpa menyentuh aturannya, karena `D-15` melarang nilai
// bisnis di-hardcode — dan aturan yang menyetujui uang adalah yang paling tidak pantas
// tertanam di dalam kode.
type AutoPolicy struct {
	// Enabled menyalakan persetujuan otomatis. Bawaannya MATI.
	//
	// Matinya disengaja. Job ini melewati seluruh kontrol otorisasi — `D-59` menetapkan
	// izin bersatuan menu, dan job tidak punya pengguna sehingga tidak ada menu yang
	// dapat diperiksa. Sesuatu yang menyetujui uang tanpa kontrol tidak boleh menyala
	// hanya karena kelalaian menyetelnya.
	Enabled bool

	// AgeDays adalah lama menganggur sebelum sebuah jenjang layak disetujui otomatis.
	// Nol berarti DefaultAutoAgeDays.
	AgeDays int
}

// Catatan atas dua batas yang SENGAJA tidak ada di sini.
//
// Sempat dibangun medan batas nilai dan batas jenjang, sebagai tempat bagi jawaban atas
// pertanyaan `TKT-B07-003` — "jenjang dan nilai mana yang boleh disetujui otomatis".
//
// Pertanyaan itu sudah dijawab Work Owner 2026-09-18, dan jawabannya: **tidak ada batas
// nilai maupun batas jenjang**. Syaratnya hanya dua — sudah menganggur cukup lama, dan
// jenjangnya belum habis (`CurrentTier <= TierCount`).
//
// Keduanya karena itu DIHAPUS, bukan dibiarkan mati. Konfigurasi yang tidak pernah dipakai
// adalah jalur yang tidak pernah diuji, dan ia menyiratkan kemampuan yang sebenarnya
// tidak diminta siapa pun. Bila kelak batas itu benar-benar dibutuhkan, menambahkannya
// kembali lebih murah daripada memelihara sesuatu yang belum tentu terpakai.

// EffectiveAge mengembalikan lama menganggur yang dipakai, dengan bawaan bila tidak diisi.
func (p AutoPolicy) EffectiveAge() time.Duration {
	days := p.AgeDays
	if days <= 0 {
		days = DefaultAutoAgeDays
	}
	return time.Duration(days) * 24 * time.Hour
}

// PendingCommittee adalah satu jenjang komite yang menunggu keputusan.
//
// Ia sengaja TIDAK memuat seluruh isi kasus komite: yang dibutuhkan aturan persetujuan
// otomatis hanyalah sejak kapan ia menganggur, jenjang keberapa, dan nilainya.
type PendingCommittee struct {
	ClaimNumber string

	// CurrentTier adalah jenjang keberapa yang sedang menunggu — `KomiteCount` di sistem
	// lama.
	CurrentTier int

	// TierCount adalah banyaknya jenjang yang harus menyetujui klaim ini —
	// `KomiteLoop` di sistem lama, yaitu jumlah baris master yang cocok.
	TierCount int

	OperatorID string
	Value      money.Money

	// WaitingSince adalah waktu jenjang ini mulai menunggu — `Komite.DateOfComitee` di
	// sistem lama.
	//
	// Perhatikan: sistem lama MENYETEL ULANG nilai ini setiap kali sebuah jenjang
	// disetujui otomatis, sehingga hitungan harinya dimulai lagi dari nol untuk jenjang
	// berikutnya. Akibatnya klaim yang seluruh jenjangnya disetujui otomatis membutuhkan
	// waktu sebanyak jumlah jenjangnya dikali ambang ini — bukan sekali saja.
	WaitingSince time.Time
}

// WithinTiers menyatakan klaim ini belum selesai melewati komitenya.
//
// Syaratnya diambil apa adanya dari `When/IsKomiteLoop-When.xml` — `KomiteCount <=
// KomiteLoop` — dan Work Owner menegaskan 2026-09-18 bahwa **inilah satu-satunya syarat
// selain lama menganggur** yang menentukan persetujuan otomatis berjalan.
//
// Ia diperiksa di sini, bukan diandaikan sudah disaring pemanggil. Sebuah aturan yang
// menyetujui uang tidak boleh bergantung pada asumsi bahwa masukannya sudah bersih.
func (c PendingCommittee) WithinTiers() bool {
	return c.TierCount > 0 && c.CurrentTier <= c.TierCount
}

// IneligibleReason menjelaskan kenapa sebuah jenjang tidak disetujui otomatis.
//
// Ia dikembalikan sebagai teks yang dapat langsung dibaca, karena keputusan untuk TIDAK
// menyetujui sama perlunya dijelaskan dengan keputusan untuk menyetujui — terutama pada
// proses yang berjalan tanpa siapa pun menyaksikannya.
type IneligibleReason string

const (
	ReasonDisabled  IneligibleReason = "persetujuan otomatis tidak diaktifkan"
	ReasonTiersDone IneligibleReason = "seluruh jenjang komite sudah dilewati"
	ReasonTooRecent IneligibleReason = "belum cukup lama menganggur"
)

// AutoAssessment adalah hasil pemeriksaan satu jenjang komite.
type AutoAssessment struct {
	Committee PendingCommittee

	Eligible bool
	Reason   IneligibleReason

	// Idle adalah lama jenjang ini sudah menunggu, dihitung sampai `now`.
	Idle time.Duration
}

// EvaluateAuto memeriksa apakah satu jenjang komite boleh disetujui tanpa manusia.
//
// # Dua syarat, dan hanya dua
//
// Work Owner menegaskan 2026-09-18 bahwa persetujuan otomatis berjalan selama
// `KomiteCount <= KomiteLoop` — yaitu selama jenjangnya belum habis — ditambah syarat lama
// menganggur pada rule aslinya. **Tidak ada batas nilai dan tidak ada batas jenjang.**
//
// Keduanya ditiru apa adanya. Yang DIPERBAIKI hanya satu hal: pelaku dicatat sebagai
// ActorSystem alih-alih dititipkan pada kalimat di kolom catatan.
func EvaluateAuto(c PendingCommittee, now time.Time, policy AutoPolicy) AutoAssessment {
	result := AutoAssessment{
		Committee: c,
		Idle:      now.Sub(c.WaitingSince),
	}

	switch {
	case !policy.Enabled:
		result.Reason = ReasonDisabled
	case !c.WithinTiers():
		result.Reason = ReasonTiersDone
	case result.Idle < policy.EffectiveAge():
		result.Reason = ReasonTooRecent
	default:
		result.Eligible = true
	}

	return result
}

// FilterAuto memeriksa banyak jenjang sekaligus dan mengembalikan seluruh
// penilaiannya — yang layak maupun yang tidak.
//
// Keduanya dikembalikan dengan sengaja. Yang tidak layak beserta alasannya adalah yang
// membuat job ini dapat diperiksa sebelum dinyalakan: seseorang dapat melihat apa yang
// AKAN disetujui, dan apa yang tidak, tanpa satu pun baris berubah.
func FilterAuto(
	pending []PendingCommittee,
	now time.Time,
	policy AutoPolicy,
) []AutoAssessment {
	result := make([]AutoAssessment, 0, len(pending))
	for _, c := range pending {
		result = append(result, EvaluateAuto(c, now, policy))
	}
	return result
}
