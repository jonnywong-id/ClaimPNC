package memory

import (
	"time"

	"claim-pnc/internal/inboxsurvey"
)

// Login contoh.
//
// # Kenapa BUKAN nama sungguhan
//
// `D-69` mengizinkan nama Operator ID ditulis di DOKUMEN supaya tiket dapat menunjuk hardcode
// mana yang dibuang. Izin itu untuk dokumen, bukan untuk data yang dijalankan — menuliskan
// login sungguhan sebagai data contoh akan memindahkan hardcode ke sistem baru lewat pintu
// belakang, tepat hal yang sedang dihapus (`D-15`).
const (
	// SampleLeaderLogin adalah adjuster yang MEMBAWAHI orang lain.
	SampleLeaderLogin = "ADJLEADER"

	// SampleMemberLogin adalah anggota di bawah leader di atas.
	SampleMemberLogin = "ADJMEMBER"

	// SampleInternalLogin adalah surveyor INTERNAL — `SURVEYTYPE = "1"`.
	//
	// Ia ada supaya keputusan Work Owner 2026-09-28 dapat dibuktikan: layar ini melayani
	// DUA populasi, dan yang membedakan keduanya adalah identitas yang masuk — bukan
	// penyaring yang dipilih pengguna.
	SampleInternalLogin = "SURVINTERN"

	// SampleOutsiderLogin adalah pengguna yang TIDAK terdaftar sebagai surveyor sama sekali.
	//
	// Tanpa baris seperti ini, ErrNotSurveyor tidak dapat dibuktikan berbeda dari antrean
	// kosong — dan itu persis perbedaan yang paling mudah hilang.
	SampleOutsiderLogin = "BUKANSURVEYOR"

	// SampleOtherSender adalah pengirim pesan dari PIHAK LAIN — petugas teknis ASM.
	//
	// Ia yang membedakan tab "belum dijawab" dari "belum dibalas ASM": keduanya berstatus
	// pesan sama, dan HANYA arah pengirimnya yang berbeda.
	SampleOtherSender = "PICTEKNIK1"
)

// Nama surveyor contoh.
//
// `SampleNearMissName` sengaja dibuat MEMUAT `SampleLeaderName` sebagai awalan. Ia yang
// membuktikan pembatas `|` pada cakupan benar-benar bekerja: tanpa pembatas, pemegang nama
// pendek akan melihat pekerjaan pemegang nama panjang.
const (
	SampleLeaderName   = "BUDI"
	SampleMemberName   = "SITI RAHAYU"
	SampleInternalName = "AGUS INTERNAL"
	SampleNearMissName = "BUDIONO SETIAWAN"
)

// at membentuk waktu UTC, supaya contoh terbaca dan urutannya mudah diperiksa dengan mata.
func at(year int, month time.Month, date, hour int) time.Time {
	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC)
}

// SampleSurveyors adalah isi `POOLDATA.MST_LOGIN_SURVEYOR` contoh.
//
// Perhatikan `SampleNearMissName`: ia terdaftar sebagai surveyor, tetapi BUKAN anggota
// leader mana pun. Itu yang membuat kebocoran cakupan — bila pembatas `|` hilang — benar-benar
// terlihat sebagai baris yang muncul di layar orang lain.
var SampleSurveyors = []SurveyorRecord{
	{Login: SampleLeaderLogin, Name: SampleLeaderName, LeaderLogin: ""},
	{Login: SampleMemberLogin, Name: SampleMemberName, LeaderLogin: SampleLeaderLogin},
	{Login: SampleInternalLogin, Name: SampleInternalName, LeaderLogin: ""},
	{Login: "ADJLAIN", Name: SampleNearMissName, LeaderLogin: ""},
}

