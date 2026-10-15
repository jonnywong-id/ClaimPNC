package inboxcompliance

import (
	"strconv"
	"strings"
	"time"
)

// Pengiriman klaim dari antrean Compliance ke Post Audit.
//
// # Alur mana yang ditiru, dan apa yang TIDAK dapat dibaca
//
// Keputusan Work Owner 2026-09-24: **mengikuti aplikasi Pega**. Di Pega, baris Post Audit
// lahir dari sisi Compliance — petugas memeriksa klaim yang menunggu di workbasket
// `CompliancePNC`, lalu meneruskannya. Itulah bentuk yang dibangun di sini.
//
// Yang perlu dinyatakan terbuka: **`Compliance_Flow` TIDAK ADA di export.** Ia dirujuk
// `Activity/SetAssignmentCompliance-Act.xml:423` sebagai `"!Compliance_Flow"`, tetapi
// direktori `Flow/` hanya memuat empat flow dan tidak satu pun di antaranya. Jadi langkah
// persis, prakondisi, dan akibat sampingannya tidak dapat dibaca — ia bagian dari `R-16`.
//
// Yang DAPAT dibaca, dan menjadi dasar bentuk di bawah:
//
//   - Kolom `POOLDATA.T_CLAIM_COMPLIANCE_H` cocok satu per satu dengan data klaim yang
//     sudah tampil di tab Compliance, ditambah satu catatan dan satu tanggal kirim.
//   - Layar Pega menampilkan catatan berbunyi "to compilance" — kalimat bebas yang diketik
//     petugas, bukan pilihan dari daftar.
//   - Tanggalnya `22/04/25 13:46`, yakni waktu pengiriman, bukan tanggal yang dipilih.
//
// # Yang sengaja TIDAK ditambahkan
//
// Tidak ada aturan yang dikarang di sini. Tidak ada penyaring kelayakan selain "klaimnya
// memang sedang menunggu di antrean Compliance", tidak ada status yang diubah pada klaimnya,
// dan tidak ada pemberitahuan yang dikirim. Ketiganya mungkin ada di `Compliance_Flow`, dan
// menebaknya berarti mengarang perilaku yang akan terlihat sebagai selisih pada gerbang 1.

// PostAuditInput adalah isian mentah dari layar, belum divalidasi.
type PostAuditInput struct {
	// Reference adalah kunci klaim yang dikirim — `PZINSKEY`, nilai yang sama dengan
	// kolom tersembunyi pada baris tab Compliance.
	//
	// Bukan nomor klaim yang dibaca orang: nomor itu tidak unik lintas sistem selama masa
	// paralel, sedangkan kunci teknis unik.
	Reference string

	// Remarks adalah Catatan yang diketik petugas.
	Remarks string
}

// PostAuditEntry adalah satu baris yang akan ditulis ke `POOLDATA.T_CLAIM_COMPLIANCE_H`.
//
// Keenam isiannya persis keenam kolom tabel itu.
type PostAuditEntry struct {
	// CaseID diisi PENYIMPANAN dari sequence, bukan oleh pemanggil.
	//
	// Ia kosong saat entry disusun dan terisi saat baris tersimpan. Membiarkan pemanggil
	// mengisinya akan membuka jalan bagi nomor yang dikarang layar.
	CaseID string

	// ClaimNumber adalah kolom `NO_KLAIM`, yang isinya kunci teknis Pega — bukan nomor
	// klaim, meski namanya begitu. Lihat WorkItem.ClaimNumber.
	ClaimNumber string

	// InsuredName dan PolicyNumber disalin dari klaimnya, tidak diketik ulang.
	//
	// Itu keputusan, bukan kemudahan: mengetiknya ulang membuka kemungkinan baris Post
	// Audit menyebut nama tertanggung yang berbeda dari klaim yang dirujuknya, dan tabel
	// ini tidak punya foreign key yang akan menolaknya.
	InsuredName  string
	PolicyNumber string

	// Remarks adalah kolom `REMARKS`.
	Remarks string

	// SentAt adalah kolom `TGL_KIRIM_POST_AUDIT`, diisi jam server lewat seam Clock.
	SentAt time.Time
}

// RemarksMaxLength mengikuti lebar kolom `REMARKS VARCHAR2(4000 BYTE)`.
//
// # Kenapa dijaga di sini, bukan dibiarkan basis data yang menolak
//
// Karena galat Oracle ORA-12899 tidak dapat ditampilkan kepada pengguna: ia menyebut nama
// kolom dan nama tabel, yakni rincian internal yang `11-CROSSCUTTING.md` §1.2 larang bocor
// ke peramban. Yang sampai ke pengguna akan berupa "Terjadi kesalahan pada sistem" — pesan
// yang tidak memberi tahu bahwa catatannya sekadar terlalu panjang.
//
// Satuannya BYTE, bukan karakter, dan itu bukan hal yang sama: kolomnya dideklarasikan
// `VARCHAR2(4000 BYTE)`, sehingga satu huruf beraksen atau satu emoji memakan lebih dari
// satu jatah. Pemeriksaan di bawah karena itu menghitung byte, bukan rune.
const RemarksMaxLength = 4000

// NewPostAuditEntry membentuk baris yang sah dari isian layar dan klaim yang dirujuknya,
// atau menyatakan apa yang salah.
//
// Klaimnya diserahkan pemanggil dalam bentuk WorkItem yang SUDAH dipastikan berada di
// antrean Compliance — pemastian itu tugas usecase, bukan fungsi ini.
func NewPostAuditEntry(
	input PostAuditInput, claim WorkItem, sentAt time.Time,
) (PostAuditEntry, error) {
	var violations []Violation

	remarks := strings.TrimSpace(input.Remarks)

	// Catatan BOLEH kosong, dan itu keputusan yang perlu disebut.
	//
	// Kolomnya `NULL`-able dan tidak ada satu pun bukti di export bahwa ia wajib. Membuatnya
	// wajib berarti mengarang aturan yang akan menahan pekerjaan petugas — lebih buruk
	// daripada baris bercatatan kosong, yang paling tidak masih dapat dilengkapi.
	if len(remarks) > RemarksMaxLength {
		violations = append(violations, Violation{
			Field: FieldRemarks,
			Message: "Catatan terlalu panjang. Maksimum " +
				strconv.Itoa(RemarksMaxLength) + " karakter.",
		})
	}

	if len(violations) > 0 {
		return PostAuditEntry{}, NewValidationError(violations)
	}

	return PostAuditEntry{
		// `NO_KLAIM` diisi dari Reference, yakni `PZINSKEY` klaimnya — BUKAN dari
		// claim.ClaimNumber, yang pada baris tab Compliance memang kosong.
		//
		// Terbalik-balik, tetapi benar: kolom bernama "No Klaim" pada tabel Post Audit
		// memang berisi kunci teknis Pega, dan layar Pega menampilkannya begitu.
		ClaimNumber: claim.Reference,

		InsuredName:  claim.InsuredName,
		PolicyNumber: claim.PolicyNumber,
		Remarks:      remarks,
		SentAt:       sentAt,
	}, nil
}
