// Package inboxxol adalah inti modul Inbox XOL.
//
// # Layar apa ini
//
// Menu `MENU_ID 53` "Inbox XOL" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk harness
// `Inbox_XOL_Harness`. Ia berada di bawah grup menu `INBOX` (`MENU_ID_LEADER 2`) dan
// diotorisasi untuk grup `IT` pada POOLDATA.M_OTORISASI_PNC.
//
// XOL — Excess of Loss — adalah treaty reasuransi non-proporsional yang menanggung
// kerugian di atas batas tertentu (`CONTEXT.md`). Layar ini **bukan** layar klaim
// perorangan: ia menghitung akumulasi klaim satu tahun perjanjian XOL, dikelompokkan
// menurut Tanggal Kejadian dan Penyebab Kerugian, lalu memperlihatkan pemberitahuan
// PLA/DLA yang sudah diterbitkan kepada para reasuradur beserta status persetujuannya.
//
// # Dua tab, dijaga peran — bukan empat tab sejajar
//
// `Section/InboxClaimXOL-Section.xml` memuat dua tab teratas, masing-masing dengan
// kondisi tampil yang membaca access group:
//
//	Inbox XOL        AccessGroup.pyAccessGroup=='GCNMFW:PncPICTeknik'
//	Inbox XOL Komite AccessGroup.pyAccessGroup=='GCNMFW:CaseManager'
//
// Tiga judul yang tampak seperti tab — "Generated DLA PLA XOL", "Cari Data DLA PLA XOL",
// dan "INSERT DOL DAN COL" — sebenarnya **tombol** di dalam tab pertama; ketiganya
// terdaftar sebagai `pyButtonLabel`, bukan sebagai tab.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/Inbox_XOL_Harness-Harness.xml        pembungkus layar, tiga defer-load
//	Section/InboxClaimXOL-Section.xml            2 tab, 6 grid beserta kolomnya, 7 tombol
//	Section/Sec_Detail_claim_XOL-Section.xml     grid rincian di balik satu baris klaim
//	Activity/GetClaimXOL-Act.xml                 perhitungan berlapis tab 1
//	Activity/GetShowDataMasterXOL-Act.xml        daftar master XOL
//	Activity/GetDataXOLKomite-Act.xml            dua grid tab Komite
//	Activity/BrowseDataXOLPLADLAGenerated-Act.xml pencarian PLA/DLA terbit
//	Activity/AddingPilihanMasterXOL-Act.xml      pemilihan master XOL di modal
//	RDB List/GetDataXOL_Calulation-SQL.xml       akumulasi per DOL + Cause of Loss
//	RDB List/GetDataXOLPerBusiness-SQL.xml       rincian per group business
//	RDB List/GetDataTrytyInwardFromUploadData-SQL.xml  rincian treaty inward
//	RDB List/BrowseAllDataXOL_PLA-SQL.xml        daftar PLA/DLA terbit
//	RDB List/GetDataXOLForKomiteApprove-SQL.xml  antrean persetujuan PLA/DLA
//	RDB List/GetDataMasterXOLForKomiteApprove-SQL.xml antrean persetujuan master XOL
//	RDB List/GetDataMasterXOL-SQL.xml            master XOL (kelas Data-ClaimData)
//	Database/GET_GROUPBUSINESS_XOL.fnc           perakitan daftar group business
//	Database/GETCURRENCYSTANDARD.fnc             pencarian kurs standar
//	Report Definition/SelectVDCauseOfLoss_RD-RD.xml  daftar Penyebab Kerugian
//
// # Kenapa nama isian di sini tidak mirip nama properti Pega
//
// Karena nama properti di layar ini menyesatkan lebih parah daripada modul mana pun
// sebelumnya — bukan singkatan tidak lazim, melainkan nama yang berarti hal lain:
//
//	.ASMFull      berarti Tanggal Kejadian      .AcceptedNo berarti Penyebab Kerugian
//	.Currency     berarti Nilai Outstanding     .CurrencyID berarti Nilai Akseptasi
//	.CurrencyName berarti Tahun XOL             .BranchOfBank berarti Group Business
//	.City         berarti ID XOL                .CityID     berarti Nama XOL
//	.CoverInsKey  berarti Nama Reasuradur       .CABANG     berarti Nama Layer
//	.ClaimFrom    berarti Tahun                 .PNCSearch  berarti Kurs
//	.ERROR        berarti Share Percent         .HASIL5     berarti Remark
//	.NOTE         berarti Alamat Surel          .ResponseCode berarti Nomor PLA/DLA
//
// Pemetaan lengkapnya ada di repo/sqlstore/inboxxol.sql dan di `docs/peta-penamaan.md`.
// Membawa nama itu ke sistem baru berarti mewariskan kekacauan yang justru menjadi alasan
// migrasi (`03-CURRENT-ARCHITECTURE.md` §4.2); yang dipakai di sini adalah padanan Inggris
// dari `CONTEXT.md` sesuai `D-19` dan `D-80`.
//
// # Modul ini TIDAK MENULIS apa pun
//
// Keputusan Work Owner 2026-09-20: tahap ini membaca saja. Empat tabel yang disentuh
// aksi tulis sistem lama — `XOL_TABLE_ALL_KLAIM`, `T_PLA_XOL`, `T_DLA_XOL`, dan
// `MST_XOL_PNC` — tetap dimiliki Pega sepenuhnya selama masa paralel, sejalan dengan
// `P-1`: satu tabel hanya boleh ditulis satu sistem.
//
// Akibatnya tombol Insert DOL dan COL, Simpan, dan Approval **digambar tetapi tidak
// menulis**, dan ditandai belum tersedia beserta alasannya. Menyembunyikannya akan
// membuat pengguna mengira fiturnya hilang; menghidupkannya akan melanggar `P-1`.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxxol/          aturan modul + seam          ← paket ini
//	inboxxol/usecase/  orkestrasi: rakit tab, cari, unduh
//	inboxxol/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxxol/http/     lapisan transport modul ini  — handler, dto, rute
package inboxxol

