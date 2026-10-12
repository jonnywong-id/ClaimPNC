// Package usecase mengorkestrasi pengelolaan Master PIC Teknik: melihat daftar, membuka
// satu baris, mencari pegawai di direktori, menambah, dan mengubah.
//
// Tugasnya tiga, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian beserta pencarian nama, lalu memanggil Repo. Ia tidak
// tahu apa pun tentang HTTP maupun SQL.
//
// # Lima aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian:
//
//   - Layar Pega tidak punya tombol hapus — `Harness/UserTeknisInbox-Harness.xml` hanya
//     memuat **Tambah** dan **Refresh**.
//   - `Database/PEGA_MST_USER_TEKNIS.prc` hanya mengenal INSERT dan UPDATE; tidak ada satu
//     pun pernyataan DELETE terhadap MST_USER_TEKNIK di seluruh export.
//   - Menghapus satu petugas akan membuat setiap klaim lama yang menyimpan `OPERATOR_ID`
//     itu kehilangan rujukan penugasannya.
//
// Persis alasan `ADR-0012` menetapkan master tidak dihapus permanen. Petugas yang berhenti
// **dinonaktifkan** lewat kolom aktif, dan itu memang yang disediakan tabelnya.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterpicteknik"
)

// Service mengelola master PIC teknik di atas seam penyimpanan per portal dan seam
// direktori pegawai.
type Service struct {
	repoSelector masterpicteknik.RepoSelector
	directory    masterpicteknik.EmployeeDirectory
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpicteknik.RepoSelector

	// Directory mencari nama pegawai beserta atasannya. Wajib.
	Directory masterpicteknik.EmployeeDirectory
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpicteknik/usecase: RepoSelector wajib diisi")
	}
	if o.Directory == nil {
		return nil, errors.New("masterpicteknik/usecase: Directory wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, directory: o.Directory}, nil
}

// List mengembalikan SELURUH petugas milik satu portal, terurut menurut ID operator.
//
// # Koreksi: tidak disaring menurut status aktif
//
// Report Definition `BrowseVMstUserTeknis_RD` memasang dua penyaring `A AND B`, dan saya
// sempat membawa penyaring B (`STS_AKTIF = '1'`) ke sini. **Itu keliru.** Layar Pega yang
// berjalan menampilkan baris ber-Status Aktif `0` berdampingan dengan `1`; keduanya
// penyaring ber-`pyPromptType=AllAccess`, yaitu nilai awal yang dapat dikosongkan
// pengguna — bukan penyaring tetap.
//
// Petugas nonaktif karena itu tetap muncul, dan kolom Status Aktif yang membedakannya.
//
// Tidak dipaginasi. Isinya puluhan baris, bukan puluhan ribu, dan Report Definition lama
// pun memuat seluruhnya sekaligus dengan batas `pyMaxRecords=500`. Penyaringan dan
// pengurutan cukup dikerjakan di layar atas data yang sudah di tangan.
func (s *Service) List(ctx context.Context, portalAlias string) ([]masterpicteknik.Technician, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterpicteknik/usecase: membaca daftar PIC teknik: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu petugas, aktif maupun tidak.
//
// Ia menggantikan `SetMstUserTeknisValue_act`, yang menjalankan `GetMasterPICTeknis` lalu
// menyalin hasilnya ke halaman `TempDcol` untuk diisi ke form.
func (s *Service) Get(ctx context.Context, portalAlias, operatorID string) (masterpicteknik.Technician, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpicteknik.Technician{}, err
	}

	technician, err := repo.Get(ctx, masterpicteknik.IDKey(operatorID))
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrNotFound) {
			return masterpicteknik.Technician{}, err
		}
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/usecase: membaca PIC %q: %w", operatorID, err)
	}
	return technician, nil
}

