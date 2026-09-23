package masterxol

import "context"

// Repo adalah seam ke penyimpanan Master XOL SATU portal.
//
// Satu instans Repo selalu terikat pada satu basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat penyaringan baris (`ADR-0030` Opsi 1). Tidak
// ada satu pun kueri di pengisinya yang menyaring berdasarkan entitas, dan memang tidak
// boleh ada.
type Repo interface {
	// List mengembalikan seluruh induk XOL TANPA anaknya — secukupnya untuk grid.
	//
	// Anaknya sengaja tidak ikut: grid di `Section/DetailXOL_sec-Section.xml` hanya
	// menampilkan ID, Tahun, Kurs, dan Remark Komite, dan menarik 18 lapisan beserta 42
	// baris reas hanya untuk menampilkan empat kolom adalah pemborosan yang akan
	// bertambah buruk seiring bertambahnya tahun treaty.
	List(ctx context.Context) ([]Master, error)

	// Get mengembalikan satu induk LENGKAP dengan bisnis, lapisan, dan reas-nya.
	//
	// Menggantikan `Activity/UpdateMasterXOL-Act.xml`, yang memuat keempat tingkat
	// sekaligus sebelum form Ubah dibuka.
	Get(ctx context.Context, id string) (Master, error)

	// Save menyimpan satu induk beserta seluruh anaknya dalam SATU transaksi.
	//
	// ID kosong berarti menambah; terisi berarti mengubah. Pembedaan itu ditiru apa
	// adanya dari `Activity/InsertUpdateMasterXOL-Act.xml`, yang mengisi
	// `TempXOL.BranchID` dengan `"UnknownID"` ketika kosong lalu menyerahkan
	// percabangannya ke procedure (`INSERT_UPDATE_MST_XOL.prc:14`).
	//
	// Sifatnya UPSERT, bukan ganti-seluruhnya: baris yang tidak disebut TIDAK dihapus.
	// Itu pun mengikuti sistem lama — penghapusan di sana adalah aksi tersendiri lewat
	// `DeleteFromTabelMst`, bukan akibat sampingan dari Simpan.
	Save(ctx context.Context, master Master) (Master, error)

	// DeleteMaster menghapus satu induk BESERTA seluruh anaknya.
	//
	// # Kenapa berkaskade, sementara sistem lama tidak
	//
	// `Activity/DeleteFromTabelMst-Act.xml.xml` menghapus satu tabel saja per pemanggilan
	// — `type="mst"` hanya menyentuh MST_XOL_PNC. Akibatnya terlihat di produksi pada
	// 2026-09-20: induk `10003` sudah terhapus, tetapi **2 lapisan dan 3 baris bisnis
	// miliknya masih ada** dan tidak dapat dicapai layar mana pun.
	//
	// Keputusan Work Owner 2026-09-20: kaskade diterapkan. Ini selisih terencana
	// terhadap Pega, dan ia dipilih karena baris yatim tidak punya satu pun kegunaan
	// sementara jumlahnya terus bertambah.
	DeleteMaster(ctx context.Context, id string) error

	// DeleteBusiness menghapus satu baris grup bisnis dari sebuah induk.
	DeleteBusiness(ctx context.Context, masterID, businessID string) error

	// DeleteLayer menghapus satu lapisan BESERTA seluruh baris reas-nya.
	DeleteLayer(ctx context.Context, masterID, layerID string) error

	// DeleteReinsurer menghapus satu baris reas dari sebuah lapisan.
	DeleteReinsurer(ctx context.Context, layerID, reinsurerID string) error

	// ListYear mengembalikan pilihan Tahun untuk dropdown di layar.
	//
	// # Kenapa sumbernya M_TREATYYEAR
	//
	// Activity pengisi dropdown-nya (`TempYear`) **tidak ada di export** (`R-16`).
	// POOLDATA.M_TREATYYEAR adalah satu-satunya master tahun treaty di basis data, dan
	// pemeriksaan pada 2026-09-20 menunjukkan **kelima tahun yang dipakai master XOL
	// seluruhnya ada di sana** — 2015, 2016, 2017, 2018, dan 2022, dari 36 tahun yang
	// tersedia (1995–2030).
	//
	// Itu bukti yang kuat tetapi bukan bukti langsung, dan pilihan ini dicatat terbuka
	// sampai Tim Pega mengirimkan activity-nya.
	ListYear(ctx context.Context) ([]string, error)

	// ListBusinessGroup mengembalikan pilihan grup bisnis untuk sebuah Type XOL.
	//
	// Menggantikan `RDB List/GetDataBisnisXol_Sql-SQL.xml` beserta penyaring yang
	// dipasang `Activity/ShowDetailGroupBisnisXol_Act-Act.xml`. Lihat BusinessGroupPattern.
	ListBusinessGroup(ctx context.Context, t Type) ([]Business, error)

	// SubmitToCommittee mencatat pengajuan sebuah induk ke komite.
	//
	// Ia menyetel PIC, STSKOMITE, dan REMARKPIC — persis tiga kolom yang disentuh
	// `Activity/UpdateStatusMasterKomitexol-Act.xml` dan
	// `Activity/SendDataMasterXOLToKomites-Act.xml`.
	SubmitToCommittee(ctx context.Context, id, pic, remark string) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis struktur treaty
// satu badan hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// BusinessGroupPattern mengembalikan pola pencarian nama grup treaty untuk sebuah
// Type XOL.
//
// # Dari mana polanya
//
// `Activity/ShowDetailGroupBisnisXol_Act-Act.xml` menyusun potongan klausa WHERE lalu
// menyisipkannya ke kueri dengan `{Asis:TempXOL.AgentID}` — perangkaian teks SQL, persis
// pola yang `docs/Steering/07-TECHNICAL-STRATEGY.md` §4.3 larang. Di sini polanya
// dikembalikan sebagai NILAI dan diikat sebagai parameter, sehingga isinya tidak pernah
// lagi menjadi bagian dari teks SQL.
//
// Ketiga cabangnya, apa adanya:
//
//	CNPSupportDoc == "1"  →  %PROPERTY%  %MOTOR%  %ENGINEERING%
//	CNPSupportDoc == "2"  →  %PA%        %GA%
//	CNPSupportDoc == "3"  →  %MARINE%    %HEAVY EQUPMENT%
//
// # Dua hal yang terlihat salah dan sengaja TIDAK diperbaiki
//
//  1. `HEAVY EQUPMENT` kurang huruf I. Dijalankan langsung ke portal ASM pada 2026-09-20,
//     ejaan yang benar maupun yang salah mengembalikan hasil yang SAMA — nama itu tidak
//     ada di POOLDATA.PROPORTIONALARRG.TREATYGROUPNAME dalam ejaan mana pun. Salah
//     ketiknya karena itu tidak berakibat apa-apa.
//  2. Penyaring Type 2 mengembalikan **hanya "AVIATION HULL"** — PA dan GA justru tidak
//     muncul. Itu cacat nyata di produksi hari ini.
//
// Keputusan Work Owner 2026-09-20: keduanya ditiru apa adanya (`P-5`). Cacat kedua
// dicatat terbuka; memperbaikinya mengubah data mana yang boleh dipilih pengguna, dan
// itu keputusan tersendiri.
//
// # Kenapa selalu tiga pola
//
// Supaya kuerinya tetap SATU teks tetap dengan tiga parameter, bukan dirangkai sesuai
// jumlah pola. Ketika polanya hanya dua, yang terakhir diulang — `A OR B OR B` bernilai
// sama dengan `A OR B`, dan tidak ada teks SQL yang perlu dibangun.
func BusinessGroupPattern(t Type) []string {
	switch t {
	case TypeProperty:
		return []string{"%PROPERTY%", "%MOTOR%", "%ENGINEERING%"}
	case TypeAccident:
		return []string{"%PA%", "%GA%", "%GA%"}
	case TypeMarine:
		return []string{"%MARINE%", "%HEAVY EQUPMENT%", "%HEAVY EQUPMENT%"}
	default:
		// Type yang belum dipilih tidak menyaring apa pun. Layar lama tidak punya cabang
		// untuk keadaan ini — dropdown bisnisnya baru menyala setelah Type dipilih — dan
		// mengembalikan seluruh grup di sini membuat layar tetap dapat dipakai tanpa
		// mengarang penyaring keempat.
		return []string{"%", "%", "%"}
	}
}

// TreatyInwardName adalah nama grup bisnis yang DITAMBAHKAN ke daftar pilihan tanpa ID.
//
// `Activity/ShowDetailGroupBisnisXol_Act-Act.xml` menambahkannya sebagai baris terakhir
// pada setiap Type, dan `Database/GET_GROUPBUSINESS_XOL.fnc:31` memakai nama yang sama
// sebagai pengganti ketika `businessgroup.note` bernilai NULL.
//
// Ia memang tidak punya ID: pemeriksaan pada 2026-09-20 menunjukkan POOLDATA.BUSINESSGROUP
// tidak memuat baris `11111` yang dipakai induk 10002, dan dua baris induk 10009
// menyimpan NULL. Keadaan itu dibawa apa adanya.
const TreatyInwardName = "TREATY INWARD"

// CommitteeSubmission adalah bahan pemberitahuan pengajuan sebuah induk XOL ke komite.
type CommitteeSubmission struct {
	MasterID     string
	Year         string
	ExchangeRate Amount

	// Remark adalah catatan PIC — isi kolom REMARKPIC.
	Remark string

	// SubmittedBy adalah identitas PIC yang mengajukan.
	SubmittedBy string
}

// Notifier adalah seam ke pemberitahuan komite.
//
// # Kenapa ia menyatakan PERISTIWA, bukan "kirim surel ke alamat ini"
//
// `docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.6 menetapkan antarmuka Notifier bicara
// dalam peristiwa domain, sehingga SIAPA penerimanya menjadi urusan konfigurasi dan bukan
// urusan pemanggil. Itu yang menutup kemungkinan pola lama terulang:
// `Activity/SendDataMasterXOLToKomites-Act.xml` menimpa penerima hasil pencarian dengan
// alamat perorangan, lalu menimpanya sekali lagi dengan alamat perorangan yang berbeda
// ketika berjalan di host dev — persis blok penimpaan yang `D-15` larang dibawa.
//
// `D-67` menegaskan tidak ada akun pribadi yang dibawa ke sistem baru. Penerimanya karena
// itu datang dari konfigurasi, bukan dari kode.
//
// Pengisinya ada di notification/ — pengirim SMTP dan sebuah tiruan yang merekam.
type Notifier interface {
	NotifyCommitteeSubmission(ctx context.Context, submission CommitteeSubmission) error
}
