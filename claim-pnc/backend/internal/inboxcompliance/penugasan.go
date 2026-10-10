package inboxcompliance

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// Perpindahan penugasan setelah Compliance memutuskan — padanan Ticket `SendtoAnalysator`,
// langkah 22 `SetComplianceResult`.
//
// # Ke mana ticket itu melompat, dan buktinya
//
//	Ticket/SendtoAnalysator-Ticket.xml      kelas ASM-FW-GCNMFW-Work-PNC, pzStatus valid
//	Flow/Register_Flow.xml                  Ticket2 menempel pada Assignment5
//	                                        "Send To Analis"
//	shape yang sama                         pyImplementation = WorkList
//
// Jadi ia **bukan** memindahkan klaim ke antrean bersama lain. Ia memindahkannya ke
// **worklist satu orang**.

// Nama tahap, diambil dari nama shape pada `Flow/Register_Flow.xml` — bukan dikarang,
// supaya baris penugasan dapat ditelusuri balik ke flow aslinya.
const (
	// StageCompliance adalah Assignment9 "Compliance", `pyWorkBasket = CompliancePNC`.
	StageCompliance = "Compliance"

	// StageSendToAnalyst adalah Assignment5 "Send To Analis", tujuan Ticket
	// `SendtoAnalysator`.
	StageSendToAnalyst = "Send To Analis"

	// StageSendToTechnician adalah **Assignment8 "Send To PIC Teknik"**, tujuan Ticket
	// `SendToPICTravel`.
	//
	// Terbaca langsung dari `Flow/Register_Flow.xml`: `pyTicketShapes` berisi
	// `pyMOName = SendToPICTravel` dengan `pyMOId = Ticket7`, dan blok itu berada di
	// dalam rowdata `REPEATINGINDEX="Assignment8"` yang ber-`pyMOName = Send To PIC
	// Teknik`.
	StageSendToTechnician = "Send To PIC Teknik"
)

// Jenis penugasan — `D-26`. Satu tugas selalu salah satu, tidak pernah keduanya.
const (
	AssignmentWorklist   = "WORKLIST"
	AssignmentWorkbasket = "WORKBASKET"
)

// Status penugasan. Yang selesai DITANDAI, tidak dihapus (`D-66`).
const (
	AssignmentWaiting = "MENUNGGU"
	AssignmentDone    = "SELESAI"
)

// Assignment adalah satu baris pada `POOLDATA.CPNC_PENUGASAN`.
type Assignment struct {
	// ID diterbitkan aplikasi.
	ID string

	// Reference adalah `PZINSKEY` klaimnya.
	Reference string

	// Stage adalah nama shape pada flow — lihat konstanta di atas.
	Stage string

	// Kind adalah WORKLIST atau WORKBASKET.
	Kind string

	// AssignedTo terisi HANYA pada WORKLIST, Workbasket HANYA pada WORKBASKET.
	// Basis data menegakkannya lewat `CK_CPNC_PENUGASAN_ISI`.
	AssignedTo string
	Workbasket string

	Status    string
	CreatedAt time.Time
}

// AssignmentMove adalah satu perpindahan: menutup tahap lama, membuka tahap baru.
//
// # Kenapa keduanya dalam SATU tipe
//
// Karena keduanya satu peristiwa, dan memisahkannya akan mengizinkan keadaan yang tidak
// boleh ada: tahap lama tertutup tanpa tahap baru terbuka — klaim hilang dari antrean
// Compliance tanpa tiba di mana pun, tanpa satu pun pesan galat.
//
// Membungkusnya di satu tipe membuat pemanggil tidak dapat mengerjakan separuhnya.
type AssignmentMove struct {
	// Closed adalah tahap yang ditinggalkan, ditandai SELESAI.
	Closed Assignment

	// Opened adalah tahap tujuan, berstatus MENUNGGU.
	Opened Assignment
}

