package registrasi

// # Baris daftar kerja — POOLDATA.T_CLAIMLIST_ADMIN
//
// Layar My Inbox (`inboxoutstanding`) membaca SATU tabel datar, `T_CLAIMLIST_ADMIN`:
// satu baris per klaim, berisi kepala klaim ditambah penugasan yang sedang berjalan.
// Tabel itu diisi proses di luar Pega — tidak satu pun rule, procedure, maupun trigger
// di export menyentuhnya — dan proses itu tidak mengenal klaim `PNCN`. Akibatnya klaim
// yang dibuka aplikasi ini tidak pernah muncul di My Inbox.
//
// Work Owner memutuskan (2026-09-28) aplikasi ini ikut MENULIS baris klaim PNCN ke tabel
// itu. Yang ditulis hanya baris ber-PZINSKEY nomor PNCN; baris Pega tidak disentuh.
//
// # Kolom yang sengaja tidak diisi
//
// Hanya kolom yang nilainya dapat diturunkan PASTI dari klaim dan tugasnya yang diisi.
// Enam kolom dibiarkan NULL karena aturan pengisinya tidak diketahui, dan hitungan
// terhadap 874 baris Pega membuktikan ia tidak dapat ditebak dari tahap:
//
//	OS_CATEGORY      "Input Estimasi" bernilai OS REGISTRASI KLAIM (163) dan OS FOLLOW UP KE (16)
//	STATUSPROGRESS1  sembilan nilai berbeda pada tahap yang sama ("Choose Surveyor")
//	STATUSPROGRESS2  idem
//	AGING            dibaca apa adanya oleh My Inbox; artinya belum dipastikan
//	DATEFORAGING_1   idem
//	PXCREATEOPNAME   NAMA operator (403 baris berbeda dari PXCREATEOPERATOR); aplikasi ini
//	                 hanya memegang identitasnya
//
// Mengisinya dengan dugaan membuat baris tampak lengkap padahal salah — kegagalan yang
// tidak menghasilkan satu pun galat.

// WorkFlowName adalah PXFLOWNAME baris klaim — nama flow Pega yang digantikan alur ini.
const WorkFlowName = "Register_Flow"

// WorkObjectClass adalah PXOBJCLASS baris klaim. My Inbox menyaring kolom ini.
const WorkObjectClass = "ASM-FW-GCNMFW-Work-PNC"

// Nilai PYSTATUSWORK sebagaimana tersimpan pada baris Pega di T_CLAIMLIST_ADMIN.
const (
	WorkStatusNew       = "New"
	WorkStatusCompleted = "Resolved-Completed"
	WorkStatusRejected  = "Resolved-Rejected"
)

// InboxEntry adalah satu baris daftar kerja: klaim beserta tugasnya yang masih terbuka.
type InboxEntry struct {
	Claim Claim

	// Task adalah tugas yang sedang menunggu; nil bila klaim tidak punya tugas terbuka.
	Task *Task

	// StageName adalah nama tahap tugas itu — PXTASKLABEL. Nama tahap alur ini sudah
	// sama dengan nama assignment di `Flow/Register_Flow.xml` ("Input Register",
	// "Choose Surveyor", "Input Estimasi"), sehingga tidak perlu pemetaan.
	StageName string
}

// NewInboxEntry menyusun baris daftar kerja sebuah klaim.
func NewInboxEntry(claim Claim, open *Task, flow Definition) InboxEntry {
	e := InboxEntry{Claim: claim, Task: open}
	if open != nil {
		if s, ok := flow.Stage(open.Stage); ok {
			e.StageName = s.Name
		}
	}
	return e
}

// Key adalah PZINSKEY — kunci primer tabel — sekaligus PYID: nomor klaimnya.
//
// Pega mengisi PZINSKEY dengan nama kelas di depan nomor ("ASM-FW-GCNMFW-WORK PNC-…");
// `D-22` menetapkan data baru tidak pernah menulis awalan itu, sehingga kuncinya nomor
// PNCN apa adanya. Nomor PNCN dan PNC tidak dapat bertabrakan, jadi baris Pega tidak
// mungkin tertimpa.
//
// Kosong untuk klaim tanpa nomor — klaim lama yang dibuka sebelum nomor terbit saat
// pembukaan. Baris seperti itu tidak ditulis: kuncinya belum ada.
func (e InboxEntry) Key() string { return e.Claim.Number }

// WorkStatus adalah PYSTATUSWORK baris ini.
//
// Klaim yang sudah selesai atau ditolak memakai status Resolved-*, sehingga My Inbox
// menyaringnya keluar. Selebihnya "New" — nilai yang dipakai 988 dari 1.016 baris Pega.
func (e InboxEntry) WorkStatus() string {
	switch e.Claim.ProcessStatus {
	case ProcessDone:
		return WorkStatusCompleted
	case ProcessRejected:
		return WorkStatusRejected
	}
	return WorkStatusNew
}

// AssignedOperator adalah PXASSIGNEDOPERATORID — pemilik tugas yang sedang berjalan.
//
// Kosong untuk tugas Workbasket yang belum diambil dan untuk klaim tanpa tugas terbuka,
// sama seperti baris Pega: kedelapan baris Pega yang PXTASKLABEL-nya kosong juga tanpa
// operator.
func (e InboxEntry) AssignedOperator() string {
	if e.Task == nil {
		return ""
	}
	return e.Task.Owner
}

// Active adalah STS_AKTIF: "0" untuk klaim yang ditandai terhapus (`ADR-0012`).
func (e InboxEntry) Active() string {
	if e.Claim.Deleted() {
		return "0"
	}
	return "1"
}
