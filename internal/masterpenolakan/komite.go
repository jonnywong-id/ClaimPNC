package masterpenolakan

import (
	"context"
	"errors"
	"strings"
)

// # Master Penolakan Komite — master kedua pada layar yang sama
//
// Berkas ini melengkapi masterpenolakan.go. Keduanya dilayani SATU layar Pega
// (`Harness/PNC_MasterTolakKlaim-Harness.xml`) yang berpindah isi lewat dua tombol:
//
//	Input Master Penolakan Klaim   FlgMasterPenolakan.FlagASO = 1  -> MST_PENOLAKAN_KLAIM_1/_2
//	Input Master Penolakan Komite  FlgMasterPenolakan.FlagASO = 2  -> MST_REJECTED_KOMITE
//
// Pemilihannya terbaca di `Activity/SetStatusMasterRejectsKlaim-Act.xml`, yang hanya
// menyetel satu properti, dan di dua syarat tampil pada
// `Section/BrowseNoteRejectClaim-Section.xml`.
//
// # Kenapa satu paket, bukan dua modul
//
// Karena satu butir menu (MENU_ID 25, `Database/m_menu_aplikasi_pnc.csv`) dan satu layar.
// Memecahnya menjadi dua modul berarti dua rakitan, dua pemilih portal, dan dua tempat
// yang harus diingat setiap kali layar itu disentuh — sementara pengguna melihatnya
// sebagai satu layar dengan dua tab.
//
// Yang TIDAK disatukan adalah datanya: tabelnya tidak sekerabat, kuncinya berbeda
// bentuk, dan tidak ada satu pun kolom yang menghubungkannya. Karena itu ia punya tipe,
// seam, dan pemilih portalnya sendiri — bukan menumpang pada Repo di masterpenolakan.go.
//
// # Asal setiap aturan di berkas ini
//
//	RDB List/GetMasterRejectedKomites-SQL.xml      daftar, ORDER BY IDMASTER
//	RDB List/MasterRejectedKlaimPNC-SQL.xml        panggil INSERTMASTERREJECTEDKOMITE
//	Database/INSERTMASTERREJECTEDKOMITE.prc        sisip & perbarui
//	Activity/GetAllDataMasterRejected-Act.xml      RDB-List ke page KomitePenolakanMaster
//	Activity/InsertMasterRejectedKomite-Act.xml    urutan langkah simpan
//
// # Penamaan ulang yang disengaja (D-19)
//
//	IDMASTER   AS "IDMaster"   -> ID
//	NOTEMASTER AS "NoteKasir"  -> Note   (tidak ada urusan dengan kasir)
//	'ubah'     AS "NOKTP"      -> dibuang; ia label tombol, bukan data, dan namanya
//	                              mengaku sebagai nomor KTP

// CommitteeRejection adalah satu baris Master Penolakan Komite —
// POOLDATA.MST_REJECTED_KOMITE.
//
// Dua kolom, dan itu memang seluruh isi tabelnya menurut
// `Database/INSERTMASTERREJECTEDKOMITE.prc:15`.
type CommitteeRejection struct {
	// ID adalah kunci baris, kolom IDMASTER.
	//
	// Ia disimpan sebagai teks di sini meski kolomnya bertipe NUMBER — procedure lama
	// mendeklarasikannya `tIDMASTER in number` (`:1`). Alasannya sama dengan seluruh
	// kunci master di aplikasi ini: kunci tidak pernah dihitung, hanya dicocokkan dan
	// ditampilkan, dan menjadikannya angka mengundang pemformatan yang tidak diminta.
	ID string

	// Note adalah keterangan penolakan komite yang dibaca petugas, kolom NOTEMASTER.
	Note string
}

// InputKomite adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Satu isian saja, persis seperti layar lama: form "Insert/Update Reject Komite" pada
// `Section/BrowseNoteRejectClaim-Section.xml` hanya mengikat
// `InsertUpdateMasterReject.NoteKasir`. ID tidak pernah diketik — pada penambahan ia
// diturunkan dari isi tabel, pada pengubahan ia diambil dari baris yang dipilih.
type InputKomite struct {
	Note string
}

// FirstIDKomite adalah nomor yang dipakai baris PERTAMA pada tabel yang masih kosong.
//
// Angka 111 direplikasi apa adanya dari `Database/INSERTMASTERREJECTEDKOMITE.prc:9`:
//
//	if count_data=0 then count_data:=111;
//
// Tidak ada keterangan apa pun di sumbernya tentang asal angka itu, dan tidak ada yang
// dapat disimpulkan dari data karena isi tabelnya tidak ikut dikirim. Ia direplikasi
// karena `P-5` menuntut perilaku dipertahankan lebih dulu, dan karena ia hanya berlaku
// pada tabel kosong — keadaan yang di produksi kemungkinan besar sudah lama lewat.
const FirstIDKomite = 111

