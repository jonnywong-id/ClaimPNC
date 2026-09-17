package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterrekening"
)

// batasBaku membatasi jumlah baris yang dibaca bila pemanggil tidak menyebut batasnya.
//
// Ia ada supaya tidak ada jalan untuk membaca seluruh tabel tanpa sengaja. Sistem lama
// memotong hasilnya di 500 baris lewat pyMaxRecords pada 54 dari 56 laporan
// (09-DATABASE-STRATEGY §6.3); angka itu dipakai kembali di sini supaya halaman
// pertama berisi jumlah baris yang sama dengan yang biasa dilihat pengguna.
const batasBaku = 500

// batasTertinggi menahan permintaan batas yang tidak masuk akal dari klien.
const batasTertinggi = 2000

// Repo membaca dan menulis POOLDATA.LST_ACCOUNT.
//
// PENULIS TUNGGAL (ADR-0004, P-1). Selama masa paralel, tabel ini masih ditulis Pega.
// Memindahkan Master Rekening ke aplikasi ini berarti kepemilikan tulis ikut berpindah
// — layar lamanya wajib dimatikan pada saat yang sama, bukan sesudahnya. Itu bagian
// dari rencana rollout, bukan detail yang dapat diurus belakangan.
type Repo struct {
	db *sql.DB
}

// RepoBaru membentuk repo; db wajib sudah terhubung.
func RepoBaru(db *sql.DB) *Repo { return &Repo{db: db} }

// Daftar membaca rekening yang cocok dengan filter beserta jumlah seluruh baris yang
// cocok sebelum dipotong paginasi.
func (r *Repo) Daftar(ctx context.Context, f masterrekening.Filter) ([]masterrekening.Rekening, int, error) {
	saring := bahanSaringan(f)

	var jumlah int
	if err := r.db.QueryRowContext(ctx, ambilKueri("rekening_jumlah"), saring...).Scan(&jumlah); err != nil {
		return nil, 0, fmt.Errorf("masterrekening/sqlstore: menghitung rekening: %w", err)
	}

	batas := f.Batas
	if batas <= 0 {
		batas = batasBaku
	}
	if batas > batasTertinggi {
		batas = batasTertinggi
	}
	lewati := f.Lewati
	if lewati < 0 {
		lewati = 0
	}

	argumen := append(append([]any(nil), saring...), lewati, batas)
	baris, err := r.db.QueryContext(ctx, ambilKueri("rekening_daftar"), argumen...)
	if err != nil {
		return nil, 0, fmt.Errorf("masterrekening/sqlstore: membaca daftar rekening: %w", err)
	}
	defer func() { _ = baris.Close() }()

	hasil := make([]masterrekening.Rekening, 0, batas)
	for baris.Next() {
		rek, err := pindaiRekening(baris)
		if err != nil {
			return nil, 0, err
		}
		hasil = append(hasil, rek)
	}
	if err := baris.Err(); err != nil {
		return nil, 0, fmt.Errorf("masterrekening/sqlstore: menelusuri daftar rekening: %w", err)
	}
	return hasil, jumlah, nil
}

// bahanSaringan menyusun sepuluh argumen saringan dalam urutan yang dituntut kedua
// kueri.
//
// Setiap saringan muncul dua kali di dalam SQL — sekali pada pemeriksaan IS NULL,
// sekali pada perbandingannya — sehingga nilainya dikirim dua kali pula. Nomor bind
// sengaja dibedakan, bukan diulang, supaya tidak bergantung pada tafsir driver
// terhadap bind bernomor sama.
func bahanSaringan(f masterrekening.Filter) []any {
	status := kosongJadiNil(string(f.Status))
	nomor := kosongJadiNil(f.NomorRekening)
	pemilik := kosongJadiNil(f.NamaPemilik)
	bank := kosongJadiNil(f.NamaBank)

	var komite any
	if f.HanyaKomiteSaya {
		komite = kosongJadiNil(f.IdentitasKomite)
	}

	return []any{
		status, status,
		nomor, nomor,
		pemilik, pemilik,
		bank, bank,
		komite, komite,
	}
}

// kosongJadiNil mengubah teks kosong menjadi NULL.
//
// Itulah yang membuat satu kueri melayani seluruh gabungan saringan tanpa merangkai
// teks SQL: saringan yang tidak diisi menjadi NULL, dan `:n IS NULL OR …` membuatnya
// tidak mempersempit apa pun.
func kosongJadiNil(s string) any {
	if potong := strings.TrimSpace(s); potong != "" {
		return potong
	}
	return nil
}

// Ambil membaca satu rekening.
func (r *Repo) Ambil(ctx context.Context, k masterrekening.Kunci) (masterrekening.Rekening, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("rekening_ambil"), k.NomorRekening, k.KodeBank)
	rek, err := pindaiRekening(baris)
	if errors.Is(err, sql.ErrNoRows) {
		return masterrekening.Rekening{}, masterrekening.ErrTidakDitemukan
	}
	if err != nil {
		return masterrekening.Rekening{}, err
	}
	return rek, nil
}

