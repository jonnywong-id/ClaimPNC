// Package masterpicteknik adalah inti modul Master PIC Teknik (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Daftar petugas teknik yang menangani klaim: siapa mereka, di grup mana, siapa
// atasannya, berapa kuota pekerjaan yang boleh dipikulnya, dan apakah ia masih aktif.
// Isi master inilah yang dipakai penugasan klaim — `SearchPICTeknik_act` memutari daftar
// ini untuk memilih PIC berikutnya.
//
// # Kunci alaminya diberikan, bukan dibuat
//
// Berbeda dari Master Status Klaim yang kodenya diterbitkan urutan, kunci di sini adalah
// `OPERATOR_ID` — identitas petugas di direktori operator. Ia DIISI pengguna dan tidak
// pernah dibuat sistem.
//
// Procedure lama `PEGA_MST_USER_TEKNIS` sempat menghitung
// `id_site || lpad(MST_USER_TEKNIS_SEQ.nextval, 6, '0')` ke dalam variabel
// `id_mst_user_teknis` — lalu **tidak pernah memakainya**. INSERT-nya tetap memakai
// `IDPega` sebagai `OPERATOR_ID`. Generator itu kode mati, dan tidak dibawa ke sini.
//
// # Nama tidak diketik, melainkan dicari
//
// `MCL_NAME` tidak pernah diisi tangan. `CNMInsertMstUserTeknis_act` mencarinya lebih
// dulu lewat `SelectMstUserTeknisMclName`:
//
//	select pyusername   as "MCL_NAME",
//	       pyuseridentifier as "OPERATOR_ID"
//	  from datapega.pr_operators
//	 where upper(pyuseridentifier) = upper(:1)
//
// dan bila hasilnya kosong, langkah berikutnya adalah `Property-Set-Messages` dengan
// keterangan "set error kalau tidak ditemukan di service". Jadi petugas yang tidak
// terdaftar di direktori operator **ditolak** — dan aturan itu dipertahankan.
//
// # Penamaan
//
// Alias sistem lama menyesatkan dan tidak dibawa masuk:
//
//	COUNTER_QUOTA2 → alias "OLD_OPERATOR_ID"   padahal sebuah ANGKA, bukan id operator
//	GROUPPANEL     → alias "IBNR"              padahal nama grup panel
//	OPERATOR_ID    → alias "MCL_Name"          pada BrowseEmailUserTeknis
//
// Pemetaan alias→kolom→domain lengkap ada di repo/sqlstore/picteknik.sql.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpicteknik

import (
	"context"
	"strings"
	"unicode/utf8"
)

// Batas panjang isian. Seluruhnya KEBUTUHAN BARU: layar Pega tidak membatasi panjang
// sama sekali. Angkanya mengikuti lebar kolom yang ada di tabel hari ini.
const (
	MaxOperatorIDLength = 64
	MaxEmailLength      = 100
	MaxGroupLength       = 50
	MaxSupervisorLength     = 64
	MaxBusinessLineLength = 50
)

// KuotaMaksimum menahan angka kuota yang tidak masuk akal.
//
// Ia bukan aturan bisnis yang ditemukan di export — tidak ada batas di sana — melainkan
// penjaga agar salah ketik tidak menghasilkan petugas berkuota jutaan yang menyedot
// seluruh antrean penugasan.
const MaxQuota = 9999

