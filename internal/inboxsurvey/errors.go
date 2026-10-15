package inboxsurvey

import "errors"

// Galat domain modul My Work.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Pada modul ini ia BUKAN gangguan kecil. Seluruh antrean disaring dengan nama surveyor
	// yang diturunkan dari login, sehingga tanpa login tidak ada antrean yang dapat dibentuk
	// sama sekali.
	ErrCallerUnknown = errors.New("inboxsurvey: identitas pemanggil tidak terbaca")

	// ErrNotSurveyor berarti pemanggil TERBACA, tetapi tidak terdaftar sebagai surveyor.
	//
	// # Kenapa ia galat tersendiri, dan bukan antrean kosong
	//
	// Karena keduanya terlihat sama di layar dan berarti hal yang sangat berbeda:
	//
	//	antrean kosong  -> "tidak ada pekerjaan untuk Anda hari ini"
	//	bukan surveyor  -> "layar ini memang bukan untuk Anda"
	//
	// Yang pertama tidak pernah dilaporkan siapa pun sebagai kerusakan, sehingga menjawab
	// dengan daftar kosong akan menyembunyikan salah pasang kewenangan sampai ada orang yang
	// kebetulan bertanya.
	//
	// Ia muncul ketika `POOLDATA.MST_LOGIN_SURVEYOR` tidak punya baris dengan `LOGIN` yang
	// cocok — pemetaan yang `RDB List/GetLoginLeaderSurveyor-SQL.xml` pakai persis begitu.
	ErrNotSurveyor = errors.New("inboxsurvey: pemanggil tidak terdaftar sebagai surveyor")

	// ErrTabUnknown berarti tab yang diminta tidak dikenal.
	//
	// Ia TIDAK dipakai jalur biasa: Filter.Normalize menjatuhkan tab tak dikenal ke
	// DefaultTab, karena tab datang dari URL dan URL yang tertinggal versi lama sebaiknya
	// membuka halaman yang masuk akal.
	//
	// Ia ada untuk pemanggil yang memang ingin menolak — perkakas dan pengujian — supaya
	// pilihan "jatuhkan ke bawaan" terbaca sebagai keputusan, bukan sebagai satu-satunya
	// perilaku yang mungkin.
	ErrTabUnknown = errors.New("inboxsurvey: tab tidak dikenal")

	// ErrKPIFilterIncomplete berarti Status Survey atau Tipe Report belum dipilih.
	//
	// Keduanya WAJIB di layar lama — ditandai bintang merah — dan tidak ada nilai bawaan yang
	// dapat dipilihkan untuk pengguna. Memilihkan salah satunya berarti menjalankan laporan
	// yang tidak diminta siapa pun, lalu menampilkan angkanya seolah itu yang dicari.
	ErrKPIFilterIncomplete = errors.New("inboxsurvey: isian wajib panel KPI belum lengkap")
)