import (
	"context"
	"strings"
)

// AdviceType membedakan Preliminary Loss Advice dari Definite Loss Advice.
//
// Keduanya tersimpan di TABEL YANG BERBEDA — `POOLDATA.T_PLA_XOL` dan
// `POOLDATA.T_DLA_XOL` — dengan susunan kolom yang sama persis. Sistem lama memilih
// tabelnya dengan merangkai nama tabel ke dalam teks SQL
// (`Activity/BrowseDataXOLPLADLAGenerated-Act.xml`); di sini tipe ini yang memilih kueri,
// sehingga tidak ada satu pun nama tabel yang berasal dari masukan pengguna.
type AdviceType string

const (
	// AdvicePLA — Preliminary Loss Advice, pemberitahuan NILAI ESTIMASI kepada
	// koasuransi/reasuransi (`CONTEXT.md`).
	AdvicePLA AdviceType = "PLA"

	// AdviceDLA — Definite Loss Advice, pemberitahuan NILAI AKSEPTASI.
	AdviceDLA AdviceType = "DLA"
)

// Valid menyatakan tipe pemberitahuan dikenal.
func (t AdviceType) Valid() bool {
	return t == AdvicePLA || t == AdviceDLA
}

// ParseAdviceType membaca tipe dari teks, tanpa peduli huruf besar-kecil.
//
// Teks kosong menghasilkan tipe kosong dan `false`. Pemanggil yang memang mengizinkan
// "kedua tipe" memeriksa kekosongannya sendiri — memberi arti khusus pada nilai kosong
// di dalam fungsi ini akan membuat salah ketik tidak terbedakan dari "tidak disaring".
func ParseAdviceType(raw string) (AdviceType, bool) {
	candidate := AdviceType(strings.ToUpper(strings.TrimSpace(raw)))
	if !candidate.Valid() {
		return "", false
	}
	return candidate, true
}

// MasterXOL adalah satu perjanjian XOL: satu tahun, satu kurs, dan sekumpulan group
// business yang ditanggungnya.
//
// Ia dipakai TIGA grid berbeda di layar yang sama, dan ketiganya menampilkan kolom yang
// berbeda dari baris yang sama:
//
//	PILIH MASTER XOL   Tahun · Kurs · Group Business            (modal tab 1)
//	Rincian klaim      Type Master · ID Master · Tahun · Kurs   (Sec_Detail_claim_XOL)
//	DATA MASTER XOL    ID XOL · Nama XOL · Tahun XOL · Kurs     (tab Komite)
//
// Satu tipe melayani ketiganya. Memecahnya menjadi tiga akan membuat satu baris master
// yang sama punya tiga bentuk yang harus dijaga tetap sejalan.
type MasterXOL struct {
	// ID — `MST_XOL_PNC.ID`. Di grid Komite bernama `.City`, di grid rincian `.BranchID`.
	ID string

	// Name — `NAMA`. Di grid Komite bernama `.CityID`, di grid rincian `.UserName`.
	Name string

	// Year — `TAHUN`, tahun perjanjian XOL.
	//
	// Teks, bukan angka: ia dibandingkan sebagai teks di seluruh kueri sistem lama
	// (`TAHUN = '2024'`), dan kolomnya tidak punya DDL yang dapat diperiksa (`R-08`).
	// Mengubahnya menjadi angka di sini berarti menebak tipe kolom yang belum terlihat.
	Year string

	// MinLimit adalah batas layer TERENDAH perjanjian ini — `MIN(LIMIT)` atas
	// `POOLDATA.MST_XOL_LAYER`, dalam mata uang perjanjian.
	//
	// Ia kolom "Min Limit" pada grid layar rincian; dikalikan kurs ia menjadi
	// "Min Limit IDR". Sumbernya `RDB List/GetDataMasterXOL-SQL.xml:11`, alias
	// `AIDiterima` dan `ClaimAmount`.
	MinLimit float64

	// ExchangeRate — `KURSVALUE`, kurs yang dipakai mengubah nilai klaim rupiah menjadi
	// mata uang perjanjian. Nilai grid "OS Value (USD)" adalah nilai rupiah DIBAGI angka
	// ini (`Activity/GetClaimXOL-Act.xml`, `.Currency := @toDecimal(.Currency)/local.kurs`).
	ExchangeRate float64

	// Type — `TYPEXOL`. Di grid rincian berjudul "Type Master".
	Type string

	// BusinessGroups adalah group business yang ditanggung perjanjian ini.
	//
	// Di sistem lama ia dirakit fungsi basis data `GET_GROUPBUSINESS_XOL`, yang
	// mengembalikan SATU TEKS berisi daftar yang dipisah koma — dan untuk tipe `'id'`
	// teks itu sudah dikutip satu per satu supaya dapat disisipkan mentah ke dalam
	// klausa `IN (...)`.
	//
	// Di sini ia SENARAI, bukan teks. Dua sebab, dan keduanya mengikat:
	//
	//  1. `D-02` melarang pemanggilan stored procedure maupun function basis data;
	//     logikanya naik ke Go.
	//  2. Teks berisi nilai yang sudah dikutip hanya berguna untuk dirangkai ke dalam
	//     SQL — persis celah yang `08-TECHNICAL-STRATEGY.md` §4.3 tutup. Senarai
	//     memaksa setiap nilai menempuh parameter binding.
	BusinessGroups []BusinessGroup

	// CommitteeStatus — `STSKOMITE`. Nilai `'0'` berarti menunggu persetujuan komite;
	// itulah penyaring grid DATA MASTER XOL pada tab Komite.
	CommitteeStatus string

	// CommitteeNote — `REMARKKOMITE`, catatan dari komite.
	CommitteeNote string

	// PIC — `PIC`, operator yang mengajukan master ini ke komite.
	PIC string

	// PICEmail dicari dari `POOLDATA.MST_USER_TEKNIK.EMAIL` berdasarkan PIC.
	PICEmail string

	// PICNote — `REMARKPIC`, catatan dari pengaju.
	PICNote string
}