// SampleRecords adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya.
//
// # Kenapa hampir seluruh baris punya pesan
//
// Karena HANYA ketiga tab komunikasi yang dapat dihitung hari ini. Baris tanpa pesan tidak
// muncul di tab mana pun, sehingga contoh tanpa pesan tidak dapat membuktikan apa pun tentang
// cakupan, urutan, maupun pencarian.
//
// Satu baris SENGAJA dibiarkan tanpa pesan (`SRV-0006`) — ia yang membuktikan penyaring tab
// benar-benar menyaring, bukan meloloskan semuanya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": setiap penyaring memperoleh baris yang membuatnya dapat
// dibuktikan bekerja.
//
//   - Satu baris milik `SampleNearMissName`, yang namanya MEMUAT nama leader sebagai awalan.
//     Bila pembatas cakupan hilang, baris itu bocor ke antrean leader.
//   - Satu baris milik anggota. Bila hierarki leader hilang, leader tidak melihatnya.
//   - Satu baris milik surveyor INTERNAL, `SURVEYTYPE = "1"`.
//   - Tiga baris berkomunikasi: dari pihak lain, dari diri sendiri belum dibalas, dan sudah
//     dibalas — ketiganya menempati tab yang berbeda.
//   - Satu baris ber-`CreatedAt` KOSONG, supaya "umur belum dapat dihitung" dapat dibedakan
//     dari nol hari. Ia ditaruh pada baris yang BENAR-BENAR tampil di sebuah tab; baris yang
//     tidak tampil di tab mana pun tidak pernah sampai ke layar dan tidak pernah teruji.
//   - Dua baris berwaktu input SAMA PERSIS, sehingga pemutus seri dapat diperiksa.
var SampleRecords = []Record{
	{
		// `CreatedAt` sengaja dibiarkan kosong — umurnya tidak dapat dihitung, dan itu BERBEDA
		// dari nol hari. Ia tetap punya pesan supaya benar-benar tampil di sebuah tab.
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleOtherSender},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0001",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0101",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0101",
			PolicyNumber:    "26.004.2026.00101",
			InsuredName:     "Rangga Contoh",
			ClassOfBusiness: "Marine Cargo",
			CauseOfLoss:     "Kerusakan Muatan",
			Location:        "Pelabuhan Tanjung Priok",
			TechnicalPIC:    SampleOtherSender,
			AdjusterPIC:     SampleLeaderName,
			DateOfLoss:      at(2026, time.August, 20, 0),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Waiting Claim Document",
		},
	},
	{
		// Milik ANGGOTA. Leader harus melihatnya; anggota lain tidak.
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleOtherSender},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0002",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0102",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0102",
			PolicyNumber:    "26.006.2026.00102",
			InsuredName:     "Melati Contoh",
			ClassOfBusiness: "Fire",
			CauseOfLoss:     "Kebakaran",
			Location:        "Bekasi",
			TechnicalPIC:    "PICTEKNIK2",
			AdjusterPIC:     SampleMemberName,
			DateOfLoss:      at(2026, time.August, 25, 0),
			CreatedAt:       at(2026, time.September, 2, 9),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Survey",
		},
	},
	{
		// Milik orang yang namanya MEMUAT nama leader sebagai awalan.
		//
		// Bila pembatas cakupan hilang, baris ini muncul di antrean leader — dan tampak
		// wajar, karena seluruh kolomnya terisi.
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleOtherSender},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0003",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0103",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0103",
			PolicyNumber:    "26.003.2026.00103",
			InsuredName:     "Dimas Contoh",
			ClassOfBusiness: "Aneka",
			CauseOfLoss:     "Kehilangan",
			Location:        "Surabaya",
			TechnicalPIC:    "PICTEKNIK3",
			AdjusterPIC:     SampleNearMissName,
			DateOfLoss:      at(2026, time.August, 26, 0),
			CreatedAt:       at(2026, time.September, 3, 10),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Initial Advice",
		},
	},
	{
		// Pesan dari DIRI SENDIRI, belum dibalas — tab "Not replied from ASM".
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleLeaderLogin},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0004",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0104",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0104",
			PolicyNumber:    "26.002.2026.00104",
			InsuredName:     "Rahmat Contoh",
			ClassOfBusiness: "Personal Accident",
			CauseOfLoss:     "Kecelakaan",
			Location:        "Bandung",
			TechnicalPIC:    SampleOtherSender,
			AdjusterPIC:     SampleLeaderName,
			DateOfLoss:      at(2026, time.July, 30, 0),
			CreatedAt:       at(2026, time.September, 4, 11),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Preliminary Advice",
		},
	},
	{
		// Pesan dari diri sendiri yang SUDAH dibalas — tab "Replied from ASM".
		Messages: []Message{
			{Status: inboxsurvey.CommunicationAnswered, Sender: SampleLeaderLogin},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0005",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0105",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0105",
			PolicyNumber:    "26.005.2026.00105",
			InsuredName:     "Ayu Contoh",
			ClassOfBusiness: "Travel",
			CauseOfLoss:     "Pembatalan Perjalanan",
			Location:        "Denpasar",
			TechnicalPIC:    "PICTEKNIK2",
			AdjusterPIC:     SampleLeaderName,
			DateOfLoss:      at(2026, time.July, 10, 0),
			CreatedAt:       at(2026, time.September, 5, 12),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Interim Report",
		},
	},
	{
		// TANPA pesan sama sekali, dan itu bukan kelalaian.
		//
		// Ketiga tab yang dapat dihitung seluruhnya berbasis komunikasi, sehingga baris ini
		// tidak muncul di tab mana pun. Ia yang membuktikan penyaring tab benar-benar
		// menyaring — tanpa baris seperti ini, penyaring yang meloloskan segalanya akan lulus
		// setiap uji di sini.
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0006",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0106",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0106",
			PolicyNumber:    "26.003.2026.00106",
			InsuredName:     "Hendra Contoh",
			ClassOfBusiness: "Aneka",
			CauseOfLoss:     "Kerusakan Mesin",
			Location:        "Semarang",
			TechnicalPIC:    "PICTEKNIK3",
			AdjusterPIC:     SampleLeaderName,
			DateOfLoss:      at(2026, time.August, 28, 0),
			CreatedAt:       at(2026, time.September, 6, 13),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Draft Final Report",
		},
	},
	{
		// Berwaktu input SAMA PERSIS dengan baris berikutnya, supaya pemutus seri teruji.
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleOtherSender},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0007",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0107",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0107",
			PolicyNumber:    "26.004.2026.00107",
			InsuredName:     "Nadia Contoh",
			ClassOfBusiness: "Marine Cargo",
			CauseOfLoss:     "Kerusakan Muatan",
			Location:        "Makassar",
			TechnicalPIC:    SampleOtherSender,
			AdjusterPIC:     SampleLeaderName,
			DateOfLoss:      at(2026, time.August, 18, 0),
			CreatedAt:       at(2026, time.September, 8, 15),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Final Report",
		},
	},
	{
		// Waktu input sama dengan baris sebelumnya; urutannya ditentukan pemutus seri.
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleOtherSender},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0008",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0108",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0108",
			PolicyNumber:    "26.006.2026.00108",
			InsuredName:     "Fajar Contoh",
			ClassOfBusiness: "Fire",
			CauseOfLoss:     "Kebakaran",
			Location:        "Medan",
			TechnicalPIC:    "PICTEKNIK2",
			AdjusterPIC:     SampleLeaderName,
			DateOfLoss:      at(2026, time.August, 15, 0),
			CreatedAt:       at(2026, time.September, 8, 15),
			SurveyorType:    inboxsurvey.SurveyorTypeLossAdjuster,
			ASMStatus:       "Invoice Fee",
		},
	},
	{
		// Milik surveyor INTERNAL — `SURVEYTYPE = "1"`.
		Messages: []Message{
			{Status: inboxsurvey.CommunicationOpen, Sender: SampleOtherSender},
		},
		Task: inboxsurvey.SurveyTask{
			SurveyID:        "ASM-FW-GCNMFW-WORK SRV-0010",
			ClaimID:         "ASM-FW-GCNMFW-WORK PNCN.26.0110",
			SurveyIndex:     "1",
			ClaimNumber:     "PNCN.26.0110",
			PolicyNumber:    "26.002.2026.00110",
			InsuredName:     "Intan Contoh",
			ClassOfBusiness: "Personal Accident",
			CauseOfLoss:     "Kecelakaan",
			Location:        "Yogyakarta",
			TechnicalPIC:    SampleOtherSender,
			AdjusterPIC:     SampleInternalName,
			DateOfLoss:      at(2026, time.September, 1, 0),
			CreatedAt:       at(2026, time.September, 9, 16),
			SurveyorType:    inboxsurvey.SurveyorTypeInternal,
			ASMStatus:       "Waiting Confirmation",
		},
	},
}

