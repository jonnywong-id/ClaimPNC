package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxacceptopenprotection"
)

// Repo membaca antrean akseptasi dan menuliskan keputusannya.
//
// # Tiga kolom yang ditulisnya, dan tidak lebih
//
// `APPROVAL_STATUS`, `RESOLVED_BY`, `RESOLVED_DATETIME`. Kolom pembuatan dimiliki modul
// `inputreqprotection` dan tidak pernah DITULIS di sini — itulah yang menjaga `P-1` tetap
// berlaku meski dua modul menyentuh satu tabel.
//
// Membacanya lain soal: form akseptasi menampilkan `OLD_DATA`, `NEW_DATA`, `OBJECT_NAME`,
// dan `BRANCH_NAME` pada panel "Detail Perubahan". `P-1` mengatur kepemilikan TULIS, bukan
// melarang pembacaan.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca satu halaman antrean akseptasi.
func (r *Repo) List(
	ctx context.Context,
	f inboxacceptopenprotection.Filter,
) (inboxacceptopenprotection.Page, error) {
	f = f.Normalize()

	// Antrean diterjemahkan menjadi SEPASANG parameter, bukan menjadi dua kueri: penanda
	// sama-atau-tidak, dan kode pembandingnya. Lihat kepala acceptance.sql.
	wantType, equal := f.Queue.TypeFilter()
	sama := 0
	if equal {
		sama = 1
	}

	// Penentu antrean dan kode pembandingnya masing-masing muncul DUA KALI di dalam kueri,
	// sehingga dikirim dua kali — driver mengikat menurut urutan kemunculan penanda, bukan
	// menurut nomornya. Mengirimnya sekali menghasilkan ORA-01008.
	argumen := append([]any{sama, wantType, sama, wantType}, searchArgs(f.Search)...)

	var total int
	if err := r.db.QueryRowContext(ctx, query("acceptance_count"), argumen...).Scan(&total); err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: menghitung antrean akseptasi: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query("acceptance_list"),
		append(append([]any(nil), argumen...), f.Offset, f.Limit)...)
	if err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca antrean akseptasi: %w", err)
	}
	defer rows.Close()

	page := make([]inboxacceptopenprotection.Protection, 0, f.Limit)
	for rows.Next() {
		p, err := scanProtection(rows, false)
		if err != nil {
			return inboxacceptopenprotection.Page{}, err
		}
		page = append(page, p)
	}
	if err := rows.Err(); err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca baris antrean akseptasi: %w", err)
	}

	return inboxacceptopenprotection.Page{Protections: page, Total: total}, nil
}

// Get membaca satu permintaan untuk form akseptasi.
func (r *Repo) Get(ctx context.Context, number string) (inboxacceptopenprotection.Protection, error) {
	return bacaProteksi(ctx, r.db, number)
}

// getTx membaca satu permintaan DI DALAM transaksi yang sedang berjalan.
//
// Dipakai `Decide` supaya yang dibaca adalah keadaan sesudah keputusan disimpan, bukan
// keadaan lama dari koneksi lain yang belum melihat transaksi ini.
func (r *Repo) getTx(
	ctx context.Context,
	tx *sql.Tx,
	number string,
) (inboxacceptopenprotection.Protection, error) {
	return bacaProteksi(ctx, tx, number)
}

// penanya menyatukan *sql.DB dan *sql.Tx supaya satu pembacaan melayani keduanya.
type penanya interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func bacaProteksi(
	ctx context.Context,
	q penanya,
	number string,
) (inboxacceptopenprotection.Protection, error) {
	row := q.QueryRowContext(ctx, query("acceptance_get"), kunci(number))

	p, err := scanProtection(row, true)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrNotFound
	}
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}
	return p, nil
}