// BusinessGroup adalah satu group business yang ditanggung sebuah perjanjian XOL.
type BusinessGroup struct {
	// ID — `MST_XOL_BUSINESS.IDBUSINESS`.
	ID string

	// Name — `POOLDATA.BUSINESSGROUP.NOTE` untuk ID tersebut.
	//
	// KOSONG bila group business-nya tidak ada di master. `GET_GROUPBUSINESS_XOL`
	// menggantinya dengan teks "TREATY INWARD" dalam keadaan itu — perilaku yang
	// dipertahankan, tetapi penggantiannya terjadi di Go, bukan di basis data. Lihat
	// DisplayName.
	Name string
}

// TreatyInwardLabel adalah nama yang dipakai ketika sebuah group business tidak punya
// keterangan di master.
//
// Sumbernya `Database/GET_GROUPBUSINESS_XOL.fnc`: `IF PNC_XOL.note is null then
// PNC_XOL.note := 'TREATY INWARD'`. Ia BUKAN nama yang diketik siapa pun — ia penanda
// bahwa barisnya berasal dari treaty inward, bukan dari group business milik sendiri.
const TreatyInwardLabel = "TREATY INWARD"

// DisplayName adalah nama yang dibaca pengguna.
func (g BusinessGroup) DisplayName() string {
	if strings.TrimSpace(g.Name) == "" {
		return TreatyInwardLabel
	}
	return g.Name
}

// BusinessGroupNames adalah nama seluruh group business, dirangkai seperti yang dilihat
// pengguna di kolom "Group Business".
//
// Pemisahnya `", "` mengikuti `GET_GROUPBUSINESS_XOL` yang memakai `',  '` — dua spasi
// setelah koma. Spasi gandanya TIDAK dibawa: ia artefak perakitan teks, bukan aturan
// bisnis, dan tidak ada satu pun kueri yang memecah teks itu kembali.
func (m MasterXOL) BusinessGroupNames() string {
	if len(m.BusinessGroups) == 0 {
		return ""
	}
	names := make([]string, 0, len(m.BusinessGroups))
	for _, group := range m.BusinessGroups {
		names = append(names, group.DisplayName())
	}
	return strings.Join(names, ", ")
}

// BusinessGroupIDs adalah kode seluruh group business perjanjian ini.
//
// Dipakai sebagai penyaring akumulasi klaim. Ia senarai supaya setiap kode menempuh
// parameter binding masing-masing.
func (m MasterXOL) BusinessGroupIDs() []string {
	if len(m.BusinessGroups) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.BusinessGroups))
	for _, group := range m.BusinessGroups {
		if clean := strings.TrimSpace(group.ID); clean != "" {
			ids = append(ids, clean)
		}
	}
	return ids
}

// AwaitingCommittee menyatakan master ini masih menunggu persetujuan komite.
//
// Penyaringnya `STSKOMITE='0'`, dibaca dari
// `RDB List/GetDataMasterXOLForKomiteApprove-SQL.xml`.
const committeeStatusPending = "0"

// AwaitingCommittee menyatakan master ini masih menunggu persetujuan komite.
func (m MasterXOL) AwaitingCommittee() bool {
	return strings.TrimSpace(m.CommitteeStatus) == committeeStatusPending
}

// ClaimSummary adalah satu baris grid "DATA XOL BASED ON DOL AND COL".
//
// Satu baris = satu Tanggal Kejadian × satu Penyebab Kerugian, dengan nilai klaim
// seluruh group business perjanjian itu sudah dijumlahkan.
type ClaimSummary struct {
	// MasterID adalah perjanjian XOL yang MELAHIRKAN baris ini.
	//
	// Ia perlu dibawa per baris karena grid menggabungkan hasil SELURUH perjanjian —
	// `Activity/GetClaimXOL-Act.xml` step 4 me-loop `MstXOL.pxResults` dan meng-APPEND
	// hasil tiap perjanjian ke satu daftar. Tanpa penanda ini, rincian di balik sebuah
	// baris tidak tahu tahun, kurs, dan group business mana yang berlaku baginya.
	//
	// Sistem lama membawanya sebagai `CaseIDCashier` dan `Notes` pada baris yang
	// di-append — nama yang tidak ada hubungannya dengan isinya. Di sini namanya
	// dibetulkan; perannya tidak.
	MasterID string

	// LossDate adalah Tanggal Kejadian — kolom `DOL` pada
	// `POOLDATA.XOL_TABLE_ALL_KLAIM`.
	//
	// TEKS, bukan tanggal, dan itu bukan kelalaian. Kolomnya memang menyimpan teks:
	// `GetDataXOL_Calulation` membacanya dengan `to_char(to_date(a.DOL,'dd/mm/yyyy'),'yyyy')`,
	// yang hanya masuk akal bila isinya teks berformat `dd/mm/yyyy`. Mengubahnya menjadi
	// time.Time di sini berarti menebak bahwa seluruh baris historis memang berformat
	// itu — dugaan yang tidak dapat diperiksa tanpa DDL (`R-08`).
	LossDate string

	// CauseOfLoss adalah Penyebab Kerugian — `CAUSEOFLOSS`, berisi DESKRIPSI-nya, bukan
	// kodenya. Kueri lama menyaringnya dengan
	// `CAUSEOFLOSS in (select DESCRIPTION from POOLDATA.V_D_CAUSE_OF_LOSS)`.
	CauseOfLoss string

	// BusinessGroup adalah nama group business perjanjian, disalin dari master.
	//
	// Ia TIDAK datang dari kueri akumulasi: `GetClaimXOL` mengisinya dari
	// `TempMst.Currency`, yaitu nama group business master XOL yang sedang dipilih.
	// Seluruh baris dalam satu perjanjian karena itu bernilai sama.
	BusinessGroup string

	// OutstandingValue adalah nilai Outstanding — `SUM(OSVALUE)` dibagi kurs master.
	//
	// Satuannya mengikuti mata uang perjanjian; judul kolomnya di layar lama tertulis
	// "OS Value (USD)".
	OutstandingValue float64

	// AcceptedValue adalah nilai Akseptasi — `SUM(AKSEPVALUE)` dibagi kurs master.
	AcceptedValue float64
}

