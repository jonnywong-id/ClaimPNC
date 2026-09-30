// Package inputacceptation adalah inti modul Acceptation Claim.
//
// # Layar apa ini
//
// Layar akseptasi satu klaim treaty NON-proporsional. Di Pega ia BUKAN butir menu melainkan
// **Flow Action** `InputAcceptation` pada kelas `ASM-FW-GCNMFW-Work-ClaimTreatyNonProp`, yang
// dijalankan ketika pengguna mengklik nomor klaim di Inbox Claim Treaty Non Prop
// (`MENU_ID 55`). Layarnya `Section/InputAcceptation-Section.xml`, berjudul
// "Acceptation Claim".
//
// Karena pintunya satu — nomor klaim di inbox — modul ini tidak punya butir menu sendiri, dan
// memang tidak boleh punya.
//
// # Ia BUKAN salinan modul Outstanding Claim
//
// `outstandingclaim` adalah layar saudaranya untuk treaty PROPORSIONAL, dan keduanya memang
// mirip di layar. Perbandingan properti kedua section: 87 lawan 87, **beririsan 54**. Yang 33
// hanya ada di sini, dan seluruhnya mesin akseptasi — `AcceptedNo`, `AcceptedDate`,
// `AcceptanceStatus`, `AdjusterFee`, `Salvage`, `TPL`, `Layer`, `CNPLimit`, `CNPMDP`,
// `CNPPctReinstate`, `CNPReinstatePremium`, dan seterusnya.
//
// Di bawah permukaan keduanya berbeda pada hal yang menentukan kuerinya:
//
//	                    Outstanding Claim (Prop)   Acceptation Claim (paket ini)
//	kelas objek kerja   Work-ClaimTreaty           Work-ClaimTreatyNonProp
//	kolom JSON          DATA_JSONBLOB              DATA_JSON
//	jumlah grid         10                         13
//	menulis             tidak                      YA — lihat Submit
//
// Menyatukan keduanya karena judulnya mirip akan memaksa satu pembaca dokumen melayani dua
// kolom JSON yang berbeda pada tabel yang sama, dan itulah cacat yang justru sedang
// ditinggalkan (utang teknis 4.6).
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Flow Action/InputAcceptation-FA.xml            pintu masuk; menunjuk section dan pra-aksi
//	Section/InputAcceptation-Section.xml           ~90 isian + 13 grid, lihat section.go
//	Activity/InputAkseptasi_PreAct-Act.xml         pra-aksi; 6 langkah, memanggil ketiganya
//	Activity/InputOutStandingClmTNP_PreAct-Act.xml menyusun ListTreaty & ListLossAllocation
//	Activity/CountLossAllocation_act-Act.xml       alokasi kerugian antar treaty (678 KB)
//	Activity/CountReinstatement_Act-Act.xml        premi reinstatement
//	Activity/SethistoryKlaimTreaty-Act.xml         riwayat klaim treaty
//	RDB List/GetDataPolisNonProp_SQL-SQL.xml       data polis dari `treatyinproduction`
//	RDB List/GetReinsuranceTypeBYName_SQL-SQL.xml  jenis reasuransi dari `REINSURANCETYPE`
//
// Seluruh rantai itu ADA di export — tidak satu pun tergantung pada rule yang hilang. Yang
// TIDAK ada disebut di bagian berikutnya.
//
// # Tiga hal yang TIDAK ada di export, dan akibatnya
//
// **Flow kelas ini.** Folder `Flow/` hanya memuat empat flow, tidak satu pun untuk
// `Work-ClaimTreatyNonProp`. Flow Action ini pun tidak punya pasca-proses. Akibatnya apa yang
// terjadi SESUDAH Submit — tahap berikutnya, ticket yang dipicu, status yang disetel — tidak
// dapat dibaca dari export sama sekali. Lihat Submit.
//
// **Halaman `TreatyInMaster` dan `OfferFacIn`.** Sembilan isian terikat padanya dan tidak
// satu pun rule di export mengisinya — lihat section.go. Modul Outstanding Claim menemukan
// penghalang yang sama pada kedelapan isian yang sama.
//
// **Report Definition `BrowseAdjusterConsultant`.** Ia memasok daftar pilihan Adjuster dan
// Consultant, dan tidak ada di export (`R-16`). Kedua isian tetap dibaca dari dokumen klaim;
// yang belum dapat dibangun adalah daftar pilihannya saat diubah.
//
// # Satu hardcode yang belum pernah tercatat
//
// `InputOutStandingClmTNP_PreAct` langkah 5 dijaga prakondisi `pyWorkPage.pyID=="CLMNP-232"`,
// dan bila benar ia menambahkan baris `FACOUT` ke `ListLossAllocation`. Satu NOMOR KLAIM
// ditanam di dalam rule dan mengubah alokasi kerugiannya. Ia tidak dibawa (`D-15`), dan
// dicatat di PlannedDifferences supaya selisihnya pada klaim itu tidak dilaporkan sebagai
// cacat.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor pustaka
// standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inputacceptation/          aturan modul + seam          ← paket ini
//	inputacceptation/usecase/  orkestrasi: rakit rincian, terima Submit
//	inputacceptation/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inputacceptation/http/     lapisan transport modul ini  — handler, dto, rute
package inputacceptation

