// Package masterpicteknik adalah inti modul Master PIC Teknik (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Daftar petugas teknik yang menangani klaim: siapa mereka, di grup mana, siapa
// atasannya, berapa kuota pekerjaan yang boleh dipikulnya, dan apakah ia masih aktif.
// Isi master inilah yang dipakai penugasan klaim — `SearchPICTeknik_act` memutari daftar
// ini untuk memilih PIC berikutnya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega dan dari procedure yang berjalan, bukan
// dikarang:
//
//	Harness/UserTeknisInbox-Harness.xml            layar "Master PIC Teknik"; tombol
//	                                               Tambah & Refresh, tanpa Hapus
//	Section/BrowseUserTeknis-Section.xml           grid dan form; TOTAL_JOB read-only
//	Section/ListUserTeknis-Section.xml             kerangka layar
//	Report Definition/BrowseVMstUserTeknis_RD      10 kolom; STS_AKTIF='1' dipatok
//	Activity/SetMstUserTeknisMstUser_act-Act.xml   pencarian nama ke direktori
//	Activity/SetMstUserTeknisValue_act-Act.xml     aksi Ubah: salin ke TempDcol
//	Activity/CNMInsertMstUserTeknis_act-Act.xml    aksi Simpan; tolak bila nama kosong
//	RDB List/GetMasterPICTeknis-SQL.xml            ambil satu baris, TANPA filter aktif
//	RDB List/UpdateMasterUserTeknis-SQL.xml        memanggil POOLDATA.PEGA_MST_USER_TEKNIS
//	Database/PEGA_MST_USER_TEKNIS.prc              source procedure-nya
//
// # Kunci alaminya diberikan, bukan dibuat
//
// Berbeda dari Master Status Klaim dan Master Tipe Surveyors yang kodenya diterbitkan
// urutan, kunci di sini adalah `OPERATOR_ID` — identitas petugas di direktori pegawai.
// Ia DIISI pengguna dan tidak pernah dibuat sistem.
//
// Procedure lama `PEGA_MST_USER_TEKNIS` sempat menghitung
// `id_site || lpad(MST_USER_TEKNIS_SEQ.nextval, 6, '0')` ke dalam variabel
// `id_mst_user_teknis` — lalu **tidak pernah memakainya**. INSERT-nya tetap memakai
// `IDPega` sebagai `OPERATOR_ID`. Generator itu kode mati, dan tidak dibawa ke sini.
//
// # Nama dan atasan tidak diketik, melainkan dicari
//
// `MCL_NAME` tidak pernah diisi tangan. `SetMstUserTeknisMstUser_act` mencarinya lebih
// dulu ke direktori pegawai, dan `CNMInsertMstUserTeknis_act` langkah 2 menolak simpan
// bila hasilnya kosong — preconditionnya `TempDcol.MCL_NAME==""`, dengan keterangan
// langkah "set error kalau tidak ditemukan di service". Petugas yang tidak terdaftar
// **ditolak**, dan aturan itu dipertahankan.
//
// Atasan pun demikian: respons direktori memuat blok `EmpLeader`, sehingga satu
// pencarian mengembalikan nama petugas SEKALIGUS atasannya.
//
// # Penamaan
//
// Alias sistem lama menyesatkan dan tidak dibawa masuk:
//
//	COUNTER_QUOTA2 → alias "OLD_OPERATOR_ID"   padahal sebuah ANGKA, bukan id operator
//	GROUPPANEL     → alias "IBNR"              padahal nama grup panel
//	OPERATOR_ID    → alias "MCL_Name"          pada BrowseEmailUserTeknis
//
// Pemetaan alias→kolom→domain lengkap ada di repo/sqlstore/technician.sql.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpicteknik

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Batas panjang isian.
//
// Seluruhnya KEBUTUHAN BARU: layar Pega tidak membatasi panjang sama sekali
// (`pyMaxLength` kosong pada setiap isian di `Section/BrowseUserTeknis-Section.xml`).
// Angkanya mengikuti lebar kolom yang wajar dan diselaraskan dengan master lain yang
// sudah dibangun, supaya batas yang dilihat pengguna seragam antarlayar.
//
// Angka yang sama diulang di frontend supaya pengguna tahu sebelum mengirim; server tetap
// yang berwenang. Bila salah satu berubah, KEDUANYA wajib ikut berubah.
const (
	MaxOperatorIDLength = 64
	MaxEmailLength      = 100
	MaxSupervisorLength = 64
)