// BusinessBreakdown adalah satu baris grid rincian di balik sebuah ClaimSummary.
//
// Sumbernya DUA kueri yang hasilnya disatukan menjadi satu daftar
// (`Activity/GetClaimXOL-Act.xml`):
//
//	GetDataXOLPerBusiness             klaim milik sendiri, per group business
//	GetDataTrytyInwardFromUploadData  klaim treaty inward, satu baris gabungan
//
// Keduanya dibedakan Source, bukan disatukan diam-diam: yang kedua berasal dari tabel
// lain (`T_CLAIM_INWARD_XOL`) dan nilainya sudah dikonversi mata uang di sumbernya.
type BusinessBreakdown struct {
	// BusinessGroup adalah nama group business — `POOLDATA.BUSINESSGROUP.NOTE`, atau
	// teks tetap "Treaty Inward" untuk baris treaty.
	BusinessGroup string

	// BusinessGroupID — `businessgroupid`. KOSONG untuk baris treaty inward.
	BusinessGroupID string

	// ClaimCount adalah jumlah klaim berbeda — `COUNT(DISTINCT claimno)` untuk klaim
	// sendiri, `COUNT(DISTINCT COMPANYNAME)` untuk treaty inward.
	//
	// Kedua hitungan itu MENGHITUNG HAL YANG BERBEDA — yang satu nomor klaim, yang lain
	// nama perusahaan — tetapi keduanya ditampilkan di kolom yang sama berjudul "Total
	// Klaim". Perbedaan itu direplikasi apa adanya (`P-5`) dan dicatat di sini supaya
	// tidak terbaca sebagai cacat saat uji kesetaraan.
	ClaimCount int

	// OutstandingValue — `SUM(os_value)` dibagi kurs master.
	OutstandingValue float64

	// AcceptedValue — `SUM(aksep_value)` dibagi kurs master.
	AcceptedValue float64

	// Source menyatakan baris ini berasal dari klaim sendiri atau treaty inward.
	Source BreakdownSource

	// RateMissing menyatakan kurs mata uang baris treaty inward tidak ditemukan.
	//
	// Ia HANYA berlaku untuk baris treaty inward, dan ia menutup cacat yang diputuskan
	// diperbaiki di `D-49` butir 5: `Database/GETCURRENCYSTANDARD.fnc:22` mengembalikan
	// `1` ketika kurs tidak ada, sehingga nilai valuta asing diperlakukan satu banding
	// satu terhadap rupiah tanpa satu pun tanda.
	//
	// Di sini nilainya TIDAK dikalikan satu diam-diam: barisnya ditandai, dan layar
	// menyatakan kursnya tidak tersedia alih-alih menampilkan angka yang salah.
	RateMissing bool
}

// BreakdownSource menyatakan asal sebuah baris rincian.
type BreakdownSource string

const (
	// SourceOwnBusiness — klaim milik sendiri, dari `POOLDATA.T_CLAIM_XOL`.
	SourceOwnBusiness BreakdownSource = "bisnis"

	// SourceTreatyInward — klaim treaty inward, dari `POOLDATA.T_CLAIM_INWARD_XOL`.
	SourceTreatyInward BreakdownSource = "treaty"
)

