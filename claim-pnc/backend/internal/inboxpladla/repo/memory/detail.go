package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/inboxpladla"
)

// Berkas ini memenuhi bagian RINCIAN pada seam inboxpladla.Repo tanpa basis data.
//
// # Apa yang benar-benar diuji di sini, dan tidak dapat diuji di tempat lain
//
// Batas kepemilikannya. Di penyimpanan SQL ia hidup di dalam teks kueri — sebagai `IN
// (SELECT … WHERE LOGIN = …)` — sehingga tidak ada uji yang dapat membuktikannya tanpa
// Oracle. Di sini ia kode Go biasa, dan setiap jalur penolakannya dapat ditembak.
//
// Itu bukan kenyamanan melainkan keharusan: layar ini dibaca PIHAK LUAR, dan satu batas
// yang hilang tidak menghasilkan galat — hanya data mitra lain yang terbuka.

// Message adalah satu baris `POOLDATA.M_KOMUNIKASI_PNC` sejauh yang dibaca modul ini.
type Message struct {
	// ID adalah `KOMUNIKASIID`.
	ID string

	// ClaimKey adalah `CASEID`.
	//
	// Perhatikan: di modul `inboxkomunikasicabang` kolom yang sama berisi NAMA KANAL
	// (`'CABANG'`). Di layar ini ia berisi kunci klaim. Satu kolom, dua arti.
	ClaimKey string

	// SenderLogin adalah `SENDER`, SenderName adalah `SENDERNAME`.
	SenderLogin string
	SenderName  string

	// RecipientCode adalah `COMMUNICATE_TO`.
	RecipientCode string

	Message   string
	CreatedAt time.Time

	// Reply, ReplierLogin, ReplierName, dan RepliedAt terisi setelah dibalas.
	Reply        string
	ReplierLogin string
	ReplierName  string
	RepliedAt    time.Time

	// Status adalah `KOMUNIKASISTATUS` — `0` belum dijawab, `1` sudah.
	Status string
}

// Document adalah satu dokumen reasuransi: `T_DOC_REAS` digabung `DATA_ATTACHFILE`.
//
// Keduanya digabung MENDATAR di sini karena hubungannya satu-ke-satu lewat
// `DOKUMENID = DATAID`. Yang diuji adalah penyaringnya, bukan cara gabungannya disusun.
type Document struct {
	// ID adalah `DATA_ATTACHFILE.DATAID`.
	ID string

	// ClaimKey, AdviceNo, dan Kind adalah kunci `T_DOC_REAS`.
	ClaimKey string
	AdviceNo string
	Kind     inboxpladla.AdviceKind

	// Login adalah `T_DOC_REAS.LOGIN` — kolom yang `GetDokumenReas` TIDAK pakai, dan yang
	// di sini DIPAKAI atas keputusan Work Owner.
	Login string

	Category    string
	SubCategory string
	Name        string
	MimeType    string
	Content     []byte
}

// SeedMessages mengisi percakapan contoh.
//
// Ia terpisah dari Seed, bukan ditambahkan sebagai parameter kedelapan, supaya data contoh
// dan uji yang sudah ada tidak perlu disentuh untuk menambah satu jenis data baru.
func (s *Store) SeedMessages(messages []Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append([]Message{}, messages...)
}

// SeedDocuments mengisi dokumen contoh.
func (s *Store) SeedDocuments(documents []Document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents = append([]Document{}, documents...)
}

// hasConversation menyatakan sebuah klaim punya percakapan yang cocok dengan tab ini.
//
// Pemanggil sudah memegang kunci baca.
func (s *Store) hasConversation(claimKey string, tab inboxpladla.Tab, login string) bool {
	for _, message := range s.messages {
		if message.ClaimKey != claimKey {
			continue
		}
		if message.Status != tab.CommunicationStatus {
			continue
		}
		if matchesRole(message, tab.CommunicationRole, login) {
			return true
		}
	}
	return false
}

// matchesRole menyatakan sebuah percakapan berada pada sisi yang diminta tab.
func matchesRole(
	message Message, role inboxpladla.CommunicationRole, login string,
) bool {
	switch role {
	case inboxpladla.RoleRecipient:
		return sameIdentity(message.RecipientCode, login)
	case inboxpladla.RoleSender:
		return sameIdentity(message.SenderLogin, login)
	default:
		return false
	}
}

// involves menyatakan sebuah percakapan menyangkut pemanggil — sisi mana pun.
//
// Layar RINCIAN memakai ini, bukan matchesRole: pemanggil berhak membaca percakapan yang
// ditujukan kepadanya MAUPUN yang ia kirim sendiri, sama seperti `detail_conversations`.
func involves(message Message, login string) bool {
	return sameIdentity(message.RecipientCode, login) ||
		sameIdentity(message.SenderLogin, login)
}

