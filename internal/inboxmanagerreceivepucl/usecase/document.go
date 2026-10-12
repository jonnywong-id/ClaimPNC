package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

// Document adalah isi layar kerja penerimaan dokumen beserta bentuk layarnya.
type Document struct {
	// Detail adalah isi berkasnya.
	Detail inboxmanagerreceivepucl.ReceiveDocument

	// Groups adalah susunan isian layar, termasuk yang terhalang.
	//
	// Ia ikut dikirim bersama isinya, bukan diminta terpisah, karena keduanya selalu dipakai
	// bersamaan — layar tidak dapat menggambar satu isian pun tanpa tahu judul dan urutannya.
	// Memisahkannya menjadi dua permintaan membuat layar menggambar setengah jadi lebih dulu.
	Groups []inboxmanagerreceivepucl.FieldGroup

	// Actions adalah tombol yang di layar lama mengubah data.
	//
	// Belum satu pun dapat dihidupkan; lihat inboxmanagerreceivepucl.WriteAction. Ia tetap
	// dikirim supaya tombolnya digambar dan penekanannya menjawab alasan.
	Actions []inboxmanagerreceivepucl.WriteAction
}

// Document mengambil isi layar kerja penerimaan dokumen untuk satu berkas.
//
// # Kenapa ia operasi tersendiri, bukan bagian dari List
//
// Karena yang dibaca memang berbeda: grid membaca sembilan kolom dari dua tabel yang
// digabung ke tabel penugasan, layar kerja membaca dua puluh tiga kolom tanpa tabel
// penugasan sama sekali. Menurunkannya dari baris grid akan membuat layar kerja menampilkan
// isian yang tidak pernah dibaca kueri mana pun.
//
// # Kunci yang kosong ditolak di sini, bukan diserahkan ke penyimpanan
//
// Penyimpanan yang menerima kunci kosong akan mengembalikan "tidak ditemukan", dan itu
// jawaban yang menyesatkan: yang salah bukan berkasnya melainkan permintaannya.
func (s *Service) Document(
	ctx context.Context,
	portalAlias string,
	caller inboxmanagerreceivepucl.Caller,
	reference string,
) (Document, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Document{}, inboxmanagerreceivepucl.ErrCallerUnknown
	}

	key := strings.TrimSpace(reference)
	if key == "" {
		return Document{}, inboxmanagerreceivepucl.ErrReferenceRequired
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Document{}, err
	}

	detail, err := repo.Document(ctx, key)
	if err != nil {
		// Galat domain diteruskan APA ADANYA supaya lapisan transport dapat memetakannya ke
		// kode HTTP yang tepat. Membungkusnya di sini akan membuat "berkas tidak ditemukan"
		// sampai ke pengguna sebagai kegagalan internal.
		if errorsIsDomain(err) {
			return Document{}, err
		}
		return Document{}, fmt.Errorf("mengambil berkas penerimaan dokumen: %w", err)
	}

	// SETIAP pembukaan dicatat, sama seperti pembukaan daftarnya.
	//
	// Di sini alasannya lebih kuat: yang dibuka bukan ringkasan satu baris melainkan ISI
	// berkas — nama tertanggung, nomor polis, kronologi kejadian, dan alamat surel pelapor.
	// Selama pemeriksaan peran belum ada (`TKT-F3-004`), jejak inilah satu-satunya kontrol
	// pengimbang (`D-59`).
	if s.logger != nil {
		s.logger.Info(
			"layar kerja penerimaan dokumen dibuka",
			slog.String("modul", "inbox-manager-receive-pucl"),
			slog.String("berkas", detail.CaseID),
			slog.String("pemanggil", clean.Login),
			slog.String("portal", portalAlias),
		)
	}

	return Document{
		Detail: detail,
		// Bentuk layar disusun UNTUK BERKAS INI, bukan daftar tetap: 14 dari 24 isian punya
		// syarat tampil di section Pega, dan empat di antaranya sudah dapat diterjemahkan
		// dari Group Panel dan nomor polis berkas ini sendiri.
		Groups:  inboxmanagerreceivepucl.DocumentFieldGroupsFor(detail),
		Actions: inboxmanagerreceivepucl.DocumentWriteActionList(),
	}, nil
}

// errorsIsDomain menyatakan galat ini milik domain modul dan layak diteruskan apa adanya.
//
// Ia dikumpulkan sebagai fungsi supaya daftarnya hidup di satu tempat: galat domain baru yang
// lupa ditambahkan di sini akan sampai ke pengguna sebagai 500, dan itu tidak menyebut
// sebabnya kepada siapa pun.
func errorsIsDomain(err error) bool {
	return errors.Is(err, inboxmanagerreceivepucl.ErrDocumentNotFound) ||
		errors.Is(err, inboxmanagerreceivepucl.ErrReferenceRequired) ||
		errors.Is(err, inboxmanagerreceivepucl.ErrCallerUnknown)
}
