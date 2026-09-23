// Package inboxcloseclaimhttp adalah lapisan transport modul Inbox Close Claim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4). Memakai
// tipe domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien, dan
// sebaliknya — penggantian nama field di domain menjadi perubahan yang merusak antarmuka.
package inboxcloseclaimhttp

import (
	"time"

	"claim-pnc/internal/inboxcloseclaim"
)

// claimDTO adalah satu baris pada layar.
//
// # Nama field berbahasa Indonesia
//
// Ia KONTRAK yang dibaca frontend, dan termasuk lima pengecualian `D-80` bersama komentar,
// nama kolom basis data, teks layar, dan variabel lingkungan. Yang berbahasa Inggris
// hanyalah nama tipe dan field Go-nya.
//
// # Yang dihitung di server, bukan di peramban
//
// `lama_hari` dan `status_tampil` diturunkan di sini. Menyerahkannya ke frontend berarti
// aturan bisnis hidup di dua tempat — dan yang di peramban tidak dapat diuji bersama
// aturan lainnya.
type claimDTO struct {
	// KlaimID adalah `PZINSKEY`. Ia dikirim ke layar karena kedua aksi membutuhkannya —
	// persis seperti tombol lama mengirim `.CaseID` sebagai parameter `casePNC`.
	//
	// Ia TIDAK ditampilkan sebagai kolom: isinya kunci teknis berisi nama kelas internal
	// Pega, dan `D-22` menghapusnya dari data bisnis untuk klaim baru.
	KlaimID string `json:"klaim_id"`

	NomorKlaim   string `json:"nomor_klaim"`      // kolom "No Klaim"
	NomorPolis   string `json:"nomor_polis"`      // kolom "No Polis"
	Tertanggung  string `json:"nama_tertanggung"` // kolom "Nama Tertanggung"
	NamaBisnis   string `json:"nama_bisnis"`      // kolom "Nama Bisnis"
	SumberBisnis string `json:"sumber_bisnis"`    // kolom "Sumber Bisnis"
	NamaCabang   string `json:"nama_cabang"`      // kolom "Nama Cabang"
	PICTeknik    string `json:"pic_teknik"`       // kolom "PIC Teknik"
	AdminPNC     string `json:"admin_pnc"`        // kolom "Admin PNC"

	// Tanggal dikirim sebagai teks ISO-8601 tanggal saja (YYYY-MM-DD), bukan timestamp.
	//
	// Kolom yang ditampilkan adalah TANGGAL, dan mengirim timestamp UTC memaksa setiap
	// tempat di frontend memutuskan sendiri cara mengubahnya ke WIB — yaitu cara paling
	// mudah membuat tanggal bergeser satu hari tanpa ada yang menyadarinya (`R-12`).
	TanggalPendaftaran string `json:"tanggal_pendaftaran"` // kolom "Tanggal Pendaftaran"
	TanggalKejadian    string `json:"tanggal_kejadian"`    // tidak ditampilkan
	TanggalTutup       string `json:"tanggal_tutup"`       // tidak ditampilkan; dasar lama_hari

	// LamaHari adalah kolom "Lama Waktu Klaim" — berapa hari klaim itu berjalan.
	//
	// Di Pega kolom itu menampilkan TANGGAL PENDAFTARAN lagi. Yang di sini adalah selisih
	// terencana yang diputuskan Work Owner 2026-09-23 — lihat `ClosedClaim.DurationDays`.
	LamaHari int `json:"lama_hari"`

	StatusProses     string `json:"status_proses"`
	StatusTampil     string `json:"status_tampil"`      // "Close" atau "Reject"
	StatusKlaimKode  string `json:"status_klaim_kode"`  // STATUSCLAIM_1 — KODE
	StatusKlaimLabel string `json:"status_klaim_label"` // labelnya dari V_STS_CLAIM

	// SudahTransfer menandai klaim pernah ditransfer ke kasir.
	//
	// Ia dikirim meski bukan kolom di layar lama: penyaring "Status Transfer" memakainya,
	// dan pengguna yang menyaring tanpa melihat penandanya di baris tidak punya cara
	// memastikan penyaringnya bekerja.
	SudahTransfer bool `json:"sudah_transfer"`

	// PermintaanTertunda adalah permintaan yang sudah diajukan atas baris ini dan belum
	// dijalankan.
	//
	// Kosong berarti belum ada. Ia yang membuat tombolnya dapat menyatakan "permintaan
	// terkirim" alih-alih membiarkan pengguna menekannya lagi karena klaimnya belum
	// berubah — dan klaim memang belum berubah, karena yang tercatat baru permintaannya.
	PermintaanTertunda []requestDTO `json:"permintaan_tertunda"`
}

