// Package komitehttp adalah lapisan transport modul Komite: bentuk respons, pemetaan
// galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp,
// portalhttp, dan masterstatushttp: foldernya `http` supaya letaknya seragam antarmodul,
// nama paketnya `komitehttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
//
// Seluruh tag `json` berbahasa Indonesia dan tetap demikian: ia **kontrak API**, bukan
// nama internal (`D-80`). Mengubahnya adalah perubahan yang merusak klien.
package komitehttp

// ThresholdDTO adalah satu baris master ambang komite yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari komite.Threshold. Memakai tipe modul langsung sebagai
// bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Kolom EMAIL dan CC pada master TIDAK ADA di sini, dan memang tidak pernah dibaca dari
// basis data — lihat catatan pada berkas kueri. `D-67` menetapkan alamat pribadi pada
// master lama tidak dibawa ke sistem baru sama sekali.
type ThresholdDTO struct {
	ID           string `json:"id"`
	Name         string `json:"nama"`
	OperatorID   string `json:"operator_id"`
	BusinessLine string `json:"lini"`

	// CommitteeType adalah kolom TYPE_KOMITE apa adanya, dan kolom itu memikul DUA ARTI:
	// pita nilai di Non-MBU, varian jalur di PA. Layar menjelaskannya lewat UsesBand
	// pada respons daftar, bukan dengan menebak dari nilainya.
	CommitteeType string `json:"jenis_komite"`

	// Nilai uang dikirim sebagai teks desimal kanonik — "50000001.00" — bukan angka
	// JSON. Angka JSON adalah floating point ganda di hampir seluruh peramban, dan
	// mengirim nilai uang lewatnya berarti menyerahkan ketepatannya kepada pembulatan
	// biner. `I-12` menetapkan nilai uang disimpan dengan presisi penuh; kontrak ini
	// menjaganya sampai ke layar.
	LowerBound string `json:"batas_bawah"`
	UpperBound string `json:"batas_atas"`

	// Tier adalah DEGREE — penentu urutan, BUKAN jumlah jenjang. Ia boleh berulang
	// dan boleh melompat.
	Tier int `json:"jenjang"`

	Active          bool `json:"aktif"`
	ForAdjustment   bool `json:"untuk_adjustment"`
	ForRegistration bool `json:"untuk_registrasi"`
	ForRejection    bool `json:"untuk_penolakan"`
	Absent          bool `json:"sedang_absen"`

	// IsApprovalTier menyatakan baris ini benar-benar ikut menyetujui nilai klaim.
	//
	// Dikirim sebagai kesimpulan, bukan dibiarkan dihitung ulang layar dari Active dan
	// ForAdjustment. Aturannya milik server, dan menduplikasinya di layar membuat dua
	// salinan yang dapat berbeda pendapat.
	IsApprovalTier bool `json:"jenjang_persetujuan"`
}

// BandPolicyDTO menjelaskan aturan pita yang sedang berlaku untuk sebuah lini.
//
// Dikirim ke layar supaya pengguna dapat MELIHAT batas pita yang dipakai, bukan
// menghafalnya. Angka yang menentukan uang tidak boleh hanya hidup di dalam kode tanpa
// pernah terlihat siapa pun.
type BandPolicyDTO struct {
	BusinessLine string `json:"lini"`
	Boundary     string `json:"batas"`
	Lower        string `json:"pita_bawah"`
	Upper        string `json:"pita_atas"`
}

// ThresholdListResponse adalah jawaban GET /api/master/ambang-komite.
type ThresholdListResponse struct {
	Thresholds []ThresholdDTO `json:"ambang"`

	// Total dan TotalTiers dikirim eksplisit, bukan dibiarkan dihitung klien.
	// Keduanya berbeda: Total adalah seluruh baris master, TotalTiers hanya yang
	// benar-benar ikut menyetujui.
	Total      int `json:"total"`
	TotalTiers int `json:"total_jenjang"`

	// BusinessLines adalah daftar lini yang punya jenjang persetujuan, diambil DARI
	// DATA. Layar memakainya mengisi pilihan lini — tidak pernah dari daftar tetap.
	BusinessLines []string `json:"lini"`

	BandPolicies []BandPolicyDTO `json:"kebijakan_pita"`

	// Mode penjenjangan yang berlaku pada portal ini — "kumulatif" atau
	// "satu-penyetuju". Layar memakainya untuk menentukan apakah isian Operator ID
	// pengaju perlu ditampilkan sama sekali.
	Mode string `json:"mode"`
}

// FindingDTO adalah satu hal yang ditemukan pada master ambang.
type FindingDTO struct {
	// Severity bernilai "cacat" atau "peringatan". Layar membedakannya lewat nilai ini,
	// bukan dengan mencocokkan teks pesan.
	Severity string `json:"tingkat"`
	Kind     string `json:"jenis"`

	BusinessLine string `json:"lini"`
	Band         string `json:"pita,omitempty"`

	Message      string   `json:"pesan"`
	ThresholdIDs []string `json:"id_ambang"`
}

