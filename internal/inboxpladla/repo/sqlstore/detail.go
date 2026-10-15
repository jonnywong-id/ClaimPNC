package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxpladla"
)

// Berkas ini memenuhi bagian RINCIAN pada seam inboxpladla.Repo.
//
// # Satu kebiasaan berlaku di seluruh berkas
//
// Setiap method meneruskan `scope.Login` ke SETIAP penanda batas yang dituntut kuerinya —
// satu penanda, satu argumen, tidak pernah satu penanda yang dirujuk dua kali.
//
// Itu bukan pemborosan. Oracle mengikat argumen menurut urutan KEMUNCULAN penanda; satu
// penanda yang dirujuk dua kali membuat jumlah kemunculan tidak lagi sama dengan jumlah
// argumen, dan seluruh bind setelahnya bergeser satu tempat tanpa satu pun galat.

// ClaimHeader mengembalikan keterangan klaim di kepala layar rincian.
func (r *Repo) ClaimHeader(
	ctx context.Context,
	scope inboxpladla.DetailScope,
) (inboxpladla.ClaimHeader, error) {
	clean := scope.Clean()

	row := r.db.QueryRowContext(ctx, query("detail_claim_header"),
		clean.ClaimKey, clean.Login, clean.Login, clean.Login, clean.Login)

	var (
		claimKey, claimNo, policyNo, insured sql.NullString
		businessName, picTeknik              sql.NullString
		statusCode, statusLabel              sql.NullString
		registerDate, lossDate               sql.NullTime
	)

	err := row.Scan(
		&claimKey, &claimNo, &policyNo, &insured, &businessName,
		&registerDate, &lossDate, &picTeknik, &statusCode, &statusLabel,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Klaim yang tidak ada dan klaim yang bukan milik pemanggil dijawab SAMA —
		// lihat inboxpladla.ErrDocumentNotFound untuk alasannya.
		return inboxpladla.ClaimHeader{}, inboxpladla.ErrRowNotFound
	case err != nil:
		return inboxpladla.ClaimHeader{}, fmt.Errorf(
			"inboxpladla/sqlstore: detail_claim_header: %w", err)
	}

	return inboxpladla.ClaimHeader{
		ClaimKey:     strings.TrimSpace(claimKey.String),
		ClaimNo:      strings.TrimSpace(claimNo.String),
		PolicyNo:     strings.TrimSpace(policyNo.String),
		Insured:      strings.TrimSpace(insured.String),
		BusinessName: strings.TrimSpace(businessName.String),
		RegisterDate: dateText(registerDate),
		LossDate:     dateText(lossDate),
		PICTeknik:    strings.TrimSpace(picTeknik.String),
		StatusCode:   strings.TrimSpace(statusCode.String),
		StatusLabel:  strings.TrimSpace(statusLabel.String),
	}, nil
}

// adviceQueryFor memilih kueri grid pemberitahuan menurut jenisnya.
func adviceQueryFor(kind inboxpladla.AdviceKind) (string, error) {
	switch kind {
	case inboxpladla.AdviceKindPLA:
		return "detail_advices_pla", nil
	case inboxpladla.AdviceKindDLA:
		return "detail_advices_dla", nil
	default:
		return "", inboxpladla.ErrAdviceKindUnknown
	}
}