import (
	"context"
	"strings"
)

// Detail adalah satu klaim treaty non-proporsional beserta SELURUH isi layar akseptasinya.
//
// Ia dibagi menjadi keadaan objek kerja, isian skalar, dan tiga belas grid — persis pembagian
// yang dipakai section. Bentuk gridnya senarai, bukan peta, karena urutan barisnya bermakna:
// urutan itulah yang dipakai Pega menghitung total di baris terakhirnya.
type Detail struct {
	// ── Keadaan objek kerja, dari DATAPEGA.PC_ASM_FW_GCNMFW_WORK ──

	// ClaimID adalah nomor klaim yang dibaca pengguna — `PYID`, mis. `CLMNP-232`.
	//
	// Ia pula kunci yang dipakai membuka layar ini: alamatnya terbaca orang dan dapat
	// disalin ke percakapan.
	ClaimID string

	// Reference adalah kunci teknis Pega — `PZINSKEY`.
	//
	// Tidak digambar. Ia dibawa karena Submit membutuhkannya untuk menunjuk objek kerja
	// yang benar — nomor klaim saja tidak cukup bila kelak ada dua objek kerja bernomor
	// sama di kelas yang berbeda.
	Reference string

	// StatusWork adalah `PYSTATUSWORK` — status ALUR KERJA Pega ("New", "Pending", …).
	//
	// Ia BUKAN Status Klaim berkode `1134`–`1166` milik master `V_STS_CLAIM`; keduanya
	// konsep berbeda (`D-18`). Nilainya tidak diterjemahkan ke label master mana pun.
	StatusWork string

	// LastUpdateOperator adalah `PXUPDATEOPERATOR` — petugas yang terakhir mengubahnya.
	LastUpdateOperator string

	// ── Isian skalar, dari POOLDATA.JSON_KLAIM.DATA_JSON ──

	// Values memuat setiap isian skalar, dikunci nama isian pada kontrak API.
	//
	// # Kenapa PETA, bukan struct berisi ~90 field
	//
	// Karena isian di layar ini tidak punya tipe yang berbeda-beda: seluruhnya digambar
	// sebagai teks, dan 120 dari 273 selnya bertanda read-only di section. Struct berisi 90
	// field bertipe sama hanya memindahkan daftar yang sama ke tempat kedua — daftar yang
	// sudah ada di section.go dan wajib sama dengannya.
	//
	// Kuncinya DIJAGA: NewDetail menolak kunci yang tidak dikenal section.go, sehingga peta
	// ini tidak dapat menampung isian yang tidak pernah digambar.
	Values map[string]string

	// ── Grid ──

	// Grids memuat setiap grid, dikunci kode grid pada section.go.
	//
	// Grid yang sumbernya kosong di dokumen JSON tetap ADA di peta ini dengan nol baris.
	// Bedanya dengan grid yang tidak ada kuncinya sama sekali bermakna: yang pertama berarti
	// "tidak ada isinya", yang kedua berarti "belum dapat dibaca".
	Grids map[string][]GridRow
}

// GridRow adalah satu baris grid.
//
// Isinya peta dari kunci kolom ke teksnya, dengan alasan yang sama seperti Detail.Values:
// setiap sel digambar sebagai teks, dan bentuk barisnya berbeda-beda antar grid.
type GridRow map[string]string

// Get mengembalikan isian bernama tertentu, atau teks kosong bila tidak ada.
func (d Detail) Get(field string) string {
	if d.Values == nil {
		return ""
	}
	return d.Values[field]
}

