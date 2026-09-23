package komite

import (
	"sort"
	"strings"

	"claim-pnc/internal/platform/money"
)

// DefaultNonMBUBandBoundary adalah nilai yang memisahkan pita 1 dari pita 2 pada lini
// Non-MBU: sampai Rp 100.000.000 masuk pita "1", di atasnya masuk pita "2".
//
// # Dari mana angka ini, tepatnya
//
// Ia ADA DI DALAM RULE PEGA, bukan kesimpulan maupun tebakan.
// `Activity/SetEmailKomite-Act.xml` menetapkannya sebagai satu ekspresi:
//
//	tempAdj.AcceptedNo := @If(tempAdj.ConvertAdjustmentValue > 100000000, 2, 1)
//
// dan `tempAdj.AcceptedNo` itulah yang dipakai menyaring
// `trim(TYPE_KOMITE) = trim({tempAdj.AcceptedNo})` di
// `RDB List/EmailKomiteBerjenjang_sql-SQL.xml`. `D-52` kemudian menegaskannya —
// "untuk komite sampai 100 Jt pakai type_komite=1" — sehingga keduanya sepakat.
//
// # Angka ini BERBEDA per entitas, dan itu wajib disadari
//
// Rule yang sama memuat kembarannya untuk entitas bermata uang dolar:
//
//	tempAdj.AcceptedNo := @If(tempAdj.ConvertAdjustmentValue > 7000, 2, 1)
//
// Jadi batas pitanya **Rp 100.000.000 pada entitas rupiah dan USD 7.000 pada entitas
// SMI**. Memakai 100.000.000 untuk seluruh portal akan salah pada portal SMI dengan
// selisih sekitar 14.000 kali lipat — lihat NonMBUBandBoundarySMI.
//
// # Kenapa angkanya masih di kode, padahal D-15 melarang nilai bisnis di-hardcode
//
// Ia TIDAK dibaca mesin penjenjangan. Ia hanya nilai bawaan yang membentuk Policy,
// dan Policy itu DIPASOK DARI LUAR — sehingga menggantinya kelak dengan baris master
// `F-4` per portal tidak menyentuh satu baris pun aturan di berkas ini.
const DefaultNonMBUBandBoundary = 100_000_000

// NonMBUBandBoundarySMI adalah kembaran USD dari batas di atas, untuk entitas SMI.
//
// # Bagaimana ia bekerja di sistem lama: TIMPA, bukan cabang
//
// Keduanya tidak berdampingan sebagai dua cabang yang setara. Di
// `Activity/SetEmailKomite-Act.xml` yang terjadi berurutan:
//
//  1. Satu step TANPA SYARAT APA PUN menetapkan
//     `tempAdj.AcceptedNo := @If(ConvertAdjustmentValue > 100000000, 2, 1)`.
//  2. Step berikutnya, bersyarat `TempGetApp.LSC_ID == "SMI"`, MENIMPANYA dengan
//     `@If(ConvertAdjustmentValue > 7000, 2, 1)`.
//
// Jadi Rp 100.000.000 adalah nilai bawaan yang berlaku bagi semua entitas, dan USD 7.000
// adalah penimpa yang hanya berlaku bagi SMI. `AcceptedNo` kemudian menjadi penyaring
// `TYPE_KOMITE` pada `EmailKomiteBerjenjang_sql`.
//
// # Bahaya yang melekat pada cara entitas dikenali
//
// `LSC_ID` tidak datang dari konfigurasi melainkan dari pencocokan NAMA SERVER:
// `GetLinkAppClaim` menjalankan `BrowseAPPName_sql`, yaitu
// `SELECT APP FROM pooldata.DB_LINK_PEGA WHERE APPIP LIKE '%<nama server>%'`, dan bila
// pencocokan itu gagal ia jatuh ke nilai bawaan `"ASM"`.
//
// Akibatnya: server SMI yang berganti nama membuat klaim berdenominasi dolar diperiksa
// terhadap ambang rupiah — meleset sekitar 14.000 kali lipat, tanpa satu pun galat.
// Inilah yang `D-75` hapus dengan mengganti pengenalan berbasis nama server menjadi
// portal.
//
// Rasio 100.000.000 ÷ 7.000 ≈ 14.285 adalah KURS YANG DIBEKUKAN KE DALAM KODE, bukan
// kurs dari master mata uang. Angkanya wajib dikonfirmasi ulang sebelum portal SMI
// dilayani — kurs itu sudah tidak mencerminkan kurs hari ini.
//
// Konstanta ini BELUM DIPAKAI di jalur mana pun; ia menunggu kebijakan disambungkan ke
// portal aktif (`TKT-F6-002`).
const NonMBUBandBoundarySMI = 7_000