// sameIdentity membandingkan dua identitas tanpa memedulikan huruf dan spasi di ujung.
//
// Sama dengan `UPPER(TRIM(…)) = UPPER(TRIM(…))` di SQL. Identitas kosong TIDAK pernah
// cocok — tanpa syarat itu, pemanggil tanpa login akan mencocokkan setiap baris yang
// kolomnya kosong.
func sameIdentity(left, right string) bool {
	l := strings.TrimSpace(left)
	r := strings.TrimSpace(right)
	if l == "" || r == "" {
		return false
	}
	return strings.EqualFold(l, r)
}

// ClaimHeader mengembalikan keterangan klaim di kepala layar rincian.
func (s *Store) ClaimHeader(
	_ context.Context,
	scope inboxpladla.DetailScope,
) (inboxpladla.ClaimHeader, error) {
	clean := scope.Clean()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, claim := range s.claims {
		if claim.Key != clean.ClaimKey {
			continue
		}
		if !s.callerMayOpenLocked(clean) {
			break
		}

		return inboxpladla.ClaimHeader{
			ClaimKey:     claim.Key,
			ClaimNo:      claim.No,
			PolicyNo:     claim.PolicyNo,
			Insured:      claim.Insured,
			BusinessName: claim.BusinessName,
			RegisterDate: dateText(claim.RegisterDate),
			LossDate:     dateText(claim.LossDate),
			PICTeknik:    claim.PICTeknik,
			StatusCode:   claim.StatusCode,
			StatusLabel:  s.labelOfLocked(claim.StatusCode),
		}, nil
	}

	// Klaim yang tidak ada dan klaim yang bukan milik pemanggil dijawab SAMA.
	return inboxpladla.ClaimHeader{}, inboxpladla.ErrRowNotFound
}

// callerMayOpenLocked menyatakan pemanggil berhak membuka rincian sebuah klaim.
//
// Syaratnya sama dengan `detail_claim_header`: sekurang-kurangnya satu pemberitahuan yang
// TERKIRIM kepadanya, ATAU satu percakapan yang menyangkutnya. Keduanya itulah yang
// membuat klaim ini muncul di salah satu daftar layar induk.
//
// Pemanggil sudah memegang kunci baca.
func (s *Store) callerMayOpenLocked(scope inboxpladla.DetailScope) bool {
	if s.hasSentAdvice(scope.ClaimKey, "pla", scope.ReinsurerCodes) ||
		s.hasSentAdvice(scope.ClaimKey, "dla", scope.ReinsurerCodes) {
		return true
	}

	for _, message := range s.messages {
		if message.ClaimKey == scope.ClaimKey && involves(message, scope.Login) {
			return true
		}
	}
	return false
}

// Advices mengembalikan isi grid PLA atau grid DLA satu klaim.
func (s *Store) Advices(
	_ context.Context,
	scope inboxpladla.DetailScope,
	kind inboxpladla.AdviceKind,
) ([]inboxpladla.AdviceRow, error) {
	if !kind.Valid() {
		return nil, inboxpladla.ErrAdviceKindUnknown
	}

	clean := scope.Clean()

	s.mu.RLock()
	defer s.mu.RUnlock()

	wanted := strings.ToLower(string(kind))
	rows := []inboxpladla.AdviceRow{}

	for _, advice := range s.advices {
		if advice.ClaimKey != clean.ClaimKey || advice.Kind != wanted {
			continue
		}

		// Hanya yang SUDAH terkirim, dan hanya kepada pemanggil. Keduanya bersama-sama
		// itulah yang menggantikan grid Pega yang memuat seluruh mitra.
		if advice.Sent != "1" {
			continue
		}
		if !containsValue(clean.ReinsurerCodes, advice.ReinsCode) {
			continue
		}

		rows = append(rows, inboxpladla.AdviceRow{
			Kind:         kind,
			No:           advice.No,
			Type:         advice.Type,
			Amount:       advice.Amount,
			AcceptanceNo: advice.AcceptanceNo,
			AdviceDate:   dateText(advice.AdviceDate),
			SentDate:     dateText(advice.SentDate),
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].AdviceDate != rows[j].AdviceDate {
			return rows[i].AdviceDate < rows[j].AdviceDate
		}
		return rows[i].No < rows[j].No
	})

	return rows, nil
}

// Documents mengembalikan dokumen satu nomor pemberitahuan.
func (s *Store) Documents(
	_ context.Context,
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
		return nil, inboxpladla.ErrDocumentNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := []inboxpladla.DocumentRow{}
	for _, document := range s.documents {
		if !s.documentBelongsLocked(document, clean, number, kind) {
			continue
		}
		rows = append(rows, inboxpladla.DocumentRow{
			ID:          document.ID,
			Category:    document.Category,
			SubCategory: document.SubCategory,
			Name:        document.Name,
			MimeType:    document.MimeType,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].ID < rows[j].ID
	})

	return rows, nil
}

