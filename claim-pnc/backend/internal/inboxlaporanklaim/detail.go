package inboxlaporanklaim

import (
	"fmt"
	"strings"
	"time"
)

// Detail adalah isian form **Input Receive Document**.
//
// # Asalnya
//
//	Flow/InputReceiveDocument.xml                       alur: Start -> assignment -> End
//	Flow Action/InputReceiveDocument-FlowAction.xml      flow action pada assignment-nya
//	Section/ViewInputReceiveDocument_sec-Section.xml     label dan properti terikatnya
//	Database/PROCINSERTDATARECIVEDKLAIM.prc              kolom yang benar-benar disimpan
//
// Section `InputReceiveDocument` yang dirender flow action itu **tidak ada di export**
// (bagian dari ±242 rule hilang, `R-16`). Yang dipakai sebagai gantinya adalah
// `ViewInputReceiveDocument_sec` — varian tampil dari form yang sama, yang memuat label
// dan properti terikat yang sama persis. Itu bukti terbaik yang tersedia, dan
// keterbatasannya dicatat di sini alih-alih ditutupi.
//
// # Tiga blok form yang TIDAK ada di sini, dan alasannya
//
//  1. **Blok data pelapor dan alamatnya** — No KTP, Nama Lengkap, Jabatan, Nama
//     Perusahaan, Pekerjaan, Negara, Provinsi, Kota, Kabupaten, Kelurahan, Kodepos.
//     Seluruhnya terikat `.ReportHE.*`, dan alamatnya bahkan page list
//     `.ReportHE.ASMAdressList`. `ReportHE` adalah area **Heavy Equipment**, yang `D-34`
//     keluarkan dari lingkup migrasi atas keputusan Work Owner.
//
//  2. **Grid dokumen** — Nama Dokumen, Jenis Dokumen, Jumlah Dokumen, Lihat Dokumen.
//     Terikat page list `.ReceiveDocument.DocumentList` dan menuntut penyimpanan dokumen
//     (`S-1`, `D-16`) yang belum ada. Yang dibawa hanya angka totalnya.
//
//  3. **Riwayat komunikasi dan progres** — `tempHistoryKomunikasi.pxResults` dan
//     `tempViewProgress.pxResults`. Keduanya menampilkan data milik modul lain, bukan
//     isian yang dikumpulkan form ini.
//
// # Pemeriksaan isian: penjaga penyimpanan, BUKAN aturan bisnis
//
// Flow action `InputReceiveDocument` **tidak punya satu pun** validate rule maupun isian
// bertanda wajib — diperiksa langsung ke berkasnya. Pega menyimpan apa pun yang diketik.
//
// Yang diperiksa di sini karena itu hanya dua hal yang melindungi penyimpanan, dan
// keduanya bukan aturan bisnis:
//
//   - **panjang teks** terhadap lebar kolom yang dibuat migrasi 0004. Tanpa ini,
//     Oracle menolaknya dengan `ORA-12899` yang tidak menyebut isian mana;
//   - **angka tidak boleh negatif** — estimasi kerugian dan jumlah dokumen.
//
// Aturan tanggal — DOL di dalam periode polis, Tanggal Lapor <= DOL + 7 hari, dan
// seterusnya — **sengaja TIDAK ada di sini**. Ia milik `B-2` Registrasi Klaim
// (`02-BUSINESS-UNDERSTANDING.md` §3.1), dan berkas laporan justru dapat masuk sebelum
// tanggalnya diketahui. Menambahkannya di sini akan menolak berkas yang di Pega diterima.
type Detail struct {
	ReceivedDate time.Time
	DateOfLoss   time.Time

	ReporterName  string
	ReporterEmail string
	ReporterPhone string
	CourierName   string

	PolicyNumber    string
	InsuredName     string
	BusinessName    string
	ReferenceNumber string

	EstimateValue Money
	LossLocation  string
	EmailSubject  string

	Chronology        string
	DamageDetail      string
	Reason            string
	NotRegisteredNote string

	DocumentCount int
}

