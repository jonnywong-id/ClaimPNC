// Package usecase mengorkestrasi pengelolaan Pelaporan Klaim: melihat daftar bertahap,
// membuka satu laporan, mencatat laporan baru, mengubahnya, mentransfernya ke ASM pusat,
// dan menautkannya ke klaim.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket pelaporanklaim dan
// tidak tahu apa pun soal HTTP maupun SQL.
//
// # Enam aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Delete. Layar Pega pun tidak punya — procedure `PROCINSERTDATARECIVEDKLAIM`
// hanya mengenal INSERT dan UPDATE, dan `ADR-0012` menetapkan penghapusan lunak
// menyeluruh. Laporan yang sudah tertaut klaim apalagi: klaimnya merujuk nomor laporan
// itu lewat `ClaimData.RCV_ID`, dan menghapusnya akan memutus rujukan yang dipakai 41
// rule.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/platform/clock"
)

// Service mengelola laporan klaim di atas satu seam penyimpanan.
type Service struct {
	repo  pelaporanklaim.Repo
	clock clock.Clock
}

// Options adalah bahan pembentuk Service.
//
// Field Clock bertipe `clock.Clock`, seam waktu milik `platform/clock` yang berada DI LUAR
// modul ini. Ia yang membuat seluruh aturan berbasis tanggal di sini dapat diuji
// deterministik, tanpa satu pun `time.Now()` di dalam modul.
type Options struct {
	Repo  pelaporanklaim.Repo
	Clock clock.Clock
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.Repo == nil {
		return nil, errors.New("pelaporanklaim/usecase: seam penyimpanan wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("pelaporanklaim/usecase: seam jam wajib diisi")
	}
	return &Service{repo: o.Repo, clock: o.Clock}, nil
}

// ListResult memuat satu halaman laporan beserta angka untuk lencana tiap tab.
//
// Keduanya dikembalikan bersamaan, bukan lewat dua endpoint. Alasannya sama dengan
// alasan sistem lama memakai satu kueri berisi enam SUM(CASE WHEN ...)
// (`RDB List/BrowseClaimRCV_Aksep-SQL.xml`): angka tiap tab harus konsisten dengan isi
// tab yang sedang terbuka. Dua permintaan terpisah dapat tiba di antara dua perubahan,
// dan pengguna melihat lencana "3" di atas tabel berisi 4 baris.
type ListResult struct {
	Page    pelaporanklaim.Page
	Summary pelaporanklaim.StageSummary
}

// List mengembalikan satu halaman laporan beserta ringkasan per tahap.
//
// BERBEDA dari Master Status Klaim, daftar ini DIPAGINASI di server. Itu bukan selera
// melainkan ukuran: master status berisi 33 baris yang bertambah beberapa per tahun,
// sedangkan laporan klaim bertambah ribuan per bulan (`D-10`) dan tidak pernah berkurang.
// Menyaring dan mengurutkannya di peramban akan mengirim seluruh riwayat ke setiap layar
// yang dibuka.
func (s *Service) List(ctx context.Context, f pelaporanklaim.Filter) (ListResult, error) {
	f = f.Normalize()

	if f.Stage != "" && !f.Stage.Known() {
		// Tahap karangan ditolak, bukan diabaikan diam-diam. Penyaring yang diabaikan
		// akan mengembalikan SELURUH baris, dan pemanggilnya mengira ia sudah tersaring.
		return ListResult{}, pelaporanklaim.NewValidationError([]pelaporanklaim.Violation{{
			Field:   "tahap",
			Message: "Tahap tidak dikenal.",
		}})
	}

	page, err := s.repo.List(ctx, f)
	if err != nil {
		return ListResult{}, fmt.Errorf("pelaporanklaim/usecase: membaca daftar laporan: %w", err)
	}

	summary, err := s.repo.Summary(ctx, f)
	if err != nil {
		return ListResult{}, fmt.Errorf("pelaporanklaim/usecase: menghitung ringkasan tahap: %w", err)
	}

	return ListResult{Page: page, Summary: summary}, nil
}

// Get mengembalikan satu laporan.
//
// Ia menggantikan `Activity/SetDataViewRCV_Act-Act.xml`, yang membuka case RCV lalu
// menyalinnya ke halaman tampilan.
func (s *Service) Get(ctx context.Context, number string) (pelaporanklaim.ClaimReport, error) {
	report, err := s.repo.Get(ctx, trim(number))
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: membaca laporan %q: %w", number, err)
	}
	return report, nil
}