// NewAssignmentMove menyusun perpindahan dari antrean Compliance ke tahap berikutnya.
//
// # Tujuannya ditentukan LINI BISNIS, bukan tombol yang ditekan
//
// `Activity/SetComplianceResult-Act.xml` memanggil `SetTicket` DUA KALI, dan keduanya
// dijaga prakondisi:
//
//	langkah 22   Ticket = SendtoAnalysator   bila `isPA_PNC`  DAN PilihanCompliance terisi
//	langkah 23   Ticket = SendToPICTravel    bila `IsTravel`  DAN PilihanCompliance terisi
//
// Activity yang SAMA dipanggil tombol "Kirim ke Analyst" (sel 79) maupun "Kirim ke PIC
// Teknik" (sel 80). Jadi yang menentukan tujuan adalah lini bisnis klaimnya, **bukan**
// tombol mana yang ditekan — dan membangunnya per-tombol akan salah pada klaim yang
// lininya tidak sesuai tombolnya.
//
// # Koreksi atas versi pertama
//
// Versi pertama memindahkan **setiap** klaim ber-`TechnicianID` ke "Send To Analis",
// tanpa memeriksa lini bisnis. Itu salah di dua arah sekaligus: klaim Travel dikirim ke
// Analis (seharusnya PIC Teknik), dan klaim lini lain — Fire, Marine, Aneka — dipindahkan
// padahal di Pega **tidak ada tiket yang menyala sama sekali**, sehingga klaimnya tetap
// di antrean Compliance.
//
// # Siapa yang menerimanya
//
// `ClaimData.UserTeknis` milik klaim itu — PIC Teknik yang sudah melekat padanya sejak
// sebelum masuk Compliance.
//
// Ini **KESIMPULAN**, bukan bacaan langsung, dan itu harus dinyatakan: router aslinya
// `PNCTeknikRouter` **hilang dari export** (`R-04`). Dasar kesimpulannya:
//
//   - `Activity/PNC_ReassignPNCTeknik-Act.xml` memperlihatkan klaim MEMBAWA `UserTeknis`
//     beserta `UserTeknisEmail`, dan memindahkan PIC berarti mengubah keduanya lalu
//     memasang Ticket — bukan memilih orang baru dari antrean.
//   - "Send To Analis" adalah tahap LANJUTAN pada klaim yang PIC-nya sudah ditetapkan di
//     tahap sebelumnya; memilih orang baru di sini akan memutus kesinambungan penanganan
//     yang justru menjadi alasan `D-26` mempertahankan Worklist.
//
// Bila kesimpulan ini keliru — misalnya router sebenarnya memilih petugas paling senggang
// lewat `ORDER BY counter_quota ASC` seperti `BrowsePICRandomTeam-SQL.xml:39-40` — yang
// berubah hanya fungsi ini, bukan tabel maupun alurnya.
//
// Nilai kedua salah berarti **tidak ada perpindahan**, dan itu keadaan sah: klaim tanpa
// PIC Teknik tidak punya tujuan, sehingga ia tetap di antrean Compliance alih-alih hilang
// ke tempat yang tidak ada.
func NewAssignmentMove(decision Decision, claim WorkItem) (AssignmentMove, bool) {
	if claim.TechnicianID == "" {
		return AssignmentMove{}, false
	}

	// `PropertyHasValue(.ClaimData.PilihanCompliance)` — prakondisi KEDUA pada langkah 22
	// dan 23. NewDecision sudah menolak pilihan kosong, sehingga di sini ia tidak pernah
	// menyala; ia tetap ditulis supaya prakondisi Pega terbaca lengkap di satu tempat,
	// dan supaya perubahan validasi kelak tidak diam-diam melewatinya.
	if decision.Choice == "" {
		return AssignmentMove{}, false
	}

	// Tahap tujuan menurut lini bisnis. Lini selain PA dan Travel TIDAK punya tiket, dan
	// karena itu tidak berpindah — klaimnya tetap menunggu di antrean Compliance.
	var tujuan string
	switch claim.GroupPanel {
	case GroupPanelPersonalAccident:
		tujuan = StageSendToAnalyst
	case GroupPanelTravel:
		tujuan = StageSendToTechnician
	default:
		return AssignmentMove{}, false
	}

	return AssignmentMove{
		Closed: Assignment{
			ID:         assignmentID(decision.Reference, StageCompliance, decision.DecidedAt),
			Reference:  decision.Reference,
			Stage:      StageCompliance,
			Kind:       AssignmentWorkbasket,
			Workbasket: WorkbasketCompliance,
			Status:     AssignmentDone,
			CreatedAt:  decision.DecidedAt,
		},
		Opened: Assignment{
			ID:         assignmentID(decision.Reference, tujuan, decision.DecidedAt),
			Reference:  decision.Reference,
			Stage:      tujuan,
			Kind:       AssignmentWorklist,
			AssignedTo: claim.TechnicianID,
			Status:     AssignmentWaiting,
			CreatedAt:  decision.DecidedAt,
		},
	}, true
}

// assignmentID menurunkan kunci baris dari datanya sendiri.
//
// # Kenapa diturunkan, bukan diacak
//
// Supaya tidak perlu seam pembangkit ID baru — satu dependensi lagi yang harus disuntikkan
// ke setiap pemanggil dan dipalsukan di setiap uji, demi nilai yang tidak pernah dibaca
// manusia.
//
// # Satu sifat yang disengaja
//
// Dua penyimpanan atas klaim, tahap, DAN waktu yang sama menghasilkan ID yang sama,
// sehingga yang kedua DITOLAK primary key. Itu bukan efek samping yang ditoleransi
// melainkan penjagaan yang dikehendaki: tanpa kunci idempotensi (`10-API-STRATEGY.md` §7),
// inilah satu-satunya yang menahan klik ganda menerbitkan dua perpindahan.
//
// Keputusan yang diubah kemudian tetap menghasilkan ID berbeda, karena waktunya berbeda.
//
// MD5 dipakai sebagai PERINGKAS, bukan sebagai pengaman — tidak ada rahasia di sini, dan
// 32 karakter heksnya muat di `VARCHAR2(64)` sementara `PZINSKEY` sendiri 255.
func assignmentID(reference, stage string, at time.Time) string {
	sidik := md5.Sum([]byte(fmt.Sprintf("%s|%s|%d", reference, stage, at.UnixNano())))
	return hex.EncodeToString(sidik[:])
}