// Rows mengembalikan baris sebuah grid, atau senarai kosong bila gridnya tidak terisi.
func (d Detail) Rows(grid string) []GridRow {
	if d.Grids == nil {
		return nil
	}
	return d.Grids[grid]
}

// NewDetail merakit satu rincian dan MENOLAK kunci yang tidak dikenal bentuk layar.
//
// Kunci yang tidak dikenal bukan sekadar kelebihan: ia isian yang tidak akan pernah digambar,
// sehingga keberadaannya di jawaban API hanya menyesatkan pembacanya. Menolaknya di sini
// membuat ketidakcocokan antara pembaca dokumen dan section.go terlihat saat uji berjalan,
// bukan saat pengguna membuka layar.
func NewDetail(
	claimID, reference, statusWork, lastUpdateOperator string,
	values map[string]string,
	gridRows map[string][]GridRow,
) (Detail, error) {
	clean := map[string]string{}
	for key, value := range values {
		if !KnownField(key) {
			return Detail{}, &UnknownFieldError{Field: key}
		}
		clean[key] = value
	}

	cleanGrids := map[string][]GridRow{}
	for code, rows := range gridRows {
		if _, known := FindGrid(code); !known {
			return Detail{}, &UnknownGridError{Grid: code}
		}
		if rows == nil {
			rows = []GridRow{}
		}
		cleanGrids[code] = rows
	}

	return Detail{
		ClaimID:            claimID,
		Reference:          reference,
		StatusWork:         statusWork,
		LastUpdateOperator: lastUpdateOperator,
		Values:             clean,
		Grids:              cleanGrids,
	}, nil
}

// Query adalah permintaan satu rincian yang sudah tervalidasi.
type Query struct {
	// ClaimID adalah nomor klaim, mis. `CLMNP-232`.
	ClaimID string

	// Caller adalah identitas pemanggil.
	//
	// Ia TIDAK menyaring apa pun — lihat NewQuery — tetapi dibawa supaya setiap pembukaan
	// dan setiap Submit dapat dicatat dengan pelakunya.
	Caller Caller
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// ClaimPrefix adalah penanda yang membedakan objek kerja klaim treaty non-proporsional dari
// objek kerja lain di tabel yang sama.
//
// Nilainya sama dengan yang dipakai Inbox Claim Treaty Non Prop, dan dengan jangkar depan
// yang sama. Ia dikumpulkan di sini supaya SQL dan penyimpanan memori tidak dapat berselisih.
//
// Ia PENTING di layar ini, bukan kerapian: `PC_ASM_FW_GCNMFW_WORK` memuat objek kerja seluruh
// jenis klaim. Tanpa penyaring ini, alamat layar dapat diisi nomor klaim PNC biasa dan layar
// akan menggambarnya dengan susunan akseptasi treaty — ~90 isian yang hampir seluruhnya
// kosong, terbaca sebagai klaim yang datanya hilang.
const ClaimPrefix = "CLMNP-"

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa rincian TIDAK disaring menurut pemanggil
//
// Karena di Pega pun tidak. Flow Action ini dijalankan atas assignment yang sedang dibuka, dan
// tidak satu pun dari keempat pra-aksinya punya prakondisi berbasis operator.
//
// Konsekuensinya diterima dengan sadar dan tidak disembunyikan: nomor klaim di sini BERURUTAN
// (`CLMNP-232`, `CLMNP-233`, …), sehingga siapa pun yang sudah masuk dapat membuka akseptasi
// klaim treaty non-prop mana pun di portalnya hanya dengan menaikkan angkanya. Yang membatasi
// bukan modul ini melainkan pemeriksaan kewenangan menu — `TKT-F3-005` — yang belum ada.
//
// Itu sebabnya setiap pembukaan DICATAT beserta pelakunya di lapisan usecase. Pencatatan bukan
// kendali, dan tidak diklaim sebagai kendali; ia yang membuat penyalahgunaannya dapat
// ditelusuri setelah terjadi.
func NewQuery(claimID string, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	clean := strings.TrimSpace(claimID)
	if clean == "" {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldClaimID,
			Message: "Nomor klaim wajib diisi.",
		}})
	}

	if !strings.HasPrefix(strings.ToUpper(clean), ClaimPrefix) {
		return Query{}, NewValidationError([]Violation{{
			Field: FieldClaimID,
			Message: "Layar ini hanya melayani klaim treaty non-proporsional, yang " +
				"nomornya berawalan " + ClaimPrefix + ".",
		}})
	}

	return Query{ClaimID: clean, Caller: cleanCaller}, nil
}

