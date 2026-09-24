// Package masterpenolakan adalah inti modul Master Penolakan Klaim.
//
// # Apa yang dimodelkan di sini
//
// Ketika sebuah klaim ditolak, alasannya diambil dari daftar baku — bukan diketik bebas
// per klaim. Daftar itulah yang dikelola modul ini, dan ia berjenjang dua tingkat:
//
//	Status Penolakan 1  POOLDATA.MST_PENOLAKAN_KLAIM_1  ID_ST (kunci), NOTE_ST
//	  └─ Status Penolakan 2  POOLDATA.MST_PENOLAKAN_KLAIM_2  ID_ND (kunci), ID_ST (induk)
//
// Berbeda dari Master Status Progres yang dua tingkatnya punya layar sendiri-sendiri, di
// sini keduanya dikelola SATU layar: tingkat 1 tidak pernah ditambahkan tersendiri
// melainkan selalu bersama tingkat 2 yang menaunginya. Itu sebabnya keduanya memakai satu
// seam Repo, bukan dua — penambahannya memang satu operasi yang tidak dapat dipecah.
//
// Layar yang sama juga mengelola master kedua yang tabelnya sama sekali berbeda —
// POOLDATA.MST_REJECTED_KOMITE. Ia ada di komite.go.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/PNC_MasterTolakKlaim-Harness.xml          layar "Master Penolakan Klaim"
//	Section/MasterRejectNoteKlaim-Section.xml          pembungkus portal
//	Section/BrowseNoteRejectClaim-Section.xml          isi sebenarnya — dua tab, grid, form
//	RDB List/BrowseStatusPenolakanKlaim2-SQL.xml       daftar, ORDER BY ID_ST ASC
//	RDB List/UpdateStatusPenolakanKlaim2-SQL.xml       ambil satu baris (namanya "Update",
//	                                                   isinya SELECT)
//	RDB List/SaveDataPenolakanKlaimMaster-SQL.xml      panggil MasterPenolakanKlaim1
//	RDB List/SaveDataPenolakanKlaimMaster2-SQL.xml     panggil MasterPenolakanKlaim2
//	Database/MASTERPENOLAKANKLAIM1.prc                 sisip tingkat 1
//	Database/MASTERPENOLAKANKLAIM2.prc                 sisip & perbarui tingkat 2
//	Activity/InsertMasterPenolakanNoteKlaim-Act.xml    urutan langkah simpan
//	Activity/BrowseStatusPenolakanKlaim_2-Act.xml      urutan langkah daftar
//	Activity/SetStatusMasterRejectsKlaim-Act.xml       pemilihan tab (FlgMasterPenolakan.FlagASO)
//
// # Penamaan ulang yang disengaja (D-19)
//
// Kueri lama mengaliaskan kedelapan kolomnya ke nama yang tidak mencerminkan isi sama
// sekali — utang teknis §4.2 `03-CURRENT-ARCHITECTURE.md`. Alias itu TIDAK dibawa:
//
//	ID_ST        AS "CaseID"          -> ParentID      (bukan nomor klaim)
//	NOTE_ST      AS "City"            -> ParentName    (bukan nama kota)
//	NOTE_ND      AS "CityID"          -> Name          (bukan kode kota)
//	ID_ND        AS "District"        -> ID            (bukan kabupaten)
//	STATUS       AS "DistrictID"      -> Status        (bukan kode kabupaten)
//	USER_INPUT   AS "UserTeknis"      -> SubmittedBy   (bukan PIC Teknik)
//	NOTEAPPROVED AS "NoteKasir"       -> ApprovalNote  (tidak ada urusan dengan kasir)
//	<derivasi>   AS "AnaylstRemarks"  -> Status.Label() (bukan catatan analis; salah eja pula)
//
// Perhatikan `City`/`CityID` di sini TIDAK berpasangan seperti dugaan yang wajar, persis
// seperti pada Master Status Progres 2: `City` adalah nama INDUK sedangkan `CityID` adalah
// nama baris ini sendiri. Itu sebabnya alias lama tidak dipakai sebagai petunjuk arti di
// mana pun pada modul ini — yang dipetakan adalah KOLOMNYA.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpenolakan

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ApprovalStatus adalah posisi sebuah Status Penolakan 2 dalam alur persetujuan.
//
// Nilainya sengaja tetap "0", "1", "2" seperti di POOLDATA.MST_PENOLAKAN_KLAIM_2:
// tabelnya masih dibaca dan ditulis sistem lama selama masa paralel (ADR-0004), sehingga
// mengubah sandi nilainya akan membuat kedua sistem membaca baris yang sama secara
// berbeda.
//
// Yang MENYETUJUI bukan layar ini. Persetujuan dikerjakan layar Inbox Manager —
// `Section/Sec_PenolakanKlaimChecker-Section.xml` yang dipakai
// `Harness/UserInbox_Harness-Harness.xml` (MENU_ID 58). Di layar ini ketiganya baca-saja
// (keputusan Work Owner 2026-09-19).
type ApprovalStatus string