// Lebar maksimum setiap isian teks, mengikuti kolom yang dibuat migrasi 0004.
//
// Angkanya diulang di frontend supaya pengguna tahu sebelum mengirim. Server tetap yang
// berwenang; pemeriksaan di layar hanya kenyamanan. Bila salah satu berubah, KEDUA tempat
// harus ikut berubah — utang yang disadari dari menduplikasi sebuah angka, dan uji di
// `detail_test.go` yang menjaganya tetap terlihat.
const (
	MaxNameLength      = 255
	MaxEmailLength     = 200
	MaxPhoneLength     = 64
	MaxPolicyLength    = 64
	MaxReferenceLength = 64
	MaxLocationLength  = 500
	MaxSubjectLength   = 1000
	MaxNoteLength      = 1000
	MaxNarrativeLength = 4000

	// MaxDocumentCount adalah batas atas yang masuk akal untuk satu berkas laporan.
	//
	// Ia bukan aturan bisnis: sistem lama tidak punya batas. Ia penjaga terhadap salah
	// ketik — angka enam digit pada "Total Jumlah Dokumen" hampir pasti bukan jumlah
	// dokumen, dan menerimanya membuat kolomnya tidak lagi berarti apa pun.
	MaxDocumentCount = 9999
)

// Clean memangkas spasi di kedua ujung setiap isian teks.
//
// Dipisahkan dari Check supaya nilai yang TERSIMPAN adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (d Detail) Clean() Detail {
	clean := d
	clean.ReporterName = strings.TrimSpace(d.ReporterName)
	clean.ReporterEmail = strings.TrimSpace(d.ReporterEmail)
	clean.ReporterPhone = strings.TrimSpace(d.ReporterPhone)
	clean.CourierName = strings.TrimSpace(d.CourierName)
	clean.PolicyNumber = strings.TrimSpace(d.PolicyNumber)
	clean.InsuredName = strings.TrimSpace(d.InsuredName)
	clean.BusinessName = strings.TrimSpace(d.BusinessName)
	clean.ReferenceNumber = strings.TrimSpace(d.ReferenceNumber)
	clean.LossLocation = strings.TrimSpace(d.LossLocation)
	clean.EmailSubject = strings.TrimSpace(d.EmailSubject)
	clean.Chronology = strings.TrimSpace(d.Chronology)
	clean.DamageDetail = strings.TrimSpace(d.DamageDetail)
	clean.Reason = strings.TrimSpace(d.Reason)
	clean.NotRegisteredNote = strings.TrimSpace(d.NotRegisteredNote)
	return clean
}

