package sqlstore

import "strings"

// Kode galat Oracle yang berarti "yang kurang BUKAN kode, melainkan sesuatu di basis data".
//
// Keduanya dikenali dari teksnya, bukan dari tipe galat driver. Alasannya bukan kemalasan:
// `godror` mengembalikan tipenya sendiri, `sqlmock` mengembalikan `errors.New`, dan uji modul
// ini berjalan di atas yang kedua. Memeriksa tipe berarti jalur ini tidak pernah teruji.
const (
	// ORA-00942 — tabel atau view tidak ada, ATAU ada tetapi tidak terlihat oleh akun ini.
	// Oracle sengaja tidak membedakan keduanya: membedakannya akan memberi tahu pemanggil
	// bahwa sebuah objek ada padahal ia tidak berhak mengetahuinya.
	oraObjectMissing = "ora-00942"

	// ORA-01031 — hak tidak cukup. Muncul saat objeknya terlihat tetapi operasinya tidak
	// diizinkan, misalnya UPDATE tanpa GRANT UPDATE.
	oraNoPrivilege = "ora-01031"

	// ORA-00904 — nama kolom tidak sah. Dibedakan karena jawabannya BUKAN "minta ke DBA"
	// melainkan "kuerinya salah" — persis kekeliruan `TOTAL_JOB` pada picteknik.sql.
	oraBadColumn = "ora-00904"
)

// needsDBA menjawab apakah galat ini berarti ada yang harus dikerjakan DBA lebih dulu.
//
// # Kenapa ini ada
//
// Tanpa pengenalan ini, dua keadaan yang SANGAT berbeda tampil sama di layar:
//
//	tabel permintaan belum dibuat      migrasi `0014` belum dijalankan
//	hak UPDATE belum diberikan         GRANT belum diberikan
//	sistemnya benar-benar rusak        yang ini saja yang 500
//
// Ketiganya sempat dijawab "Terjadi kesalahan pada sistem", dan pesan itu tidak memberi tahu
// siapa pun apa yang harus dilakukan — pengguna menyangka sistemnya rusak, padahal yang
// kurang adalah satu baris yang sedang ditunggu dari DBA.
func needsDBA(err error) bool {
	if err == nil {
		return false
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, oraObjectMissing) || strings.Contains(lower, oraNoPrivilege)
}

// badColumn menjawab apakah galatnya kekeliruan KUERI, bukan kekurangan hak.
//
// Dipisah dari needsDBA karena tindak lanjutnya berlawanan: yang satu menunggu orang lain,
// yang satu lagi menuntut kuerinya diperbaiki. Menyamakannya akan membuat kueri yang salah
// menunggu DBA selamanya.
func badColumn(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), oraBadColumn)
}