// IntegrityResponse adalah jawaban GET /api/master/ambang-komite/integritas.
type IntegrityResponse struct {
	Findings []FindingDTO `json:"temuan"`

	// Dihitung server supaya layar tidak perlu menyaring sendiri untuk tahu apakah ada
	// yang gawat.
	DefectCount  int `json:"jumlah_cacat"`
	WarningCount int `json:"jumlah_peringatan"`
}

// ApproverDTO adalah satu orang yang harus menyetujui.
type ApproverDTO struct {
	// Order selalu berurutan tanpa lompatan, 1 sampai jumlah penyetuju.
	Order int `json:"urutan"`

	// Tier adalah DEGREE dari master. Ia boleh berulang dan boleh melompat — lihat
	// AmbiguousOrder pada respons.
	Tier int `json:"jenjang"`

	Name       string `json:"nama"`
	OperatorID string `json:"operator_id"`

	// LowerBound adalah ambang yang membuat orang ini ikut. Dikirim supaya layar dapat
	// menampilkan ALASAN seseorang masuk daftar, bukan hanya hasilnya.
	LowerBound string `json:"batas_bawah"`

	Absent bool `json:"sedang_absen"`

	// ThresholdID menunjuk baris master asalnya, supaya hasil hitungan dapat ditelusuri
	// balik ke datanya saat ada yang meragukannya.
	ThresholdID string `json:"id_ambang"`
}

// TieringResponse adalah jawaban GET /api/komite/penjenjangan.
type TieringResponse struct {
	Value        string `json:"nilai"`
	BusinessLine string `json:"lini"`

	// Mode bernilai "kumulatif" atau "satu-penyetuju", dan ia ditentukan per PORTAL —
	// bukan per lini. Layar menjelaskan hasilnya berbeda pada keduanya, sehingga nilai
	// ini dikirim alih-alih dibiarkan disimpulkan dari panjang daftar penyetuju.
	Mode string `json:"mode"`

	// Candidates hanya terisi pada mode satu-penyetuju: seluruh orang yang LAYAK dipilih
	// pada jenjang terendah, sebelum satu di antaranya diacak.
	//
	// Ia dikirim karena yang dapat diperiksa pada mode itu bukan SIAPA yang terpilih —
	// itu acak — melainkan apakah KUMPULAN yang layak sudah benar.
	Candidates []ApproverDTO `json:"kandidat,omitempty"`

	// ExcludedApplicant menyebut operator yang diminta dikecualikan karena dialah
	// yang mengajukan. Kosong berarti tidak ada yang diminta dikecualikan.
	ExcludedApplicant string `json:"dikecualikan_penginput,omitempty"`

	// Excluded memuat orang yang BENAR-BENAR keluar karena pengecualian itu.
	//
	// Dibedakan dari ExcludedApplicant dengan sengaja: yang pertama menyatakan siapa
	// yang diminta dikecualikan, yang kedua apakah permintaan itu berakibat. Penginput
	// yang bukan anggota komite tidak mengubah apa pun, dan layar tidak boleh menyiratkan
	// sebaliknya.
	Excluded []ApproverDTO `json:"tersingkir,omitempty"`

	// UsesBand membedakan "lini ini tidak memakai pita" dari "pitanya gagal
	// dihitung", supaya layar tidak perlu menebak arti Band yang kosong.
	UsesBand bool   `json:"berpita_nilai"`
	Band     string `json:"pita,omitempty"`

	Approvers []ApproverDTO `json:"penyetuju"`
	TierCount int           `json:"jumlah_jenjang"`

	// NoApprovers dikirim sebagai KEADAAN, bukan galat: nilai yang tidak menemukan
	// satu pun jenjang adalah temuan yang harus dilihat Work Owner, dan layar
	// menampilkannya sebagai peringatan mencolok alih-alih halaman gagal.
	NoApprovers bool `json:"tanpa_penyetuju"`

	// AmbiguousOrder menyala bila ada dua penyetuju ber-DEGREE sama.
	//
	// Kueri sistem lama mengurutkan dengan ORDER BY DEGREE saja, sehingga saat seri
	// urutannya ditentukan basis data dan dapat berubah antar eksekusi. Keadaan itu
	// benar-benar ada pada master yang berlaku. Melaporkannya membuat perbedaan urutan
	// terhadap Pega pada kasus seri tidak terbaca sebagai cacat saat uji kesetaraan.
	AmbiguousOrder bool `json:"urutan_tidak_pasti"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Code dimaksudkan untuk dibaca program, Message untuk dibaca manusia. Klien membedakan
// jenis galat lewat Code — bukan dengan mencocokkan teks Message.
//
// Bentuknya sengaja sama persis dengan authhttp.ErrorResponse dan
// masterstatushttp.ErrorResponse. Menyatukan ketiganya menjadi satu tipe bersama adalah
// lingkup `TKT-F1-004`, kontrak galat yang mengikat seluruh aplikasi, dan tiket itu masih
// terhalang keputusan Work Owner. Sampai itu diputuskan, beberapa tipe yang berbentuk
// sama lebih jujur daripada satu tipe bersama yang menyiratkan kontraknya sudah ada.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Details hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Details []ViolationDTO `json:"detail,omitempty"`
}