// Pita yang berlaku pada lini Non-MBU, sesuai isi kolom TYPE_KOMITE di master.
const (
	BandLower = "1"
	BandUpper = "2"
)

// Mode menyatakan CARA penyetuju dipilih, dan ia ditentukan per PORTAL — bukan per lini.
//
// Di sistem lama pemilihannya ada di
// `Activity/SetListComiteeClaimPerObjAdj-Act.xml`:
//
//	bila TempGetApp.LSC_ID == "SIMASNET"  → SetEmailKomiteSimasnet
//	selain itu                            → SetEmailKomite
//
// dan `LSC_ID` dibaca dari `POOLDATA.DB_LINK_PEGA` dengan mencocokkan nama server.
// `D-75` mengganti pengenalan berbasis nama server itu dengan **portal**, sehingga di
// sistem baru mode ini melekat pada portal yang sedang melayani.
type Mode string

const (
	// ModeCumulative adalah perilaku Non-MBU, PA, Travel, dan Bonding: SETIAP jenjang
	// yang ambang bawahnya sudah terlampaui ikut menyetujui.
	ModeCumulative Mode = "kumulatif"

	// ModeSingleApprover adalah perilaku entitas Simasnet: dipilih TEPAT SATU penyetuju,
	// diacak di antara jenjang terendah yang memenuhi syarat, dan operator yang
	// menginput DIKECUALIKAN.
	//
	// # Kenapa entitas itu bekerja begitu
	//
	// Work Owner menjelaskan 2026-09-19: di Simasnet, **komitenya adalah tim klaim itu
	// sendiri**. Yang mengajukan dan yang menyetujui berasal dari kumpulan orang yang
	// sama, sehingga mengeluarkan si pengaju bukan tambahan melainkan INTI aturannya —
	// tanpa itu seseorang akan menyetujui pekerjaannya sendiri.
	//
	// Pengacakannya mengikuti dari situ: bila tiga orang sama-sama berwenang, mengambil
	// satu secara acak menyebar beban alih-alih selalu menjatuhkannya pada orang yang
	// sama.
	//
	// Perhatikan bahwa alasan itu TIDAK berlaku pada entitas lain, tempat komitenya
	// adalah kelompok senior tersendiri — bukan tim klaim. Lihat catatan pada
	// excludeApplicant.
	//
	// Ia bukan variasi kecil dari ModeCumulative melainkan aturan yang berbeda sama
	// sekali, dan buktinya utuh di satu kueri:
	//
	//	SELECT * FROM (
	//	  SELECT … FROM POOLDATA.EMAILKOMITE
	//	   WHERE STS_ADJ='1' AND STS_AKTIF='1'
	//	     AND trim(TYPE_BUSINESS)=trim(:lini)
	//	     AND LIMIT_BOTTOM <= :nilai
	//	     AND OPERATOR_ID != :operator_yang_menginput
	//	   ORDER BY degree, dbms_random.value)
	//	 WHERE rownum = 1
	//
	// `RDB List/EmailKomiteBerjenjangSimasnet_sql-SQL.xml`, dengan kedua potongan
	// dinamisnya diisi `Activity/SetEmailKomiteSimasnet-Act.xml`:
	// `"AND OPERATOR_ID!='" + OperatorID.pyUserIdentifier + "'"` dan `"WHERE rownum = 1"`.
	//
	// Perhatikan pula: mode ini TIDAK menyaring TYPE_KOMITE sama sekali, sehingga tidak
	// mengenal pita.
	ModeSingleApprover Mode = "satu-penyetuju"
)

