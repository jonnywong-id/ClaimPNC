package masterstatusprogres

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// # Master Status Progres 2 — tingkat kedua
//
// Berkas ini melengkapi masterstatusprogres.go, yang sejak awal sudah menyebut bahwa tingkat
// dua "menempel pada modul yang sama karena kuncinya merujuk tingkat 1". Itulah yang
// dikerjakan di sini: satu paket domain, dua tingkat, karena penambahan tingkat 2 TIDAK
// dapat dilakukan tanpa membaca tingkat 1 lebih dulu (lihat Input2 dan Repo2.InsertNew).
//
//	Status Progres 1  POOLDATA.GCNM_MST_PROGRESS_KLAIM  ID_PROGRESS (kunci)
//	  └─ Status Progres 2  POOLDATA.GCNM_MST_PROGRESS    ID_MST (kunci), ID_PROGRESS (induk)
//
// Perhatikan nama tabelnya: yang berakhiran `_KLAIM` adalah tingkat SATU. Tingkat dua
// memakai nama yang lebih pendek. Tertukar sekali saja berarti layar ini menulis ke
// tabel induknya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/StatusProgress2-Harness.xml             layar "Master Status Progress 2"
//	Section/MasterStatusProgress2-Section.xml        judul layar, tombol Tambah & Refresh
//	Section/BrowseStatusProgress2-Section.xml        grid dan form modal
//	RDB List/BrowseStatusProgress2-SQL.xml           daftar, ORDER BY ID_MST ASC
//	RDB List/InsertStatusProgress2-SQL.xml           sisip
//	RDB List/UpdateStatusProgress2-SQL.xml           ambil satu baris (namanya "Update",
//	                                                 isinya SELECT)
//	RDB List/BrowseIDStatusProgress-SQL.xml          MAX(ID_MST)+1
//	RDB List/BrowseMstProgress1-SQL.xml              sumber dropdown induk
//	Activity/InsertMstStatusProgress2_act-Act.xml    urutan langkah sisip
//	Activity/UpdateMstStatusProgress2_act-Act.xml    urutan langkah sunting — lihat §
//	                                                 "Kenapa tidak ada penyuntingan"
//
// # Penamaan ulang yang disengaja (D-19)
//
// Kueri lama mengaliaskan kelima kolomnya ke nama yang tidak mencerminkan isi sama
// sekali — utang teknis §4.2 `03-CURRENT-ARCHITECTURE.md` dalam bentuknya yang paling
// parah di seluruh modul yang sudah dikerjakan. Alias itu TIDAK dibawa:
//
//	ID_MST         AS "CaseID"      -> ID          (bukan nomor klaim)
//	STS_PROGRESS1  AS "City"        -> ParentName   (bukan nama kota)
//	STS_PROGRESS2  AS "CityID"      -> Nama        (bukan kode kota)
//	ID_PROGRESS    AS "District"    -> ParentID     (bukan kabupaten)
//	TIPE           AS "DistrictID"  -> Tipe        (bukan kode kabupaten)
//
// Perhatikan bahwa "City"/"CityID" dan "District"/"DistrictID" di sini TIDAK berpasangan
// seperti dugaan yang wajar: `City` adalah nama induk sedangkan `CityID` adalah nama
// baris ini sendiri. Itu sebabnya alias lama tidak boleh dipakai sebagai petunjuk arti.

// ProgressStatus2 adalah satu baris master status progres tingkat 2.
type ProgressStatus2 struct {
	// ID adalah kunci baris, kolom ID_MST.
	//
	// Bentuknya BERBEDA dari ID tingkat 1: di sini angka polos tanpa awalan. Lihat
	// FormatID2.
	ID string

	// Nama adalah keterangan status tingkat 2 yang dibaca petugas, kolom STS_PROGRESS2.
	Name string

	// ParentID merujuk Status Progres 1, kolom ID_PROGRESS.
	ParentID string

	// ParentName adalah SALINAN nama induk pada saat baris ini disimpan, kolom
	// STS_PROGRESS1.
	//
	// Ia salinan, bukan hasil join — dan itu perilaku sistem lama yang dipertahankan
	// apa adanya atas keputusan Work Owner 2026-09-18. Rinciannya di Repo2.InsertNew.
	ParentName string

	// Tipe adalah kolom TIPE, dibaca apa adanya dan tidak pernah ditulis.
	//
	// Artinya TIDAK diketahui: di seluruh export ia hanya muncul pada dua SELECT
	// (`BrowseStatusProgress2-SQL.xml` dan `UpdateStatusProgress2-SQL.xml`), tanpa satu
	// pun INSERT, UPDATE, maupun penyaring yang memakainya. DDL tabelnya belum diterima
	// (R-08), sehingga tidak ada pula daftar nilai sahnya.
	//
	// Ia tetap dibaca dan ditampilkan supaya nilai yang benar-benar tersimpan terlihat
	// petugas. Ia TIDAK pernah ikut ditulis, sehingga baris lama tidak kehilangan
	// nilainya hanya karena disentuh layar baru.
	Kind string
}