// CariNomor membaca seluruh baris dengan nomor rekening tertentu, tanpa peduli banknya.
func (r *Repo) CariNomor(ctx context.Context, nomor string) ([]masterrekening.Rekening, error) {
	baris, err := r.db.QueryContext(ctx, ambilKueri("rekening_cari_nomor"), strings.TrimSpace(nomor))
	if err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: mencari nomor rekening: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []masterrekening.Rekening
	for baris.Next() {
		rek, err := pindaiRekening(baris)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, rek)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: menelusuri hasil pencarian nomor: %w", err)
	}
	return hasil, nil
}

// Simpan menyisipkan rekening baru.
func (r *Repo) Simpan(ctx context.Context, rek masterrekening.Rekening) error {
	_, err := r.db.ExecContext(ctx, ambilKueri("rekening_sisip"),
		rek.NomorRekening,
		rek.NamaPemilik,
		rek.NamaBank,
		rek.CabangBank,
		rek.AlamatBank,
		rek.KodeBank,
		rek.TipeRekening,
		sandiAktif(rek.Aktif),
		string(rek.Status),
		rek.KomiteApproval,
		rek.Email,
		rek.EmailPenginput,
		rek.Telepon,
		rek.NIK,
		rek.IDDokumen,
		rek.Catatan,
		rek.DiinputOleh,
		rek.DiinputPada.UTC(),
		rek.DiubahOleh,
		rek.StatusLayanan,
		rek.KodeBankLama,
		rek.NomorRekeningLama,
		rek.NamaPemilikLama,
	)
	if err != nil {
		return fmt.Errorf("masterrekening/sqlstore: menyisipkan rekening: %w", err)
	}
	return nil
}

// Perbarui menulis ulang rekening yang sudah ada.
func (r *Repo) Perbarui(ctx context.Context, rek masterrekening.Rekening) error {
	var diputuskan any
	if rek.DiputuskanPada != nil {
		diputuskan = rek.DiputuskanPada.UTC()
	}

	hasil, err := r.db.ExecContext(ctx, ambilKueri("rekening_perbarui"),
		rek.NamaPemilik,
		rek.NamaBank,
		rek.CabangBank,
		rek.AlamatBank,
		rek.TipeRekening,
		sandiAktif(rek.Aktif),
		string(rek.Status),
		rek.KomiteApproval,
		diputuskan,
		rek.Email,
		rek.EmailPenginput,
		rek.Telepon,
		rek.NIK,
		rek.IDDokumen,
		rek.Catatan,
		rek.DiubahOleh,
		rek.StatusLayanan,
		rek.IDRekeningKasir,
		rek.ResponsKasir,
		rek.KodeBankLama,
		rek.NomorRekeningLama,
		rek.NamaPemilikLama,
		rek.NomorRekening,
		rek.KodeBank,
	)
	if err != nil {
		return fmt.Errorf("masterrekening/sqlstore: memperbarui rekening: %w", err)
	}
	return pastikanTersentuh(hasil, masterrekening.ErrTidakDitemukan)
}

// HapusYangDitolak membuang baris bekas penolakan komite.
//
// Syarat APPROVAL = '2' ditegakkan di dalam kueri. Bila tidak ada baris yang tersentuh,
// artinya barisnya tidak ada ATAU statusnya bukan ditolak; keduanya dijawab
// ErrSudahDiputuskan karena keduanya berarti hal yang sama bagi pemanggil: baris ini
// tidak boleh dibuang.
func (r *Repo) HapusYangDitolak(ctx context.Context, k masterrekening.Kunci) error {
	hasil, err := r.db.ExecContext(ctx, ambilKueri("rekening_hapus_yang_ditolak"), k.NomorRekening, k.KodeBank)
	if err != nil {
		return fmt.Errorf("masterrekening/sqlstore: membuang pengajuan yang ditolak: %w", err)
	}
	return pastikanTersentuh(hasil, masterrekening.ErrSudahDiputuskan)
}

// PeriksaTabel menyatakan apakah POOLDATA.LST_ACCOUNT dapat dijangkau akun aplikasi.
//
// Dipakai mode -periksa pada binary, yang menguji kesiapan tanpa menulis apa pun.
func (r *Repo) PeriksaTabel(ctx context.Context) (bool, error) {
	var jumlah int
	if err := r.db.QueryRowContext(ctx, ambilKueri("rekening_periksa_tabel")).Scan(&jumlah); err != nil {
		return false, fmt.Errorf("masterrekening/sqlstore: memeriksa tabel LST_ACCOUNT: %w", err)
	}
	return jumlah > 0, nil
}

func pastikanTersentuh(hasil sql.Result, bila error) error {
	// Sebagian driver tidak melaporkan jumlah baris yang tersentuh. Bila begitu,
	// ketiadaan angka BUKAN bukti bahwa tidak ada yang berubah — memperlakukannya
	// sebagai galat akan menolak penulisan yang sebenarnya berhasil.
	jumlah, err := hasil.RowsAffected()
	if err != nil {
		return nil
	}
	if jumlah == 0 {
		return bila
	}
	return nil
}