// Recorder adalah siapa yang sedang mencatat laporan.
//
// Ia datang dari sesi, bukan dari badan permintaan. Membiarkannya dikirim klien berarti
// siapa pun dapat mengaku mencatat atas nama orang lain — dan karena `D-59` menghapus
// pemisahan tugas, jejak siapa-mencatat-apa adalah satu-satunya kontrol yang tersisa.
type Recorder struct {
	// Login adalah pengenal operator, setara `pxCreateOperator` di sistem lama.
	Login string

	// BranchCode membatasi laporan pada cabang pencatatnya.
	//
	// Di sistem lama ia dibaca dari `GetIDCabang` lewat DB Link `@ASMD`
	// (`Activity/CreateNewCaseRCV-Act.xml` step 6). API penggantinya (`D-25`, `R-03`)
	// belum ada, sehingga nilainya hari ini datang dari profil pengguna dan boleh kosong.
	BranchCode string
}

// Record menyimpan laporan baru dan mengembalikannya lengkap dengan nomor yang dibuat
// penyimpanan.
//
// Nomor TIDAK diterima dari pemanggil, mengikuti pola yang sama dengan seluruh modul
// lain: pada penambahan ia dibuat penyimpanan, pada pengubahan ia berada di jalur URL.
// Menerima nomor dari luar akan membuat dua laporan memperebutkan nomor yang sama.
//
// # Empat field yang diisi sistem, bukan pengguna
//
// Pencatat, cabang, waktu pencatatan, dan tahap awal. Keempatnya sengaja ditimpa di sini
// alih-alih dipercaya dari badan permintaan — lihat Recorder.
func (s *Service) Record(ctx context.Context, report pelaporanklaim.ClaimReport, by Recorder) (pelaporanklaim.ClaimReport, error) {
	report = report.Clean()

	if err := pelaporanklaim.NewValidationError(report.Validate()); err != nil {
		return pelaporanklaim.ClaimReport{}, err
	}

	now := s.clock.Now().UTC()

	report.Number = ""
	report.CreatedBy = trim(by.Login)
	report.CreatedAt = now
	report.UpdatedAt = now

	// Cabang dari PROFIL menang atas isian form.
	//
	// Cabang menentukan siapa yang kelak boleh melihat laporan ini
	// (`11-SECURITY.md` §3.2). Membiarkan form menimpanya berarti petugas dapat mencatat
	// laporan ke cabang lain — dan batas data yang dibangun `TKT-F3-005` akan berdiri di
	// atas nilai yang dipilih sendiri oleh yang dibatasi.
	//
	// Isian form dipakai HANYA bila profil tidak membawanya. Itu bukan celah melainkan
	// keharusan: pengguna non-karyawan tidak punya cabang sama sekali —
	// `POOLDATA.M_LOGIN_PNC` tidak memuatnya — dan tanpa jalan mundur ini seluruh laporan
	// yang dicatat broker akan tercatat tanpa cabang.
	if branch := trim(by.BranchCode); branch != "" {
		report.BranchCode = branch
	}

	// Laporan baru SELALU mulai dari tahap paling awal, apa pun yang dikirim klien.
	// Membiarkan klien menyatakan dirinya "sudah ditransfer" akan melewati satu-satunya
	// langkah yang mencatat kapan dan oleh siapa transfer itu terjadi.
	report.Transferred = false
	report.TransferredAt = nil
	report.ClaimNumber = ""
	report.RegisteredAt = nil
	report.Outcome = pelaporanklaim.OutcomeNone

	saved, err := s.repo.Insert(ctx, report)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNumberTaken) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: mencatat laporan: %w", err)
	}
	return saved, nil
}