// Lookup mencari seorang pegawai di direktori tanpa menyimpan apa pun.
//
// Ia menggantikan `SetMstUserTeknisMstUser_act`, yang berjalan di layar begitu ID operator
// diisi dan mengisi `TempDcol.MCL_NAME`. Memisahkannya sebagai aksi tersendiri membuat
// petugas melihat nama yang ditemukan SEBELUM menekan Simpan — bukan menerima penolakan
// setelah seluruh form diisi.
//
// Hasilnya tidak dipercaya begitu saja saat menyimpan: Create dan Update mencarinya ulang,
// karena nama yang dikirim klien tidak dapat dijamin berasal dari pencarian ini.
func (s *Service) Lookup(ctx context.Context, portalAlias, operatorID string) (masterpicteknik.Employee, error) {
	// Portal diperiksa meski pencarian tidak menyentuh penyimpanan. Alamat layanan
	// direktori dibaca per entitas, dan permintaan tanpa portal yang sah tidak boleh
	// diam-diam memakai portal utama (`R-20`).
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpicteknik.Employee{}, err
	}

	// Master dicoba LEBIH DULU, dan itu atas instruksi Work Owner: nama diambil dari
	// `mst_user_teknis`.
	//
	// Alasan teknisnya menguatkan: direktori HCQ dipanggil lewat `HCQ-LOGIN`, yang
	// endpoint-nya `ValEmpPassOldStructure` — pemeriksa password, bukan pencari pegawai.
	// Ia hanya mengembalikan data pegawai bila passwordnya benar, sedangkan pencarian ini
	// tidak punya password siapa pun. Menggantungkan nama sepenuhnya padanya membuat
	// sebagian besar pencarian nihil tanpa sebab yang terbaca pengguna.
	if employee, ok := fromMaster(ctx, repo, operatorID); ok {
		return employee, nil
	}
	return s.employee(ctx, portalAlias, operatorID)
}

// fromMaster membaca petugas dari master itu sendiri.
//
// # Barisnya ADA sudah cukup — namanya boleh kosong
//
// Sempat dituntut `MCL_NAME` terisi, dengan alasan baris tanpa nama tidak menjawab
// "siapa orang ini". Itu keliru dan langsung menggigit: `JESSEJUANFRITZ` ADA di
// `POOLDATA.MST_USER_TEKNIK`, tetapi karena namanya kosong ia dilempar ke direktori HCQ —
// yang tidak dapat menjawab — lalu muncul sebagai "tidak terdaftar". Petugas yang jelas
// terdaftar dinyatakan tidak terdaftar.
//
// `MCL_NAME` memang tidak selalu terisi: `Database/PEGA_MST_USER_TEKNIS.prc:36-37`
// menulisnya dari parameter `TNAMA`, dan parameter itu berasal dari pencarian yang boleh
// nihil. Baris bernama kosong karena itu normal, bukan baris rusak.
//
// Jawabannya kini: barisnya ada → itu orangnya, apa pun isi namanya.
func fromMaster(ctx context.Context, repo masterpicteknik.Repo, operatorID string) (masterpicteknik.Employee, bool) {
	row, err := repo.Get(ctx, masterpicteknik.IDKey(operatorID))
	if err != nil {
		return masterpicteknik.Employee{}, false
	}
	employee := masterpicteknik.Employee{
		OperatorID:   row.OperatorID,
		Name:         row.Name,
		Email:        row.Email,
		SupervisorID: row.Supervisor,
	}

	// Nama atasan ikut diambil bila atasannya juga ada di master, supaya layar dapat
	// menampilkan nama orang dan bukan sekadar ID. Gagalnya tidak berakibat apa pun:
	// atasan boleh saja tidak terdaftar sebagai PIC teknik.
	if employee.SupervisorID != "" {
		if supervisor, err := repo.Get(ctx, masterpicteknik.IDKey(employee.SupervisorID)); err == nil {
			employee.SupervisorName = supervisor.Name
		}
	}
	return employee, true
}