// PICTeknik adalah satu baris master petugas teknik.
type PICTeknik struct {
	// IDOperator adalah OPERATOR_ID — kunci alaminya, sekaligus identitas petugas di
	// direktori operator. Tetap seumur hidup baris ini.
	OperatorID string

	// Nama adalah MCL_NAME. Ia DITURUNKAN dari direktori operator, bukan diketik.
	Name string

	// Email adalah alamat surel petugas. Dipakai seluruh pemberitahuan yang ditujukan
	// kepadanya — termasuk peringatan kegagalan integrasi Kasir.
	Email string

	// LiniBisnis adalah TYPE_BUSINESS. Ia teks bebas, bukan pilihan tertutup: satu-
	// satunya nilai yang benar-benar muncul di export adalah "NONMBU", dan mengarang
	// daftar pilihan dari satu contoh akan menolak nilai sah yang belum terlihat.
	BusinessLine string

	// Grup adalah TEAM_GROUP, kelompok kerja petugas.
	Group string

	// Atasan adalah ATASAN — diisi OPERATOR_ID atasannya.
	Supervisor string

	// Kuota adalah COUNTER_QUOTA, banyaknya pekerjaan yang boleh dipikul petugas ini.
	Quota int

	// KuotaLuar adalah COUNTER_QUOTA2.
	//
	// Namanya di sistem lama — alias "OLD_OPERATOR_ID" — menyesatkan: isinya ANGKA,
	// bukan identitas. `SetTotalJobMstUserTeknis` mengisinya dengan
	// `sum(total_job)` dari sistem luar lewat DB link
	// (`new_general.m_user_job@opjava.sinarmas.co.id`), yaitu beban kerja petugas yang
	// sama di aplikasi lain.
	//
	// DB link itu sendiri TIDAK dibawa ke sini — `ADR-0008` menetapkan DB link diganti
	// API, dan API-nya belum ada. Untuk sekarang nilainya dikelola sebagai isian biasa,
	// persis seperti procedure lama yang menerimanya sebagai parameter.
	ExternalQuota int

	// GrupPanel adalah GROUPPANEL, dialias "IBNR" pada SELECT lama.
	//
	// HANYA DIBACA. Procedure penulis `PEGA_MST_USER_TEKNIS` tidak pernah menulis kolom
	// ini — baik pada cabang INSERT maupun UPDATE. Menjadikannya dapat diubah di sini
	// berarti menambah perilaku yang tidak pernah ada.
	GrupPanel string

	// Aktif menyatakan petugas masih menerima penugasan.
	Active bool
}

// SandiAktif adalah isi kolom STS_AKTIF untuk petugas yang aktif.
//
// Nilainya "1", BUKAN "Ya"/"Tidak" seperti STS_AKTIF pada POOLDATA.LST_ACCOUNT. Dua
// tabel berbeda memakai sandi berbeda untuk kolom bernama sama, dan itu justru alasan
// nilainya ditulis sebagai konstanta bernama di sini: `STS_AKTIF = '1'` muncul 17 kali
// di seluruh export, tidak satu pun memakai "Ya".
const ActiveCode = "1"

// Bersih mengembalikan salinan dengan spasi tepi dibuang.
func (p PICTeknik) Clean() PICTeknik {
	p.OperatorID = strings.TrimSpace(p.OperatorID)
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.TrimSpace(p.Email)
	p.BusinessLine = strings.TrimSpace(p.BusinessLine)
	p.Group = strings.TrimSpace(p.Group)
	p.Supervisor = strings.TrimSpace(p.Supervisor)
	p.GrupPanel = strings.TrimSpace(p.GrupPanel)
	return p
}

// KunciID adalah bentuk IDOperator yang dipakai membandingkan dan mencari.
//
// Perbandingan mengabaikan besar-kecil huruf, mengikuti kueri lamanya yang memang
// menulis `upper(pyuseridentifier) = upper(:1)`.
func IDKey(id string) string {
	return strings.ToUpper(strings.TrimSpace(id))
}

// Repo adalah seam ke penyimpanan master PIC teknik.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memori. Antarmukanya berbicara dalam
// istilah domain, bukan istilah SQL.
type Repo interface {
	// Daftar mengembalikan seluruh petugas, terurut menurut IDOperator.
	List(ctx context.Context) ([]PICTeknik, error)

	// Ambil mengembalikan satu petugas, atau ErrTidakDitemukan.
	Get(ctx context.Context, operatorID string) (PICTeknik, error)

	// Sisip menyimpan petugas baru. Mengembalikan ErrSudahAda bila IDOperator-nya
	// sudah dipakai.
	Insert(ctx context.Context, p PICTeknik) (PICTeknik, error)

	// Perbarui mengubah petugas yang sudah ada. IDOperator, Nama, dan GrupPanel tidak
	// ikut berubah. Mengembalikan ErrTidakDitemukan bila tidak ada.
	Update(ctx context.Context, p PICTeknik) (PICTeknik, error)
}