// Decide menuliskan keputusan akseptasi.
//
// Bila tidak ada baris yang tersentuh, sebabnya DIBEDAKAN lewat pembacaan ulang: tidak ada,
// sudah diputuskan orang lain, atau belum lengkap. Ketiganya menuntut pesan berbeda —
// terutama yang kedua, karena petugas yang kalah cepat pada antrean bersama tidak sedang
// melakukan kesalahan.
func (r *Repo) Decide(
	ctx context.Context,
	number string,
	d inboxacceptopenprotection.Decision,
	by string,
	at time.Time,
) (inboxacceptopenprotection.Protection, error) {
	if !d.Valid() {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrUnknownDecision
	}

	// Keputusan dan penerapannya ke klaim berjalan dalam SATU transaksi.
	//
	// Keduanya menyentuh tabel berbeda, dan memisahkannya akan menghasilkan keadaan yang
	// tidak dapat dipulihkan sendiri: keputusan tersimpan sementara DOL klaim tidak berubah,
	// atau sebaliknya. Tidak satu pun dari keduanya menghasilkan galat yang terlihat
	// pengguna — hanya data yang tidak lagi bersesuaian.
	//
	// `08-TECHNICAL-STRATEGY.md` §4.5 menaruh kepemilikan transaksi di lapisan aplikasi.
	// Di sini ia turun ke adapter karena keduanya satu pernyataan tak terpisahkan bagi
	// pemanggil: `Decide` menjanjikan keputusan BESERTA akibatnya.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membuka transaksi keputusan: %w", err)
	}
	// Rollback setelah Commit tidak berakibat apa pun; yang dijaga adalah jalur GAGAL, dan
	// jalur itu punya banyak cabang keluar.
	defer func() { _ = tx.Rollback() }()

	hasil, err := tx.ExecContext(ctx, query("acceptance_decide"),
		d.Status(), strings.TrimSpace(by), at.UTC(), kunci(number))
	if err != nil {
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: menyimpan keputusan akseptasi: %w", err)
	}

	tersentuh, err := hasil.RowsAffected()
	if err != nil {
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca jumlah baris tersentuh: %w", err)
	}

	if tersentuh == 0 {
		existing, getErr := r.Get(ctx, number)
		switch {
		case getErr != nil:
			return inboxacceptopenprotection.Protection{}, getErr
		case !existing.Pending():
			return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrAlreadyDecided
		default:
			return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrIncomplete
		}
	}

	// Proteksi dibaca DI DALAM transaksi: yang menentukan perubahan klaim adalah tipe dan
	// detail perubahannya, dan keduanya harus dibaca dari keadaan yang sedang dikunci.
	sesudah, err := r.getTx(ctx, tx, number)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	if err := r.applyToClaim(ctx, tx, sesudah, d); err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	if err := tx.Commit(); err != nil {
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: menyimpan keputusan akseptasi: %w", err)
	}

	return sesudah, nil
}

// applyToClaim menerapkan perubahan yang disetujui ke data klaim.
//
// Aturannya ADA DI DOMAIN (`LossDateToApply`), bukan di sini: adapter hanya menjalankan.
// Menaruh "tipe '7' berarti ubah DOL" di dalam SQL akan menyembunyikan aturan bisnis di
// tempat yang tidak dibaca siapa pun saat menelusuri perilaku.
func (r *Repo) applyToClaim(
	ctx context.Context,
	tx *sql.Tx,
	p inboxacceptopenprotection.Protection,
	d inboxacceptopenprotection.Decision,
) error {
	tanggal, perlu := inboxacceptopenprotection.LossDateToApply(p, d)
	if !perlu {
		return nil
	}

	// Kuncinya ID_CLAIM, bukan nomor klaim: `T_CLAIM_PNC` dikunci `CLAIMID`, yang bagi klaim
	// warisan berbentuk `ASM-FW-GCNMFW-WORK PNC-1865`. Memakai nomor klaim di sini tidak akan
	// menemukan baris mana pun, dan tidak menemukan apa pun tidak menghasilkan galat.
	rujukan := strings.TrimSpace(p.ClaimReference)
	if rujukan == "" {
		// Proteksi tanpa ID_CLAIM tidak dapat ditautkan ke barisnya. Diperlakukan sama
		// dengan klaim yang tidak ditemukan — keputusannya dibatalkan, bukan diteruskan.
		return inboxacceptopenprotection.ErrClaimNotSynced
	}

	hasil, err := tx.ExecContext(ctx, query("claim_apply_loss_date"),
		tanggal, kunci(rujukan))
	if err != nil {
		return fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: menerapkan tanggal kejadian ke klaim: %w", err)
	}

	tersentuh, err := hasil.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca jumlah baris klaim tersentuh: %w", err)
	}
	if tersentuh == 0 {
		// Klaimnya tidak ada di tabel datar — keadaan yang MUNGKIN hari ini: baru 1.014 dari
		// 7.703 klaim ada di sana (`kolom-t-claimlist-admin.md`).
		//
		// Keputusannya DIBATALKAN, tidak diteruskan. Menyimpan persetujuan atas perubahan
		// yang tidak pernah diterapkan menghasilkan klaim yang DOL-nya berbeda dari yang
		// disetujui — dan tidak ada gejala yang menandainya. Lebih baik petugas menerima
		// penolakan yang menjelaskan sebabnya.
		return inboxacceptopenprotection.ErrClaimNotSynced
	}

	return nil
}