// sandiAktif memetakan status aktif ke sandi kolom STS_AKTIF.
//
// Sandinya "Ya" dan "Tidak", BUKAN "1" dan "0". Nilainya diambil dari activity
// SetTipeRekening, yang mengisi daftar pilihan layar lama:
//
//	TempTipeBank.pxResults(<APPEND>).NomorKontrak = "Ya"    → dipakai TempBank.CaseID
//	TempTipeBank.pxResults(<APPEND>).NomorKontrak = "Tidak"     (alias CaseID = STS_AKTIF)
//
// Ia tidak dapat dipilih bebas selama Pega masih membaca kolom yang sama.
func sandiAktif(aktif bool) string {
	if aktif {
		return "Ya"
	}
	return "Tidak"
}

// bacaAktif menafsirkan isi kolom STS_AKTIF.
//
// Perbandingannya tanpa peduli besar-kecil huruf, dan "1" ikut diterima: kolom ini
// sudah dipakai bertahun-tahun oleh beberapa rule, dan menolak mengenali baris lama
// hanya karena ejaannya berbeda akan menampilkan rekening aktif sebagai nonaktif —
// yang berarti petugas mengira rekening itu tidak dapat dipakai membayar klaim.
func bacaAktif(nilai string) bool {
	switch strings.ToUpper(strings.TrimSpace(nilai)) {
	case "YA", "1", "Y", "AKTIF":
		return true
	default:
		return false
	}
}

// pemindai menyatukan *sql.Row dan *sql.Rows sehingga satu fungsi pemindaian melayani
// keduanya. Tanpa ini, 27 kolom harus ditulis dua kali dan kedua salinannya harus
// diingat untuk diubah bersama-sama.
type pemindai interface {
	Scan(tujuan ...any) error
}

func pindaiRekening(p pemindai) (masterrekening.Rekening, error) {
	var (
		rek        masterrekening.Rekening
		aktif      sql.NullString
		status     sql.NullString
		diputuskan sql.NullTime
		diinput    sql.NullTime
		respons    sql.NullString

		nama, bank, cabang, alamat, kodeBank   sql.NullString
		tipe, komite, email, emailInput, telp  sql.NullString
		nik, dokumen, catatan, olehInput, oleh sql.NullString
		layanan, idKasir, flag                 sql.NullString
		bankLama, nomorLama, pemilikLama       sql.NullString
	)

	err := p.Scan(
		&rek.NomorRekening,
		&nama,
		&bank,
		&cabang,
		&alamat,
		&kodeBank,
		&tipe,
		&aktif,
		&status,
		&komite,
		&diputuskan,
		&email,
		&emailInput,
		&telp,
		&nik,
		&dokumen,
		&catatan,
		&olehInput,
		&diinput,
		&oleh,
		&layanan,
		&idKasir,
		&respons,
		&flag,
		&bankLama,
		&nomorLama,
		&pemilikLama,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return masterrekening.Rekening{}, err
		}
		return masterrekening.Rekening{}, fmt.Errorf("masterrekening/sqlstore: membaca baris rekening: %w", err)
	}

	rek.NomorRekening = strings.TrimSpace(rek.NomorRekening)
	rek.NamaPemilik = teks(nama)
	rek.NamaBank = teks(bank)
	rek.CabangBank = teks(cabang)
	rek.AlamatBank = teks(alamat)
	rek.KodeBank = teks(kodeBank)
	rek.TipeRekening = teks(tipe)
	rek.Aktif = bacaAktif(teks(aktif))
	rek.Status = masterrekening.StatusApproval(teks(status))
	rek.KomiteApproval = teks(komite)
	rek.Email = teks(email)
	rek.EmailPenginput = teks(emailInput)
	rek.Telepon = teks(telp)
	rek.NIK = teks(nik)
	rek.IDDokumen = teks(dokumen)
	rek.Catatan = teks(catatan)
	rek.DiinputOleh = teks(olehInput)
	rek.DiubahOleh = teks(oleh)
	rek.StatusLayanan = teks(layanan)
	rek.IDRekeningKasir = teks(idKasir)
	rek.FlagPerubahan = teks(flag)
	rek.KodeBankLama = teks(bankLama)
	rek.NomorRekeningLama = teks(nomorLama)
	rek.NamaPemilikLama = teks(pemilikLama)

	// Pemangkasan yang dulu dilakukan SUBSTR/INSTR di dalam SQL kini terjadi di sini.
	rek.ResponsKasir = masterrekening.PangkasResponsKasir(teks(respons))

	if diinput.Valid {
		rek.DiinputPada = diinput.Time.UTC()
	}
	if diputuskan.Valid {
		t := diputuskan.Time.UTC()
		rek.DiputuskanPada = &t
	}
	return rek, nil
}

func teks(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return strings.TrimSpace(n.String)
}

var _ masterrekening.Repo = (*Repo)(nil)