// MaxClaimCounter menahan angka pencacah klaim yang tidak masuk akal.
//
// Ia bukan aturan bisnis yang ditemukan di export — tidak ada batas di sana — melainkan
// penjaga agar salah ketik tidak menghasilkan pencacah berisi jutaan, yang akan membuat
// petugas itu tidak pernah terpilih lagi oleh `ORDER BY counter_quota ASC`.
const MaxClaimCounter = 9999

// GroupCodes adalah seluruh nilai sah kolom TEAM_GROUP, berlabel "Kelompok".
//
// Bukan tebakan dari data: dibaca dari `Property/TEAM_GROUP_property.xml`, yang
// mendaftarkannya sebagai Local List ber-`pyStandardValue` A, B, dan C. Itulah yang
// mengisi dropdown di layar Pega.
var GroupCodes = []string{"A", "B", "C"}

// BusinessCodes adalah seluruh nilai sah kolom TYPE_BUSINESS, berlabel "Bisnis".
//
// Dibaca dari `Property/TYPE_BUSINESS_property.xml`. Urutannya sengaja mengikuti urutan
// `pyStandardValue` di berkas itu, bukan diurutkan ulang menurut abjad — dropdown-nya
// tampil dalam urutan yang sama dengan layar lama.
var BusinessCodes = []string{"NONMBU", "TRAVEL", "PA", "BONDING"}

// ATASAN sengaja TIDAK punya daftar nilai.
//
// `Property/ATASAN_property.xml` adalah properti Text biasa tanpa `pyStandardValue`; yang
// membatasi isiannya di layar adalah autocomplete atas daftar operator
// (`Section/BrowseUserTeknis-Section.xml:20650` menampilkan `.MCL_NAME`, menyimpan
// `.OPERATOR_ID`). Karena itu isinya divalidasi sebagai teks, bukan sebagai pilihan.

// ActiveCode adalah isi kolom STS_AKTIF untuk petugas yang aktif.
//
// Nilainya "1", BUKAN "Ya"/"Tidak" seperti STS_AKTIF pada POOLDATA.LST_ACCOUNT. Dua tabel
// berbeda memakai sandi berbeda untuk kolom bernama sama, dan itu justru alasan nilainya
// ditulis sebagai konstanta bernama di sini: `STS_AKTIF = '1'` muncul 17 kali di seluruh
// export, tidak satu pun memakai "Ya".
const ActiveCode = "1"

// InactiveCode adalah isi kolom STS_AKTIF untuk petugas yang tidak lagi menerima
// penugasan.
const InactiveCode = "0"