// Create mendaftarkan petugas baru.
//
// Urutannya mengikuti `CNMInsertMstUserTeknis_act` dan tidak boleh dibalik:
//
//  1. Periksa kelengkapan isian.
//  2. Cari namanya di direktori — TANPA menggagalkan penyimpanan bila tidak ketemu.
//  3. Baru simpan, dengan nama yang berasal dari direktori, bukan dari isian.
//
// Nama TIDAK diterima dari pemanggil. Menerimanya berarti master ini dapat memuat nama
// yang tidak cocok dengan direktori, dan setiap layar yang menampilkan penugasan akan
// menyebut orang yang berbeda dari yang sesungguhnya bertugas.
//
// # Koreksi: pencarian nama tidak lagi menolak
//
// Langkah 2 sempat MENOLAK pengajuan ketika ID tidak ada di direktori. Itu lebih ketat
// daripada acuannya: `Activity/SetMstUserTeknisMstUser_act-Act.xml` hanya menjalankan
// `Page-New` → `Property-Set` → `RDB-List SelectMstUserTeknisMclName` → `Property-Set
// TempDcol.MCL_NAME`. Tidak ada satu pun langkah yang memeriksa hasilnya; nihil berarti
// nama kosong, dan penyimpanan tetap berjalan.
func (s *Service) Create(ctx context.Context, portalAlias string, t masterpicteknik.Technician) (masterpicteknik.Technician, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpicteknik.Technician{}, err
	}

	t = t.Clean()
	if err := masterpicteknik.NewValidationError(masterpicteknik.Check(t)); err != nil {
		return masterpicteknik.Technician{}, err
	}

	t = applyDirectory(t, s.employeeBestEffort(ctx, portalAlias, t.OperatorID))

	// PanelGroup dan Workload tidak pernah ditulis: yang pertama tidak disentuh procedure
	// lama pada cabang mana pun, yang kedua milik view dan tidak ada di tabelnya. Keduanya
	// dikosongkan di sini supaya nilai yang terlanjur dikirim klien tidak diam-diam
	// menjadi perilaku baru.
	t.PanelGroup = ""
	t.Workload = 0

	saved, err := repo.Insert(ctx, t)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrAlreadyExists) {
			return masterpicteknik.Technician{}, err
		}
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/usecase: menambah PIC teknik: %w", err)
	}
	return saved, nil
}

// Update mengganti data petugas yang sudah ada.
//
// ID operator dan nama tidak dapat berubah. ID adalah kunci alaminya — mengubahnya akan
// memutus setiap klaim lama yang menyimpannya. Nama dimiliki direktori, bukan master ini;
// ia disegarkan ulang dari sana setiap kali disimpan, supaya master tidak perlahan
// menyimpang dari sumbernya.
func (s *Service) Update(
	ctx context.Context,
	portalAlias, operatorID string,
	t masterpicteknik.Technician,
) (masterpicteknik.Technician, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpicteknik.Technician{}, err
	}
	operatorID = masterpicteknik.IDKey(operatorID)

	// Keberadaan diperiksa lebih dulu supaya mengubah petugas yang tidak ada dijawab
	// "tidak ditemukan", bukan galat direktori yang menyesatkan.
	existing, err := repo.Get(ctx, operatorID)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrNotFound) {
			return masterpicteknik.Technician{}, err
		}
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/usecase: membaca PIC %q: %w", operatorID, err)
	}

	// ID diambil dari jalur URL, bukan dari badan permintaan: dua sumber untuk satu nilai
	// berarti keduanya dapat berbeda, dan yang menang menjadi soal urutan baca.
	t = t.Clean()
	t.OperatorID = existing.OperatorID

	if err := masterpicteknik.NewValidationError(masterpicteknik.Check(t)); err != nil {
		return masterpicteknik.Technician{}, err
	}

	// Nama disegarkan dari direktori bila ketemu; bila tidak, nama yang SUDAH tersimpan di
	// master dipertahankan. Tanpa penopang itu, satu petugas yang tidak ada di direktori
	// akan kehilangan namanya hanya karena kolom lain disunting.
	t.Name = firstNonEmpty(s.employeeBestEffort(ctx, portalAlias, t.OperatorID).Name, existing.Name)

	// Kedua nilai yang tidak dikelola layar ini dipertahankan apa adanya dari baris yang
	// sudah tersimpan, bukan diambil dari permintaan.
	t.PanelGroup = existing.PanelGroup
	t.Workload = existing.Workload

	saved, err := repo.Update(ctx, t)
	if err != nil {
		if errors.Is(err, masterpicteknik.ErrNotFound) {
			return masterpicteknik.Technician{}, err
		}
		return masterpicteknik.Technician{}, fmt.Errorf("masterpicteknik/usecase: mengubah PIC %q: %w", operatorID, err)
	}
	return saved, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterpicteknik/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// applyDirectory menyalin hasil pencarian ke baris yang akan disimpan.