// Repo adalah seam ke akseptasi klaim treaty non-proporsional pada SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat kueri
// (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas, dan
// memang tidak boleh ada.
type Repo interface {
	// Find mengembalikan satu rincian akseptasi.
	//
	// Klaim yang tidak ada menghasilkan ErrNotFound, bukan Detail kosong: Detail kosong
	// terbaca di layar sebagai "klaim tanpa isi", padahal yang benar adalah "klaim tidak
	// ditemukan".
	Find(ctx context.Context, q Query) (Detail, error)

	// Save menyimpan perubahan akseptasi.
	//
	// # Kenapa operasi ini ADA di seam, berbeda dari modul Outstanding Claim
	//
	// Karena Work Owner memutuskan layar ini dibangun PENUH termasuk Submit (2026-09-30).
	// Modul Outstanding Claim sengaja tidak punya operasi tulis sama sekali; di sini ia ada.
	//
	// Yang TIDAK berubah oleh keputusan itu adalah `P-1`: selama masa paralel, tabel objek
	// kerja dan POOLDATA.JSON_KLAIM masih ditulis Pega. Karena itu pengisi seam ini menolak
	// menulis sampai kepemilikan tabelnya benar-benar berpindah — penolakannya dinyatakan
	// lewat ErrWriteNotOwned, bukan didiamkan. Lihat repo/sqlstore.
	Save(ctx context.Context, cmd SaveCommand) error
}

// SaveCommand adalah muatan Submit yang sudah tervalidasi.
//
// Ia dipisah dari Detail dengan sengaja: yang boleh dikirim balik petugas hanyalah isian dan
// sel yang BENAR-BENAR dapat diubah di section — 11 isian skalar dan sebagian sel pada lima
// grid. Menerima Detail utuh berarti menerima 120 sel read-only sebagai muatan tulis, dan
// nilai yang dihitung server akan dapat ditimpa klien tanpa satu pun tanda.
type SaveCommand struct {
	// Query membawa nomor klaim dan pemanggilnya.
	Query Query

	// Reference adalah kunci teknis objek kerja yang ditulis.
	//
	// Ia dibaca ulang dari penyimpanan saat Submit, BUKAN diterima dari klien. Kunci teknis
	// yang datang dari klien adalah kunci yang dapat ditukar klien.
	Reference string

	// Values memuat isian skalar yang diubah, dikunci nama isian pada kontrak API.
	//
	// Hanya kunci yang ada di EditableFields() yang diterima; selebihnya ditolak.
	Values map[string]string

	// Grids memuat baris grid yang diubah, dikunci kode grid.
	//
	// Hanya kolom ber-`Editable` yang diterima. Baris dikirim UTUH per grid — bukan sebagai
	// selisih — karena urutan baris bermakna di layar ini dan selisih tanpa urutan tidak
	// dapat diterapkan kembali dengan pasti.
	Grids map[string][]GridRow
}