// SampleKPI adalah isi `POOLDATA.DETAIL_KPI_ADJUSTER` contoh.
//
// Dua baris per adjuster pada tahun yang sama, supaya RATA-RATA benar-benar dihitung — satu
// baris saja tidak dapat membedakan rata-rata dari penjumlahan.
var SampleKPI = []KPIRecord{
	{
		Adjuster: SampleLeaderName,
		Category: inboxsurvey.KPITypeFinal,
		Year:     "2026",
		Row: inboxsurvey.KPIRow{
			SurveyScheduling: 90, ImmediateAdvice: 80, PreliminaryAdvice: 85,
			InterimReport: 75, ProgressUpdate: 70, CommunicationResponse: 95,
			ProposeAdjustment: 88, FinalReport: 92, Value: 86,
		},
	},
	{
		Adjuster: SampleLeaderName,
		Category: inboxsurvey.KPITypeFinal,
		Year:     "2026",
		Row: inboxsurvey.KPIRow{
			SurveyScheduling: 80, ImmediateAdvice: 70, PreliminaryAdvice: 75,
			InterimReport: 65, ProgressUpdate: 60, CommunicationResponse: 85,
			ProposeAdjustment: 78, FinalReport: 82, Value: 76,
		},
	},
	{
		Adjuster: SampleMemberName,
		Category: inboxsurvey.KPITypeFinal,
		Year:     "2025",
		Row: inboxsurvey.KPIRow{
			SurveyScheduling: 70, ImmediateAdvice: 65, PreliminaryAdvice: 68,
			InterimReport: 60, ProgressUpdate: 55, CommunicationResponse: 72,
			ProposeAdjustment: 66, FinalReport: 74, Value: 69,
		},
	},
	{
		// Kategori BUKAN "FINAL". Ia harus TERTOLAK oleh ringkasan Final dan Kuartal, dan
		// hanya muncul pada ringkasan Outstanding tanpa penyaring kategori.
		Adjuster: SampleLeaderName,
		Category: "OUTSTANDING",
		Year:     "2026",
		Row: inboxsurvey.KPIRow{
			SurveyScheduling: 10, ImmediateAdvice: 10, PreliminaryAdvice: 10,
			InterimReport: 10, ProgressUpdate: 10, CommunicationResponse: 10,
			ProposeAdjustment: 10, FinalReport: 10, Value: 10,
		},
	},
	{
		// Milik adjuster DI LUAR cakupan. Bila penyaring cakupan hilang pada KPI, baris ini
		// ikut terhitung — dan papan penilaian menjadi bocor.
		Adjuster: SampleNearMissName,
		Category: inboxsurvey.KPITypeFinal,
		Year:     "2026",
		Row: inboxsurvey.KPIRow{
			SurveyScheduling: 100, ImmediateAdvice: 100, PreliminaryAdvice: 100,
			InterimReport: 100, ProgressUpdate: 100, CommunicationResponse: 100,
			ProposeAdjustment: 100, FinalReport: 100, Value: 100,
		},
	},
}

// NewSampleStore membentuk pembaca berisi seluruh data contoh.
func NewSampleStore() *Store {
	return NewStore(SampleRecords, SampleSurveyors, SampleKPI)
}
