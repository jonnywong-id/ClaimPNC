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
	MaxOperatorIDLength   = 64
	MaxEmailLength        = 100
	MaxGroupLength        = 50
	MaxSupervisorLength   = 64
	MaxBusinessLineLength = 50
)

// MaxQuota menahan angka kuota yang tidak masuk akal.
//
// Ia bukan aturan bisnis yang ditemukan di export — tidak ada batas di sana — melainkan
// penjaga agar salah ketik tidak menghasilkan petugas berkuota jutaan yang menyedot
// seluruh antrean penugasan.
const MaxQuota = 9999

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

	// BusinessLine adalah TYPE_BUSINESS. Ia teks bebas, bukan pilihan tertutup:
	// satu-satunya nilai yang benar-benar muncul di export adalah "NONMBU", dan mengarang
	// daftar pilihan dari satu contoh akan menolak nilai sah yang belum terlihat.
	BusinessLine string

	// Group adalah TEAM_GROUP, kelompok kerja petugas.
	Group string

	// Supervisor adalah ATASAN — diisi OPERATOR_ID atasannya, bukan namanya.
	//
	// Di layar Pega ia autocomplete yang MENAMPILKAN `MCL_NAME` tetapi MENYIMPAN
	// `OPERATOR_ID` (`Section/BrowseUserTeknis-Section.xml:20642`). Di sini nilainya
	// diusulkan dari blok `EmpLeader` respons direktori, dan tetap dapat diubah petugas.
	Supervisor string

	// Quota adalah COUNTER_QUOTA, banyaknya pekerjaan yang boleh dipikul petugas ini.
	Quota int

	// ExternalQuota adalah COUNTER_QUOTA2.
	//
	// Namanya di sistem lama — alias "OLD_OPERATOR_ID" — menyesatkan: isinya ANGKA, bukan
	// identitas. Procedure lamanya pun menerimanya sebagai `TJOB2 number`.
	//
	// `SetTotalJobMstUserTeknis` mengisinya dengan `sum(total_job)` dari sistem luar lewat
	// DB link (`new_general.m_user_job@opjava.sinarmas.co.id`), yaitu beban kerja petugas
	// yang sama di aplikasi lain. DB link itu TIDAK dibawa ke sini — `ADR-0008` menetapkan
	// DB link diganti API, dan API-nya belum ada.
	//
	// Sampai API itu ada ia dikelola sebagai isian biasa, dan itu bukan penyimpangan:
	// layar Pega pun menandainya `pyReadOnly=false`
	// (`Section/BrowseUserTeknis-Section.xml:22351`).
	ExternalQuota int

	// PanelGroup adalah GROUPPANEL, dialias "IBNR" pada `GetMasterPICTeknis`.
	//
	// HANYA DIBACA. Procedure penulis `PEGA_MST_USER_TEKNIS` tidak pernah menulis kolom
	// ini — baik pada cabang INSERT maupun UPDATE — dan form Pega tidak memuatnya.
	// Menjadikannya dapat diubah di sini berarti menambah perilaku yang tidak pernah ada.
	PanelGroup string

	// Workload adalah TOTAL_JOB, beban pekerjaan petugas yang sebenarnya — berbeda dari
	// Quota yang menyatakan batas yang BOLEH dipikul.
	//
	// HANYA DIBACA, dan hanya terisi pada daftar: ia kolom milik view
	// POOLDATA.V_MST_USER_TEKNIS dan tidak ada di tabelnya. Layar Pega pun menandainya
	// `pyReadOnly=true` (`Section/BrowseUserTeknis-Section.xml:19414`), dan
	// `GetMasterPICTeknis` yang mengisi form tidak menyertakannya sama sekali.
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

	// Surel wajib. Perbedaan yang DISENGAJA dari sistem lama, yang menerima kosong:
	// petugas tanpa surel tidak dapat menerima satu pun pemberitahuan penugasan, dan
	// ketiadaannya baru ketahuan saat pemberitahuan gagal terkirim — jauh dari layar ini.
	if t.Email == "" {
		add(FieldEmail, "Email wajib diisi.")
	} else if !EmailPlausible(t.Email) {
		add(FieldEmail, "Format email tidak benar.")
	} else if utf8.RuneCountInString(t.Email) > MaxEmailLength {
		add(FieldEmail, "Email paling panjang "+strconv.Itoa(MaxEmailLength)+" karakter.")
	}

	if utf8.RuneCountInString(t.BusinessLine) > MaxBusinessLineLength {
		add(FieldBusinessLine, "Lini bisnis paling panjang "+strconv.Itoa(MaxBusinessLineLength)+" karakter.")
	}
	if utf8.RuneCountInString(t.Group) > MaxGroupLength {
		add(FieldGroup, "Grup paling panjang "+strconv.Itoa(MaxGroupLength)+" karakter.")
	}
	if utf8.RuneCountInString(t.Supervisor) > MaxSupervisorLength {
		add(FieldSupervisor, "Atasan paling panjang "+strconv.Itoa(MaxSupervisorLength)+" karakter.")
	}

	if t.Quota < 0 || t.Quota > MaxQuota {
		add(FieldQuota, "Kuota harus antara 0 dan "+strconv.Itoa(MaxQuota)+".")
	}
	if t.ExternalQuota < 0 || t.ExternalQuota > MaxQuota {
		add(FieldExternalQuota, "Kuota sistem lain harus antara 0 dan "+strconv.Itoa(MaxQuota)+".")
	}

	// Petugas tidak boleh menjadi atasan dirinya sendiri. Bukan aturan yang tertulis di
	// export, melainkan akibat langsung dari cara penugasan menelusuri rantai atasan:
	// rujukan ke diri sendiri membuatnya berputar tanpa henti.
	if t.Supervisor != "" && IDKey(t.Supervisor) == IDKey(t.OperatorID) {
		add(FieldSupervisor, "Petugas tidak boleh menjadi atasan dirinya sendiri.")
	}

	return violation
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