// NewSaveCommand membentuk muatan Submit yang sah, atau menyatakan apa yang salah.
//
// Setiap isian dan setiap kolom diperiksa terhadap section.go. Isian yang tidak dapat diubah
// DITOLAK, bukan diabaikan diam-diam: mengabaikannya membuat pengguna mengira perubahannya
// tersimpan, dan pada layar yang menetapkan nilai akseptasi itu kekeliruan bernilai uang.
func NewSaveCommand(
	q Query,
	reference string,
	values map[string]string,
	gridRows map[string][]GridRow,
) (SaveCommand, error) {
	violations := []Violation{}

	editable := map[string]bool{}
	for _, key := range EditableFields() {
		editable[key] = true
	}

	cleanValues := map[string]string{}
	for key, value := range values {
		switch {
		case !KnownField(key):
			violations = append(violations, Violation{
				Field:   key,
				Message: "Isian ini tidak ada di layar akseptasi.",
			})
		case !editable[key]:
			violations = append(violations, Violation{
				Field:   key,
				Message: "Isian ini hanya ditampilkan dan tidak dapat diubah dari layar.",
			})
		default:
			cleanValues[key] = value
		}
	}

	cleanGrids := map[string][]GridRow{}
	for code, rows := range gridRows {
		grid, known := FindGrid(code)
		if !known {
			violations = append(violations, Violation{
				Field:   code,
				Message: "Tabel ini tidak ada di layar akseptasi.",
			})
			continue
		}

		allowed := map[string]bool{}
		for _, column := range grid.Columns {
			if column.Editable {
				allowed[column.Key] = true
			}
		}

		cleanRows := make([]GridRow, 0, len(rows))
		for index, row := range rows {
			cleanRow := GridRow{}
			for key, value := range row {
				if !allowed[key] {
					violations = append(violations, Violation{
						Field: code + "." + key,
						Message: "Kolom ini hanya ditampilkan dan tidak dapat diubah " +
							"dari layar (baris " + itoa(index+1) + ").",
					})
					continue
				}
				cleanRow[key] = value
			}
			cleanRows = append(cleanRows, cleanRow)
		}
		cleanGrids[code] = cleanRows
	}

	if len(violations) > 0 {
		return SaveCommand{}, NewValidationError(violations)
	}

	return SaveCommand{
		Query:     q,
		Reference: strings.TrimSpace(reference),
		Values:    cleanValues,
		Grids:     cleanGrids,
	}, nil
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke paket domain.
//
// Ia dipakai HANYA untuk nomor baris pada pesan pelanggaran, dan nomor baris selalu positif.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan nilai akseptasi
// dan pembagian reasuransi satu badan hukum kepada petugas badan hukum lain tanpa satu pun
// pesan galat (`R-20`) — dan pada layar yang juga MENULIS, ia berarti menulisi klaim milik
// entitas yang salah.
type RepoSelector func(portalAlias string) (Repo, error)

// PlannedDifferences adalah selisih terhadap sistem lama yang DIPUTUSKAN, bukan cacat.
//
// Ia dikirim ke layar dan ditampilkan kepada pengguna. Selisih yang hanya tercatat di komentar
// akan dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan layar baru
// dengan Pega berdampingan.
var PlannedDifferences = []string{
	"Sembilan isian pada blok Treaty Information SELALU kosong dan ditandai belum " +
		"tersedia. Kedelapan isian .TreatyInMaster.* dan satu isian .OfferFacIn.* berada di " +
		"halaman TERSENDIRI pada objek kerja, bukan di .ClaimData, sehingga tidak ikut " +
		"tersimpan di dokumen JSON klaim — dan tidak satu pun rule di export mengisinya. " +
		"Modul Outstanding Claim menemukan penghalang yang sama pada isian yang sama.",

	"Prakondisi berisi NOMOR KLAIM tidak dibawa. InputOutStandingClmTNP_PreAct langkah 5 " +
		"dijaga pyWorkPage.pyID==\"CLMNP-232\" dan menambahkan satu baris FACOUT ke daftar " +
		"alokasi kerugian hanya untuk klaim itu. Satu nomor klaim yang ditanam di dalam rule " +
		"adalah hardcode yang D-15 tetapkan menjadi master data; di sini ia dihilangkan, " +
		"sehingga klaim CLMNP-232 akan menampilkan satu baris alokasi lebih sedikit " +
		"daripada di Pega.",

	"Daftar pilihan Adjuster dan Consultant belum dapat digambar. Report Definition " +
		"BrowseAdjusterConsultant yang memasoknya tidak ada di export (R-16). Nilai yang " +
		"sudah tersimpan tetap ditampilkan; yang belum ada adalah daftar pilihannya.",

	"Kedua kotak pencari InputData.CARI31 dan InputData.CARI32 tidak dibawa. Keduanya " +
		"pencari ID adjuster dan ID konsultan pada halaman SEMENTARA InputData, bukan bagian " +
		"klaim: nilainya hanya ada selama layar terbuka dan tidak pernah tersimpan.",

	"Submit belum menulis ke basis data. Selama Pega dan sistem baru berjalan " +
		"berdampingan, tabel objek kerja dan POOLDATA.JSON_KLAIM hanya boleh ditulis satu " +
		"sistem (P-1), dan keduanya masih dimiliki Pega. Perpindahan kepemilikannya menempuh " +
		"D-63 — permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA — dan sampai itu " +
		"terjadi Submit menolak dengan alasan alih-alih menyimpan diam-diam.",

	"Apa yang terjadi SESUDAH Submit belum dapat ditiru. Flow untuk kelas " +
		"Work-ClaimTreatyNonProp tidak ada di export dan Flow Action ini tidak punya " +
		"pasca-proses, sehingga tahap berikutnya, ticket yang dipicu, dan status yang disetel " +
		"tidak diketahui. Yang dibangun adalah PENYIMPANAN isiannya; perpindahan tahap " +
		"menunggu flow-nya tiba dari Tim Pega.",
}