// Technician adalah satu baris master petugas teknik.
type Technician struct {
	// OperatorID adalah OPERATOR_ID — kunci alaminya, sekaligus identitas petugas di
	// direktori pegawai. Tetap seumur hidup baris ini.
	OperatorID string

	// Name adalah MCL_NAME. Ia DITURUNKAN dari direktori pegawai, bukan diketik.
	Name string

	// Email adalah alamat surel petugas. Dipakai seluruh pemberitahuan yang ditujukan
	// kepadanya.
	//
	// Ia DIKETIK, bukan diturunkan — `SetMstUserTeknisMstUser_act` langkah 9 hanya
	// menyalin nama dan id, tidak menyentuh surel.
	Email string

	// BusinessLine adalah TYPE_BUSINESS, berlabel **"Bisnis"** di layar.
	//
	// Isinya teks bebas di sisi penyimpanan — layar Pega menyajikannya sebagai dropdown,
	// tetapi daftar pilihannya tidak ada di export. Nilai yang benar-benar terlihat:
	// "NONMBU" dan "BONDING".
	BusinessLine string

	// Group adalah TEAM_GROUP, berlabel **"Kelompok"** di layar.
	//
	// Nilai yang benar-benar terlihat berupa huruf tunggal: "A", "B", "C".
	Group string

	// Supervisor adalah ATASAN — diisi OPERATOR_ID atasannya, bukan namanya.
	//
	// Di layar Pega ia autocomplete yang MENAMPILKAN `MCL_NAME` tetapi MENYIMPAN
	// `OPERATOR_ID` (`Section/BrowseUserTeknis-Section.xml:20642`). Di sini nilainya
	// diusulkan dari blok `EmpLeader` respons direktori, dan tetap dapat diubah petugas.
	Supervisor string

	// ClaimCounterBelow1M adalah COUNTER_QUOTA, berlabel **"Counter Klaim <1M"**.
	//
	// # Ini PENCACAH, bukan kuota
	//
	// Nama kolomnya menyesatkan dan sempat menyesatkan saya juga. Labelnya di layar
	// (`Section/BrowseUserTeknis-Section.xml:21683`) menyebutnya pencacah klaim, dan
	// pemakaiannya membenarkan itu: `RDB List/BrowsePICRandomTeam-SQL.xml:39-40` memilih
	// petugas dengan `ORDER BY counter_quota ASC` — yang paling sedikit menangani klaim —
	// lalu `AddTJobCounterPIC_SQL` menaikkannya satu. Tidak ada satu pun tempat yang
	// membandingkannya sebagai batas.
	//
	// Pemisahan "<1M" dan ">1M" adalah nilai klaim di bawah dan di atas **Rp 1 Miliar**,
	// ambang yang sama dengan Notice of Large Losses.
	ClaimCounterBelow1M int

	// ClaimCounterAbove1M adalah COUNTER_QUOTA2, berlabel **"Counter Klaim >1M"**.
	//
	// Aliasnya di sistem lama — "OLD_OPERATOR_ID" — menyesatkan dua kali: ia bukan
	// identitas (procedure menerimanya sebagai `TJOB2 number`), dan ia bukan "kuota sistem
	// lain" seperti yang sempat saya tulis. Ia pencacah klaim bernilai di atas Rp 1 Miliar.
	//
	// `SetTotalJobMstUserTeknis` pernah mengisinya dari sistem luar lewat DB link
	// (`new_general.m_user_job@opjava.sinarmas.co.id`); DB link itu TIDAK dibawa ke sini
	// (`ADR-0008`). Layar Pega menandainya `pyReadOnly=false`
	// (`Section/BrowseUserTeknis-Section.xml:22351`), jadi ia tetap dapat diisi petugas.
	ClaimCounterAbove1M int

	// PanelGroup adalah GROUPPANEL, dialias "IBNR" pada `GetMasterPICTeknis`.
	//
	// HANYA DIBACA. Procedure penulis `PEGA_MST_USER_TEKNIS` tidak pernah menulis kolom
	// ini — baik pada cabang INSERT maupun UPDATE — dan form Pega tidak memuatnya.
	// Menjadikannya dapat diubah di sini berarti menambah perilaku yang tidak pernah ada.
	PanelGroup string

	// Workload adalah TOTAL_JOB, kolom milik view POOLDATA.V_MST_USER_TEKNIS yang tidak
	// ada di tabelnya.
	//
	// **TIDAK DITAMPILKAN DI LAYAR MANA PUN**, dan itu mengikuti Pega: grid-nya memuat
	// tujuh kolom dan TOTAL_JOB bukan salah satunya, sementara form-nya pun tidak
	// memuatnya. Ia ikut dibaca karena Report Definition lama memang menyebutnya dan
	// karena mengambilnya tidak berbiaya — bukan karena ada yang memakainya.
	//
	// Jangan menampilkannya tanpa keputusan Work Owner: menambahkan kolom yang tidak ada
	// di sistem lama adalah persis yang membuat layar ini sempat berbeda dari aslinya.
	Workload int

	// Active menyatakan petugas masih menerima penugasan.
	Active bool
}

// Clean mengembalikan salinan dengan spasi tepi dibuang.
func (t Technician) Clean() Technician {
	t.OperatorID = strings.TrimSpace(t.OperatorID)
	t.Name = strings.TrimSpace(t.Name)
	t.Email = strings.TrimSpace(t.Email)
	t.BusinessLine = strings.TrimSpace(t.BusinessLine)
	t.Group = strings.TrimSpace(t.Group)
	t.Supervisor = strings.TrimSpace(t.Supervisor)
	t.PanelGroup = strings.TrimSpace(t.PanelGroup)
	return t
}