// MaxNoteLengthKomite adalah panjang maksimum kolom NOTEMASTER.
//
// ASUMSI yang disadari, dengan alasan yang sama seperti MaxNameLength: DDL
// POOLDATA.MST_REJECTED_KOMITE belum ada (R-08), dan procedure lama menerimanya sebagai
// `varchar2` tanpa panjang.
//
// Angka yang sama diulang di `CommitteeRejectionForm.tsx`.
const MaxNoteLengthKomite = 100

// ErrKomiteNotFound: baris Master Penolakan Komite yang diminta tidak ada.
//
// Ia galat tersendiri, bukan ErrNotFound yang sama, supaya pesan yang sampai ke pengguna
// menyebut master yang benar. Kedua tab hidup di satu layar, dan "tidak ditemukan" tanpa
// keterangan akan membuat petugas mencari di tab yang salah.
var ErrKomiteNotFound = errors.New("masterpenolakan: penolakan komite tidak ditemukan")

// Clean memangkas spasi di kedua ujung isian.
func (i InputKomite) Clean() InputKomite {
	return InputKomite{Note: strings.TrimSpace(i.Note)}
}

// Check menjalankan seluruh aturan isian dan mengembalikan semua pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// Kedua aturannya — wajib dan terbatas panjang — TIDAK ADA di sistem lama: layar Pega
// meneruskan isian apa adanya, sehingga catatan kosong pun tersimpan. Keduanya
// ditambahkan mengikuti keputusan yang sama pada Master Status Klaim (2026-09-17), yang
// juga menambahkan aturan wajib-isi pada master yang sebelumnya menerima apa saja.
func (i InputKomite) Check() error {
	if violation := checkText(i.Note, "catatan", "Note Komite Reject"); len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// FormatIDKomite menyusun IDMASTER dari nomor urut berikutnya.
//
// Angka polos, mengikuti `Database/INSERTMASTERREJECTEDKOMITE.prc:11`
// (`select max(IDMASTER)+1`). Kolomnya bertipe NUMBER, sehingga tidak ada awalan nol
// maupun pemadatan lebar yang perlu ditiru.
func FormatIDKomite(sequence int) string { return FormatID(sequence) }

// RepoKomite adalah seam ke penyimpanan Master Penolakan Komite SATU portal.
//
// # Kenapa TIDAK ada Delete
//
// Alasannya sama dengan Repo: seluruh export tidak memuat satu pun `DELETE` terhadap
// POOLDATA.MST_REJECTED_KOMITE, layar lama hanya punya tombol "ubah", dan tabelnya tidak
// punya kolom penanda terhapus yang dapat dipakai `D-66`.
type RepoKomite interface {
	// List mengembalikan seluruh baris, terurut seperti kueri lama.
	List(ctx context.Context) ([]CommitteeRejection, error)

	// Get mengembalikan satu baris; ErrKomiteNotFound bila tidak ada.
	Get(ctx context.Context, id string) (CommitteeRejection, error)

	// InsertNew menurunkan ID dari isi tabel lalu menyisipkan barisnya.
	//
	// Penurunan ID berada di dalam satu operasi repo dengan alasan yang sama seperti
	// Repo.InsertNew: nomornya diturunkan dari isi tabel itu sendiri.
	//
	// Input sudah harus bersih dan lolos Check.
	InsertNew(ctx context.Context, input InputKomite) (CommitteeRejection, error)

	// Update menyimpan perubahan catatan pada baris yang sudah ada.
	//
	// Hanya NOTEMASTER yang berubah; IDMASTER adalah kunci dan tidak pernah di-SET —
	// persis `Database/INSERTMASTERREJECTEDKOMITE.prc:19`, yang memakainya hanya sebagai
	// penyaring WHERE.
	//
	// ErrKomiteNotFound bila barisnya hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, id string, input InputKomite) (CommitteeRejection, error)
}

// RepoSelectorKomite memilih RepoKomite milik satu portal entitas.
//
// Alasannya sama persis dengan RepoSelector: portal yang tidak dikenal atau koneksinya
// belum hidup WAJIB menghasilkan galat, tidak pernah dialihkan ke portal utama sebagai
// cadangan (R-20).
type RepoSelectorKomite func(portalAlias string) (RepoKomite, error)
