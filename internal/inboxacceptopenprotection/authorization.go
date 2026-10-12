package inboxacceptopenprotection

import (
	"context"
	"strings"
)

// Kewenangan layar Inbox Accept Open Protection.
//
// # Dari mana aturannya
//
// Layar lama dibatasi `When/IsOpenProtectionPNC-When.xml`, yang membandingkan
// `AccessGroup.pyAccessGroup` dengan LIMA nilai. Di dalamnya, pemisahan antrean ditentukan
// syarat tampil kedua grid pada `Section/InputProtection_Section-Section.xml`:
//
//	:1592  NON PREMI  AccessGroup != 'GCNMFW:PncCollection'
//	:6036  PREMI      AccessGroup == 'GCNMFW:PncCollection'
//
// # Sumber datanya di sistem baru
//
// `POOLDATA.M_LOGIN_GROUP_PNC` (`LOGIN_ID`, `GROUP_ID`). `GROUP_ID` berisi nama access group
// Pega **tanpa awalan `GCNMFW:`** — Work Owner, 2026-09-25. Jadi `GCNMFW:PncCollection`
// menjadi `PncCollection`.
//
// Tabelnya sudah ada dan sudah dipakai modul `menu` untuk menyusun pohon menu; yang baru di
// sini hanyalah PEMAKAIANNYA sebagai kewenangan.
//
// # Kenapa ini bukan sekadar menyembunyikan menu
//
// Otorisasi sistem lama hanyalah penyembunyian menu — `pyPrivilegeName` terisi pada 1 dari
// 902 activity. Siapa pun yang mengetahui alamat sebuah layar dapat membukanya. Pemeriksaan
// di sini terjadi di SERVER pada setiap permintaan, sehingga mengetahui alamatnya tidak lagi
// cukup (`D-59`).

// GroupPremiumQueue adalah satu-satunya group yang melihat antrean PREMI.
//
// Dibiarkan sebagai konstanta, bukan dibaca dari master: ia PERCABANGAN, bukan label. Sama
// alasannya dengan kode tipe proteksi di atas — tidak ada kolom di mana pun yang menyatakan
// "group ini yang menangani premi".
const GroupPremiumQueue = "PncCollection"

// screenGroups adalah kelima access group yang boleh membuka layar ini.
//
// Disimpan sebagai himpunan berkunci HURUF KECIL. Pega tidak konsisten kapitalisasinya —
// `D-58` mencatat `ViewClaimPNC`/`VIEWCLAIMPNC` dan `PncReceive`/`PNCRECEIVE` hidup
// berdampingan — dan menuntut sistem baru menormalkannya menjadi satu identitas per peran.
//
// Perbandingan yang peka huruf besar-kecil akan menolak pengguna yang sah hanya karena
// barisnya diketik dengan kapitalisasi lain, dan penolakan itu tampak seperti gangguan.
var screenGroups = map[string]struct{}{
	"pnccollection":  {},
	"casemanager":    {},
	"pncopcgeneral":  {},
	"pnckomite":      {},
	"administrators": {},
}

// normalizeGroup merapikan satu GROUP_ID menjadi bentuk yang dibandingkan.
//
// Awalan `GCNMFW:` ikut dibuang meski Work Owner menetapkan `GROUP_ID` tidak memuatnya:
// barisnya diisi manusia, dan satu baris yang telanjur memuat awalan akan menolak pengguna
// yang sah tanpa satu pun galat. Menerima keduanya tidak melonggarkan apa pun — yang
// dibandingkan tetap nama group yang sama.
func normalizeGroup(group string) string {
	g := strings.TrimSpace(group)
	if titik := strings.LastIndex(g, ":"); titik >= 0 {
		g = g[titik+1:]
	}
	return strings.ToLower(strings.TrimSpace(g))
}