// Randomizer memilih satu dari sekian kemungkinan yang setara.
//
// # Kenapa ini seam, bukan panggilan langsung ke math/rand
//
// Pengacakan adalah satu-satunya sumber ketidakpastian di seluruh modul ini, dan
// membiarkannya menembus lapisan domain akan membuat aturan penjenjangan tidak dapat
// diuji — hasil yang berbeda tiap kali dijalankan tidak dapat dibandingkan dengan apa pun.
//
// Dengan seam, pengujian memakai pemilih tetap dan produksi memakai pemilih sungguhan,
// sementara ATURANNYA — siapa saja yang layak, dan berapa yang dipilih — tetap satu dan
// dapat diperiksa.
type Randomizer interface {
	// Pick mengembalikan indeks dalam rentang [0, count). Pemanggil menjamin count > 0.
	Pick(count int) int
}

// FixedRandomizer selalu memilih indeks yang sama. Dipakai pengujian, dan sebagai
// perilaku bawaan yang aman bila tidak ada randomizer yang dipasok.
type FixedRandomizer int

// Pick mengembalikan indeks tetap, dijaga tetap berada di dalam rentang.
func (f FixedRandomizer) Pick(count int) int {
	if count <= 0 {
		return 0
	}
	index := int(f) % count
	if index < 0 {
		index += count
	}
	return index
}

// BandPolicy menyatakan bagaimana sebuah lini memilih pita nilai sebelum jenjang
// diakumulasi.
type BandPolicy struct {
	// Boundary adalah nilai TERTINGGI yang masih masuk pita bawah. Nilai tepat di batas
	// masuk pita bawah; satu satuan di atasnya sudah masuk pita atas.
	Boundary money.Money

	Lower string
	Upper string
}

// Select mengembalikan pita untuk sebuah nilai klaim.
func (b BandPolicy) Select(value money.Money) string {
	if value <= b.Boundary {
		return b.Lower
	}
	return b.Upper
}

// Policy mengumpulkan aturan yang berlaku pada satu portal.
//
// Lini yang TIDAK ada di dalam Bands tidak mengenal pita sama sekali, dan jenjangnya
// diakumulasi langsung atas seluruh baris lini tersebut.
type Policy struct {
	// Mode kosong diperlakukan sebagai ModeCumulative.
	Mode Mode

	Bands map[BusinessLine]BandPolicy
}

// EffectiveMode mengembalikan mode, dengan kumulatif sebagai bawaan.
func (p Policy) EffectiveMode() Mode {
	if p.Mode == "" {
		return ModeCumulative
	}
	return p.Mode
}

// DefaultPolicy mengembalikan kebijakan portal rupiah: kumulatif, dan HANYA Non-MBU
// yang memakai pita.
//
// # Kenapa pita hanya Non-MBU, dan kenapa ini tidak boleh diseragamkan
//
// `D-70` membatasinya setelah diperiksa terhadap isi master. Dihitung dari
// `Database/emailkomite.csv`, memberlakukan pemilihan pita ke semua lini akan
// menghasilkan:
//
//	PA Rp 5.000.000        1 penyetuju  →  0 penyetuju   klaim mandek
//	PA Rp 75.000.000       3 penyetuju  →  2 penyetuju
//	PA Rp 150.000.000      4 penyetuju  →  2 penyetuju
//	Travel Rp 150.000.000  3 penyetuju  →  0 penyetuju   klaim mandek
//
// Sebabnya terlihat di data: pada PA, kolom TYPE_KOMITE berselang-seling 2 · 1 · 1 · 2
// menaiki tangga — ia membedakan PA reguler dari PA TKI, bukan pita nilai.
func DefaultPolicy() Policy {
	return Policy{
		Mode: ModeCumulative,
		Bands: map[BusinessLine]BandPolicy{
			BusinessLineNonMBU: {
				Boundary: money.FromRupiah(DefaultNonMBUBandBoundary),
				Lower:    BandLower,
				Upper:    BandUpper,
			},
		},
	}
}

// SimasnetPolicy mengembalikan kebijakan portal Simasnet: satu penyetuju, diacak,
// dan TANPA pita sama sekali.
//
// Ketiadaan pita bukan kelalaian — kueri Simasnet memang tidak menyaring TYPE_KOMITE.
func SimasnetPolicy() Policy {
	return Policy{Mode: ModeSingleApprover}
}