// ── Pemindaian ───────────────────────────────────────────────────────────────────

// pemindai menyatukan *sql.Row dan *sql.Rows supaya scanProtection melayani keduanya.
type pemindai interface {
	Scan(dest ...any) error
}

// scanProtection membaca satu baris menjadi Protection.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: tabelnya membolehkan NULL pada
// semuanya, dan memindainya ke string biasa akan gagal pada baris pertama yang kolomnya
// kosong.
//
// # Tiga field yang SENGAJA tidak terisi di sini
//
// `InsuredName`, `PolicyStart`, dan `PolicyEnd` ditampilkan form akseptasi
// (`Section/AcceptProtectionSection-Section.xml` memuat "Nama Tertanggung", "Start Date
// Time", "End Date Time"), tetapi ketiganya milik SNAPSHOT POLIS — bukan milik tabel ini.
//
// Mengambilnya menuntut pembacaan ke data polis, yang ada di bounded context lain (`D-04`)
// dan modulnya belum terpasang. Sampai itu ada, ketiganya kosong dan form menampilkannya
// sebagai tanda hubung. Itu lebih jujur daripada mengisinya dengan tebakan.
func scanProtection(row pemindai, denganPolis bool) (inboxacceptopenprotection.Protection, error) {
	var (
		id                    string
		nopolis, noklaim      sql.NullString
		idKlaim               sql.NullString
		tipe                  sql.NullString
		namaTipe              sql.NullString
		dibuatPada            sql.NullTime
		notes, dibuatOleh     sql.NullString
		dataLama, dataBaru    sql.NullString
		namaObjek, namaCabang sql.NullString
		status                sql.NullString
		diputuskanPada        sql.NullTime
		diputuskanOleh        sql.NullString

		tertanggung            sql.NullString
		polisMulai, polisAkhir sql.NullTime
	)

	// Kolom polis HANYA ada di kueri detail.
	//
	// Daftar tidak menampilkannya, dan menambahkan tiga subkueri berkorelasi ke sana berarti
	// tiga pembacaan tambahan per baris — enam puluh untuk satu halaman dua puluh baris —
	// demi nilai yang tidak pernah dilihat siapa pun.
	kolom := []any{&id, &nopolis, &noklaim, &idKlaim, &tipe, &namaTipe, &dibuatPada,
		&notes, &dibuatOleh, &dataLama, &dataBaru, &namaObjek, &namaCabang,
		&status, &diputuskanPada, &diputuskanOleh}
	if denganPolis {
		kolom = append(kolom, &tertanggung, &polisMulai, &polisAkhir)
	}

	// namaTipe datang dari LEFT JOIN ke master; NULL berarti kodenya kosong ATAU tidak
	// terdaftar. Keduanya diperlakukan sama di sini — pembedanya urusan lapisan tampilan.
	if err := row.Scan(kolom...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inboxacceptopenprotection.Protection{}, err
		}
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: memindai antrean akseptasi: %w", err)
	}

	p := inboxacceptopenprotection.Protection{
		Number:       strings.TrimSpace(id),
		PolicyNumber: teks(nopolis),
		ClaimNumber:  teks(noklaim),
		// ID_CLAIM dipakai sebagai KUNCI saat menerapkan perubahan ke `T_CLAIM_PNC`;
		// ia tidak ditampilkan layar.
		ClaimReference: teks(idKlaim),
		Type:           teks(tipe),
		TypeName:       teks(namaTipe),
		Note:           teks(notes),
		CreatedBy:      teks(dibuatOleh),
		Change: decodeChangeDetail(
			teks(tipe), teks(dataLama), teks(dataBaru), teks(namaObjek), teks(namaCabang)),
		AcceptStatus: teks(status),
		AcceptedBy:   teks(diputuskanOleh),
		InsuredName:  teks(tertanggung),
	}
	if dibuatPada.Valid {
		p.InputDate = dibuatPada.Time
	}
	if diputuskanPada.Valid {
		waktu := diputuskanPada.Time
		p.AcceptedAt = &waktu
	}
	if polisMulai.Valid {
		waktu := polisMulai.Time
		p.PolicyStart = &waktu
	}
	if polisAkhir.Valid {
		waktu := polisAkhir.Time
		p.PolicyEnd = &waktu
	}

	return p, nil
}