// requestDTO adalah satu permintaan yang menunggu dijalankan.
type requestDTO struct {
	ID          string `json:"id"`
	Jenis       string `json:"jenis"`  // "reopen" atau "salin"
	Status      string `json:"status"` // "menunggu"
	Alasan      string `json:"alasan"`
	Pemohon     string `json:"pemohon"`
	PemohonNama string `json:"pemohon_nama"`

	// Pada adalah waktu permintaan, dalam WIB dan lengkap sampai menit.
	//
	// Berbeda dari kolom tanggal di atas yang hanya tanggal: yang ditanyakan orang tentang
	// sebuah permintaan adalah "kapan tepatnya", bukan "hari apa".
	Pada string `json:"pada"`

	// Efek adalah niat yang tercatat — apa yang akan berubah saat permintaan dijalankan.
	//
	// Dikirim ke layar supaya pengguna dapat membaca akibat yang ia minta, bukan sekadar
	// nama tombol yang ia tekan.
	EfekStatusKerja string `json:"efek_status_kerja"`
	EfekStatusKlaim string `json:"efek_status_klaim"`
	LingkupSalin    string `json:"lingkup_salin"`
}

// listResponse adalah badan respons daftar.
type listResponse struct {
	Klaim []claimDTO `json:"klaim"`
	Total int        `json:"total"`

	// PermintaanTerbaca false berarti permintaan tertunda TIDAK dapat dibaca.
	//
	// Layar wajib menyatakannya. Bila tidak, penanda "permintaan terkirim" yang tidak
	// muncul akan terbaca sebagai "belum pernah diminta" — dan pengguna mengajukannya lagi,
	// yang justru keadaan yang hendak dicegah.
	//
	// Keadaan ini NYATA hari ini: tabel permintaan dibuat migrasi `0006` yang belum
	// dijalankan DBA di lingkungan mana pun.
	PermintaanTerbaca bool `json:"permintaan_terbaca"`

	// BolehMengajukan false berarti pemanggil TIDAK dapat mengajukan ReOpen maupun Copy
	// Klaim.
	//
	// Ia dikirim supaya layar dapat menonaktifkan kedua tombolnya BESERTA ALASANNYA,
	// alih-alih membiarkan pengguna menekan tombol yang pasti dijawab 403. Tombol yang
	// tampak dapat ditekan lalu selalu gagal terbaca sebagai gangguan sistem, bukan sebagai
	// kewenangan yang memang tidak ada.
	//
	// Hari ini ia **selalu false** — penjaganya When rule `IsGCNMUser`, yang isinya `1 = 2`.
	// Lihat `inboxcloseclaim/authorization.go`.
	BolehMengajukan bool `json:"boleh_mengajukan"`

	// AlasanTidakBoleh terisi hanya saat BolehMengajukan false.
	//
	// Datang dari DOMAIN, bukan ditulis di layar: keterangan sebuah aturan tidak boleh hidup
	// terpisah dari aturannya.
	AlasanTidakBoleh string `json:"alasan_tidak_boleh,omitempty"`

	// PelaksanaBelumAda menyatakan bahwa permintaan yang tercatat BELUM ADA yang
	// menjalankannya.
	//
	// Ditetapkan Work Owner 2026-09-23: siapa yang mengeksekusi permintaan di sisi Pega
	// belum ditentukan. Layar wajib menyatakannya — pengguna yang mengajukan lalu menunggu
	// perubahan yang tidak akan datang akan melaporkannya sebagai kegagalan.
	PelaksanaBelumAda bool `json:"pelaksana_belum_ada"`

	// SelisihTerencana menyebut perbedaan yang disengaja terhadap layar Pega.
	//
	// Ia dinyatakan kepada pengguna, bukan disembunyikan sebagai detail teknis (`D-54`).
	SelisihTerencana []string `json:"selisih_terencana"`
}

// metadataResponse adalah keterangan bentuk layar: isi ketiga dropdown penyaring.
//
// # Kenapa bentuk penyaring datang dari server
//
// Karena kelima pilihan lini bisnis adalah HASIL PEMBACAAN activity Pega, dan tempat
// pembacaan itu tercatat adalah backend (`inboxcloseclaim.BusinessLine`). Menyalinnya ke
// layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat
// yang lain diperbaiki.
type metadataResponse struct {
	LiniBisnis     []pilihanDTO `json:"lini_bisnis"`
	StatusTransfer []pilihanDTO `json:"status_transfer"`
	StatusBayar    []pilihanDTO `json:"status_bayar"`
}

// pilihanDTO adalah satu butir dropdown.
type pilihanDTO struct {
	Nilai string `json:"nilai"`
	Label string `json:"label"`
}