// Approver adalah satu orang yang harus menyetujui, beserta tempatnya dalam antrean.
type Approver struct {
	// Order adalah posisi menyetujui, 1 sampai jumlah penyetuju. Ia SELALU berurutan
	// tanpa lompatan, berbeda dari Tier.
	Order int

	// Tier adalah DEGREE dari master. Ia dapat berulang dan dapat melompat; ia bukan
	// penomoran antrean.
	Tier int

	Name       string
	OperatorID string

	// LowerBound adalah ambang yang membuat orang ini ikut. Dikirim ke layar supaya
	// pengguna dapat melihat ALASAN seseorang masuk daftar, bukan hanya hasilnya.
	LowerBound money.Money

	// Absent dibawa apa adanya dari master. Ia TIDAK menyaring siapa pun — lihat
	// catatan pada Threshold.Absent.
	Absent bool

	// ThresholdID menunjuk baris master asalnya, supaya hasil hitungan dapat ditelusuri
	// balik ke datanya saat ada yang meragukannya.
	ThresholdID string
}

// Tiering adalah hasil perhitungan untuk satu nilai klaim pada satu lini.
type Tiering struct {
	Value        money.Money
	BusinessLine BusinessLine
	Mode         Mode

	// Band terisi HANYA untuk lini yang memakainya. Kosong berarti lini ini memang
	// tidak mengenal pita — bukan berarti pitanya gagal dihitung.
	Band string

	// UsesBand membedakan kedua keadaan di atas secara tegas, supaya layar tidak
	// perlu menebak arti Band yang kosong.
	UsesBand bool

	// Approvers berurutan sesuai antrean menyetujui. Panjangnya adalah jumlah jenjang.
	Approvers []Approver

	// Candidates hanya terisi pada ModeSingleApprover: seluruh orang yang LAYAK dipilih
	// pada jenjang terendah, sebelum satu di antaranya diacak.
	//
	// Ia dilaporkan karena yang dapat diperiksa pada mode ini bukan SIAPA yang terpilih
	// — itu acak — melainkan apakah KUMPULAN yang layak sudah benar. Tanpa ini, hasil
	// mode tersebut tidak dapat diuji sama sekali.
	Candidates []Approver

	// ExcludedApplicant menyebut operator yang tidak ikut dipertimbangkan karena
	// dialah yang mengajukan. Kosong berarti tidak ada yang dikecualikan.
	ExcludedApplicant string

	// Excluded memuat orang yang BENAR-BENAR dikeluarkan oleh pengecualian di atas.
	//
	// Ia dibedakan dari ExcludedApplicant dengan sengaja: yang pertama menyatakan
	// siapa yang diminta dikecualikan, yang kedua menyatakan apakah permintaan itu
	// berakibat. Keduanya berbeda — penginput yang bukan anggota komite tidak mengubah
	// apa pun, dan layar tidak boleh menyiratkan sebaliknya.
	//
	// Isinya penting justru saat hasilnya kosong: bila seluruh jenjang tersingkir,
	// inilah satu-satunya keterangan yang menjelaskan KENAPA klaim itu tidak punya
	// penyetuju.
	Excluded []Approver

	// AmbiguousOrder menyala pada ModeCumulative bila ada dua penyetuju atau lebih
	// ber-DEGREE SAMA.
	//
	// Kueri sistem lama mengurutkan dengan `ORDER BY DEGREE` saja, sehingga saat DEGREE
	// seri, urutannya ditentukan basis data dan dapat berubah antar eksekusi. Keadaan
	// itu BENAR-BENAR ADA pada master yang berlaku — Non-MBU pita 1 memiliki dua baris
	// ber-DEGREE 1 (ID 7 dan ID 1).
	//
	// Modul ini mengurutkan secara pasti, dan menyalakan penanda ini supaya perbedaan
	// urutan terhadap Pega pada kasus seri tidak terbaca sebagai cacat saat uji
	// kesetaraan.
	AmbiguousOrder bool
}

// TierCount adalah banyaknya persetujuan yang dibutuhkan.
func (t Tiering) TierCount() int { return len(t.Approvers) }

// NoApprovers berarti tidak satu pun jenjang cocok untuk nilai ini.
//
// Ia dilaporkan sebagai KEADAAN, bukan galat, karena pemanggil yang berbeda menanganinya
// berbeda: layar simulasi menampilkannya sebagai peringatan yang mencolok, sedangkan
// alur klaim kelak harus menolak melanjutkan. Menjadikannya galat akan memaksa layar
// simulasi menampilkan kegagalan padahal yang terjadi adalah temuan.
func (t Tiering) NoApprovers() bool { return len(t.Approvers) == 0 }