// Input2 adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Hanya DUA isian, persis seperti layar lama: `Section/BrowseStatusProgress2-Section.xml`
// hanya mengikat `TempInputStatus2.CityID` (induk, dropdown) dan `TempInputStatus2.District`
// (nama). Ketiga nilai lainnya diturunkan sistem:
//
//	ID         dari MAX(ID_MST)+1
//	ParentName  disalin dari baris induk yang dibaca ulang
//	Tipe       tidak pernah diisi
type Input2 struct {
	// Nama adalah keterangan status tingkat 2 yang diketik pengguna.
	Name string

	// ParentID adalah ID Status Progres 1 yang dipilih pengguna dari dropdown.
	ParentID string
}

// MaxNameLength2 adalah panjang maksimum kolom STS_PROGRESS2.
//
// Angkanya disamakan dengan MaxNameLength tingkat 1, dan itu ASUMSI yang disadari —
// bukan angka yang diterima dari Work Owner seperti halnya tingkat 1. DDL
// POOLDATA.GCNM_MST_PROGRESS belum ada (R-08), dan kedua kolom menyimpan hal yang
// sejenis pada tabel yang sekerabat.
//
// Bila basis data ternyata menerima lebih pendek, penolakannya datang dari basis data
// dan terbaca sebagai galat teknis, bukan sebagai pesan yang menuntun pengguna. Itu
// kekurangan yang diterima sampai DDL-nya tiba — bukan alasan menebak angka yang lebih
// longgar, karena menebak longgar justru memindahkan kegagalannya ke tempat yang lebih
// sulit dibaca.
//
// Angka yang sama diulang di `FormStatusProgres2.tsx`. Bila berubah, KEDUA tempat harus
// ikut berubah.
const MaxNameLength2 = 100

// Galat khusus tingkat 2.
var (
	// ErrParentNotFound: Status Progres 1 yang dirujuk tidak ada.
	//
	// Ia dibedakan dari ErrNotFound supaya layar dapat mengatakan induk MANA yang
	// bermasalah. Keduanya dipetakan ke kode HTTP yang berbeda pula: baris yang dirujuk
	// pengguna lewat dropdown adalah isian yang salah (422), bukan sumber daya yang
	// tidak ada (404).
	ErrParentNotFound = errors.New("masterstatusprogres: status progres 1 induk tidak ditemukan")
)