//
// Nama SELALU menang atas apa pun yang dikirim klien. Atasan hanya diisi bila klien
// mengosongkannya — petugas boleh menimpa usulan direktori, sama seperti layar Pega yang
// menandai isian atasan `pyReadOnly=false`. Atasan dari direktori adalah atasan menurut
// struktur organisasi, dan itu belum tentu atasan penanganan klaim.
func applyDirectory(t masterpicteknik.Technician, employee masterpicteknik.Employee) masterpicteknik.Technician {
	t.Name = employee.Name
	if t.Supervisor == "" {
		t.Supervisor = employee.SupervisorID
	}
	return t
}

// firstNonEmpty mengembalikan nilai pertama yang tidak kosong.
func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// employeeBestEffort mencari pegawai TANPA menggagalkan penyimpanan.
//
// # Kenapa kegagalannya ditelan di sini
//
// Acuannya menelan kegagalan yang sama. `Activity/SetMstUserTeknisMstUser_act-Act.xml`
// menjalankan `RDB-List SelectMstUserTeknisMclName` lalu langsung menyalin hasilnya ke
// `TempDcol.MCL_NAME`; tidak ada langkah yang memeriksa `pxResultCount`, dan tidak ada
// satu pun `pyRequired=true` di seluruh Section. Nihil berarti nama kosong — bukan
// penolakan.
//
// Membuatnya menolak berakibat nyata, bukan sekadar lebih ketat: setiap petugas yang
// sudah terdaftar di master tetapi tidak ditemukan di direktori — berhenti bekerja,
// berpindah entitas, atau ID-nya berubah — menjadi TIDAK DAPAT disunting sama sekali,
// termasuk untuk dinonaktifkan. Padahal menonaktifkannya justru yang perlu dilakukan.
//
// Pencarian yang HASILNYA diperlihatkan ke pengguna tetap melaporkan galatnya apa adanya;
// itu Lookup, dan ia memakai employee() di bawah.
func (s *Service) employeeBestEffort(ctx context.Context, portalAlias, operatorID string) masterpicteknik.Employee {
	if repo, err := s.repoSelector(portalAlias); err == nil {
		if employee, ok := fromMaster(ctx, repo, operatorID); ok {
			return employee
		}
	}

	employee, err := s.directory.Lookup(ctx, portalAlias, operatorID)
	if err != nil {
		return masterpicteknik.Employee{}
	}
	return employee
}

// employee mencari pegawai dan mengubah ketiadaannya menjadi galat validasi pada kolom
// yang benar.
//
// Ketiadaan di direktori BUKAN galat sistem melainkan isian yang salah: yang perlu
// diperbaiki pengguna adalah ID operatornya. Karena itu ia dilaporkan sebagai pelanggaran
// pada field id_operator, sehingga layar menandai kolom itu — bukan menampilkan pesan umum
// yang tidak menunjuk ke mana pun.
//
// Dua kegagalan lain diteruskan apa adanya. Keduanya bukan kesalahan pengguna, dan
// menyamarkannya sebagai isian salah akan menyuruhnya memperbaiki sesuatu yang sudah
// benar.
func (s *Service) employee(ctx context.Context, portalAlias, operatorID string) (masterpicteknik.Employee, error) {
	employee, err := s.directory.Lookup(ctx, portalAlias, operatorID)
	switch {
	case err == nil:
		return employee, nil

	case errors.Is(err, masterpicteknik.ErrEmployeeUnknown):
		return masterpicteknik.Employee{}, masterpicteknik.NewValidationError([]masterpicteknik.Violation{{
			Field:   masterpicteknik.FieldOperatorID,
			Message: "ID operator tidak terdaftar di direktori pegawai.",
		}})

	case errors.Is(err, masterpicteknik.ErrDirectoryUnreachable),
		errors.Is(err, masterpicteknik.ErrDirectoryNotConfigured):
		return masterpicteknik.Employee{}, err

	default:
		return masterpicteknik.Employee{}, fmt.Errorf("masterpicteknik/usecase: mencari pegawai %q: %w", operatorID, err)
	}
}