// requestBody adalah badan permintaan ReOpen atau Copy Klaim.
//
// # Kenapa klaim disebut di BADAN, bukan di jalur URL
//
// Kunci klaim warisan berbentuk `ASM-FW-GCNMFW-WORK PNC-9001` — **mengandung spasi**, dan
// nama kelas internal Pega di dalamnya. Menaruhnya di jalur URL menuntut penyandian yang
// benar di setiap sisi, dan satu sisi yang keliru menghasilkan klaim yang tidak ditemukan
// alih-alih galat yang jelas.
//
// Badan permintaan tidak punya persoalan itu. Ia juga yang membuat `alasan` — yang boleh
// panjang — punya tempat yang wajar.
type requestBody struct {
	Jenis   string `json:"jenis"`
	KlaimID string `json:"klaim_id"`
	Alasan  string `json:"alasan"`
}

// requestResponse adalah jawaban atas permintaan yang berhasil dicatat.
type requestResponse struct {
	Permintaan requestDTO `json:"permintaan"`

	// Pesan menjelaskan apa yang SUDAH dan BELUM terjadi.
	//
	// Ia ada karena ketidaksamaan di sini mudah disalahpahami: tombolnya berhasil ditekan,
	// tetapi klaimnya belum berubah. Tanpa kalimat ini, pengguna menutup layar dengan
	// mengira klaimnya sudah terbuka kembali.
	Pesan string `json:"pesan"`
}

// toClaimDTO mengubah satu klaim menjadi bentuk kiriman.
func toClaimDTO(
	c inboxcloseclaim.ClosedClaim,
	pending []inboxcloseclaim.ClaimRequest,
	now time.Time,
	loc *time.Location,
) claimDTO {
	requests := make([]requestDTO, 0, len(pending))
	for _, request := range pending {
		requests = append(requests, toRequestDTO(request, loc))
	}

	return claimDTO{
		KlaimID:            c.ClaimID,
		NomorKlaim:         c.ClaimNumber,
		NomorPolis:         c.PolicyNumber,
		Tertanggung:        c.InsuredName,
		NamaBisnis:         c.BusinessName,
		SumberBisnis:       c.BusinessSource,
		NamaCabang:         c.BranchName,
		PICTeknik:          c.TechnicalPIC,
		AdminPNC:           c.AdminPNC,
		TanggalPendaftaran: formatDate(&c.RegisteredAt, loc),
		TanggalKejadian:    formatDate(c.LossDate, loc),
		TanggalTutup:       formatDate(closingDate(c), loc),
		LamaHari:           c.DurationDays(now, loc),
		StatusProses:       c.ProcessStatus,
		StatusTampil:       string(c.DisplayStatus()),
		StatusKlaimKode:    c.ClaimStatusCode,
		StatusKlaimLabel:   c.ClaimStatusLabel,
		SudahTransfer:      c.TransferredToCashier,
		PermintaanTertunda: requests,
	}
}

// closingDate memilih tanggal tutup yang ditampilkan, mengikuti urutan yang sama dengan
// DurationDays: tanggal penutupan bisnis lebih dulu, waktu penyelesaian Pega sebagai
// cadangan.
//
// Mengembalikan nil bila keduanya kosong — dan sel kosong di layar, bukan tanggal hari ini,
// karena "belum diketahui" bukan "hari ini".
func closingDate(c inboxcloseclaim.ClosedClaim) *time.Time {
	if c.ClosedAt != nil && !c.ClosedAt.IsZero() {
		return c.ClosedAt
	}
	if c.ResolvedAt != nil && !c.ResolvedAt.IsZero() {
		return c.ResolvedAt
	}
	return nil
}

func toRequestDTO(r inboxcloseclaim.ClaimRequest, loc *time.Location) requestDTO {
	return requestDTO{
		ID:              r.ID,
		Jenis:           string(r.Kind),
		Status:          string(r.Status),
		Alasan:          r.Reason,
		Pemohon:         r.ActorLogin,
		PemohonNama:     r.ActorName,
		Pada:            formatDateTime(r.RequestedAt, loc),
		EfekStatusKerja: r.EffectWorkStatus,
		EfekStatusKlaim: r.EffectClaimStatus,
		LingkupSalin:    r.CopyScope,
	}
}

// formatDate mengubah waktu UTC menjadi tanggal WIB berbentuk YYYY-MM-DD.
//
// Inilah SATU-SATUNYA tempat konversi zona waktu terjadi pada modul ini, bersama
// formatDateTime di bawahnya. Penyimpanan memakai UTC dan tampilan memakai WIB; menyebar
// konversinya akan mengulang persis cacat `Set7Hours` sistem lama, yang menambah tujuh jam
// manual di 118 titik pada 36 activity sehingga satu tempat yang lupa menggeser tanggal
// tanpa terdeteksi.
func formatDate(t *time.Time, loc *time.Location) string {
	if t == nil || t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}

// formatDateTime mengubah waktu UTC menjadi tanggal dan jam WIB.
func formatDateTime(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}