// Options adalah keterangan tambahan yang hanya dibutuhkan sebagian mode.
type Options struct {
	// Applicant adalah OPERATOR_ID orang yang mengajukan. Ia DIKECUALIKAN dari calon
	// penyetuju pada KEDUA mode.
	//
	// Di sistem lama pengecualian ini hanya ada pada jalur Simasnet. Work Owner
	// menetapkan 2026-09-18 bahwa ia diberlakukan ke seluruh entitas — perubahan
	// perilaku yang disengaja, dan akibatnya dijelaskan pada excludeApplicant.
	Applicant string

	// Randomizer dipakai ModeSingleApprover. Bila nil, dipakai FixedRandomizer(0) sehingga
	// hasilnya dapat diulang — pilihan yang aman, karena diam-diam menjadi acak jauh
	// lebih berbahaya daripada diam-diam menjadi tetap.
	Randomizer Randomizer
}

// Determine menghitung siapa saja yang harus menyetujui sebuah nilai klaim, memakai mode
// yang ditetapkan kebijakan dan tanpa keterangan tambahan.
func Determine(
	value money.Money,
	line BusinessLine,
	thresholds []Threshold,
	policy Policy,
) (Tiering, error) {
	return DetermineWith(value, line, thresholds, policy, Options{})
}

// DetermineWith menghitung penyetuju dengan keterangan tambahan.
//
// # Langkah yang berlaku untuk KEDUA mode
//
//  1. Ambil hanya baris yang merupakan jenjang persetujuan — STS_AKTIF dan STS_ADJ
//     keduanya menyala, dan DEGREE-nya bukan nol.
//  2. Ambil hanya baris lini yang diminta.
//  3. Ambil hanya baris yang LowerBound <= nilai klaim.
//
// # Yang berbeda sesudahnya
//
// ModeCumulative — bila lini memakai pita, pitanya dipilih dari nilai klaim dan akumulasi
// berjalan DI DALAM pita itu saja. Seluruh baris yang tersisa menjadi penyetuju,
// diurutkan menurut DEGREE.
//
// ModeSingleApprover — tidak ada pita. Operator yang menginput dikeluarkan, lalu dari
// jenjang TERENDAH yang tersisa dipilih TEPAT SATU secara acak.
//
// LIMIT_TOP tidak dipakai pada satu langkah pun; pemakaiannya hanya di integrity.go.
//
// # Kenapa pengurutannya lebih pasti daripada Pega
//
// Kueri lama memakai `ORDER BY DEGREE` saja. Pada master yang berlaku, Non-MBU pita 1
// memiliki DUA baris ber-DEGREE 1, sehingga urutan keduanya diserahkan kepada basis data
// dan dapat berbeda antar eksekusi.
//
// Di sini seri dipecahkan secara pasti: LowerBound lebih kecil lebih dulu — yang secara
// bisnis memang masuk akal, karena jenjang berambang lebih rendah adalah yang menyetujui
// lebih awal — lalu ID sebagai pemecah terakhir. Ini TIDAK mengubah SIAPA yang menyetujui
// pada mode kumulatif, hanya URUTANNYA saat seri, dan setiap kali hal itu terjadi ia
// dilaporkan lewat AmbiguousOrder.
func DetermineWith(
	value money.Money,
	line BusinessLine,
	thresholds []Threshold,
	policy Policy,
	opts Options,
) (Tiering, error) {
	line = line.Normalized()

	if err := NewValidationError(ValidateInput(value, line)); err != nil {
		return Tiering{}, err
	}

	result := Tiering{Value: value, BusinessLine: line, Mode: policy.EffectiveMode()}

	// Langkah 1 dan 2 — jenjang persetujuan pada lini yang diminta.
	lineExists := false
	sameLine := make([]Threshold, 0, len(thresholds))
	for _, t := range thresholds {
		t = t.Normalized()
		if t.BusinessLine != line {
			continue
		}
		lineExists = true
		if !t.IsApprovalTier() {
			continue
		}
		sameLine = append(sameLine, t)
	}
	if !lineExists {
		// Dibedakan dari "ada tetapi tidak ada yang cocok": yang ini salah ketik atau
		// lini baru yang belum diisi, dan pemanggil pantas diberi tahu bedanya.
		return Tiering{}, ErrUnknownBusinessLine
	}

	if result.Mode == ModeCumulative {
		// Pita hanya berlaku pada mode kumulatif; kueri Simasnet tidak menyaring
		// TYPE_KOMITE sama sekali.
		if rule, banded := policy.Bands[line]; banded {
			result.UsesBand = true
			result.Band = rule.Select(value)

			filtered := sameLine[:0:0]
			for _, t := range sameLine {
				if t.CommitteeType == result.Band {
					filtered = append(filtered, t)
				}
			}
			sameLine = filtered
		}
	}

	// Langkah 3 — akumulasi menurut batas bawah.
	matched := sameLine[:0:0]
	for _, t := range sameLine {
		if t.LowerBound <= value {
			matched = append(matched, t)
		}
	}

	sortDeterministic(matched)

	// Langkah 4 — keluarkan orang yang mengajukan.
	matched, excluded := excludeApplicant(matched, opts.Applicant)
	result.ExcludedApplicant = strings.TrimSpace(opts.Applicant)
	result.Excluded = toApprovers(excluded)

	if result.Mode == ModeSingleApprover {
		return pickOne(result, matched, opts), nil
	}

	result.Approvers = toApprovers(matched)
	result.AmbiguousOrder = hasTiedTier(matched)
	return result, nil
}

