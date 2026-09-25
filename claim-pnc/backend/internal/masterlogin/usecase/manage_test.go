package usecase_test

import (
	"context"
	"errors"
	"testing"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterlogin/repo/memory"
	"claim-pnc/internal/masterlogin/usecase"
)

const portalUji = "ASM"

// layanan menyiapkan Service beserta repo memori berisi baris yang diberikan.
func layanan(t *testing.T, rows []masterlogin.SurveyorLogin) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewRepo(memory.Options{Rows: rows})
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterlogin.Repo, error) {
			if alias != portalUji {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return service, repo
}

// tim adalah satu leader beserta satu anggotanya, isi awal yang dipakai kebanyakan uji.
func tim() []masterlogin.SurveyorLogin {
	return []masterlogin.SurveyorLogin{
		{
			Name:        "Budi Hartono",
			Login:       "BudiHartono",
			Email:       "budi@contoh.invalid",
			Phone:       "021-5550101",
			Address:     "Jl. Melati",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "",
		},
		{
			Name:        "Rina Ayu",
			Login:       "RinaAyu",
			Email:       "rina@contoh.invalid",
			Phone:       "021-5550102",
			Address:     "Jl. Kenanga",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "BudiHartono",
		},
	}
}

func isianSah() masterlogin.Input {
	return masterlogin.Input{
		Name:    "Agus Pratama",
		Email:   "agus@contoh.invalid",
		Phone:   "021-5550103",
		Address: "Jl. Anggrek",
	}
}

// TestCreateDerivesLoginFromName membuktikan LOGIN diturunkan SERVER, bukan diterima dari
// klien.
//
// Di Pega penurunannya terjadi di layar lalu dikirim kembali sebagai isian biasa, sehingga
// permintaan yang tidak datang dari layar dapat mengirim LOGIN apa pun. Kunci baris tidak
// boleh bergantung pada kejujuran klien; lihat masterlogin.DeriveLogin.
func TestCreateDerivesLoginFromName(t *testing.T) {
	service, _ := layanan(t, tim())

	saved, err := service.Create(context.Background(), portalUji, isianSah(),
		usecase.Actor{Login: "RinaAyu"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if saved.Login != "AgusPratama" {
		t.Fatalf("Login = %q, mau %q", saved.Login, "AgusPratama")
	}
}

// TestCreateWritesMemberStatus membuktikan STSLOGIN selalu "Member".
//
// `Activity/CNMInsertMstLoginSurveyor_act` menetapkan `TempLoginSurvey.ObjectName :=
// "Member"` tanpa syarat apa pun, dan isian itu tidak pernah digambar di layar.
func TestCreateWritesMemberStatus(t *testing.T) {
	service, _ := layanan(t, tim())

	saved, err := service.Create(context.Background(), portalUji, isianSah(),
		usecase.Actor{Login: "RinaAyu"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if saved.LoginStatus != masterlogin.LoginStatusMember {
		t.Fatalf("LoginStatus = %q, mau %q", saved.LoginStatus, masterlogin.LoginStatusMember)
	}
}

// TestCreateInheritsLeaderOfTheSaver membuktikan LOGINLEADER diturunkan dari LEADER milik
// pengguna yang menyimpan — bukan dari login pengguna itu sendiri.
//
// Inilah bagian yang paling mudah salah dibaca. `GetLoginLeaderSurveyor` berbunyi:
//
//	select loginleader from pooldata.mst_login_surveyor where login = {OperatorID...}
//
// Rina bertim pada Budi. Saat Rina menambahkan Agus, Agus ikut bertim pada BUDI — bukan
// pada Rina. Bila uji ini gagal dengan nilai "RinaAyu", penurunannya terbalik.
func TestCreateInheritsLeaderOfTheSaver(t *testing.T) {
	service, _ := layanan(t, tim())

	saved, err := service.Create(context.Background(), portalUji, isianSah(),
		usecase.Actor{Login: "RinaAyu"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if saved.LeaderLogin != "BudiHartono" {
		t.Fatalf("LeaderLogin = %q, mau %q — penurunannya mungkin terbalik",
			saved.LeaderLogin, "BudiHartono")
	}
}

// TestCreateAllowsSaverWithoutOwnRow membuktikan pengguna yang TIDAK terdaftar sebagai
// surveyor tetap dapat menambah — barisnya tersimpan dengan LOGINLEADER kosong.
//
// Perilaku Pega apa adanya (`P-5`): kuerinya tidak mengembalikan apa pun, dan langkah
// berikutnya menyalin nilai yang tidak ada. Menolaknya akan menghalangi petugas admin
// menambahkan surveyor sama sekali.
func TestCreateAllowsSaverWithoutOwnRow(t *testing.T) {
	service, _ := layanan(t, tim())

	saved, err := service.Create(context.Background(), portalUji, isianSah(),
		usecase.Actor{Login: "PetugasAdmin"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if saved.LeaderLogin != "" {
		t.Fatalf("LeaderLogin = %q, mau kosong", saved.LeaderLogin)
	}
}

// TestCreateRejectsDuplicateLogin membuktikan LOGIN ganda ditolak.
//
// "Budi.Hartono" dan "Budi Hartono" adalah dua nama yang terlihat berbeda dan menghasilkan
// LOGIN yang sama persis — itulah bentuk bentrok yang paling mengejutkan di modul ini, dan
// itulah yang diuji di sini alih-alih nama yang jelas-jelas sama.
func TestCreateRejectsDuplicateLogin(t *testing.T) {
	service, _ := layanan(t, tim())

	input := isianSah()
	input.Name = "Budi.Hartono"

	_, err := service.Create(context.Background(), portalUji, input,
		usecase.Actor{Login: "RinaAyu"}, nil)
	if !errors.Is(err, masterlogin.ErrLoginTaken) {
		t.Fatalf("Create = %v, mau ErrLoginTaken", err)
	}
}

// TestCreateRejectsDuplicateLoginIgnoringCase membuktikan pemeriksaan keunikan TIDAK peka
// huruf besar-kecil.
//
// Dua kunci yang hanya berbeda huruf besar-kecilnya akan bertabrakan pada basis data yang
// membandingkannya tanpa membedakan; lihat banner masterlogin.sql.
func TestCreateRejectsDuplicateLoginIgnoringCase(t *testing.T) {
	service, _ := layanan(t, tim())

	input := isianSah()
	input.Name = "budi hartono"

	_, err := service.Create(context.Background(), portalUji, input,
		usecase.Actor{Login: "RinaAyu"}, nil)
	if !errors.Is(err, masterlogin.ErrLoginTaken) {
		t.Fatalf("Create = %v, mau ErrLoginTaken", err)
	}
}

// TestSaveRejectsChangedName membuktikan Nama tidak dapat diubah lewat server, bukan hanya
// terkunci di layar.
//
// Penguncian di antarmuka adalah kenyamanan tampilan; permintaan yang tidak datang dari
// layar itu tidak tersentuh olehnya. Lihat masterlogin.ErrNameLocked.
func TestSaveRejectsChangedName(t *testing.T) {
	service, _ := layanan(t, tim())

	input := masterlogin.Input{
		Name:  "Budi Hartono Baru",
		Email: "budi@contoh.invalid",
		Phone: "021-5550101",
	}

	_, err := service.Save(context.Background(), portalUji, "BudiHartono", input,
		usecase.Actor{Login: "BudiHartono"}, nil)
	if !errors.Is(err, masterlogin.ErrNameLocked) {
		t.Fatalf("Save = %v, mau ErrNameLocked", err)
	}
}

// TestSaveKeepsDerivedColumns membuktikan ketiga kolom yang diturunkan server TIDAK berubah
// saat menyimpan — bahkan ketika barisnya disunting berkali-kali.
//
// Ketiganya diambil dari BARIS YANG TERSIMPAN, bukan dari permintaan, sehingga penyuntingan
// tidak pernah dapat memindahkan seseorang ke tim lain maupun mengubah perannya.
func TestSaveKeepsDerivedColumns(t *testing.T) {
	service, _ := layanan(t, tim())

	input := masterlogin.Input{
		Name:    "Rina Ayu",
		Email:   "rina.baru@contoh.invalid",
		Phone:   "0812-000",
		Address: "Jl. Baru",
	}

	saved, err := service.Save(context.Background(), portalUji, "RinaAyu", input,
		usecase.Actor{Login: "BudiHartono"}, nil)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	switch {
	case saved.Login != "RinaAyu":
		t.Errorf("Login berubah menjadi %q", saved.Login)
	case saved.LoginStatus != masterlogin.LoginStatusMember:
		t.Errorf("LoginStatus berubah menjadi %q", saved.LoginStatus)
	case saved.LeaderLogin != "BudiHartono":
		t.Errorf("LeaderLogin berubah menjadi %q", saved.LeaderLogin)
	}

	if saved.Email != "rina.baru@contoh.invalid" || saved.Phone != "0812-000" ||
		saved.Address != "Jl. Baru" {
		t.Errorf("ketiga isian yang boleh berubah tidak tersimpan: %+v", saved)
	}
}

// TestSavePreservesUnknownLoginStatus membuktikan nilai STSLOGIN di luar "Member"
// DIPERTAHANKAN, bukan ditimpa.
//
// Tidak satu pun rule di export menuliskan nilai lain (`R-16`), tetapi baris lama dapat
// memuatnya. Menimpanya dengan "Member" berarti modul ini diam-diam mengubah peran
// seseorang pada baris yang petugas hanya ingin perbarui nomor teleponnya.
func TestSavePreservesUnknownLoginStatus(t *testing.T) {
	rows := tim()
	rows[0].LoginStatus = "Leader"
	service, _ := layanan(t, rows)

	saved, err := service.Save(context.Background(), portalUji, "BudiHartono",
		masterlogin.Input{
			Name:  "Budi Hartono",
			Email: "budi@contoh.invalid",
			Phone: "021-999",
		},
		usecase.Actor{Login: "BudiHartono"}, nil)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if saved.LoginStatus != "Leader" {
		t.Fatalf("LoginStatus = %q, mau %q — nilai lama harus dipertahankan",
			saved.LoginStatus, "Leader")
	}
}

// TestSaveRejectsMissingRow membuktikan penyimpanan atas baris yang sudah tidak ada
// menghasilkan ErrNotFound, bukan diam-diam berhasil.
func TestSaveRejectsMissingRow(t *testing.T) {
	service, _ := layanan(t, tim())

	_, err := service.Save(context.Background(), portalUji, "TidakAda",
		masterlogin.Input{Name: "X", Email: "x@contoh.invalid", Phone: "1"},
		usecase.Actor{Login: "BudiHartono"}, nil)
	if !errors.Is(err, masterlogin.ErrNotFound) {
		t.Fatalf("Save = %v, mau ErrNotFound", err)
	}
}

// TestListReturnsEveryRow membuktikan daftar memuat SELURUH baris, bukan hanya satu tim.
//
// Cakupan itu adalah rekonstruksi — rule yang mengisi grid Pega tidak ada di export
// (`R-16`) — dan uji ini mengikatnya supaya perubahannya kelak menjadi keputusan yang
// terlihat, bukan pergeseran yang tidak disadari. Lihat masterlogin.Filter.
func TestListReturnsEveryRow(t *testing.T) {
	service, _ := layanan(t, tim())

	list, err := service.List(context.Background(), portalUji, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("baris = %d, mau 2", len(list))
	}
}

// TestListSearchesNameLoginAndEmail membuktikan ketiga kolom itu ikut dicari — dan Telp
// tidak.
//
// Kedua adapter repo harus sepakat soal ini; bila salah satu menambah kolom, uji yang lulus
// terhadap memori tidak lagi berlaku terhadap SQL.
func TestListSearchesNameLoginAndEmail(t *testing.T) {
	service, _ := layanan(t, tim())

	kasus := []struct {
		nama    string
		cari    string
		mauRows int
	}{
		{"menurut nama", "rina", 1},
		{"menurut login", "BudiHartono", 1},
		{"menurut email", "rina@contoh", 1},
		{"tidak menurut telp", "5550102", 0},
		{"tanpa penyaring", "", 2},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			list, err := service.List(context.Background(), portalUji, k.cari)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(list) != k.mauRows {
				t.Fatalf("cari %q = %d baris, mau %d", k.cari, len(list), k.mauRows)
			}
		})
	}
}

// TestUnknownPortalIsRejected membuktikan portal yang tidak dikenal DITOLAK, bukan jatuh ke
// portal utama.
//
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestUnknownPortalIsRejected(t *testing.T) {
	service, _ := layanan(t, tim())

	if _, err := service.List(context.Background(), "SIMASNET", ""); err == nil {
		t.Fatal("List pada portal tak dikenal berhasil, mau galat")
	}
	if err := service.EnsurePortalReady("SIMASNET"); err == nil {
		t.Fatal("EnsurePortalReady pada portal tak dikenal berhasil, mau galat")
	}
}