// Update mengganti isi laporan yang sudah ada.
//
// # Apa yang tidak dapat diubah lewat jalur ini
//
// Nomor, pencatat, waktu pencatatan, dan SELURUH penanda daur hidup. Yang terakhir punya
// jalurnya sendiri — Transfer dan LinkClaim — supaya perpindahan tahap tidak dapat
// terjadi sebagai efek samping penyuntingan biasa.
//
// # Kenapa laporan yang sudah diregistrasi ditolak
//
// Sejak klaim terbit, yang berlaku adalah data klaimnya. Membiarkan laporannya tetap
// dapat disunting berarti dua versi nomor polis dan nama tertanggung hidup berdampingan,
// dan tidak ada yang tahu mana yang dipakai. Sistem lama menutup ini dengan cara yang
// berbeda dan lebih lemah: layarnya dikunci `StatusLock`, tetapi kuncinya DAPAT DILEWATI
// operator berjabatan "PA" atau bercabang kantor pusat
// (`Activity/Pre_ActReceiveDocument-Act.xml` step 3-4). Pengecualian itu tidak dibawa —
// ia pagar kewenangan yang ditulis di dalam kode, persis yang `D-15` larang.
func (s *Service) Update(ctx context.Context, number string, report pelaporanklaim.ClaimReport) (pelaporanklaim.ClaimReport, error) {
	number = trim(number)

	existing, err := s.repo.Get(ctx, number)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: membaca laporan %q: %w", number, err)
	}
	if existing.IsRegistered() {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrAlreadyRegistered
	}

	report = report.Clean()
	if err := pelaporanklaim.NewValidationError(report.Validate()); err != nil {
		return pelaporanklaim.ClaimReport{}, err
	}

	// Yang boleh berubah disalin ke atas baris yang sudah ada, bukan sebaliknya. Arah ini
	// yang membuat field di luar daftar tidak mungkin ikut tertimpa — termasuk field yang
	// ditambahkan kemudian dan lupa didaftarkan di sini.
	next := existing
	next.ReporterName = report.ReporterName
	next.SenderEmail = report.SenderEmail
	next.SenderPhone = report.SenderPhone
	next.CourierName = report.CourierName
	next.EmailSubject = report.EmailSubject
	next.PolicyNumber = report.PolicyNumber
	next.InsuredName = report.InsuredName
	next.InsuredEmail = report.InsuredEmail
	next.BusinessCode = report.BusinessCode
	next.GroupPanel = report.GroupPanel
	next.ReferenceNumber = report.ReferenceNumber
	next.LossDate = report.LossDate
	next.LossLocation = report.LossLocation
	next.Chronology = report.Chronology
	next.DamageDetails = report.DamageDetails
	next.DriverLicense = report.DriverLicense
	next.EstimatedValue = report.EstimatedValue
	next.ClaimType = report.ClaimType
	next.DocumentCount = report.DocumentCount
	next.DocumentReceivedDate = report.DocumentReceivedDate
	next.NotTransferredReason = report.NotTransferredReason
	next.NotRegisteredNote = report.NotRegisteredNote
	next.UpdatedAt = s.clock.Now().UTC()

	saved, err := s.repo.Update(ctx, next)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: mengubah laporan %q: %w", number, err)
	}
	return saved, nil
}

// Transfer menandai laporan sudah dikirim ke ASM pusat.
//
// Ia menggantikan pengisian `ReceiveDocument.StatusLock` dan
// `ReceiveDocument.DateOfSendASM` di sistem lama, yang di sana terjadi sebagai efek
// samping penyimpanan layar — tanpa langkah tersendiri dan tanpa apa pun yang mencegah
// laporan ditransfer dua kali.
//
// Di sini ia menjadi aksi sendiri yang IDEMPOTEN pada hasilnya tetapi TIDAK DIAM: laporan
// yang sudah ditransfer dijawab ErrAlreadyTransferred, bukan ditransfer ulang dengan
// tanggal baru. Menimpa tanggal transfer berarti menghapus kapan ia benar-benar dikirim.
func (s *Service) Transfer(ctx context.Context, number string) (pelaporanklaim.ClaimReport, error) {
	number = trim(number)

	report, err := s.repo.Get(ctx, number)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: membaca laporan %q: %w", number, err)
	}
	if report.IsRegistered() {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrAlreadyRegistered
	}
	if report.Transferred {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrAlreadyTransferred
	}

	now := s.clock.Now().UTC()
	report.Transferred = true
	report.TransferredAt = &now
	report.UpdatedAt = now

	saved, err := s.repo.Update(ctx, report)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: mentransfer laporan %q: %w", number, err)
	}
	return saved, nil
}