// DirektoriOperator adalah seam ke daftar operator tempat nama petugas dicari.
//
// Sistem lama membacanya dari `DATAPEGA.PR_OPERATORS`, tabel milik Pega. Selama masa
// paralel tabel itu masih hidup dan masih menjadi sumber kebenaran nama operator; ketika
// Pega dimatikan, yang diganti hanyalah pengisi seam ini.
type OperatorDirectory interface {
	// NamaOperator mengembalikan nama petugas. Mengembalikan ErrOperatorTidakDikenal
	// bila identitas itu tidak terdaftar.
	OperatorName(ctx context.Context, operatorID string) (string, error)
}

// Periksa mengumpulkan SELURUH pelanggaran aturan sekaligus, bukan berhenti pada yang
// pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku: sistem lama menampilkan seluruh
// pesan validasi sekaligus (docs/Steering/12-CROSSCUTTING.md §1.2 butir 1).
//
// Keberadaan petugas di direktori operator TIDAK diperiksa di sini — ia menuntut
// memanggil seam, sedangkan fungsi ini murni dan dapat diuji tanpa apa pun.
// Pemeriksaannya ada di usecase.
func Check(p PICTeknik) []Violation {
	p = p.Clean()
	var violation []Violation

	create := func(field, message string) {
		violation = append(violation, Violation{Field: field, Message: message})
	}

	if p.OperatorID == "" {
		create(FieldOperatorID, "ID operator wajib diisi.")
	} else if utf8.RuneCountInString(p.OperatorID) > MaxOperatorIDLength {
		create(FieldOperatorID, "ID operator paling panjang "+itoa(MaxOperatorIDLength)+" karakter.")
	}

	if p.Email == "" {
		create(FieldEmail, "Email wajib diisi.")
	} else if !EmailLooksValid(p.Email) {
		create(FieldEmail, "Format email tidak benar.")
	} else if utf8.RuneCountInString(p.Email) > MaxEmailLength {
		create(FieldEmail, "Email paling panjang "+itoa(MaxEmailLength)+" karakter.")
	}

	if utf8.RuneCountInString(p.BusinessLine) > MaxBusinessLineLength {
		create(FieldBusinessLine, "Lini bisnis paling panjang "+itoa(MaxBusinessLineLength)+" karakter.")
	}
	if utf8.RuneCountInString(p.Group) > MaxGroupLength {
		create(FieldGroup, "Grup paling panjang "+itoa(MaxGroupLength)+" karakter.")
	}
	if utf8.RuneCountInString(p.Supervisor) > MaxSupervisorLength {
		create(FieldSupervisor, "Atasan paling panjang "+itoa(MaxSupervisorLength)+" karakter.")
	}

	if p.Quota < 0 || p.Quota > MaxQuota {
		create(FieldQuota, "Kuota harus antara 0 dan "+itoa(MaxQuota)+".")
	}
	if p.ExternalQuota < 0 || p.ExternalQuota > MaxQuota {
		create(FieldExternalQuota, "Kuota sistem lain harus antara 0 dan "+itoa(MaxQuota)+".")
	}

	// Petugas tidak boleh menjadi atasan dirinya sendiri. Bukan aturan yang tertulis di
	// export, melainkan akibat langsung dari cara `SearchPICTeknik_act` menelusuri
	// rantai atasan: rujukan ke diri sendiri membuatnya berputar tanpa henti.
	if p.Supervisor != "" && IDKey(p.Supervisor) == IDKey(p.OperatorID) {
		create(FieldSupervisor, "Petugas tidak boleh menjadi atasan dirinya sendiri.")
	}

	return violation
}

// EmailMasukAkal memeriksa bentuk alamat surel sekadarnya.
//
// Sengaja longgar: satu-satunya cara membuktikan sebuah alamat benar adalah mengirim
// surel ke sana, dan validasi yang terlalu ketat justru menolak alamat yang sah.
func EmailLooksValid(address string) bool {
	address = strings.TrimSpace(address)
	i := strings.IndexByte(address, '@')
	if i <= 0 || i == len(address)-1 {
		return false
	}
	domain := address[i+1:]
	if strings.ContainsRune(domain, '@') {
		return false
	}
	j := strings.IndexByte(domain, '.')
	return j > 0 && j < len(domain)-1
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke lapisan domain
// hanya untuk pesan galat.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