const (
	// StatusPending — sudah diajukan, checker belum memutuskan. Nilai awal setiap baris.
	StatusPending ApprovalStatus = "0"
	// StatusApproved — checker menyetujui.
	StatusApproved ApprovalStatus = "1"
	// StatusRejected — checker menolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status sebagaimana dibaca pengguna.
//
// Ketiga teksnya disalin APA ADANYA dari derivasi kueri lama
// (`RDB List/BrowseStatusPenolakanKlaim2-SQL.xml`):
//
//	case when A.STATUS='1' then 'APPROVED' when A.STATUS='2' then 'REJECTED' ELSE 'MENUNGGU' end
//
// Termasuk huruf besarnya. `D-13` menetapkan teks yang dilihat pengguna mengikuti layar
// Pega supaya petugas tidak perlu belajar ulang; menerjemahkannya menjadi "Disetujui"
// akan membuat kolom yang sama terbaca berbeda di dua layar selama masa paralel.
func (s ApprovalStatus) Label() string {
	switch s {
	case StatusApproved:
		return "APPROVED"
	case StatusRejected:
		return "REJECTED"
	default:
		// Kueri lama pun memakai ELSE, bukan WHEN '0' — baris ber-STATUS kosong, NULL,
		// atau bernilai lain apa pun ikut terbaca MENUNGGU. Ditiru apa adanya supaya
		// baris warisan yang kolomnya tidak terisi tidak menghasilkan sel kosong yang
		// tampak seperti cacat layar.
		return "MENUNGGU"
	}
}

// Known menyatakan status ini termasuk salah satu dari tiga yang sah.
//
// Dipakai penyimpanan, bukan oleh Label: Label sengaja memaafkan nilai tak dikenal
// sebagaimana kueri lama, sedangkan penulisan tidak boleh ikut memaafkannya.
func (s ApprovalStatus) Known() bool {
	return s == StatusPending || s == StatusApproved || s == StatusRejected
}

// RejectionStatus adalah satu baris Status Penolakan 1 — POOLDATA.MST_PENOLAKAN_KLAIM_1.
//
// Hanya dua kolom, dan itu memang seluruh isi tabelnya menurut
// `Database/MASTERPENOLAKANKLAIM1.prc:9`.
type RejectionStatus struct {
	// ID adalah kunci baris, kolom ID_ST.
	ID string

	// Name adalah keterangan yang dibaca petugas, kolom NOTE_ST.
	Name string
}

// RejectionStatus2 adalah satu baris Status Penolakan 2 —
// POOLDATA.MST_PENOLAKAN_KLAIM_2.
type RejectionStatus2 struct {
	// ID adalah kunci baris, kolom ID_ND.
	ID string

	// Name adalah keterangan tingkat 2 yang dibaca petugas, kolom NOTE_ND.
	Name string

	// ParentID merujuk Status Penolakan 1, kolom ID_ST.
	ParentID string

	// ParentName adalah SALINAN nama induk pada saat baris ini disimpan, kolom NOTE_ST.
	//
	// Ia salinan, bukan hasil join — `Database/MASTERPENOLAKANKLAIM2.prc:9` memang
	// menuliskan kedua kolom sekaligus. Denormalisasi itu dipertahankan karena tabelnya
	// masih dibaca sistem lama, yang mengambil teksnya dari kolom ini dan bukan dengan
	// menelusuri induknya.
	ParentName string

	// Status adalah kolom STATUS. Diisi penyimpanan, tidak pernah dari layar ini.
	Status ApprovalStatus

	// SubmittedBy adalah kolom USER_INPUT — siapa yang mengajukan.
	//
	// Di sistem lama ia `OperatorID.pyUserIdentifier`
	// (`Activity/InsertMasterPenolakanNoteKlaim-Act.xml`), yaitu login operator Pega.
	// Padanannya di sini adalah login pemanggil, bukan NIK-nya: kolomnya sudah berisi
	// login pada baris-baris lama.
	SubmittedBy string

	// SubmittedAt adalah kolom TANGGALKIRIM.
	SubmittedAt time.Time

	// ApprovedBy adalah kolom APPROVEBY — diisi layar Inbox Manager, baca-saja di sini.
	ApprovedBy string

	// ApprovedAt adalah kolom TANGGAL_APPROVE; nil bila belum pernah diputuskan.
	ApprovedAt *time.Time

	// ApprovalNote adalah kolom NOTEAPPROVED — catatan checker, baca-saja di sini.
	ApprovalNote string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// # Kenapa induknya punya DUA isian, padahal layar lama hanya punya satu
//
// Layar lama memuat dua kotak teks bebas: "Status Penolakan 1" dan "Status Penolakan 2"
// (`Section/BrowseNoteRejectClaim-Section.xml`, keduanya `pxTextArea`). Isi kotak pertama
// selalu diteruskan ke `MasterPenolakanKlaim1` — dan procedure itu **menyisipkan baris
// baru pada kedua cabang IF-nya**, tidak pernah memperbarui:
//
//	Database/MASTERPENOLAKANKLAIM1.prc:9   INSERT ... VALUES (id_mst, tnotest)
//	Database/MASTERPENOLAKANKLAIM1.prc:14  INSERT ... VALUES (tid_st, tnotest)
//
// Akibatnya setiap simpan — termasuk setiap pengubahan — menerbitkan satu baris tingkat 1
// baru, dan baris tingkat 2 dipindahkan ke sana. Baris tingkat 1 lama menggantung tanpa
// satu pun yang merujuknya, dan tabelnya tumbuh tanpa batas.
//
// Work Owner memutuskan pada 2026-09-19 untuk MEMPERBAIKINYA, bukan mereplikasinya:
// induk dipilih dari daftar yang sudah ada, dan baris tingkat 1 baru hanya lahir bila
// pengguna memang meminta yang baru. Itu sebabnya ada dua isian di sini —
// ParentID untuk yang sudah ada, ParentName untuk yang baru. Perbaikan ini masuk daftar
// perbaikan eksplisit `P-5` sebagaimana `D-49`, sehingga selisih yang muncul pada uji
// kesetaraan gerbang 1 sudah disetujui di muka.
type Input struct {
	// Name adalah Status Penolakan 2 yang diketik pengguna — kolom NOTE_ND.
	Name string

	// ParentID adalah Status Penolakan 1 yang DIPILIH pengguna dari daftar.
	//
	// Kosong berarti pengguna memilih membuat induk baru; ParentName yang dipakai.
	ParentID string

	// ParentName adalah Status Penolakan 1 BARU yang diketik pengguna.
	//
	// Hanya dibaca bila ParentID kosong. Bila ParentID terisi, nama induk diambil dari
	// baris yang dipilih — bukan dari sini — supaya salinan di kolom NOTE_ST tidak
	// pernah berbeda dari induk yang dirujuknya.
	ParentName string
}

// Submission adalah Input yang sudah bersih, ditambah jejak siapa dan kapan.
//
// Keduanya tidak berasal dari layar: pelakunya diambil dari sesi pemanggil, dan waktunya
// dari seam Clock. Menerimanya dari peramban berarti mempercayai klien atas dua nilai
// yang justru menjadi jejak pertanggungjawaban.
type Submission struct {
	Input

	// By mengisi kolom USER_INPUT.
	By string

	// At mengisi kolom TANGGALKIRIM. Disimpan UTC; lihat berkas .sql.
	At time.Time
}

// MaxNameLength adalah panjang maksimum kolom NOTE_ND dan NOTE_ST.
//
// Angkanya ASUMSI yang disadari, bukan panjang kolom yang diterima dari DBA: DDL
// POOLDATA.MST_PENOLAKAN_KLAIM_1 dan _2 belum ada (R-08), dan
// `Database/MASTERPENOLAKANKLAIM2.prc` menerima parameternya sebagai `varchar2` tanpa
// panjang sehingga tidak memberi petunjuk apa pun.
//
// Angka 100 dipilih karena sama dengan MaxNameLength pada Master Status Progres, yang
// DITETAPKAN Work Owner untuk kolom sejenis pada tabel yang sekerabat. Bila basis data
// ternyata menerima lebih pendek, penolakannya datang dari basis data dan terbaca sebagai
// galat teknis, bukan sebagai pesan yang menuntun pengguna — kekurangan yang diterima
// sampai DDL-nya tiba.
//
// Angka yang sama diulang di `RejectionForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah; uji di masterpenolakan_test.go yang menjaganya tetap terlihat.
const MaxNameLength = 100

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris Status Penolakan 2 yang diminta tidak ada.
	ErrNotFound = errors.New("masterpenolakan: status penolakan tidak ditemukan")

	// ErrParentNotFound: Status Penolakan 1 yang dirujuk tidak ada.
	//
	// Dibedakan dari ErrNotFound supaya layar dapat mengatakan induk MANA yang bermasalah,
	// dan supaya keduanya dijawab kode HTTP yang berbeda: baris yang dipilih pengguna dari
	// daftar adalah isian yang salah (422), bukan sumber daya yang hilang (404).
	ErrParentNotFound = errors.New("masterpenolakan: status penolakan 1 induk tidak ditemukan")

	// ErrIDTaken: ID yang akan disisipkan sudah dipakai baris lain.
	ErrIDTaken = errors.New("masterpenolakan: ID status penolakan sudah dipakai")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (P-5). `InputRegister_act` sistem lama memeriksa
// belasan aturan lalu menampilkan semuanya bersamaan; mengembalikan satu galat per
// percobaan akan membuat pengguna menebak-nebak isian mana lagi yang salah
// (`11-CROSSCUTTING.md` §1.2 aturan 1).
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterpenolakan: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (i Input) Clean() Input {
	return Input{
		Name:       strings.TrimSpace(i.Name),
		ParentID:   strings.TrimSpace(i.ParentID),
		ParentName: strings.TrimSpace(i.ParentName),
	}
}

// WantsNewParent menyatakan pengguna meminta Status Penolakan 1 yang baru.
//
// Input sudah harus melewati Clean lebih dulu.
func (i Input) WantsNewParent() bool { return i.ParentID == "" }

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// Keberadaan induknya TIDAK diperiksa di sini: itu menuntut pembacaan basis data, dan
// domain tidak boleh melakukannya. Pemeriksaannya ada di Repo.InsertNew dan Repo.Update,
// yang memang sudah membaca baris induk untuk menyalin namanya.
func (i Input) Check() error {
	var violation []Violation

	violation = append(violation, checkText(i.Name, "nama", "Status Penolakan 2")...)

	// Induk diperiksa menurut cara pengguna memilihnya. Kalau ia memilih dari daftar,
	// yang wajib adalah pilihannya; kalau ia membuat baru, yang wajib adalah namanya.
	// Memeriksa keduanya sekaligus akan menuntut isian yang memang sengaja dikosongkan.
	if i.WantsNewParent() {
		violation = append(violation, checkText(i.ParentName, "nama_status_1", "Status Penolakan 1")...)
	} else if i.ParentName != "" {
		// Keduanya terisi berarti layar mengirim dua sumber untuk satu nilai, dan yang
		// mana yang menang menjadi pertanyaan yang tidak perlu ada. Ditolak terang-terangan
		// alih-alih dipilih diam-diam.
		violation = append(violation, Violation{
			Field:   "nama_status_1",
			Message: "Pilih Status Penolakan 1 yang sudah ada, atau isi yang baru — tidak keduanya.",
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// checkText menjalankan dua aturan yang sama pada setiap isian teks: wajib dan terbatas.
//
// Ia satu fungsi supaya kedua pesan berbunyi sama persis di kedua isian — pesan yang
// disusun sendiri di setiap tempat akan berbeda kata begitu salah satunya disunting.
func checkText(value, field, label string) []Violation {
	switch {
	case value == "":
		return []Violation{{
			Field:   field,
			Message: label + " wajib diisi.",
		}}
	case len(value) > MaxNameLength:
		return []Violation{{
			Field:   field,
			Message: fmt.Sprintf("%s paling panjang %d karakter.", label, MaxNameLength),
		}}
	}
	return nil
}

// FormatID menyusun ID_ST — kunci Status Penolakan 1 — dari nomor urut berikutnya.
//
// Angka polos tanpa awalan dan tanpa pemadatan lebar, direplikasi dari
// `Database/MASTERPENOLAKANKLAIM1.prc:6`:
//
//	select nvl(max(to_number(ID_ST)),0)+1 into id_mst from MST_PENOLAKAN_KLAIM_1;
//
// Perhatikan `to_number` di dalamnya: procedure lama sendiri memperlakukan kolom teks itu
// sebagai angka, sehingga bentuk "01" atau "001" justru akan menyimpang darinya.
func FormatID(sequence int) string { return strconv.Itoa(sequence) }

// FormatID2 menyusun ID_ND — kunci Status Penolakan 2 — dari nomor urut berikutnya.
//
// Bentuknya sama dengan tingkat 1, dari `Database/MASTERPENOLAKANKLAIM2.prc:6`. Ia
// fungsi tersendiri meski isinya sama, karena keduanya menomori tabel yang berbeda dan
// tidak ada jaminan keduanya tetap sama bila salah satu kelak berubah.
func FormatID2(sequence int) string { return strconv.Itoa(sequence) }

// NextSequence mencari nomor urut berikutnya yang belum dipakai.
//
// # Kenapa nomornya dihitung di Go, bukan dengan MAX di SQL
//
// Procedure lama memakai `NVL(MAX(TO_NUMBER(ID_ST)),0)+1`. `TO_NUMBER` atas kolom teks
// akan gagal dengan ORA-01722 begitu SATU baris saja memuat nilai yang bukan angka — dan
// karena kedua kolom itu bertipe teks tanpa constraint yang diketahui (R-08), tidak ada
// yang mencegahnya. Menghitungnya di Go membuat baris semacam itu dilewati alih-alih
// menghentikan seluruh penambahan.
//
// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan
// menabraknya lebih buruk daripada melewatinya.
//
// `format` dipakai untuk menguji calon terhadap daftar terpakai, sehingga fungsi ini
// melayani kedua tingkat tanpa menduplikasi perulangannya.
func NextSequence(used []string, format func(int) string) string {
	taken := make(map[string]bool, len(used))
	highest := 0
	for _, id := range used {
		clean := strings.TrimSpace(id)
		taken[clean] = true
		if number, err := strconv.Atoi(clean); err == nil && number > highest {
			highest = number
		}
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk tabel
	// yang sudah memuat ID berbentuk lain, supaya baris baru tidak menabraknya.
	for number := highest + 1; ; number++ {
		if candidate := format(number); !taken[candidate] {
			return candidate
		}
	}
}

// Repo adalah seam ke penyimpanan Master Penolakan Klaim SATU portal.
//
// # Kenapa satu seam untuk dua tabel
//
// Karena penambahannya memang satu operasi yang tidak dapat dipecah: menyimpan satu
// Status Penolakan 2 menuntut induknya ada — dan bila pengguna meminta induk baru, induk
// itu harus lahir lebih dulu di dalam transaksi yang sama. Memecahnya menjadi dua seam
// berarti lapisan aplikasi yang menjahitnya, dan jahitan itu tidak dapat dibungkus satu
// transaksi tanpa membocorkan `*sql.Tx` ke luar repo.
//
// Sistem lama pun memperlakukannya sebagai satu langkah:
// `Activity/InsertMasterPenolakanNoteKlaim-Act.xml` memanggil `MasterPenolakanKlaim1`
// lalu langsung memakai ID hasilnya untuk `MasterPenolakanKlaim2`.
//
// # Kenapa TIDAK ada Delete
//
// Seluruh export tidak memuat satu pun `DELETE` terhadap kedua tabel ini, layar lama pun
// tidak punya tombolnya, dan keduanya tidak punya kolom penanda terhapus yang dapat
// dipakai `D-66`. Baris Status Penolakan 2 juga dirujuk klaim yang sudah ditolak
// memakainya; menghapusnya akan memutus rujukan itu (`ADR-0012`).
type Repo interface {
	// ListParent mengembalikan seluruh Status Penolakan 1, untuk daftar pilihan di layar.
	ListParent(ctx context.Context) ([]RejectionStatus, error)

	// List mengembalikan seluruh Status Penolakan 2, terurut seperti kueri lama.
	List(ctx context.Context) ([]RejectionStatus2, error)

	// Get mengembalikan satu Status Penolakan 2; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (RejectionStatus2, error)

	// InsertNew menyimpan satu Status Penolakan 2 baru beserta induknya bila diminta.
	//
	// Urutannya: induk dipastikan ada (atau dibuat), ID diturunkan dari isi tabel, lalu
	// barisnya disisipkan — ketiganya dalam satu operasi repo, karena nomornya diturunkan
	// dari isi tabel itu sendiri dan memisahkannya membuka lubang balapan yang justru
	// sedang ditutup.
	//
	// Status baris baru SELALU StatusPending, tidak pernah dari pemanggil: itu yang
	// ditulis `Database/MASTERPENOLAKANKLAIM2.prc:10`, dan yang berwenang mengubahnya
	// adalah layar Inbox Manager.
	//
	// Submission sudah harus bersih dan lolos Check.
	InsertNew(ctx context.Context, s Submission) (RejectionStatus2, error)

	// Update menyimpan perubahan pada Status Penolakan 2 yang sudah ada.
	//
	// # Ia MENGEMBALIKAN baris ke antrean persetujuan
	//
	// `Database/MASTERPENOLAKANKLAIM2.prc:14` menyetel `A.STATUS='0'` dan
	// `A.TANGGALKIRIM=sysdate` pada setiap pengubahan. Artinya mengubah teks penolakan
	// membatalkan persetujuan yang sudah ada dan mengirimnya kembali ke checker.
	//
	// Perilaku itu DIPERTAHANKAN atas keputusan Work Owner 2026-09-19, dan memang masuk
	// akal secara bisnis: teks yang sudah disetujui tidak boleh berubah diam-diam.
	//
	// Yang TIDAK ikut dibersihkan — APPROVEBY, TANGGAL_APPROVE, dan NOTEAPPROVED —
	// dibiarkan apa adanya persis seperti procedure lama. Akibatnya baris berstatus
	// MENUNGGU masih menampilkan nama penyetuju sebelumnya; itu jejak keputusan yang
	// pernah ada, bukan keadaan yang berlaku, dan layar menyebutnya demikian.
	//
	// ErrNotFound bila barisnya hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, id string, s Submission) (RejectionStatus2, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (R-20).
type RepoSelector func(portalAlias string) (Repo, error)
