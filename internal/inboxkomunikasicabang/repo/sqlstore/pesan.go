package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Branches mengembalikan daftar cabang yang dapat dipilih sebagai tujuan pesan baru.
//
// Tanpa batas cabang — lihat catatan pada seam `Repo.Branches`. Yang dibatasi adalah
// percakapan, bukan daftar cabang.
func (r *Repo) Branches(ctx context.Context) ([]inboxkomunikasicabang.BranchOption, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("branch_options"))
	if err != nil {
		return nil, fmt.Errorf("inboxkomunikasicabang/sqlstore: membaca daftar cabang: %w", err)
	}
	defer func() { _ = rows.Close() }()

	options := []inboxkomunikasicabang.BranchOption{}
	for rows.Next() {
		var code, name, email sql.NullString
		if err := rows.Scan(&code, &name, &email); err != nil {
			return nil, fmt.Errorf(
				"inboxkomunikasicabang/sqlstore: memindai baris cabang: %w", err)
		}

		options = append(options, inboxkomunikasicabang.BranchOption{
			Code:  strings.TrimSpace(text(code)),
			Name:  strings.TrimSpace(text(name)),
			Email: strings.TrimSpace(text(email)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menutup daftar cabang: %w", err)
	}

	return options, nil
}

// SendMessage membuat percakapan BARU.
//
// # Kenapa SATU transaksi, dan kenapa di sini alasannya lebih tajam daripada pada Reply
//
// Ketiga pernyataannya saling bergantung, dan yang kedua membaca `MAX(KOMUNIKASIID)` —
// nomor yang baru saja diterbitkan basis data. Tanpa transaksi, dua pengiriman yang berjalan
// bersamaan membaca nomor yang SAMA, dan riwayat yang satu tertaut ke percakapan yang lain.
//
// Sistem lama menjalankannya sebagai tiga langkah activity tanpa transaksi apa pun. Cacat
// itu tidak akan terlihat pada pengujian satu pengguna, dan tidak menghasilkan satu pun
// galat ketika terjadi.
//
// # Tingkat isolasi
//
// Transaksi dibuka dengan tingkat isolasi BAWAAN, bukan Serializable. Itu keputusan sadar:
// pada Oracle bawaannya Read Committed, yang TIDAK mencegah dua transaksi membaca `MAX()`
// yang sama bila keduanya berjalan benar-benar bersamaan.
//
// Yang menutup celah itu sepenuhnya bukan tingkat isolasi melainkan `INSERT ... RETURNING`,
// dan itu menuntut nama sequence atau trigger yang menerbitkan nomornya — tidak terbaca dari
// export karena DDL-nya belum ada (`R-08`). Sampai DDL itu tiba, yang ada di sini sudah
// lebih baik daripada sistem lama, dan keterbatasannya dinyatakan alih-alih disembunyikan.
func (r *Repo) SendMessage(
	ctx context.Context,
	command inboxkomunikasicabang.NewMessageCommand,
	origin string,
) (string, error) {
	cleanOrigin := strings.TrimSpace(origin)
	if cleanOrigin == "" {
		return "", errors.New(
			"inboxkomunikasicabang/sqlstore: cabang asal kosong; pesan tidak dikirim")
	}

	recipient := strings.TrimSpace(command.RecipientCode())
	if recipient == "" {
		return "", errors.New(
			"inboxkomunikasicabang/sqlstore: cabang tujuan kosong; pesan tidak dikirim")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: memulai transaksi pesan: %w", err)
	}
	// Tanpa syarat — setelah Commit berhasil ia tidak melakukan apa pun, sehingga setiap
	// jalur keluar tertutup, termasuk yang ditambahkan kemudian oleh orang yang lupa.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		getQuery("message_insert"),
		inboxkomunikasicabang.CaseOpen,          // :1 kanal
		command.Sender.Login,                    // :2 pengirim
		command.Message,                         // :3 isi pesan
		command.Sender.Name,                     // :4 nama pengirim
		inboxkomunikasicabang.StatusNotAnswered, // :5 status
		recipient,                               // :6 tujuan
		cleanOrigin,                             // :7 asal
	); err != nil {
		return "", fmt.Errorf("inboxkomunikasicabang/sqlstore: menyimpan pesan: %w", err)
	}

	// Nomornya dibaca DI DALAM transaksi yang sama. Membacanya di luar akan mengembalikan
	// nomor milik pengiriman orang lain yang kebetulan selesai lebih dulu.
	var latest sql.NullString
	if err := tx.QueryRowContext(ctx, getQuery("message_max_id")).Scan(&latest); err != nil {
		return "", fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: membaca nomor percakapan baru: %w", err)
	}

	id := strings.TrimSpace(text(latest))
	if id == "" {
		// Tabelnya kosong padahal baris baru saja disisipkan — keadaan yang tidak mungkin
		// terjadi bila INSERT-nya benar-benar berhasil. Ia ditolak, bukan diteruskan dengan
		// nomor kosong yang akan menautkan riwayat ke tidak ke mana-mana.
		return "", errors.New(
			"inboxkomunikasicabang/sqlstore: nomor percakapan baru tidak terbaca setelah " +
				"penyimpanan")
	}

	if _, err := tx.ExecContext(
		ctx,
		getQuery("message_history_insert"),
		command.Sender.Login, // :1 pengirim
		command.Message,      // :2 isi pesan
		id,                   // :3 nomor percakapan
		// :4 kode cabang TUJUAN — dan di sinilah ia benar-benar berisi kode cabang,
		// berbeda dari jalur balasan yang mengisinya dengan CASEID. Lihat catatan pada
		// berkas .sql.
		command.BranchCode,
	); err != nil {
		return "", fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: mencatat riwayat pesan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf(
			"inboxkomunikasicabang/sqlstore: menutup transaksi pesan: %w", err)
	}

	return id, nil
}