// LinkClaim mencatat bahwa laporan ini sudah menjadi klaim.
//
// # Kenapa ia ada sekarang padahal modul registrasi belum dibangun
//
// Di sistem lama arah penautannya TERBALIK dari yang terbaca sepintas: bukan layar
// laporan yang mencatat nomor klaim, melainkan KLAIM yang membuka case laporan lalu
// menulis ke dalamnya (`Activity/UpdateRCVCase-Act.xml` — Obj-Open-By-Handle atas
// `"ASM-FW-GCNMFW-WORK " + ClaimData.RCV_ID`, lalu menyalin polis, tertanggung, tanggal
// kejadian, dan kronologi).
//
// Modul `B-2` Input Register kelak memanggil aksi ini. Sampai ia ada, aksi ini tetap
// berguna: laporan yang klaimnya sudah dibuat di Pega dapat ditandai supaya tidak
// tertinggal di tab "belum diregistrasi" selamanya.
//
// # Yang sengaja TIDAK ditiru
//
// `UpdateRCVCase` juga MENIMPA nomor polis, nama tertanggung, tanggal kejadian, dan
// kronologi pada laporan dengan nilai dari klaim. Itu menghapus apa yang benar-benar
// dilaporkan pelapor, dan dengan begitu menghapus satu-satunya cara mengetahui bahwa
// pelapor semula menyebut polis yang keliru. Di sini laporan dibiarkan apa adanya; yang
// ditambahkan hanya nomor klaim dan tanggalnya.
func (s *Service) LinkClaim(ctx context.Context, number, claimNumber string) (pelaporanklaim.ClaimReport, error) {
	number = trim(number)
	claimNumber = trim(claimNumber)

	if claimNumber == "" {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.NewValidationError([]pelaporanklaim.Violation{{
			Field:   pelaporanklaim.FieldClaimNumber,
			Message: "Nomor klaim wajib diisi.",
		}})
	}

	report, err := s.repo.Get(ctx, number)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: membaca laporan %q: %w", number, err)
	}
	if report.IsRegistered() {
		// Penautan ulang ditolak. Satu laporan melahirkan satu klaim; menautkannya ke
		// klaim kedua akan membuat dua klaim mengaku berasal dari laporan yang sama, dan
		// tidak ada apa pun sesudahnya yang dapat membedakan mana yang benar.
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrAlreadyRegistered
	}

	now := s.clock.Now().UTC()
	report.ClaimNumber = claimNumber
	report.RegisteredAt = &now
	report.UpdatedAt = now

	// Registrasi mengandaikan laporannya sudah sampai ASM. Laporan yang diregistrasi
	// tanpa pernah ditransfer — mungkin karena dicatat langsung di pusat — ditandai
	// ditransfer pada saat yang sama, supaya tidak ada laporan berklaim yang tetap
	// terbaca "belum ditransfer".
	if !report.Transferred {
		report.Transferred = true
		report.TransferredAt = &now
	}

	saved, err := s.repo.Update(ctx, report)
	if err != nil {
		if errors.Is(err, pelaporanklaim.ErrNotFound) {
			return pelaporanklaim.ClaimReport{}, err
		}
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/usecase: menautkan laporan %q ke klaim: %w", number, err)
	}
	return saved, nil
}

// trim membuang spasi tepi dari masukan sebelum ia menyentuh penyimpanan.
func trim(s string) string { return strings.TrimSpace(s) }
