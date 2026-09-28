package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Uji di berkas ini menjaga peminjaman identitas mitra pada layar Inbox PLA DLA
// (`MENU_ID 45`).
//
// # Kenapa ia layak diuji, padahal isinya tiga baris
//
// Ia mengganti SATU nilai yang menentukan seluruh penyaring modul — daftar, ringkasan,
// rincian, dokumen, dan komunikasi semuanya turun dari Login. Penggantian yang terlalu
// longgar membuat pemakai lain ikut melihat klaim mitra; penggantian yang terlalu ketat
// membuat setelannya tampak diabaikan, yang persis kegagalan yang baru saja diperbaiki.

// Tanpa setelan, login pemanggil tidak disentuh sama sekali.
func TestWithoutABorrowTheCallerLoginIsUntouched(t *testing.T) {
	var none reinsurerBorrow

	require.Equal(t, "JONNY", none.apply("JONNY"))
	require.Equal(t, "", none.apply(""))
}

// Setelan yang setengah terisi TIDAK meminjam apa pun.
//
// Ia sudah ditolak `config.Load`, tetapi penjagaan kedua di sini murah: perakitan yang
// diam-diam meminjam "ke login kosong" akan membuat SETIAP pemanggil ditolak, dan
// penolakan itu terbaca seperti kerusakan penyaring.
func TestAHalfFilledBorrowNeverTakesEffect(t *testing.T) {
	require.Equal(t, "JONNY",
		newReinsurerBorrow("JONNY", "", nil).apply("JONNY"))
	require.Equal(t, "JONNY",
		newReinsurerBorrow("", "TUGURE", nil).apply("JONNY"))
	require.Equal(t, "JONNY",
		newReinsurerBorrow("   ", "   ", nil).apply("JONNY"))
}

// Login yang meminjam berjalan sebagai mitra; login lain tidak tersentuh.
//
// Baris kedua yang menjaga bagian terpenting: peminjaman ini BUKAN pelonggaran gerbang.
// Pemakai lain tetap dicocokkan dengan loginnya sendiri, dan karena itu tetap ditolak
// bila ia bukan mitra terdaftar.
func TestOnlyTheNamedLoginBorrowsThePartnerIdentity(t *testing.T) {
	borrow := newReinsurerBorrow("JONNY", "TUGUREASURANSIINDONESIA", nil)

	require.Equal(t, "TUGUREASURANSIINDONESIA", borrow.apply("JONNY"))
	require.Equal(t, "ORANGLAIN", borrow.apply("ORANGLAIN"),
		"peminjaman berlaku bagi SATU login, bukan melonggarkan penyaringnya")
	require.Equal(t, "", borrow.apply(""))
}

// Besar-kecil huruf dan spasi di ujung tidak menggagalkan peminjaman.
//
// Alasannya sama seperti `F-3` menormalkan nama peran: `T_ACCESS_GROUP_PNC` dan rule Pega
// terbukti tidak konsisten soal kapitalisasi. Setelan yang gagal karena satu huruf akan
// terbaca seperti setelan yang diabaikan — dan "diabaikan diam-diam" adalah kegagalan
// yang seluruh perubahan ini ada untuk menutupnya.
func TestTheBorrowIgnoresCaseAndSurroundingSpaces(t *testing.T) {
	borrow := newReinsurerBorrow("  JONNY  ", "TUGURE", nil)

	for _, login := range []string{"JONNY", "jonny", "Jonny", "  jOnNy  "} {
		require.Equal(t, "TUGURE", borrow.apply(login),
			"login %q seharusnya tetap meminjam", login)
	}
}
