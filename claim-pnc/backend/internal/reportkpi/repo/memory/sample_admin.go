package memory

import "claim-pnc/internal/reportkpi"

// SampleAdminRows adalah baris contoh tab KPI Admin.
//
// # Seluruh isinya KARANGAN
//
// Nomor klaim, nomor polis, dan nama bisnis tidak diambil dari produksi (`D-69`).
//
// # Angkanya dipilih supaya kartu skornya DAPAT DIPERIKSA DENGAN TANGAN
//
// Itu yang membedakan contoh ini dari sekadar "beberapa baris supaya layar tidak kosong":
// kartu skor adalah hasil hitungan bertingkat — persentase, tangga nilai, bobot, subtotal,
// lalu rasio pencapaian — dan satu tingkat yang salah tidak terlihat sebagai kerusakan.
//
// Dengan angka di bawah, kelompok NON-MBU pada Maret 2026 menghasilkan:
//
//	leader   1 dari 4 melewati SLA  ->  25%    -> nilai 0 (di atas 2)
//	member   0 dari 3 melewati SLA  ->   0%    -> nilai 5 (di bawah 0,5)
//	subtotal leader 0 · member 40   ->  total 40
//	pencapaian 40 / ((3/5)*90) = 0,74  ->  TIDAK TERCAPAI TARGET
//
// Angka-angka itu dituliskan di uji, sehingga kekeliruan pada tingkat mana pun langsung
// terlihat sebagai uji yang gagal — bukan sebagai kartu yang tampak masuk akal.
func SampleAdminRows() []AdminRow {
	return []AdminRow{
		// ── NON-MBU, leader: 4 klaim, 1 melewati SLA ────────────────────────────
		//
		// Ambang "melewati SLA" di sini `> 1` hari kerja. Baris pertama TEPAT di angka 1
		// dan karena itu TIDAK melewati — batas yang paling mudah keliru dibaca sebagai
		// `>=`.
		adminNonMBU("CONTOH-ADM-0001", "leader", true, "2026-03-02", 1.0),
		adminNonMBU("CONTOH-ADM-0002", "leader", true, "2026-03-05", 0.5),
		adminNonMBU("CONTOH-ADM-0003", "leader", true, "2026-03-11", 0.25),
		adminNonMBU("CONTOH-ADM-0004", "leader", true, "2026-03-19", 2.5),

		// ── NON-MBU, member: 3 klaim, tidak satu pun melewati SLA ───────────────
		//
		// Nol melewati SLA adalah hasil TERBAIK, dan itu menghasilkan nilai 5. Ia ada di
		// sini supaya perbedaan antara "nol" dan "tidak ada" terlihat di layar: yang ini
		// nol sungguhan, bukan kolom yang tidak terisi.
		adminNonMBU("CONTOH-ADM-0005", "member", false, "2026-03-03", 0.5),
		adminNonMBU("CONTOH-ADM-0006", "member", false, "2026-03-14", 0.75),
		adminNonMBU("CONTOH-ADM-0007", "member", false, "2026-03-28", 1.0),

		// ── NON-MBU di LUAR periode contoh ──────────────────────────────────────
		//
		// Tanpa baris yang tertolak, penyaring yang tidak bekerja sama sekali tidak dapat
		// dibedakan dari penyaring yang bekerja.
		adminNonMBU("CONTOH-ADM-0008", "leader", true, "2026-02-27", 9),
		adminNonMBU("CONTOH-ADM-0009", "member", false, "2026-04-02", 9),

		// ── PA: 3 klaim pada periode contoh ─────────────────────────────────────
		//
		// Ambangnya `> 0`, BUKAN `> 1` seperti NON-MBU — sehingga baris ber-TAT 0,5 pun
		// sudah terhitung melewati SLA. Perbedaan ambang itu ditiru apa adanya, dan
		// baris pertama ada justru untuk membuktikannya.
		adminPA("CONTOH-ADM-0101", "2026-03-04", 0.5, 0.5, true, false),
		adminPA("CONTOH-ADM-0102", "2026-03-17", 0, 0, true, false),
		adminPA("CONTOH-ADM-0103", "2026-03-23", 3, 4, true, false),

		// ── PA di dalam rentang 2023 yang TERTANAM ──────────────────────────────
		//
		// Kedua baris ini berada JAUH di luar periode contoh, tetapi tetap ikut menghitung
		// "TOTAL KLAIM BAYAR" — karena pembagi itu terkunci pada 2023 di dalam kueri lama.
		// Tanpa baris seperti ini, keanehan tersebut tidak akan pernah terlihat di layar
		// pengembangan. Lihat reportkpi_admin.sql.
		adminPA("CONTOH-ADM-0104", "2023-05-10", 2, 2, true, true),
		adminPA("CONTOH-ADM-0105", "2023-09-21", 1.5, 3, true, true),

		// ── PA tanpa akseptasi ──────────────────────────────────────────────────
		//
		// `NOAKSEPTASI IS NULL` — ia ikut TOTAL KLAIM REGIST tetapi TIDAK ikut pencacah
		// pembayaran mana pun.
		adminPA("CONTOH-ADM-0106", "2026-03-26", 0.25, 0, false, false),
	}
}

// adminNonMBU menyusun satu baris contoh kelompok NON-MBU.
func adminNonMBU(claim, flag string, leader bool, registerDate string, aging float64) AdminRow {
	return AdminRow{
		Group:         reportkpi.AdminGroupNonMBU,
		ClaimNumber:   claim,
		PolicyNumber:  "CONTOH-POL-" + claim[len(claim)-4:],
		BusinessName:  "BISNIS CONTOH NON MBU",
		TeamFlag:      flag,
		Leader:        leader,
		RegisterDate:  registerDate,
		TransferDate:  registerDate,
		RegisterAging: aging,
		AdminName:     "ADMINCONTOH",
		ClaimStatus:   "Outstanding",
	}
}

// adminPA menyusun satu baris contoh kelompok PA.
func adminPA(
	claim, registerDate string,
	registerAging, paymentAging float64,
	hasPayment, legacyWindow bool,
) AdminRow {
	row := AdminRow{
		Group:                 reportkpi.AdminGroupPA,
		ClaimNumber:           claim,
		PolicyNumber:          "CONTOH-POL-" + claim[len(claim)-4:],
		RegisterDate:          registerDate,
		ReceiveDate:           registerDate,
		RegisterAging:         registerAging,
		PaymentAging:          paymentAging,
		HasPayment:            hasPayment,
		AdminName:             "ADMINCONTOHPA",
		ClaimStatus:           "Close",
		InLegacyPaymentWindow: legacyWindow,
	}

	if hasPayment {
		row.LODReceiveDate = registerDate
		row.AcceptanceDate = registerDate
	}

	// Penanda rentang 2023 dihitung dari tanggalnya sendiri bila pemanggil tidak
	// menyatakannya, supaya contoh dan aturannya tidak dapat berselisih.
	if !legacyWindow {
		row.InLegacyPaymentWindow = adminWithinYear(registerDate, 2023)
	}
	return row
}