// documentBelongsLocked menegakkan SELURUH rantai kepemilikan satu dokumen.
//
// Keempat syaratnya sama persis dengan `detail_documents`, dan keempatnya diperiksa —
// bukan sebagian. Syarat terakhir yang paling mudah terlupa: nomor pemberitahuannya harus
// benar-benar TERKIRIM kepada pemanggil, bukan sekadar tercatat atas namanya.
//
// Pemanggil sudah memegang kunci baca.
func (s *Store) documentBelongsLocked(
	document Document,
	scope inboxpladla.DetailScope,
	adviceNo string,
	kind inboxpladla.AdviceKind,
) bool {
	if document.ClaimKey != scope.ClaimKey ||
		document.AdviceNo != adviceNo ||
		document.Kind != kind {
		return false
	}
	if !sameIdentity(document.Login, scope.Login) {
		return false
	}
	return s.adviceSentToCallerLocked(scope, adviceNo)
}

// adviceSentToCallerLocked menyatakan sebuah nomor pemberitahuan memang terkirim kepada
// pemanggil — jenis apa pun.
//
// Pemanggil sudah memegang kunci baca.
func (s *Store) adviceSentToCallerLocked(
	scope inboxpladla.DetailScope, adviceNo string,
) bool {
	for _, advice := range s.advices {
		if advice.ClaimKey != scope.ClaimKey || advice.No != adviceNo {
			continue
		}
		if advice.Sent != "1" {
			continue
		}
		if containsValue(scope.ReinsurerCodes, advice.ReinsCode) {
			return true
		}
	}
	return false
}

// DocumentContent mengembalikan ISI satu dokumen.
func (s *Store) DocumentContent(
	_ context.Context,
	scope inboxpladla.DetailScope,
	documentID string,
) (inboxpladla.DocumentContent, error) {
	clean := scope.Clean()
	id := strings.TrimSpace(documentID)
	if id == "" {
		return inboxpladla.DocumentContent{}, inboxpladla.ErrDocumentNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, document := range s.documents {
		if document.ID != id {
			continue
		}
		// Rantai kepemilikannya diperiksa ULANG seluruhnya — id dokumen adalah angka,
		// dan angka dapat ditebak.
		if !s.documentBelongsLocked(document, clean, document.AdviceNo, document.Kind) {
			break
		}
		return inboxpladla.DocumentContent{
			Name:     document.Name,
			MimeType: document.MimeType,
			Content:  append([]byte{}, document.Content...),
		}, nil
	}

	return inboxpladla.DocumentContent{}, inboxpladla.ErrDocumentNotFound
}

// Conversations mengembalikan riwayat komunikasi satu klaim yang menyangkut pemanggil.
func (s *Store) Conversations(
	_ context.Context,
	scope inboxpladla.DetailScope,
) ([]inboxpladla.Conversation, error) {
	clean := scope.Clean()

	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := []inboxpladla.Conversation{}
	for _, message := range s.messages {
		if message.ClaimKey != clean.ClaimKey || !involves(message, clean.Login) {
			continue
		}

		rows = append(rows, inboxpladla.Conversation{
			ID:          message.ID,
			CreatedAt:   dateText(message.CreatedAt),
			SenderName:  message.SenderName,
			Message:     message.Message,
			Reply:       message.Reply,
			ReplierName: message.ReplierName,
			RepliedAt:   dateText(message.RepliedAt),

			Answered: message.Status == inboxpladla.CommunicationAnswered,

			// Sama dengan penyimpanan SQL: yang diperiksa adalah ISI balasannya, bukan
			// penandanya. Lihat buildConversation di sqlstore/detail.go.
			CanReply: strings.TrimSpace(message.Reply) == "",
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].CreatedAt != rows[j].CreatedAt {
			return rows[i].CreatedAt < rows[j].CreatedAt
		}
		return rows[i].ID < rows[j].ID
	})

	return rows, nil
}

// Reply menyimpan balasan atas satu percakapan.
//
// Ketiga pemagarannya sama dengan `detail_reply`: kunci klaim, keterlibatan pemanggil, dan
// balasan yang belum ada.
func (s *Store) Reply(
	_ context.Context,
	scope inboxpladla.DetailScope,
	command inboxpladla.ReplyCommand,
) error {
	clean := scope.Clean()

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.messages {
		message := s.messages[i]

		if message.ID != command.ConversationID ||
			message.ClaimKey != clean.ClaimKey ||
			!involves(message, clean.Login) {
			continue
		}

		if strings.TrimSpace(message.Reply) != "" {
			return inboxpladla.ErrConversationAlreadyAnswered
		}

		s.messages[i].Reply = command.Message
		s.messages[i].ReplierLogin = command.Replier.Login
		s.messages[i].ReplierName = command.ReplierName()
		s.messages[i].RepliedAt = command.RepliedAt
		s.messages[i].Status = inboxpladla.AnsweredStatus
		return nil
	}

	return inboxpladla.ErrConversationNotFound
}
