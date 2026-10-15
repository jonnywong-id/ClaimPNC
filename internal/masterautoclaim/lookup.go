package masterautoclaim

import (
	"context"
	"errors"
)

// ErrBankNotFound: nama bank yang dikirim tidak ada di GENERAL.LST_BANK_GROUP.
//
// Ia galat lookup, bukan galat master auto claim, sehingga tempatnya di berkas ini.
// Lapisan aplikasi mengubahnya menjadi pelanggaran pada isian `nama_bank`; lihat
// OneViolation.
var ErrBankNotFound = errors.New("masterautoclaim: bank tidak ada di master bank")

// # Empat tabel acuan yang dibaca modul ini
//
// Keempatnya milik sistem lain dan aplikasi ini HANYA MEMBACA (ADR-0004, penulis tunggal
// per tabel). Tidak satu pun disentuh oleh penyimpanan master auto claim:
//
//	POOLDATA.AGENT            Sumber Bisnis — asal INISIALID dan NAMA_PENERIMA
//	POOLDATA.CLIENT           Client        — asal CLIENTID dan CLIENTNAME
//	GENERAL.LST_BANK_GROUP    Bank          — asal BANK_PENERIMA
//	POOLDATA.EMAILKOMITE      Komite        — asal KOMITE
//
// Ketiga yang pertama adalah lookup yang dipakai pengguna saat mengisi form; yang
// keempat diisi sistem tanpa pernah terlihat pengguna.

// BusinessSource adalah satu Sumber Bisnis pada POOLDATA.AGENT.
//
// Ia yang menjadi kunci master auto claim: `ID` di sini disimpan sebagai
// `M_AUTO_CLAIM_PNC.INISIALID`, dan dicocokkan dengan `T_GENERAL.SOURCEOFBUSINESS`
// milik polis saat klaim otomatis dibuat.
type BusinessSource struct {
	// ID adalah kolom POOLDATA.AGENT.ID.
	ID string

	// Name adalah kolom POOLDATA.AGENT.CLIENTNAME.
	//
	// Perhatikan namanya: kolom bernama CLIENTNAME pada tabel AGENT BUKAN nama client.
	// Ia nama agen atau sumber bisnis, dan ia yang disimpan sebagai NAMA_PENERIMA.
	// Master client yang sebenarnya adalah POOLDATA.CLIENT — lihat Client di bawah.
	Name string
}

// Client adalah satu tertanggung pada POOLDATA.CLIENT.
type Client struct {
	// ID adalah kolom POOLDATA.CLIENT.ID, disimpan sebagai CLIENTID.
	ID string

	// Name adalah kolom POOLDATA.CLIENT.NAME, disimpan sebagai CLIENTNAME.
	Name string
}

