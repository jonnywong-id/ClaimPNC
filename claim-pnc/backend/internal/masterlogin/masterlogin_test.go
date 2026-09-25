package masterlogin_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"claim-pnc/internal/masterlogin"
)

// TestDeriveLoginMatchesPegaExpression membuktikan penurunan LOGIN sama persis dengan
// ekspresi Pega.
//
// Rujukannya `Activity/SetLoginSurveyor_act-Act.xml`:
//
//	local.login := @replaceAll(@replaceAll(@replaceAll(@replaceAll(
//	                 TempLoginSurvey.SurveyName," ",""),".",""),",",""),"-","")
//
// Ia uji TERPENTING di modul ini: LOGIN adalah kunci baris, dan penurunan yang berbeda
// sedikit saja akan membuat setiap login yang diterbitkan sesudahnya tidak sebentuk dengan
// login lama yang sudah ada di tabel.
func TestDeriveLoginMatchesPegaExpression(t *testing.T) {
	kasus := []struct {
		nama  string
		input string
		mau   string
	}{
		{"spasi dibuang", "Budi Hartono", "BudiHartono"},
		{"dua spasi berurutan", "Rina  Ayu", "RinaAyu"},
		{"titik dibuang", "Budi.Hartono", "BudiHartono"},
		{"koma dibuang", "Hartono, Budi", "HartonoBudi"},
		{"tanda hubung dibuang", "Siti Nur-Halimah", "SitiNurHalimah"},
		{"keempatnya sekaligus", "A. B, C-D E", "ABCDE"},
		{"huruf besar-kecil TIDAK diubah", "bUdI hArToNo", "bUdIhArToNo"},
		{"spasi ujung dipangkas lebih dulu", "  Budi  ", "Budi"},
		{"angka dipertahankan", "Budi 2", "Budi2"},
		{"garis bawah TIDAK dibuang", "Budi_Hartono", "Budi_Hartono"},
		{"apostrof TIDAK dibuang", "O'Brien", "O'Brien"},
		{"kosong tetap kosong", "", ""},
		{"seluruhnya karakter yang dibuang", "- . , ", ""},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			if got := masterlogin.DeriveLogin(k.input); got != k.mau {
				t.Fatalf("DeriveLogin(%q) = %q, mau %q", k.input, got, k.mau)
			}
		})
	}
}

// TestDeriveLoginIsStable membuktikan penurunan bersifat idempoten.
//
// Ia bukan sekadar sifat matematis yang rapi: layar menghitung LOGIN untuk DITAMPILKAN
// sementara pengguna mengetik, dan server menghitungnya lagi untuk DISIMPAN. Bila
// menurunkan dari hasil penurunan menghasilkan nilai lain, kedua perhitungan itu dapat
// berbeda — dan yang terlihat pengguna bukan yang tersimpan.
func TestDeriveLoginIsStable(t *testing.T) {
	for _, nama := range []string{"Budi Hartono", "A. B, C-D", "  Rina  Ayu  ", "O'Brien"} {
		once := masterlogin.DeriveLogin(nama)
		twice := masterlogin.DeriveLogin(once)
		if once != twice {
			t.Fatalf("DeriveLogin tidak stabil untuk %q: %q lalu %q", nama, once, twice)
		}
	}
}

// TestCheckRejectsEmptyRequiredFields membuktikan ketiga isian wajib benar-benar ditolak
// saat kosong.
//
// Ketiganya dibaca dari `pyRequired = true` pada kontrolnya masing-masing di
// `Section/BrowseLoginSurveyor-Section.xml`.
func TestCheckRejectsEmptyRequiredFields(t *testing.T) {
	err := masterlogin.Input{}.Clean().Check()

	var validation *masterlogin.ValidationError
	if !asValidation(err, &validation) {
		t.Fatalf("Check() = %v, mau ValidationError", err)
	}

	mau := map[string]bool{"nama": false, "email": false, "telp": false}
	for _, p := range validation.Violation {
		if _, dikenal := mau[p.Field]; !dikenal {
			t.Fatalf("pelanggaran pada isian tak terduga: %q", p.Field)
		}
		mau[p.Field] = true
	}
	for isian, ada := range mau {
		if !ada {
			t.Errorf("isian %q tidak dilaporkan sebagai wajib", isian)
		}
	}
}

// TestCheckReportsEveryViolationAtOnce membuktikan SELURUH pelanggaran dikirim bersamaan,
// bukan yang pertama saja.
//
// `P-5` menuntutnya, dan alasannya bukan kerapian: layar menyorot isian satu per satu, dan
// itu hanya berguna bila seluruh pelanggaran sampai sekaligus. Mengembalikan satu galat per
// percobaan akan membuat pengguna menekan Simpan empat kali untuk mengetahui empat hal.
func TestCheckReportsEveryViolationAtOnce(t *testing.T) {
	err := masterlogin.Input{
		Address: strings.Repeat("x", masterlogin.MaxAddressLength+1),
	}.Clean().Check()

	var validation *masterlogin.ValidationError
	if !asValidation(err, &validation) {
		t.Fatalf("Check() = %v, mau ValidationError", err)
	}
	// nama, email, telp kosong + alamat kelewat panjang.
	if len(validation.Violation) != 4 {
		t.Fatalf("pelanggaran = %d, mau 4: %v", len(validation.Violation), validation.Violation)
	}
}