// IDKey adalah bentuk OperatorID yang dipakai membandingkan dan mencari.
//
// Perbandingan mengabaikan besar-kecil huruf, mengikuti kueri pencarian nama lamanya yang
// memang menulis `upper(pyuseridentifier) = upper(:1)`.
//
// Ini sekaligus PERBAIKAN yang disengaja terhadap ketidakkonsistenan sistem lama:
// `PEGA_MST_USER_TEKNIS` memeriksa keberadaan baris dengan `operator_id = IDPega` tanpa
// UPPER, sementara pencarian namanya memakai UPPER di kedua sisi. Akibatnya "BUDI" dan
// "budi" dapat menjadi DUA baris di master yang sama, padahal keduanya orang yang sama
// bagi direktori. Di sini keduanya satu identitas.
func IDKey(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

// Employee adalah jawaban direktori pegawai atas satu identitas.
//
// Ia sengaja memuat atasan: respons direktori memuat blok `EmpLeader`, sehingga satu
// pencarian menjawab dua pertanyaan sekaligus dan form tidak perlu menembak dua kali.
type Employee struct {
	// OperatorID adalah identitas sebagaimana dikenal direktori. Dipakai apa adanya,
	// bukan yang diketik pengguna, supaya besar-kecil hurufnya seragam dengan sumbernya.
	OperatorID string

	Name  string
	Email string

	// SupervisorID dan SupervisorName datang dari blok EmpLeader. Keduanya dapat KOSONG
	// — pucuk pimpinan tidak punya atasan, dan tidak setiap respons mengisinya.
	SupervisorID   string
	SupervisorName string
}

// Repo adalah seam ke penyimpanan master PIC teknik SATU portal.
//
// Satu instans Repo selalu terikat pada satu basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat penyaringan baris (`ADR-0030` Opsi 1). Tidak
// ada satu pun kueri di pengisinya yang menyaring berdasarkan entitas, dan memang tidak
// boleh ada.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Antarmukanya berbicara dalam istilah domain — List, Get, Insert,
// Update — bukan istilah SQL.
type Repo interface {
	// List mengembalikan petugas yang AKTIF saja, terurut menurut OperatorID.
	//
	// Penyaringan "aktif saja" ada di sini, bukan di usecase, karena ia bagian dari
	// kueri: Report Definition lama pun mematoknya sebagai penyaring tetap
	// (`STS_AKTIF = '1'`), bukan sebagai pilihan pemanggil.
	List(ctx context.Context) ([]Technician, error)

	// Get mengembalikan satu petugas, aktif maupun tidak, atau ErrNotFound.
	//
	// TIDAK menyaring status aktif — `GetMasterPICTeknis` pun tidak. Itulah yang membuat
	// petugas yang telanjur dinonaktifkan masih dapat dibuka dan diaktifkan kembali,
	// meski ia tidak lagi muncul di daftar.
	Get(ctx context.Context, operatorID string) (Technician, error)

	// Insert menyimpan petugas baru. ErrAlreadyExists bila OperatorID-nya sudah dipakai.
	Insert(ctx context.Context, t Technician) (Technician, error)

	// Update mengubah petugas yang sudah ada. OperatorID, PanelGroup, dan Workload tidak
	// ikut berubah. ErrNotFound bila tidak ada.
	Update(ctx context.Context, t Technician) (Technician, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// EmployeeDirectory adalah seam ke direktori pegawai tempat nama petugas dicari.
//
// # Kenapa bukan DATAPEGA.PR_OPERATORS
//
// Sistem lama mencari dua tingkat: `SelectMstUserTeknisMclName` ke tabel operator milik
// Pega lebih dulu, lalu — bila kosong — layanan REST lewat `ConnectRestPNC_act`.
//
// Keputusan Work Owner 2026-09-19 meninggalkan tabel operator Pega seluruhnya dan memakai
// layanan REST saja. Alasannya lurus dengan arah migrasi: `DATAPEGA.PR_OPERATORS` adalah
// tabel milik engine Pega, dan bergantung padanya berarti modul ini ikut mati ketika Pega
// dimatikan.
//
// portalAlias ikut karena alamat layanannya dibaca per entitas dari
// POOLDATA.GCNM_CONNECT_REST — satu aplikasi melayani empat badan hukum, dan masing-masing
// boleh punya endpoint sendiri (`ADR-0030`).
type EmployeeDirectory interface {
	// Lookup mengembalikan data pegawai. ErrEmployeeUnknown bila identitas itu tidak
	// terdaftar; ErrDirectoryUnreachable bila direktorinya tidak dapat dihubungi.
	Lookup(ctx context.Context, portalAlias, operatorID string) (Employee, error)
}

// Check mengumpulkan SELURUH pelanggaran aturan sekaligus, bukan berhenti pada yang
// pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama menampilkan
// seluruh pesan validasi sekaligus, dan mengembalikannya satu per satu akan membuat
// pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// Keberadaan petugas di direktori TIDAK diperiksa di sini — ia menuntut memanggil seam,
// sedangkan fungsi ini murni dan dapat diuji tanpa apa pun. Pemeriksaannya ada di usecase.
func Check(t Technician) []Violation {
	t = t.Clean()
	var violation []Violation

	add := func(field, message string) {
		violation = append(violation, Violation{Field: field, Message: message})
	}

	if t.OperatorID == "" {
		add(FieldOperatorID, "ID operator wajib diisi.")
	} else if utf8.RuneCountInString(t.OperatorID) > MaxOperatorIDLength {
		add(FieldOperatorID, "ID operator paling panjang "+strconv.Itoa(MaxOperatorIDLength)+" karakter.")
	}

	// Surel TIDAK wajib, dan itu koreksi.
	//
	// Sempat diwajibkan di sini sebagai "perbedaan yang disengaja". Itu keliru:
	// `Section/BrowseUserTeknis-Section.xml` tidak memuat satu pun `pyRequired=true` —
	// nol isian wajib di seluruh layar — dan kolomnya NULLABLE. Mewajibkannya membuat
	// baris lama yang surelnya kosong TIDAK DAPAT disunting sama sekali, karena petugas
	// dipaksa mengarang surel hanya untuk mengubah kolom lain.
	//
	// Formatnya tetap diperiksa bila diisi; yang dicabut hanya kewajiban mengisinya.
	if t.Email != "" && !EmailPlausible(t.Email) {
		add(FieldEmail, "Format email tidak benar.")
	} else if utf8.RuneCountInString(t.Email) > MaxEmailLength {
		add(FieldEmail, "Email paling panjang "+strconv.Itoa(MaxEmailLength)+" karakter.")
	}

	// Kelompok dan Bisnis adalah PILIHAN, bukan teks bebas — keduanya Local List pada
	// Property rule-nya. Pemeriksaannya di sini, bukan hanya di layar, karena pemanggilan
	// langsung ke API tidak melewati dropdown mana pun.
	//
	// Kosong tetap diterima: layar Pega menyajikan pilihan `--Pilih--`, dan kolomnya pun
	// NULLABLE.
	if t.Group != "" && !oneOf(t.Group, GroupCodes) {
		add(FieldGroup, "Kelompok harus salah satu dari "+strings.Join(GroupCodes, ", ")+".")
	}
	if t.BusinessLine != "" && !oneOf(t.BusinessLine, BusinessCodes) {
		add(FieldBusinessLine, "Bisnis harus salah satu dari "+strings.Join(BusinessCodes, ", ")+".")
	}
	if utf8.RuneCountInString(t.Supervisor) > MaxSupervisorLength {
		add(FieldSupervisor, "Atasan paling panjang "+strconv.Itoa(MaxSupervisorLength)+" karakter.")
	}

	if t.ClaimCounterBelow1M < 0 || t.ClaimCounterBelow1M > MaxClaimCounter {
		add(FieldClaimCounterBelow1M, "Counter Klaim <1M harus antara 0 dan "+strconv.Itoa(MaxClaimCounter)+".")
	}
	if t.ClaimCounterAbove1M < 0 || t.ClaimCounterAbove1M > MaxClaimCounter {
		add(FieldClaimCounterAbove1M, "Counter Klaim >1M harus antara 0 dan "+strconv.Itoa(MaxClaimCounter)+".")
	}

	// Petugas tidak boleh menjadi atasan dirinya sendiri. Bukan aturan yang tertulis di
	// export, melainkan akibat langsung dari cara penugasan menelusuri rantai atasan:
	// rujukan ke diri sendiri membuatnya berputar tanpa henti.
	if t.Supervisor != "" && IDKey(t.Supervisor) == IDKey(t.OperatorID) {
		add(FieldSupervisor, "Petugas tidak boleh menjadi atasan dirinya sendiri.")
	}

	return violation
}

// oneOf memeriksa keanggotaan dengan mengabaikan besar-kecil huruf.
//
// Mengabaikannya disengaja: data lama dapat menyimpan "nonmbu" sementara Property rule
// menulis "NONMBU", dan menolak baris yang sebenarnya sah hanya karena hurufnya berbeda
// akan membuat petugas tidak dapat menyunting barisnya sendiri.
func oneOf(value string, allowed []string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	for _, candidate := range allowed {
		if strings.ToUpper(candidate) == value {
			return true
		}
	}
	return false
}

// EmailPlausible memeriksa bentuk alamat surel sekadarnya.
//
// Sengaja longgar: satu-satunya cara membuktikan sebuah alamat benar adalah mengirim surel
// ke sana, dan validasi yang terlalu ketat justru menolak alamat yang sah.
func EmailPlausible(address string) bool {
	address = strings.TrimSpace(address)
	at := strings.IndexByte(address, '@')
	if at <= 0 || at == len(address)-1 {
		return false
	}
	domain := address[at+1:]
	if strings.ContainsRune(domain, '@') {
		return false
	}
	dot := strings.IndexByte(domain, '.')
	return dot > 0 && dot < len(domain)-1
}