// Bank adalah satu bank pada GENERAL.LST_BANK_GROUP.
//
// Tabel yang sama dibaca modul Master Rekening. Tipe ini sengaja TIDAK dipakai bersama:
// modul tidak saling mengimpor, dan tipe bersama akan membuat perubahan di satu modul
// menyeret modul lain. Yang dipakai bersama adalah tabelnya, bukan kodenya.
type Bank struct {
	// Code adalah kolom LBG_ID. Ia TIDAK pernah disimpan ke master auto claim —
	// tabelnya tidak punya kolom kode bank. Ia hanya dipakai memastikan bank yang
	// dikirim layar benar-benar dipilih dari daftar (Input.BankCode).
	Code string

	// Name adalah kolom BANK_GROUP. Inilah yang disimpan sebagai BANK_PENERIMA.
	Name string
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan sebuah pencarian lookup.
//
// Sistem lama tidak membatasinya sama sekali: `GetClientName-SQL.xml` mengembalikan
// seluruh baris POOLDATA.AGENT yang cocok, dan kata kunci sependek satu huruf akan
// menariknya nyaris seluruhnya. Pada master berbaris banyak itu memuat ribuan baris ke
// memori aplikasi dan ke peramban, untuk daftar yang hanya akan dipilih satu.
//
// Batasnya di sini, bukan di frontend: memotong di peramban berarti barisnya sudah
// terlanjur dibaca, dikirim, dan diurai.
const MaxLookupRows = 50

// MinLookupKeyword adalah panjang minimum kata kunci pencarian lookup.
//
// Sistem lama menerima kata kunci kosong, dan `like '%%'` mengembalikan seluruh tabel.
// Dua huruf cukup untuk membuat hasilnya bermakna tanpa membuat pencarian terasa rewel.
const MinLookupKeyword = 2

// LookupRepo adalah seam ke keempat tabel acuan, SATU portal.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang
// tersedia" alih-alih "apa isi master ini" — dan karena keempat tabelnya dimiliki
// sistem lain. Keduanya tetap dipilih bersama lewat RepoSelector, karena keduanya
// selalu berasal dari koneksi entitas yang sama.
type LookupRepo interface {
	// SearchBusinessSources mencari Sumber Bisnis menurut nama atau ID.
	//
	// Asalnya `RDB List/GetClientName-SQL.xml`:
	//
	//	select id as "Country", clientname as "CountryID" from pooldata.agent
	//	 where replace(clientname,'.','') like '%{ASIS:TempDcol.ATASAN}%'
	//	    or id = {TempDcol.ATASAN}
	//
	// `TempDcol.ATASAN` adalah kata kunci yang sudah di-uppercase dan dibuang titiknya
	// oleh Activity/GetClientName step 3:
	//
	//	@toUpperCase(@replaceAll(Param.name,".",""))
	//
	// Kedua perlakuan itu dipertahankan — membuang titik membuat "PT. ABC" dan "PT ABC"
	// sama-sama ketemu, dan itu memang perilaku yang berguna.
	SearchBusinessSources(ctx context.Context, keyword string) ([]BusinessSource, error)

	// SearchClients mencari tertanggung menurut nama atau ID.
	//
	// Asalnya `RDB List/GetClientName2-SQL.xml`, berbentuk sama persis dengan yang di
	// atas tetapi atas POOLDATA.CLIENT dan kolom NAME.
	SearchClients(ctx context.Context, keyword string) ([]Client, error)

	// ListBanks mengembalikan seluruh bank.
	//
	// Tanpa pencarian: daftarnya pendek dan tidak berubah-ubah, dan layar lama pun
	// memakainya sebagai autocomplete atas satu data page yang dimuat penuh
	// (`DataPage/D_BankGroup-DataPage.xml`).
	ListBanks(ctx context.Context) ([]Bank, error)

	// FindBankByName mencari satu bank menurut namanya; ErrBankNotFound bila tidak ada.
	//
	// Inilah padanan pemeriksaan "Nama bank jangan diketik manual" pada
	// InsertMstAutoClaim_act step 3. Pega memeriksa bahwa autocomplete-nya sempat
	// menghasilkan sebuah kode; di sini diperiksa bahwa nama yang akan TERSIMPAN
	// benar-benar ada di master bank — pemeriksaan yang sama maksudnya, tetapi tidak
	// dapat ditipu dengan mengirim kode karangan bersama nama karangan.
	//
	// Pencocokannya mengabaikan besar-kecil huruf dan spasi tepi; lihat berkas .sql.
	FindBankByName(ctx context.Context, name string) (Bank, error)

	// Committee mengembalikan operator komite yang berwenang atas baris baru.
	//
	// Asalnya `RDB List/GetKomiteAutoKlaim-SQL.xml`:
	//
	//	select operator_id as "UserAdmin" from emailkomite
	//	 where type_business='BONDING' and sts_aktif='1'
	//
	// lalu `Activity/InsertMstAutoClaim_act` step 7 mengambil BARIS PERTAMA saja:
	//
	//	TempInputAutoClaim.FlagASO := TempKomite.pxResults(1).UserAdmin
	//
	// # Dua keanehan yang direplikasi atas keputusan Work Owner 2026-09-19
	//
	//  1. `type_business='BONDING'` tertanam di kueri, padahal modul ini bukan lini
	//     Bonding. Tampak sisa salin-tempel, dan ia persis bentuk hardcode yang D-15
	//     perintahkan menjadi master atau konfigurasi. Dibiarkan apa adanya supaya uji
	//     kesetaraan gerbang 1 tidak melihat selisih; dicatat sebagai utang teknis.
	//  2. Tanpa ORDER BY. Bila barisnya lebih dari satu, penyetujunya ditentukan urutan
	//     yang tidak dijamin apa pun — dan dapat berbeda antar pemanggilan.
	//
	// Teks kosong bukan galat: bila tidak ada baris komite yang aktif, sistem lama pun
	// menyimpan KOMITE kosong. Akibatnya baris itu tidak pernah muncul di tab Komite
	// Approval dan tertahan di Waiting Approval selamanya. Perilakunya dipertahankan,
	// tetapi pemanggil MENCATATNYA di log — lihat usecase.Service.Create.
	Committee(ctx context.Context) (string, error)
}