// TestCheckRejectsNameThatDerivesEmptyLogin membuktikan nama yang seluruhnya terdiri atas
// karakter yang dibuang DITOLAK.
//
// Padanan langkah kedua `Activity/CNMInsertMstLoginSurveyor_act`, yang menolak penyimpanan
// saat `TempLoginSurvey.SurveyorID == ""`. Tanpa uji ini, nama seperti "- . -" akan lolos
// sebagai nama yang terisi dan menghasilkan baris berkunci kosong — baris yang tidak dapat
// dibuka siapa pun.
func TestCheckRejectsNameThatDerivesEmptyLogin(t *testing.T) {
	err := masterlogin.Input{
		Name:  "- . , ",
		Email: "a@contoh.invalid",
		Phone: "021",
	}.Clean().Check()

	var validation *masterlogin.ValidationError
	if !asValidation(err, &validation) {
		t.Fatalf("Check() = %v, mau ValidationError", err)
	}
	if len(validation.Violation) != 1 || validation.Violation[0].Field != "nama" {
		t.Fatalf("pelanggaran = %v, mau satu pada isian nama", validation.Violation)
	}
}

// TestCheckAcceptsValidInput membuktikan isian yang sah benar-benar lolos — termasuk Alamat
// yang kosong, karena ia memang tidak wajib (`pyRequired = false`).
func TestCheckAcceptsValidInput(t *testing.T) {
	err := masterlogin.Input{
		Name:    "Budi Hartono",
		Email:   "budi@contoh.invalid",
		Phone:   "021-5550101",
		Address: "",
	}.Clean().Check()

	if err != nil {
		t.Fatalf("Check() = %v, mau nil", err)
	}
}

// TestCheckDoesNotValidateEmailShape membuktikan bentuk surel TIDAK diperiksa.
//
// Tidak ada satu pun rule di export yang memeriksanya. Menambahkan pemeriksaan bentuk
// berarti menolak alamat yang selama ini diterima sistem lama, dan itu selisih perilaku
// yang tidak diminta siapa pun (`P-5`).
//
// Uji ini menjaga ketiadaan itu tetap DISENGAJA: bila kelak seseorang menambahkan
// pemeriksaan bentuk, uji ini gagal dan ia harus memutuskannya sebagai perubahan, bukan
// menambahkannya sambil lalu.
func TestCheckDoesNotValidateEmailShape(t *testing.T) {
	err := masterlogin.Input{
		Name:  "Budi",
		Email: "bukan alamat surel",
		Phone: "021",
	}.Clean().Check()

	if err != nil {
		t.Fatalf("Check() = %v, mau nil — bentuk surel tidak diperiksa", err)
	}
}

// TestCleanTrimsEveryField membuktikan keempat isian dipangkas, bukan sebagian.
//
// Satu isian yang terlewat akan tersimpan berspasi ujung, dan pada Nama akibatnya berlanjut:
// LOGIN diturunkan darinya, dan perbandingan kunci memakai TRIM di satu sisi saja tidak
// selalu menolongnya.
func TestCleanTrimsEveryField(t *testing.T) {
	clean := masterlogin.Input{
		Name:    "  Budi  ",
		Email:   "  budi@contoh.invalid  ",
		Phone:   "  021  ",
		Address: "  Jl. Melati  ",
	}.Clean()

	mau := masterlogin.Input{
		Name:    "Budi",
		Email:   "budi@contoh.invalid",
		Phone:   "021",
		Address: "Jl. Melati",
	}
	if clean != mau {
		t.Fatalf("Clean() = %+v, mau %+v", clean, mau)
	}
}

// TestMaxNameLengthMatchesFrontendForm membuktikan batas panjang Nama di backend dan di
// layar tidak pernah berbeda.
//
// Angkanya sengaja diulang di dua tempat — server berwenang, layar memberi tahu lebih dulu —
// dan duplikasi itu hanya aman bila ada yang menjaganya. Tanpa uji ini, salah satu dapat
// berubah sendirian dan pengguna menerima penolakan atas isian yang layar nyatakan sah.
//
// Pola yang sama dipakai masterkategorisparepart.
func TestMaxNameLengthMatchesFrontendForm(t *testing.T) {
	berkas := filepath.Join("..", "..", "..", "frontend", "src", "modules",
		"master-login", "SurveyorLoginForm.tsx")

	isi, err := os.ReadFile(berkas)
	if err != nil {
		t.Skipf("berkas form tidak terbaca (%v); uji dilewati", err)
	}

	const penanda = "const MAX_NAME_LENGTH = "
	mulai := strings.Index(string(isi), penanda)
	if mulai < 0 {
		t.Fatalf("penanda %q tidak ditemukan di %s", penanda, berkas)
	}

	sisa := string(isi)[mulai+len(penanda):]
	akhir := strings.IndexAny(sisa, "\r\n")
	if akhir < 0 {
		akhir = len(sisa)
	}

	angka, err := strconv.Atoi(strings.TrimSpace(sisa[:akhir]))
	if err != nil {
		t.Fatalf("nilai MAX_NAME_LENGTH tidak terbaca sebagai angka: %v", err)
	}
	if angka != masterlogin.MaxNameLength {
		t.Fatalf("MAX_NAME_LENGTH di form = %d, masterlogin.MaxNameLength = %d",
			angka, masterlogin.MaxNameLength)
	}
}

// asValidation membungkus errors.As supaya maksud ujinya terbaca di tempat pemanggilan.
func asValidation(err error, target **masterlogin.ValidationError) bool {
	if err == nil {
		return false
	}
	one, ok := err.(*masterlogin.ValidationError)
	if !ok {
		return false
	}
	*target = one
	return true
}