// Advice adalah satu baris PLA atau DLA yang sudah diterbitkan kepada satu reasuradur.
//
// Satu Tanggal Kejadian + Penyebab Kerugian dapat menghasilkan banyak baris: satu per
// reasuradur per layer, dan satu lagi setiap kali pemberitahuannya direvisi.
type Advice struct {
	// Number adalah nomor yang dibaca pengguna — `NO_PLADLAXOL`, ditambah nomor revisi
	// bila revisinya bukan `'0'`.
	//
	// Perakitannya ada di kueri lama sebagai `CASE REVISI WHEN '0' THEN NO_PLADLAXOL
	// ELSE NO_PLADLAXOL || ' / ' || REVISI END`. Di sini nomor dan revisinya dibawa
	// TERPISAH dan dirakit di Go — pemformatan tampilan tidak dikerjakan basis data
	// (`08-TECHNICAL-STRATEGY.md` §4.3).
	Number string

	// Revision — `REVISI`. Teks `'0'` berarti belum pernah direvisi.
	Revision string

	// ReinsurerID — `IDREAS`.
	ReinsurerID string

	// ReinsurerName — `NAMAREAS`. Di grid berjudul "Nama Insurance".
	ReinsurerName string

	// LayerID — `IDLAYER`.
	LayerID string

	// LayerName — `NAMALAYER`.
	LayerName string

	// Year — `TAHUN`, tahun perjanjian XOL.
	Year string

	// CauseOfLoss — `CAUSEOFLOSS`.
	CauseOfLoss string

	// ExchangeRate — `KURS`, kurs yang berlaku saat pemberitahuan diterbitkan.
	ExchangeRate float64

	// SharePercent — `PERCENT`, bagian reasuradur ini.
	//
	// Angka, bukan teks: kueri lama merangkainya menjadi `PERCENT || ' %'` di dalam SQL,
	// dan satuan yang menempel pada nilai membuatnya tidak dapat dijumlahkan maupun
	// diurutkan. Tanda persennya ditambahkan saat ditampilkan.
	SharePercent float64

	// Email adalah alamat surel reasuradur.
	//
	// Kueri lama mencarinya berjenjang: kolom `EMAIL` pada baris pemberitahuan, dan bila
	// kosong barulah `POOLDATA.T_REINSURER.EMAIL`. Kedua langkah itu dipertahankan.
	//
	// # Satu penimpaan yang TIDAK dibawa
	//
	// `Activity/BrowseDataXOLPLADLAGenerated-Act.xml` menimpa alamat ini dengan satu
	// alamat tetap apabila operator yang membuka layar bernama tertentu. Itu hardcode
	// identitas di jalur produksi, persis yang `D-15` larang dan `D-67` tetapkan tidak
	// dibawa ke sistem baru. Tidak ada padanannya di modul ini.
	Email string

	// Country adalah negara reasuradur — `POOLDATA.T_REINSURER.COUNTRY`.
	Country string

	// Remark — `REMARKREAS`, catatan dari reasuradur.
	Remark string

	// ApprovalNote — `REMARKAPPROVE`, catatan dari yang menyetujui.
	ApprovalNote string

	// PICNote — `REMARKPIC`, catatan dari PIC teknis.
	PICNote string

	// Limit — `LIMIT_XOL`, batas layer yang berlaku pada pemberitahuan ini.
	Limit string

	// ApprovalStatus — `STATUSAPPROVE`. Nilai `'0'` berarti belum disetujui; itulah
	// penyaring antrean persetujuan pada tab Komite.
	ApprovalStatus string

	// MasterID — `IDMASTER`, menunjuk kembali ke MST_XOL_PNC.ID.
	MasterID string

	// InputBy — `USERINPUT`, operator yang menerbitkan pemberitahuan.
	InputBy string

	// InputByEmail dicari dari `POOLDATA.MST_USER_TEKNIK.EMAIL` berdasarkan InputBy.
	InputByEmail string

	// IssuedOn adalah tanggal pemberitahuan diterbitkan — `TGLINSERT`, dalam bentuk
	// teks `dd/mm/yyyy`.
	//
	// Ia TIDAK ditampilkan di grid mana pun, dan tetap dibawa karena antrean persetujuan
	// mengelompokkan menurut kolom ini (`MAX(TO_CHAR(TGLINSERT,'dd/mm/yyyy'))`). Tanpa
	// membawanya, penyimpanan memori tidak dapat merakit antrean yang sama dengan SQL —
	// dan dua penyimpanan yang merakit antrean berbeda berarti layar berperilaku
	// berbeda tergantung ada tidaknya Oracle.
	//
	// Bentuknya teks, bukan time.Time, supaya cacat pengurutan teks pada antrean
	// persetujuan dapat direplikasi apa adanya. Lihat ApprovalItem.LastInsertedAt.
	IssuedOn string

	// Type membedakan PLA dari DLA. Ia tidak datang dari kolom mana pun — ia ditentukan
	// TABEL mana yang dibaca.
	Type AdviceType
}

// DisplayNumber adalah nomor pemberitahuan seperti yang dibaca pengguna.
//
// Revisi `'0'`, kosong, atau hanya spasi diperlakukan sama: belum pernah direvisi.
func (a Advice) DisplayNumber() string {
	revision := strings.TrimSpace(a.Revision)
	if revision == "" || revision == "0" {
		return a.Number
	}
	return a.Number + " / " + revision
}

// ApprovalItem adalah satu baris antrean persetujuan pemberitahuan pada tab Komite.
//
// Satu baris = satu tahun × satu penyebab kerugian × satu tipe, bukan satu pemberitahuan.
// Kueri lama mengelompokkannya dengan `GROUP BY tahun, causeofloss` dan mengambil
// `MAX(TGLINSERT)` — sehingga komite menyetujui SEKUMPULAN pemberitahuan sekaligus,
// bukan satu per satu.
type ApprovalItem struct {
	// Year — `TAHUN`. Di grid berjudul "Date Of Loss", dan judul itu MENYESATKAN:
	// isinya tahun perjanjian, bukan tanggal kejadian. Judulnya direplikasi apa adanya
	// (`P-5`); namanya di kode dibetulkan.
	Year string

	// CauseOfLoss — `CAUSEOFLOSS`.
	CauseOfLoss string

	// Type membedakan PLA dari DLA. Kueri lama menyatukan keduanya dengan `UNION` dan
	// menandai asalnya lewat kolom literal `'DLA'` dan `'PLA'`.
	Type AdviceType

	// LastInsertedAt adalah tanggal penerbitan terakhir dalam kelompok ini —
	// `MAX(TO_CHAR(TGLINSERT,'dd/mm/yyyy'))`.
	//
	// Teks, dengan alasan yang sama seperti ClaimSummary.LossDate: yang dikelompokkan
	// adalah hasil `TO_CHAR`, sehingga nilai terbesarnya adalah teks terbesar — bukan
	// tanggal terbaru. Pada format `dd/mm/yyyy` keduanya BERBEDA: `31/01/2024` lebih
	// besar daripada `01/12/2024` sebagai teks. Cacat itu direplikasi (`P-5`) dan
	// dicatat di sini supaya tidak terbaca sebagai cacat baru.
	LastInsertedAt string
}