// excludeApplicant membuang baris milik orang yang mengajukan, dan mengembalikan yang
// dibuang supaya pengecualiannya dapat dilaporkan alih-alih terjadi diam-diam.
//
// # Kenapa ini berlaku pada SEMUA mode
//
// Di sistem lama, penyaring `AND OPERATOR_ID != …` hanya ada pada satu kueri — jalur
// Simasnet (`Activity/SetEmailKomiteSimasnet-Act.xml`). Work Owner menetapkan 2026-09-18
// bahwa aturan itu **diberlakukan ke seluruh entitas**.
//
// Ini PERUBAHAN PERILAKU yang disengaja, bukan peniruan.
//
// # Kenapa akibatnya berbeda di luar Simasnet
//
// Di Simasnet, komitenya ADALAH tim klaim — yang mengajukan dan yang menyetujui satu
// kumpulan orang, sehingga mengeluarkan si pengaju menyisakan rekan-rekannya.
//
// Di entitas lain, komitenya kelompok senior TERSENDIRI dan tangganya tipis: dihitung
// dari `Database/emailkomite.csv`, setiap lini punya jenjang terendah yang diisi SATU
// orang saja. Bila orang itu yang mengajukan, tidak ada rekan yang tersisa — klaimnya
// berakhir tanpa penyetuju sama sekali, dan itu menghentikannya.
//
// Ketujuh keadaan itu diuji di TestPengecualianPenginputDapatMenghabiskanSeluruhPenyetuju
// bukan supaya dianggap benar, melainkan supaya terlihat. Yang tersingkir karena itu
// dilaporkan, dan layar membedakannya dari "master tidak menjangkau nilai ini".
//
// # Keputusan itu DITEGASKAN ULANG setelah akibatnya dihitung
//
// Pertanyaannya diajukan kembali pada 2026-09-19, lengkap dengan ketujuh keadaan di atas
// dan tiga jalan keluar yang ditawarkan — melengkapi master, membatasi kembali ke
// Simasnet saja, atau menaikkan klaim ke jenjang berikutnya. Work Owner memilih
// **tetap seperti ini**: klaim yang diajukan satu-satunya penyetujunya akan berhenti, dan
// ditangani manual.
//
// Karena itu perilaku di bawah BUKAN kelalaian yang menunggu diperbaiki. Ia keputusan
// yang diambil setelah akibatnya diketahui, dan catatan ini ada supaya pembaca berikutnya
// tidak "memperbaikinya" tanpa bertanya.
func excludeApplicant(rows []Threshold, applicant string) (remaining, excluded []Threshold) {
	key := OperatorKey(applicant)
	if key == "" {
		return rows, nil
	}

	remaining = rows[:0:0]
	for _, t := range rows {
		// Perbandingan mengabaikan besar-kecil huruf dan spasi tepi. Alasannya nyata:
		// `docs/Steering/11-SECURITY.md` §3.1 mencatat nama access group Pega muncul
		// dalam dua kapitalisasi berbeda, dan OPERATOR_ID pada master bahkan ada yang
		// diakhiri baris baru. Perbandingan yang terlalu ketat akan gagal mengecualikan
		// penginput — dan gagalnya TIDAK terlihat, karena hasilnya tetap berupa nama
		// yang masuk akal.
		if OperatorKey(t.OperatorID) == key {
			excluded = append(excluded, t)
			continue
		}
		remaining = append(remaining, t)
	}
	return remaining, excluded
}