// ── Detail perubahan ─────────────────────────────────────────────────────────────

// tanggalTeks adalah bentuk tanggal di dalam OLD_DATA dan NEW_DATA.
//
// WAJIB sama dengan konstanta senama di `inputreqprotection/repo/sqlstore` — modul itu yang
// MENULIS kedua kolom, modul ini yang membacanya. Dua bentuk yang tidak sepakat menghasilkan
// tanggal yang selalu gagal dibaca, dan kegagalannya diam: panelnya tampil kosong, bukan
// bergalat.
//
// Bentuknya sendiri dipilih di sisi penulis; lihat kepala `protection.sql` modul itu.
const tanggalTeks = "2006-01-02"

// decodeChangeDetail membaca sepasang kolom serbaguna menurut tipe proteksi.
//
// Tipe selain '7' dan '8' mengembalikan detail KOSONG meski kedua kolom kebetulan terisi:
// form Pega tidak memunculkan panelnya bagi tipe lain, dan penulisnya pun tidak pernah
// mengisinya di sana. Menampilkan isi yang tidak seharusnya ada akan membuat petugas
// menimbang data yang bukan bagian dari permintaan.
//
// Tanggal yang tidak dapat dibaca menjadi nil, BUKAN galat. Baris warisan Pega tidak punya
// kolom asal untuk OLD_DATA/NEW_DATA (`kolom-open-protection.md` §9), dan satu baris
// berformat asing tidak boleh mematikan seluruh antrean.
func decodeChangeDetail(protectionType, lama, baru, objek, cabang string) inboxacceptopenprotection.ChangeDetail {
	if !inboxacceptopenprotection.ShowsChangeDetail(protectionType) {
		return inboxacceptopenprotection.ChangeDetail{}
	}

	d := inboxacceptopenprotection.ChangeDetail{ObjectName: objek, BranchName: cabang}

	switch strings.TrimSpace(protectionType) {
	case inboxacceptopenprotection.TypeChangeLossDate:
		d.LossDateBefore = bacaTanggal(lama)
		d.LossDateAfter = bacaTanggal(baru)
	case inboxacceptopenprotection.TypeChangeCauseOfLoss:
		d.CauseOfLossBefore = lama
		d.CauseOfLossAfter = baru
	}

	return d
}

func bacaTanggal(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parsed, err := time.Parse(tanggalTeks, s)
	if err != nil {
		return nil
	}
	return &parsed
}

// kunci merapikan nomor proteksi menjadi bentuk yang dibandingkan kueri.
//
// Kueri membandingkannya dengan `UPPER(TRIM(ID)) = :n`, sehingga perapiannya harus terjadi
// DI SINI juga — dua tempat yang tidak sepakat menghasilkan pencarian yang selalu gagal
// tanpa satu pun galat.
func kunci(number string) string {
	return strings.ToUpper(strings.TrimSpace(number))
}

// searchArgs menyusun keempat argumen kotak pencarian.
//
// Kata pencarian muncul EMPAT KALI di dalam kueri — sekali sebagai penentu apakah
// pencariannya aktif, tiga kali sebagai pola LIKE — dan driver mengikat menurut urutan
// kemunculan penanda.
//
// Polanya dibentuk di Go supaya tanda persen dan garis bawah yang diketik pengguna tidak
// menjadi wildcard tanpa disengaja.
func searchArgs(search string) []any {
	search = strings.TrimSpace(search)
	if search == "" {
		return []any{nil, nil, nil, nil}
	}

	pola := "%" + escapeLike(strings.ToUpper(search)) + "%"
	return []any{strings.ToUpper(search), pola, pola, pola}
}

// escapeLike menetralkan wildcard LIKE di dalam kata pencarian.
//
// Karakternya dibuang, bukan diloloskan — kueri tidak memakai klausa ESCAPE. Konsekuensinya
// mencari "50%" menemukan yang memuat "50", dan itu perilaku yang masuk akal bagi kotak
// pencarian bebas.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "%", "")
	return strings.ReplaceAll(s, "_", "")
}

func teks(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}