// CauseOfLoss adalah satu Penyebab Kerugian yang dapat dipilih.
//
// Sumbernya report definition `SelectVDCauseOfLoss_RD` pada kelas
// `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS`, yang membaca `POOLDATA.V_D_CAUSE_OF_LOSS`.
type CauseOfLoss struct {
	// ID — `D_COL_ID`.
	ID string

	// Description — `DESCRIPTION`. Inilah yang tersimpan di kolom `CAUSEOFLOSS` pada
	// tabel klaim XOL — bukan kodenya.
	Description string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// ClaimFilter memilih akumulasi klaim milik satu perjanjian XOL.
type ClaimFilter struct {
	// Year adalah tahun perjanjian. Kueri lama membandingkannya dengan
	// `to_char(to_date(DOL,'dd/mm/yyyy'),'yyyy')`.
	Year string

	// BusinessGroupIDs adalah kode group business perjanjian itu. KOSONG berarti
	// perjanjiannya tidak menanggung group business mana pun, dan hasilnya kosong —
	// BUKAN "seluruh group business". Lihat Empty.
	BusinessGroupIDs []string
}

// Empty menyatakan penyaring ini tidak akan pernah mengembalikan baris.
//
// Ia diperiksa SEBELUM kueri dijalankan. Tanpa itu, klausa `IN ()` kosong akan menjadi
// SQL yang tidak sah di Oracle dan mengembalikan seluruh baris di sebagian dialek lain —
// dua perilaku yang sama-sama salah, dan yang kedua membocorkan data perjanjian lain.
func (f ClaimFilter) Empty() bool {
	return strings.TrimSpace(f.Year) == "" || len(f.BusinessGroupIDs) == 0
}

// BreakdownFilter memilih rincian di balik satu baris ClaimSummary.
type BreakdownFilter struct {
	// LossDate adalah Tanggal Kejadian dalam bentuk teks `dd/mm/yyyy`, persis seperti
	// tersimpan.
	LossDate string

	// CauseOfLoss adalah DESKRIPSI penyebab kerugian, bukan kodenya.
	CauseOfLoss string

	// BusinessGroupIDs adalah kode group business perjanjian.
	BusinessGroupIDs []string
}

// Empty menyatakan penyaring ini tidak akan pernah mengembalikan baris klaim sendiri.
//
// Treaty inward TIDAK memakai group business, sehingga ia tetap dapat dijalankan meski
// senarainya kosong — lihat TreatyEmpty.
func (f BreakdownFilter) Empty() bool {
	return f.TreatyEmpty() || len(f.BusinessGroupIDs) == 0
}

// TreatyEmpty menyatakan penyaring ini tidak cukup untuk mencari baris treaty inward.
func (f BreakdownFilter) TreatyEmpty() bool {
	return strings.TrimSpace(f.LossDate) == "" || strings.TrimSpace(f.CauseOfLoss) == ""
}

// AdviceFilter memilih pemberitahuan PLA/DLA yang sudah diterbitkan.
type AdviceFilter struct {
	// Year adalah tahun perjanjian — kolom `TAHUN`.
	Year string

	// CauseOfLoss adalah deskripsi penyebab kerugian — kolom `CAUSEOFLOSS`.
	CauseOfLoss string

	// Type memilih tabel yang dibaca. WAJIB terisi: tidak ada satu pun kueri sistem lama
	// yang membaca kedua tabel sekaligus, dan menyatukannya di sini akan menghasilkan
	// daftar yang tidak pernah ada padanannya di sistem lama.
	Type AdviceType
}

// Empty menyatakan penyaring ini belum cukup untuk dijalankan.
func (f AdviceFilter) Empty() bool {
	return strings.TrimSpace(f.Year) == "" ||
		strings.TrimSpace(f.CauseOfLoss) == "" ||
		!f.Type.Valid()
}

// Repo adalah seam ke penyimpanan Inbox XOL SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di
// tingkat kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut
// entitas, dan memang tidak boleh ada.
//
// # Dua operasi menulis, sisanya membaca
//
// Batas yang ditetapkan Work Owner 2026-09-20 berbunyi "modul ini membaca saja", dan
// alasannya — kepemilikan tabel masih di Pega selama masa paralel (`P-1`) — tetap berlaku
// untuk tabel yang tidak disebut di sini. Dua tabel dikecualikan atas permintaan Work
// Owner, masing-masing setelah seluruh rantai rule-nya ada di export:
//
//	T_SALVAGE_MBU         UploadSalvageMBU  unggahan "Upload MBU Salvage"
//	XOL_TABLE_ALL_KLAIM   InsertDolCol      modal "INSERT DOL DAN COL"
//
// Pengecualiannya per OPERASI, bukan pencabutan larangan: operasi tulis berikutnya tetap
// harus ditambahkan di sini secara sadar, dan uji di repo/sqlstore/query_test.go
// menggagalkan kueri tulis yang tidak terdaftar.
//
// # Yang TETAP tidak ditulis, dan kenapa itu penting
//
// T_PLA_XOL, T_DLA_XOL, dan MST_XOL_PNC. Selama Pega masih menulis XOL_TABLE_ALL_KLAIM,
// dua sistem kini menulis satu tabel — `P-1` dilanggar secara sadar, bukan tanpa sengaja.
// Akibat yang harus disadari: baris yang sama dapat disisipkan dua kali bila aksi yang
// sama dijalankan di kedua aplikasi, dan gridnya akan menjumlahkan keduanya.
type Repo interface {
	// ListMasterXOL mengembalikan seluruh perjanjian XOL beserta group business-nya,
	// diurutkan menurut ID.
	ListMasterXOL(ctx context.Context) ([]MasterXOL, error)

	// SummarizeClaims mengembalikan akumulasi klaim per Tanggal Kejadian dan Penyebab
	// Kerugian. Nilainya masih dalam RUPIAH — pembagian dengan kurs terjadi di usecase.
	SummarizeClaims(ctx context.Context, filter ClaimFilter) ([]ClaimSummary, error)

	// BreakdownByBusiness mengembalikan rincian klaim milik sendiri per group business.
	// Nilainya masih dalam rupiah.
	BreakdownByBusiness(ctx context.Context, filter BreakdownFilter) ([]BusinessBreakdown, error)

	// BreakdownTreatyInward mengembalikan rincian klaim treaty inward.
	//
	// Nilainya SUDAH dikonversi ke mata uang perjanjian di dalam kueri, karena setiap
	// barisnya punya mata uang sendiri yang tidak terbawa ke hasil. Ia karena itu TIDAK
	// ikut dibagi kurs master di usecase.
	BreakdownTreatyInward(ctx context.Context, filter BreakdownFilter) ([]BusinessBreakdown, error)

	// SearchAdvice mengembalikan pemberitahuan PLA atau DLA yang sudah diterbitkan.
	SearchAdvice(ctx context.Context, filter AdviceFilter) ([]Advice, error)

	// ListPendingAdviceApproval mengembalikan antrean persetujuan pemberitahuan —
	// seluruh kelompok yang `STATUSAPPROVE`-nya masih `'0'`.
	ListPendingAdviceApproval(ctx context.Context) ([]ApprovalItem, error)

	// ListPendingMasterApproval mengembalikan antrean persetujuan master XOL — seluruh
	// perjanjian yang `STSKOMITE`-nya masih `'0'`.
	ListPendingMasterApproval(ctx context.Context) ([]MasterXOL, error)

	// SummarizeBusiness mengembalikan grid "Summary Data XOL" — group business yang
	// menanggung klaim pada satu Tanggal Kejadian dan Penyebab Kerugian.
	SummarizeBusiness(ctx context.Context, filter SummaryFilter) ([]SummaryBusiness, error)

	// ListClaims mengembalikan grid "No Klaim" — satu baris per klaim pada satu Tanggal
	// Kejadian dan Penyebab Kerugian, klaim sendiri dan treaty inward disatukan.
	ListClaims(ctx context.Context, filter ClaimListFilter) ([]ClaimListItem, error)

	// ExportClaimDetail mengembalikan isi berkas "Export to Excel" satu baris rincian.
	ExportClaimDetail(ctx context.Context, filter ExportFilter) (ExportTable, error)

	// ListCauseOfLoss mengembalikan daftar Penyebab Kerugian yang dapat dipilih.
	ListCauseOfLoss(ctx context.Context) ([]CauseOfLoss, error)

	// CurrencyIDByName mencari ID mata uang dari namanya, untuk unggahan MBU Salvage.
	//
	// Mata uang yang tidak dikenal dijawab teks kosong TANPA galat: barisnya ditolak
	// beserta sebabnya, dan satu nama salah ketik tidak menggugurkan seluruh berkas.
	CurrencyIDByName(ctx context.Context, name string) (string, error)

	// UploadSalvageMBU menyisipkan baris hasil "Upload MBU Salvage".
	//
	// Ia satu-satunya operasi TULIS modul ini. Seluruh barisnya masuk dalam satu
	// transaksi — berkas tersimpan seluruhnya, atau tidak sama sekali.
	UploadSalvageMBU(ctx context.Context, rows []SalvageInsert) error

	// InsertDolCol menyisipkan baris hasil modal "INSERT DOL DAN COL".
	//
	// Seluruh barisnya — satu per group business perjanjian — masuk dalam SATU transaksi.
	// Sistem lama mengulang RDB-Save tanpa transaksi, sehingga kegagalan di group business
	// ketiga meninggalkan dua baris tersimpan dan sisanya tidak, tanpa cara mengetahui di
	// mana batasnya (`D-68`).
	InsertDolCol(ctx context.Context, rows []DolColInsert) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti membaca nilai klaim dan
// nama reasuradur satu badan hukum dari basis data badan hukum lain tanpa satu pun pesan
// galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// SummaryBusiness adalah satu baris grid "Summary Data XOL" pada layar rincian.
//
// Satu baris = satu group business yang menanggung klaim pada tanggal kejadian dan
// penyebab kerugian yang sedang dibuka. Ia TIDAK membawa nilai uang: kueri lama
// (`GetBusinessnameXOLForSummerry`) hanya mengembalikan kode dan namanya, dan grid di
// layar lama pun hanya berkolom "Business Name".
type SummaryBusiness struct {
	// BusinessGroupID — `BUSINESSGROUPID` pada klaim sendiri, `BUSINESSID` pada treaty
	// inward. Keduanya disatukan UNION oleh kuerinya.
	BusinessGroupID string

	// BusinessGroupName — nama dari `POOLDATA.BUSINESS`, atau teks tetap
	// "Treaty Inward" untuk baris treaty.
	BusinessGroupName string
}

// SummaryFilter menyaring grid "Summary Data XOL".
//
// Hanya dua kolom, dan keduanya berasal dari BARIS yang sedang dibuka di grid
// "DATA XOL BASED ON DOL AND COL" — bukan dari perjanjian XOL. Group business TIDAK ikut
// menyaring: grid ini justru yang menyebutkan group business mana saja yang terlibat.
type SummaryFilter struct {
	// LossDate — Tanggal Kejadian berformat `dd/mm/yyyy`.
	LossDate string

	// CauseOfLoss — DESKRIPSI penyebab kerugian, bukan kodenya.
	CauseOfLoss string
}

// Empty menyatakan penyaring belum lengkap, sehingga kuerinya tidak perlu dijalankan.
func (f SummaryFilter) Empty() bool {
	return strings.TrimSpace(f.LossDate) == "" || strings.TrimSpace(f.CauseOfLoss) == ""
}

// ClaimListItem adalah satu baris grid "No Klaim" pada layar rincian.
//
// Sumbernya DUA: klaim milik sendiri dari `T_CLAIM_XOL`, dan klaim treaty inward dari
// `T_CLAIM_INWARD_XOL` — disatukan UNION oleh `BrowserT_claim_xolDesc`.
type ClaimListItem struct {
	// ClaimNo — nomor klaim. Untuk baris treaty inward ia NAMA PERUSAHAAN, bukan nomor:
	// klaim inward tidak punya nomor klaim ASM, dan kueri lama memang mengisinya dengan
	// `COMPANYNAME`.
	ClaimNo string

	// CurrencyName — nama mata uang dari `POOLDATA.CURRENCY`, bukan kodenya.
	CurrencyName string

	// Source membedakan kedua asal baris, sama seperti pada BusinessBreakdown.
	Source BreakdownSource

	OutstandingValue float64
	AcceptedValue    float64

	// RateMissing menandai baris treaty inward yang kursnya tidak ditemukan. Nilainya
	// TIDAK ditampilkan saat ini benar (`D-48`).
	RateMissing bool
}

// ClaimListFilter menyaring grid "No Klaim".
//
// Hanya dua kolom, dan keduanya milik BARIS yang sedang dibuka. Group business TIDAK ikut
// menyaring: grid ini memuat seluruh klaim pada tanggal dan penyebab kerugian itu, lintas
// group business.
//
// Mata uang dasar konversi treaty inward TIDAK ada di sini — ia milik repo
// (`DefaultBaseCurrencyID`), sama seperti pada BreakdownFilter.
type ClaimListFilter struct {
	// LossDate — Tanggal Kejadian berformat `dd/mm/yyyy`.
	LossDate string

	// CauseOfLoss — DESKRIPSI penyebab kerugian.
	CauseOfLoss string
}

// Empty menyatakan penyaring belum lengkap, sehingga kuerinya tidak perlu dijalankan.
func (f ClaimListFilter) Empty() bool {
	return strings.TrimSpace(f.LossDate) == "" || strings.TrimSpace(f.CauseOfLoss) == ""
}

// ExportFilter menyaring isi "Export to Excel" pada layar rincian.
//
// BusinessGroupID menentukan KUERI MANA yang dipakai, bukan sekadar menyaring —
// `GenerateDetailClaimBusinessXOL` bercabang atasnya:
//
//	'10004'		MBU		export_mbu
//	'treaty'	treaty inward	export_treaty_inward
//	selainnya	group biasa	export_per_business
type ExportFilter struct {
	// LossDate — Tanggal Kejadian berformat `dd/mm/yyyy`.
	LossDate string

	// CauseOfLoss — DESKRIPSI penyebab kerugian.
	CauseOfLoss string

	// BusinessGroupID — kode group business, atau teks `treaty` untuk baris inward.
	BusinessGroupID string
}

// ExportBusinessGroupTreaty adalah nilai khusus BusinessGroupID untuk baris treaty
// inward. Ia BUKAN kode group business — `GenerateDetailClaimBusinessXOL` memakainya
// sebagai penanda cabang (`Param.grpbzid=="treaty"`).
const ExportBusinessGroupTreaty = "treaty"

// ExportBusinessGroupMBU adalah kode group business MBU, yang punya kuerinya sendiri.
const ExportBusinessGroupMBU = "10004"

// Empty menyatakan penyaring belum lengkap, sehingga kuerinya tidak perlu dijalankan.
func (f ExportFilter) Empty() bool {
	return strings.TrimSpace(f.LossDate) == "" ||
		strings.TrimSpace(f.CauseOfLoss) == "" ||
		strings.TrimSpace(f.BusinessGroupID) == ""
}

// ExportTable adalah isi berkas unduhan: satu baris judul, lalu barisnya.
//
// # Kenapa teks, bukan tipe bernama
//
// Karena ketiga varian export punya kolom yang BERBEDA — 42, 14, dan 13 — dan satu-satunya
// pembacanya adalah penulis CSV. Menjadikannya tiga struct berarti 69 field yang tidak
// pernah dibaca kode mana pun, dan tiga tempat baru yang bisa menyimpang dari daftar
// kolom di activity.
type ExportTable struct {
	// Headers adalah judul kolom yang dibaca PENGGUNA, dari `CSVPropHeaders`.
	Headers []string

	// Rows sudah berurutan sesuai Headers.
	Rows [][]string
}

// Jenis berkas contoh unggahan, sesuai `Param.jenis` tombol "Format ..." di layar lama
// (`Section/DetailValueClaimXOL-Section.xml:7094` dan `:7480`).
const (
	UploadTemplateMBUSalvage = "1"
	UploadTemplateInward     = "2"
)

// uploadTemplates memuat judul kolom kedua berkas contoh unggahan.
//
// Keduanya disalin apa adanya dari `CSVPropHeaders` di
// `Activity/DownloadFormatForAllUploadingFile-Act.xml` — baris 301 untuk `jenis=1` dan
// 499 untuk `jenis=2`.
//
// # Kenapa hanya judul kolom, tanpa satu pun baris
//
// Karena sistem lama pun demikian. Activity-nya memanggil `pxConvertResultsToCSV` atas
// `Code-Pega-List` yang TIDAK pernah diisi, sehingga berkas yang turun berisi baris judul
// saja. Ia memang berkas contoh untuk diisi pengguna, bukan ekspor data.
//
// `CSVProperties` di activity itu sengaja TIDAK disalin: isinya properti ber-alias yang
// tidak mencerminkan kolomnya (`CaseID;NamaSurveyor` berulang tiga belas kali untuk
// `jenis=2`), dan karena daftar barisnya kosong, properti itu tidak pernah terbaca.
var uploadTemplates = map[string][]string{
	UploadTemplateMBUSalvage: {
		"ClaimNo", "DateOfLoss", "Currency", "SALVAGEOS", "SALVAGEAKSEP", "CauseOfLoss",
	},
	UploadTemplateInward: {
		"pyCompany", "Currency", "IsReservedClaim", "OwnRiskValue", "QsTreaty",
		"DeductibleValue", "ClaimAmount", "ASMShare", "ClaimAmountShareASM",
		"PaidClaimAmountShare", "BalanceClaimASMShare", "DateOfLoss", "CauseOfLoss",
	},
}

// uploadTemplateNames adalah nama berkasnya, dari elemen `FileName` di activity yang sama.
var uploadTemplateNames = map[string]string{
	UploadTemplateMBUSalvage: "Format Upload Salvage MBU",
	UploadTemplateInward:     "Format Upload Inward",
}

// UploadTemplate mengembalikan berkas contoh unggahan beserta namanya.
//
// Jenis yang tidak dikenal ditolak, bukan dijawab berkas kosong: berkas contoh tanpa
// judul kolom akan dibuka pengguna, diisi, lalu ditolak pembacanya — kegagalan yang
// muncul jauh dari sebabnya.
func UploadTemplate(kind string) (ExportTable, string, bool) {
	headers, known := uploadTemplates[strings.TrimSpace(kind)]
	if !known {
		return ExportTable{}, "", false
	}
	return ExportTable{Headers: headers}, uploadTemplateNames[strings.TrimSpace(kind)], true
}
