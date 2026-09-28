package db

import "strings"

// Kode galat Oracle untuk objek yang tidak ada.
//
// Ketiganya dibedakan Oracle sendiri, dan ketiganya berarti hal yang sama bagi aplikasi:
// **skema belum lengkap**, bukan permintaan yang salah.
const (
	// oraTableMissing — tabel atau view tidak ada.
	oraTableMissing = "ORA-00942"

	// oraSequenceMissing — sequence tidak ada.
	//
	// Ia dipisahkan dari yang di atas karena migrasi dapat gagal separuh jalan: tabelnya
	// terbentuk, sequence-nya tidak. Tanpa kode ini, keadaan separuh itu jatuh kembali ke
	// galat umum — persis kelas cacat yang berkas ini ada untuk menutupnya.
	oraSequenceMissing = "ORA-02289"

	// oraProcedureMissing — procedure atau function tidak ada.
	oraProcedureMissing = "ORA-04043"

	// oraDuplicateKey — kunci utama atau indeks unik dilanggar.
	oraDuplicateKey = "ORA-00001"
)

// IsDuplicateKey menyatakan apakah galat berarti kunci yang disisipkan sudah dipakai.
//
// Dipakai modul yang menerbitkan nomor tanpa sequence: tabrakan bukan kegagalan yang
// harus dilaporkan ke pengguna, melainkan tanda untuk mengambil nomor berikutnya.
// Membedakannya dari kegagalan lain penting — mengulang penyisipan yang gagal karena
// sebab LAIN hanya akan mengulang kegagalan yang sama lima kali.
func IsDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), oraDuplicateKey)
}

// IsMissingObject menyatakan apakah galat basis data berarti objeknya belum dibuat.
//
// # Kenapa pengetahuan ini tinggal di sini
//
// Modul bisnis tidak boleh mengenali driver — `03-FUTURE-ARCHITECTURE.md` §2 menetapkan
// Domain tidak tahu apa pun tentang SQL, dan adapter-nya pun sebaiknya tidak terikat pada
// SATU driver tertentu. Paket ini sudah memegang pilihan driver (lihat komentar paket),
// sehingga pertukaran go-ora ↔ godror tetap menyentuh satu paket saja.
//
// # Kenapa dicocokkan sebagai teks, bukan sebagai tipe galat
//
// Mencocokkan tipe menuntut impor paket internal driver, dan itu justru mengikat aplikasi
// pada driver yang `08-TECHNICAL-STRATEGY.md` §1 sebut akan ditukar. Kode `ORA-nnnnn`
// sebaliknya adalah kontrak Oracle sendiri: nomornya sama pada driver mana pun, dan tidak
// berubah antar versi. Yang dicocokkan adalah **kodenya**, bukan kalimat pesannya — pesan
// dapat berganti bahasa mengikuti NLS, kode tidak.
func IsMissingObject(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	for _, code := range []string{oraTableMissing, oraSequenceMissing, oraProcedureMissing} {
		if strings.Contains(message, code) {
			return true
		}
	}
	return false
}