// pickOne menerapkan sisa aturan Simasnet: ambil jenjang terendah, acak satu di
// antaranya.
//
// Pengecualian penginput sudah dikerjakan sebelum fungsi ini dipanggil, karena ia kini
// berlaku pada kedua mode.
func pickOne(result Tiering, eligible []Threshold, opts Options) Tiering {
	if len(eligible) == 0 {
		result.Approvers = nil
		result.Candidates = nil
		return result
	}

	// `ORDER BY degree, dbms_random.value` diikuti `WHERE rownum = 1` berarti: ambil
	// jenjang TERENDAH, lalu acak di antara yang seri pada jenjang itu.
	lowestTier := eligible[0].Tier
	candidates := eligible[:0:0]
	for _, t := range eligible {
		if t.Tier == lowestTier {
			candidates = append(candidates, t)
		}
	}

	result.Candidates = toApprovers(candidates)

	randomizer := opts.Randomizer
	if randomizer == nil {
		randomizer = FixedRandomizer(0)
	}
	chosen := candidates[randomizer.Pick(len(candidates))]

	result.Approvers = toApprovers([]Threshold{chosen})
	return result
}

// sortDeterministic menyusun baris menjadi urutan yang tidak pernah bergantung pada urutan
// datangnya dari penyimpanan.
func sortDeterministic(rows []Threshold) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Tier != rows[j].Tier {
			return rows[i].Tier < rows[j].Tier
		}
		if rows[i].LowerBound != rows[j].LowerBound {
			return rows[i].LowerBound < rows[j].LowerBound
		}
		return rows[i].ID < rows[j].ID
	})
}

func toApprovers(rows []Threshold) []Approver {
	result := make([]Approver, 0, len(rows))
	for i, t := range rows {
		result = append(result, Approver{
			Order:       i + 1,
			Tier:        t.Tier,
			Name:        t.Name,
			OperatorID:  t.OperatorID,
			LowerBound:  t.LowerBound,
			Absent:      t.Absent,
			ThresholdID: t.ID,
		})
	}
	return result
}

func hasTiedTier(rows []Threshold) bool {
	seen := map[int]bool{}
	for _, t := range rows {
		if seen[t.Tier] {
			return true
		}
		seen[t.Tier] = true
	}
	return false
}

// ValidateInput mengumpulkan SELURUH pelanggaran pada masukan, bukan berhenti pada yang
// pertama.
func ValidateInput(value money.Money, line BusinessLine) []Violation {
	var violations []Violation

	if value < 0 {
		violations = append(violations, Violation{
			Field:   FieldValue,
			Message: "Nilai klaim tidak boleh kurang dari nol.",
		})
	}
	if line.Normalized() == "" {
		violations = append(violations, Violation{
			Field:   FieldBusinessLine,
			Message: "Lini bisnis wajib dipilih.",
		})
	}

	return violations
}

// ListBusinessLines mengembalikan seluruh lini yang punya jenjang persetujuan di master,
// terurut.
//
// Dipakai layar untuk mengisi pilihan lini. Daftarnya datang DARI DATA, tidak pernah
// dari daftar tetap di dalam kode (`D-15`) — lini yang ditambahkan ke master langsung
// muncul tanpa rilis ulang.
func ListBusinessLines(thresholds []Threshold) []BusinessLine {
	seen := map[BusinessLine]bool{}
	for _, t := range thresholds {
		t = t.Normalized()
		if t.IsApprovalTier() {
			seen[t.BusinessLine] = true
		}
	}

	result := make([]BusinessLine, 0, len(seen))
	for l := range seen {
		result = append(result, l)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