// Check menjalankan seluruh pemeriksaan dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Detail sudah harus melewati Clean lebih dulu.
//
// Seluruh pelanggaran dikumpulkan, bukan berhenti pada yang pertama: form ini memuat
// tujuh belas isian, dan mengembalikan satu galat per percobaan akan membuat pengguna
// menebak isian mana lagi yang salah (`P-5`, `11-CROSSCUTTING.md` §1.2 aturan 1).
func (d Detail) Check() error {
	var violation []Violation

	limit := []struct {
		field string
		label string
		value string
		max   int
	}{
		{"nama_pelapor", "Nama Pengirim / Pelapor Dokumen", d.ReporterName, MaxNameLength},
		{"email_pelapor", "Email Pengirim", d.ReporterEmail, MaxEmailLength},
		{"telepon_pelapor", "No. HP Pengirim", d.ReporterPhone, MaxPhoneLength},
		{"nama_kurir", "Nama Kurir ASM", d.CourierName, MaxNameLength},
		{"nomor_polis", "Nomor Polis", d.PolicyNumber, MaxPolicyLength},
		{"tertanggung", "Nama Tertanggung", d.InsuredName, MaxNameLength},
		{"nama_bisnis", "Nama Bisnis", d.BusinessName, MaxNameLength},
		{"nomor_rujukan", "No. Referensi/Placing Slip", d.ReferenceNumber, MaxReferenceLength},
		{"lokasi_kejadian", "Lokasi Kejadian", d.LossLocation, MaxLocationLength},
		{"subjek_email", "Subject Email", d.EmailSubject, MaxSubjectLength},
		{"kronologis", "Kronologis Kejadian", d.Chronology, MaxNarrativeLength},
		{"rincian_kerusakan", "Rincian Kerusakan", d.DamageDetail, MaxNarrativeLength},
		{"alasan", "Keterangan Belum Transfer", d.Reason, MaxNoteLength},
		{"keterangan_belum_registrasi", "Keterangan Belum Registrasi", d.NotRegisteredNote, MaxNoteLength},
	}

	for _, l := range limit {
		// Dihitung dalam RUNE, bukan byte: satu huruf beraksen memakan dua byte, dan
		// menghitungnya sebagai dua karakter akan menolak isian yang sebenarnya pendek.
		if len([]rune(l.value)) > l.max {
			violation = append(violation, Violation{
				Field:   l.field,
				Message: fmt.Sprintf("%s paling panjang %d karakter.", l.label, l.max),
			})
		}
	}

	if d.EstimateValue < 0 {
		violation = append(violation, Violation{
			Field:   "estimasi_kerugian",
			Message: "Estimasi kerugian tidak boleh negatif.",
		})
	}

	switch {
	case d.DocumentCount < 0:
		violation = append(violation, Violation{
			Field:   "jumlah_dokumen",
			Message: "Total jumlah dokumen tidak boleh negatif.",
		})
	case d.DocumentCount > MaxDocumentCount:
		violation = append(violation, Violation{
			Field:   "jumlah_dokumen",
			Message: fmt.Sprintf("Total jumlah dokumen paling banyak %d.", MaxDocumentCount),
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// DetailOf membaca isian form dari sebuah berkas yang sudah tersimpan.
//
// Dipakai saat form dibuka: yang digambar adalah isi berkasnya, bukan form kosong.
func DetailOf(r ClaimReport) Detail {
	return Detail{
		ReceivedDate:      r.ReceivedDate,
		DateOfLoss:        r.DateOfLoss,
		ReporterName:      r.ReporterName,
		ReporterEmail:     r.ReporterEmail,
		ReporterPhone:     r.ReporterPhone,
		CourierName:       r.CourierName,
		PolicyNumber:      r.PolicyNumber,
		InsuredName:       r.InsuredName,
		BusinessName:      r.BusinessName,
		ReferenceNumber:   r.ReferenceNumber,
		EstimateValue:     r.EstimateValue,
		LossLocation:      r.LossLocation,
		EmailSubject:      r.EmailSubject,
		Chronology:        r.Chronology,
		DamageDetail:      r.DamageDetail,
		Reason:            r.Reason,
		NotRegisteredNote: r.NotRegisteredNote,
		DocumentCount:     r.DocumentCount,
	}
}

// Apply menuliskan isian form ke atas berkas yang sudah ada.
//
// Yang TIDAK pernah tersentuh: nomor berkas, nomor klaim, cabang, asal, penanda
// diserahkan, dan seluruh jejak pembuatannya. Form ini mengisi berkas, bukan
// memindahkannya — perpindahan tahap adalah tindakan tersendiri.
func (d Detail) Apply(r ClaimReport) ClaimReport {
	result := r
	result.ReceivedDate = d.ReceivedDate
	result.DateOfLoss = d.DateOfLoss
	result.ReporterName = d.ReporterName
	result.ReporterEmail = d.ReporterEmail
	result.ReporterPhone = d.ReporterPhone
	result.CourierName = d.CourierName
	result.PolicyNumber = d.PolicyNumber
	result.InsuredName = d.InsuredName
	result.BusinessName = d.BusinessName
	result.ReferenceNumber = d.ReferenceNumber
	result.EstimateValue = d.EstimateValue
	result.LossLocation = d.LossLocation
	result.EmailSubject = d.EmailSubject
	result.Chronology = d.Chronology
	result.DamageDetail = d.DamageDetail
	result.Reason = d.Reason
	result.NotRegisteredNote = d.NotRegisteredNote
	result.DocumentCount = d.DocumentCount
	return result
}