// CanOpenScreen menyatakan pemanggil boleh membuka layar ini sama sekali.
//
// Pemanggil TANPA satu pun group ditolak. Arahnya dipilih sengaja: layar ini menyetujui
// pembukaan proteksi, dan `D-59` menetapkan tidak ada pemisahan tugas formal — sehingga
// memperlakukan "belum terdaftar" sebagai "boleh" berarti siapa pun yang berhasil masuk
// dapat menyetujui.
func CanOpenScreen(groups []string) bool {
	for _, g := range groups {
		if _, ok := screenGroups[normalizeGroup(g)]; ok {
			return true
		}
	}
	return false
}

// CanOpenQueue menyatakan pemanggil boleh membuka antrean tertentu.
func CanOpenQueue(groups []string, q Queue) bool {
	for _, allowed := range QueuesFor(groups) {
		if allowed == q {
			return true
		}
	}
	return false
}

// QueuesFor menyebut antrean mana saja yang boleh dibuka pemanggil.
//
// # Banyak group per login — DIKONFIRMASI Work Owner, 2026-09-25
//
// Pega membandingkan `AccessGroup.pyAccessGroup` — access group AKTIF, TUNGGAL. Satu
// operator berada di tepat satu access group pada satu saat, sehingga ia melihat tepat satu
// grid.
//
// `M_LOGIN_GROUP_PNC` berkunci (`LOGIN_ID`, `GROUP_ID`), dan Work Owner menegaskan **satu
// login memang boleh mengikuti banyak GROUP_ID**. Jadi keadaan "punya PncCollection
// sekaligus CaseManager" bukan kemungkinan teoretis melainkan bentuk yang dikehendaki — dan
// ia tidak punya padanan di Pega.
//
// Yang berlaku: **gabungan**, bukan salah satu. Orang yang mengikuti `PncCollection`
// mendapat PREMI; yang mengikuti group lain dari kelima itu mendapat NON PREMI; yang
// mengikuti keduanya mendapat keduanya.
//
// Gabungan tidak pernah MENGHILANGKAN akses yang di Pega ada. Memilih salah satu akan
// menuntut aturan urutan yang tidak berdasar di sumber mana pun, dan tebakan yang keliru di
// sana menutup antrean bagi orang yang berhak — kegagalan yang tampak seperti daftar kosong,
// bukan seperti penolakan.
func QueuesFor(groups []string) []Queue {
	var premium, nonPremium bool

	for _, g := range groups {
		n := normalizeGroup(g)
		if _, ok := screenGroups[n]; !ok {
			continue
		}
		if n == normalizeGroup(GroupPremiumQueue) {
			premium = true
		} else {
			nonPremium = true
		}
	}

	// Urutannya mengikuti urutan grid di layar lama: NON PREMI lebih dulu.
	var hasil []Queue
	if nonPremium {
		hasil = append(hasil, QueueNonPremium)
	}
	if premium {
		hasil = append(hasil, QueuePremium)
	}
	return hasil
}

// GroupReader membaca access group yang diikuti sebuah login.
//
// # Kenapa seam-nya dideklarasikan DI SINI
//
// Modul `menu` sudah membaca tabel yang sama untuk keperluannya sendiri. Mengimpornya dari
// sini akan membuat dua modul domain saling bergantung — yang dilarang
// `08-TECHNICAL-STRATEGY.md` §2. Yang diulang hanyalah satu kueri sebaris; yang dihindari
// adalah kopling yang tidak dapat diurai lagi.
//
// # Koneksi UTAMA, bukan koneksi portal
//
// `D-78` menetapkan satu identitas berlaku di keempat portal, sehingga keanggotaan group
// tinggal bersama identitas — bukan di dalam basis data tiap entitas. Modul `menu` pun
// dipasang pada koneksi utama. Membacanya dari koneksi portal akan membuat kewenangan
// seseorang berubah-ubah mengikuti entitas yang sedang dibuka.
type GroupReader interface {
	GroupsOf(ctx context.Context, login string) ([]string, error)
}