// Clean memangkas spasi di kedua ujung setiap isian.
func (i Input2) Clean() Input2 {
	return Input2{
		Name:     strings.TrimSpace(i.Name),
		ParentID: strings.TrimSpace(i.ParentID),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// Keberadaan induknya TIDAK diperiksa di sini: itu menuntut pembacaan basis data, dan
// domain tidak boleh melakukannya. Pemeriksaannya ada di Repo2.InsertNew, yang memang
// sudah membaca baris induk untuk menyalin namanya.
func (i Input2) Check() error {
	var violation []Violation

	switch {
	case i.Name == "":
		violation = append(violation, Violation{
			Field:   "nama",
			Message: "Nama status progres 2 wajib diisi.",
		})
	case len(i.Name) > MaxNameLength2:
		violation = append(violation, Violation{
			Field:   "nama",
			Message: fmt.Sprintf("Nama status progres 2 paling panjang %d karakter.", MaxNameLength2),
		})
	}

	if i.ParentID == "" {
		violation = append(violation, Violation{
			Field:   "id_induk",
			Message: "Status Progres 1 wajib dipilih.",
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// FormatID2 menyusun ID_MST dari nomor urut berikutnya.
//
// # Kenapa TANPA awalan "0", padahal tingkat 1 memakainya
//
// Bukan kelalaian, melainkan perbedaan yang nyata di sistem lama dan sudah diverifikasi
// dua kali — sekali dari rule, sekali dari data:
//
//	tingkat 1  Activity/InsertMstStatusProgress1_act  TempInputStatus.CaseID  := "0"+.City
//	tingkat 2  Activity/InsertMstStatusProgress2_act  TempInputStatus2.CaseID := .District
//
// Tingkat 2 memakai nilai `NVL(MAX(B.ID_MST),0)+1` APA ADANYA, tanpa perangkaian apa pun.
// Dikuatkan data nyata yang beredar di kueri lain: `GetDataProgressClaim-SQL.xml`
// menyaring `G.STATUS_PROGRESS2 not in ('2','24','60')` — angka polos, bukan "02".
//
// Memakai FormatID tingkat 1 di sini karena itu akan menerbitkan ID yang TIDAK dapat
// dicocokkan dengan baris `GCNM_PROGRESS_CLAIM.STATUS_PROGRESS2` yang sudah ada.
//
// CACAT YANG TIDAK IKUT TERBAWA: karena tanpa awalan, tingkat 2 luput dari persoalan
// urutan teks yang menghinggapi tingkat 1 ("010" mendahului "09"). Pengurutannya tetap
// dilakukan basis data mengikuti kueri lama, bukan diurutkan ulang di frontend.
func FormatID2(sequence int) string {
	return fmt.Sprintf("%d", sequence)
}

// Repo2 adalah seam ke penyimpanan master status progres tingkat 2 SATU portal.
//
// # Kenapa hanya List dan InsertNew
//
// Karena hanya itu yang benar-benar dilakukan sistem lama terhadap tabel ini.
//
// Seluruh export TIDAK memuat satu pun pernyataan yang mengubah isi
// POOLDATA.GCNM_MST_PROGRESS setelah barisnya tersimpan — tidak ada UPDATE, tidak ada
// DELETE. Yang tampak seperti penyuntingan ternyata bukan:
//
//	RDB List/UpdateStatusProgress2-SQL.xml      namanya "Update", isinya SELECT satu baris
//	Activity/UpdateMstStatusProgress2_act       menyiapkan lima Local.* lalu memanggil
//	                                            UpdateStatusProgress2_sql
//	RDB List/UpdateStatusProgress2_sql-SQL.xml  UPDATE POOLDATA.GCNM_PROGRESS_CLAIM
//	                                            SET JSONSTATUS_PROGRESS2 = ...
//
// Pernyataan terakhir itu menyentuh TABEL LAIN — GCNM_PROGRESS_CLAIM adalah catatan
// progres milik satu klaim, bukan master. Ia pun menyaring dengan
// `{tempSearchProgress.AnalystDoctorRemaks}` dan `{tempSearchProgress.Email}`, dua page
// klipboard yang tidak diisi activity itu maupun section layarnya; kelima `Local.*` yang
// disiapkan dengan cermat tidak pernah dipakai satu pun.
//
// Akibatnya di sistem lama: menekan "Update" pada layar Master Status Progress 2 tidak
// mengubah apa pun pada tabel master.
//
// # Yang DIPERTAHANKAN dan yang TIDAK
//
// Keputusan Work Owner 2026-09-18: jalankan as-is. Karena itu tingkat 2 di sini juga
// tidak mengubah baris master setelah tersimpan — perilakunya setara, dan uji kesetaraan
// gerbang 1 tidak akan melihat selisih.
//
// Satu bagian sengaja TIDAK direproduksi: pemanggilan UPDATE ke GCNM_PROGRESS_CLAIM.
// Menyalinnya berarti membawa pernyataan yang, bila kedua page itu kebetulan terisi sisa
// nilai dari layar lain dalam sesi yang sama, MENIMPA catatan progres sebuah klaim
// dengan isian layar master. Yang direplikasi adalah hasil yang teramati — baris master
// tidak berubah — bukan jalur yang menghasilkannya.
//
// Menambahkan penyuntingan kelak adalah perubahan yang menambah, bukan membongkar:
// satu method pada seam ini, satu kueri, satu rute.
type Repo2 interface {
	// List mengembalikan seluruh baris tingkat 2, terurut seperti kueri lama.
	List(ctx context.Context) ([]ProgressStatus2, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (ProgressStatus2, error)

	// InsertNew membaca induknya, menurunkan ID, lalu menyisipkan barisnya.
	//
	// Ketiganya berada di dalam SATU operasi repo dengan alasan yang sama seperti
	// tingkat 1: nomornya diturunkan dari isi tabel, sehingga memisahkan pembacaan dari
	// penulisan membuka lubang balapan yang justru sedang ditutup.
	//
	// Pembacaan induk ikut ke dalamnya karena dua hal yang tidak dapat dipisahkan:
	// memastikan induknya ADA (ErrParentNotFound bila tidak), dan MENYALIN namanya
	// ke kolom STS_PROGRESS1 — persis langkah "cari status progress 1" lalu
	// "set status progress 1" pada Activity/InsertMstStatusProgress2_act.
	//
	// Salinan nama itu adalah denormalisasi sistem lama yang dipertahankan atas
	// keputusan Work Owner 2026-09-18. Konsekuensinya disadari dan dicatat supaya tidak
	// dikira rancangan: mengganti nama sebuah Status Progres 1 TIDAK memperbarui salinan
	// di baris-baris tingkat 2 yang sudah ada, sehingga keduanya dapat berbeda. Bahwa
	// perbedaan itu benar-benar terjadi di produksi terbaca dari kueri laporan yang
	// membacanya — `GetDataOutstandingperCabangExport-SQL.xml` memakai
	// `SELECT id_progress, MAX(sts_progress1) ... GROUP BY id_progress`, dan MAX hanya
	// diperlukan bila baris ber-id_progress sama menyimpan nama yang berlainan.
	//
	// Input sudah harus bersih dan lolos Check.
	InsertNew(ctx context.Context, input Input2) (ProgressStatus2, error)
}

// RepoSelector2 memilih Repo2 milik satu portal entitas.
//
// Alasannya sama persis dengan RepoSelector: portal yang tidak dikenal atau koneksinya
// belum hidup WAJIB menghasilkan galat, tidak pernah dialihkan ke portal utama sebagai
// cadangan (R-20).
type RepoSelector2 func(portalAlias string) (Repo2, error)