// Advices mengembalikan isi grid PLA atau grid DLA satu klaim.
func (r *Repo) Advices(
	ctx context.Context,
	scope inboxpladla.DetailScope,
	kind inboxpladla.AdviceKind,
) ([]inboxpladla.AdviceRow, error) {
	name, err := adviceQueryFor(kind)
	if err != nil {
		return nil, err
	}

	clean := scope.Clean()

	rows, err := r.db.QueryContext(ctx, query(name), clean.ClaimKey, clean.Login)
	if err != nil {
		return nil, fmt.Errorf("inboxpladla/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	items := []inboxpladla.AdviceRow{}
	for rows.Next() {
		var (
			adviceNo, adviceType, amount sql.NullString
			acceptanceNo                 sql.NullString
			adviceDate, sentDate         sql.NullTime
		)

		if err := rows.Scan(
			&adviceNo, &adviceType, &amount, &acceptanceNo, &adviceDate, &sentDate,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxpladla/sqlstore: %s: memindai baris: %w", name, err)
		}

		items = append(items, inboxpladla.AdviceRow{
			Kind:         kind,
			No:           strings.TrimSpace(adviceNo.String),
			Type:         strings.TrimSpace(adviceType.String),
			Amount:       strings.TrimSpace(amount.String),
			AcceptanceNo: strings.TrimSpace(acceptanceNo.String),
			AdviceDate:   dateText(adviceDate),
			SentDate:     dateText(sentDate),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladla/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return items, nil
}

// Documents mengembalikan dokumen satu nomor pemberitahuan.
func (r *Repo) Documents(
	ctx context.Context,
	scope inboxpladla.DetailScope,
	adviceNo string,
	kind inboxpladla.AdviceKind,
) ([]inboxpladla.DocumentRow, error) {
	if !kind.Valid() {
		return nil, inboxpladla.ErrAdviceKindUnknown
	}

	clean := scope.Clean()
	number := strings.TrimSpace(adviceNo)
	if number == "" {
		// Nomor pemberitahuan kosong akan mencocokkan baris `T_DOC_REAS` yang nomornya
		// pun kosong — bila ada. Ia ditolak lebih dulu, bukan diserahkan ke basis data.
		return nil, inboxpladla.ErrDocumentNotFound
	}

	rows, err := r.db.QueryContext(ctx, query("detail_documents"),
		clean.ClaimKey, number, string(kind), clean.Login, clean.Login)
	if err != nil {
		return nil, fmt.Errorf("inboxpladla/sqlstore: detail_documents: %w", err)
	}
	defer rows.Close()

	items := []inboxpladla.DocumentRow{}
	for rows.Next() {
		var (
			id, category, subCategory sql.NullString
			name, mimeType            sql.NullString
		)

		if err := rows.Scan(&id, &category, &subCategory, &name, &mimeType); err != nil {
			return nil, fmt.Errorf(
				"inboxpladla/sqlstore: detail_documents: memindai baris: %w", err)
		}

		items = append(items, inboxpladla.DocumentRow{
			ID:          strings.TrimSpace(id.String),
			Category:    strings.TrimSpace(category.String),
			SubCategory: strings.TrimSpace(subCategory.String),
			Name:        strings.TrimSpace(name.String),
			MimeType:    strings.TrimSpace(mimeType.String),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladla/sqlstore: detail_documents: membaca hasil: %w", err)
	}

	return items, nil
}

// DocumentContent mengembalikan ISI satu dokumen.
func (r *Repo) DocumentContent(
	ctx context.Context,
	scope inboxpladla.DetailScope,
	documentID string,
) (inboxpladla.DocumentContent, error) {
	clean := scope.Clean()
	id := strings.TrimSpace(documentID)
	if id == "" {
		return inboxpladla.DocumentContent{}, inboxpladla.ErrDocumentNotFound
	}

	row := r.db.QueryRowContext(ctx, query("detail_document_content"),
		id, clean.ClaimKey, clean.Login, clean.Login)

	var (
		name, mimeType sql.NullString
		content        []byte
	)

	err := row.Scan(&name, &mimeType, &content)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return inboxpladla.DocumentContent{}, inboxpladla.ErrDocumentNotFound
	case err != nil:
		return inboxpladla.DocumentContent{}, fmt.Errorf(
			"inboxpladla/sqlstore: detail_document_content: %w", err)
	}

	return inboxpladla.DocumentContent{
		Name:     strings.TrimSpace(name.String),
		MimeType: strings.TrimSpace(mimeType.String),
		Content:  content,
	}, nil
}

// Conversations mengembalikan riwayat komunikasi satu klaim yang menyangkut pemanggil.
func (r *Repo) Conversations(
	ctx context.Context,
	scope inboxpladla.DetailScope,
) ([]inboxpladla.Conversation, error) {
	clean := scope.Clean()

	rows, err := r.db.QueryContext(ctx, query("detail_conversations"),
		clean.ClaimKey, clean.Login, clean.Login)
	if err != nil {
		return nil, fmt.Errorf("inboxpladla/sqlstore: detail_conversations: %w", err)
	}
	defer rows.Close()

	items := []inboxpladla.Conversation{}
	for rows.Next() {
		var (
			id, senderName, message   sql.NullString
			reply, replierName, state sql.NullString
			createdAt, repliedAt      sql.NullTime
		)

		if err := rows.Scan(
			&id, &createdAt, &senderName, &message,
			&reply, &replierName, &repliedAt, &state,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxpladla/sqlstore: detail_conversations: memindai baris: %w", err)
		}

		items = append(items, buildConversation(
			strings.TrimSpace(id.String),
			dateText(createdAt),
			strings.TrimSpace(senderName.String),
			strings.TrimSpace(message.String),
			strings.TrimSpace(reply.String),
			strings.TrimSpace(replierName.String),
			dateText(repliedAt),
			strings.TrimSpace(state.String),
		))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxpladla/sqlstore: detail_conversations: membaca hasil: %w", err)
	}

	return items, nil
}

// buildConversation menyusun satu baris percakapan beserta kedua penandanya.
//
// Ia dipisahkan supaya penyimpanan memori dan penyimpanan SQL memutuskan "sudah dijawab"
// dan "boleh dibalas" dengan cara yang SAMA PERSIS. Keputusan yang disalin ke dua tempat
// akan bergeser pada perubahan berikutnya, dan yang bergeser di sini menggambar tombol
// balas pada percakapan yang akan menolaknya.
func buildConversation(
	id, createdAt, senderName, message, reply, replierName, repliedAt, state string,
) inboxpladla.Conversation {
	answered := state == inboxpladla.CommunicationAnswered

	return inboxpladla.Conversation{
		ID:          id,
		CreatedAt:   createdAt,
		SenderName:  senderName,
		Message:     message,
		Reply:       reply,
		ReplierName: replierName,
		RepliedAt:   repliedAt,

		Answered: answered,

		// Boleh dibalas ketika BELUM ada isi balasannya.
		//
		// Yang diperiksa adalah ISI, bukan penandanya, dan itu disengaja: pemagaran di
		// dalam `detail_reply` pun memeriksa `REPLYMESSAGE IS NULL`. Memakai penanda di
		// sini akan menggambar tombol balas pada percakapan yang pernyataannya akan
		// menolak — keduanya DAPAT berbeda pada data lama.
		CanReply: reply == "",
	}
}

// Reply menyimpan balasan atas satu percakapan.
//
// # Kenapa ia memeriksa JUMLAH BARIS, bukan membaca lebih dulu
//
// Karena membaca-lalu-menulis tidak menutup dua permintaan yang datang bersamaan: keduanya
// akan membaca "belum dijawab" lalu menulis, dan yang kedua menimpa yang pertama pada kolom
// yang sama. Pemagarannya karena itu berada di dalam klausa `WHERE` pernyataan tulisnya
// sendiri, dan jumlah baris terpengaruh itulah jawabannya.
//
// Kueri baca hanya dijalankan SESUDAHNYA, dan hanya untuk membedakan sebab penolakannya —
// ketika tidak ada lagi yang dapat rusak karenanya.
func (r *Repo) Reply(
	ctx context.Context,
	scope inboxpladla.DetailScope,
	command inboxpladla.ReplyCommand,
) error {
	clean := scope.Clean()

	result, err := r.db.ExecContext(ctx, query("detail_reply"),
		command.Message,
		command.Replier.Login,
		command.RepliedAt,
		command.ReplierName(),
		inboxpladla.AnsweredStatus,
		clean.ClaimKey,
		clean.Login,
		clean.Login,
		command.ConversationID,
	)
	if err != nil {
		return fmt.Errorf("inboxpladla/sqlstore: detail_reply: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"inboxpladla/sqlstore: detail_reply: membaca jumlah baris: %w", err)
	}
	if affected > 0 {
		return nil
	}

	// Tidak ada baris tersentuh. Sebabnya dua, dan keduanya menuntut kalimat yang berbeda
	// bagi pengguna.
	var total int
	if err := r.db.QueryRowContext(ctx, query("detail_conversation_exists"),
		clean.ClaimKey, command.ConversationID, clean.Login, clean.Login,
	).Scan(&total); err != nil {
		return fmt.Errorf(
			"inboxpladla/sqlstore: detail_conversation_exists: %w", err)
	}

	if total == 0 {
		return inboxpladla.ErrConversationNotFound
	}
	return inboxpladla.ErrConversationAlreadyAnswered
}
