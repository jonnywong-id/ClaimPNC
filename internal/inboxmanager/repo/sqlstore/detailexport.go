package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/inboxmanager"
)

// WithDetailRules memasang aturan fee adjuster dan nama tahapan progres yang dipakai ekspor
// "Export Data Detail Klaim". Lihat inboxmanager.DetailRules.
//
// Tanpa ini ekspor tetap berjalan; hanya "Adjuster Fee" dan "Status Progress 2" yang kosong.
func (r *Repo) WithDetailRules(rules inboxmanager.DetailRules) *Repo {
	r.rules = rules
	return r
}

// DetailExport mengalirkan baris berkas "Export Data Detail Klaim".
//
// # Urutan kerjanya
//
// Tangga fee dan nama tahapan dibaca SEKALI lebih dulu — fungsi basis data aslinya membacanya
// sekali PER BARIS. Lalu kueri utama dibaca baris demi baris, setiap baris dilengkapi kolom
// turunannya (deriveDetailRow), dan langsung diteruskan ke emit.
func (r *Repo) DetailExport(
	ctx context.Context,
	q inboxmanager.DetailExportQuery,
	emit func(inboxmanager.DetailExportRow) error,
) error {
	fee := r.detailFeeScale(ctx)
	progress := r.detailProgressNames(ctx)

	line := q.LineBusiness
	if q.AllLines {
		line = ""
	}
	args := append(lineFlags(line),
		boolFlag(!q.HasPeriod()), q.From, q.Until,
	)

	rows, err := r.db.QueryContext(ctx, query("detail_export"), args...)
	if err != nil {
		return wrapQueryError("menjalankan kueri detail_export", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("membaca kolom detail_export: %w", err)
	}

	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	for rows.Next() {
		for i := range values {
			values[i] = nil
		}
		if err := rows.Scan(pointers...); err != nil {
			return fmt.Errorf("membaca baris detail_export: %w", err)
		}

		raw := detailRaw{text: map[string]string{}, date: map[string]time.Time{}}
		for i, name := range columns {
			raw.put(name, values[i])
		}

		if err := emit(deriveDetailRow(raw, fee, progress)); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("menelusuri hasil detail_export: %w", err)
	}
	return nil
}

// detailFeeScale membaca tangga fee adjuster. Nil bila aturannya tidak dipasang atau
// tangganya tidak terbaca — kolom "Adjuster Fee" lalu kosong, bukan nol, seperti modul
// Report Klaim.
func (r *Repo) detailFeeScale(ctx context.Context) inboxmanager.FeeScale {
	if r.rules.NewFeeScale == nil {
		return nil
	}
	rows, err := r.db.QueryContext(ctx, query("detail_fee_scale"))
	if err != nil {
		return nil
	}
	defer rows.Close()

	bands := []inboxmanager.FeeBand{}
	for rows.Next() {
		var index sql.NullInt64
		var amount, fee sql.NullFloat64
		if err := rows.Scan(&index, &amount, &fee); err != nil {
			return nil
		}
		if index.Valid && amount.Valid && fee.Valid {
			bands = append(bands, inboxmanager.FeeBand{
				Index: int(index.Int64), LossAmount: amount.Float64, Fee: fee.Float64,
			})
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return r.rules.NewFeeScale(bands)
}

// detailProgressNames membaca nama tahapan progres. Nil bila aturannya tidak dipasang atau
// masternya tidak terbaca — kolom "Status Progress 2" lalu kosong.
func (r *Repo) detailProgressNames(ctx context.Context) inboxmanager.ProgressNames {
	if r.rules.NewProgressNames == nil {
		return nil
	}
	rows, err := r.db.QueryContext(ctx, query("detail_progress_names"))
	if err != nil {
		return nil
	}
	defer rows.Close()

	names := map[string]string{}
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil
		}
		if id.Valid {
			names[strings.TrimSpace(id.String)] = name.String
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return r.rules.NewProgressNames(names)
}

// detailRaw adalah satu baris kueri sebelum dilengkapi, dipisah menurut jenis nilainya.
type detailRaw struct {
	text map[string]string
	date map[string]time.Time
}

// put menyimpan satu nilai dari driver.
//
// Angka datang dari go-ora sebagai TEKS presisi penuh — diperiksa langsung 2026-10-14 — dan
// disimpan apa adanya, tanpa melewati float (`I-12`).
func (r detailRaw) put(name string, value any) {
	switch v := value.(type) {
	case nil:
	case time.Time:
		r.date[name] = v
	case string:
		r.text[name] = v
	case []byte:
		r.text[name] = string(v)
	case int64:
		r.text[name] = strconv.FormatInt(v, 10)
	case float64:
		r.text[name] = strconv.FormatFloat(v, 'f', -1, 64)
	case fmt.Stringer:
		r.text[name] = v.String()
	default:
		r.text[name] = fmt.Sprint(v)
	}
}

// detailCopied adalah kolom yang ditulis apa adanya oleh activity (`X <= .X`).
var detailCopied = []string{
	"CaseID", "pyStatusMessage", "AlasanDokterRejectRCL", "TKI", "Usulan", "ResponseMsg",
	"AnalystRemaksInvestigator", "AnaylstRemarks", "City", "CauseOfLoss", "CityID",
	"ClaimEstimate", "CommentKomiteClosecase", "DaftarObjek", "Location", "pyID",
	"DistrictID", "Province", "AgingAmount", "ReporterTelp", "ResponseNote",
	"StatusKomunikasi", "pyNote", "UserBusinessPA", "OwnRisk", "SuspiciousComment",
	"StatusAnalystRemarks", "TypeOfAddress", "TypeOldNoHp", "Resources", "viewnotelp",
	"HideKTP", "HidePelapor",
}

// detailCommaToSpace adalah kolom yang activity-nya tulis `@replaceAll(.X, ",", " ")`.
var detailCommaToSpace = []string{"AnalystDoctorRemaks", "ProvinceID"}

// detailDecimalComma adalah kolom yang activity-nya tulis `@replaceAll(.X, ".", ",")` —
// pemisah desimal Indonesia untuk Excel.
var detailDecimalComma = []string{
	"CountryID", "Currency", "District", "DokumenLengkap", "ExGratiaNote", "FlagReject",
	"InsuredRelationship", "PolicyObjectLocationRWNote", "IsBackCFS", "isComplianceTransfer",
	"IsTransferAnalisator", "IsTransferPIC", "InsuredRelationshipOthers", "KomiteStatus",
	"LokasiSurveyor", "LossCoverage", "NamaDokumen", "NamaSurveyor", "NIK", "NoKTP", "ProdKe",
	"Password", "RCV_ID", "ReportAddress", "StatusEkonomis", "Remark", "RWID",
}

// detailORProducts memetakan keempat kolom "… OR" yang dikalikan ke nilai pengalinya.
//
// Activity-nya: `ReportDescription <= @replaceAll(@toDecimal(.Password) *
// @toDecimal(.ReportDescription), ".", ",")`, dan seterusnya — `.ReportDescription` di
// ruas kanan adalah FAKTOR OR dari kueri, bukan hasilnya.
var detailORProducts = map[string]string{
	"ReportDescription": "Password",
	"StatusClaim":       "RCV_ID",
	"TelpTertanggung":   "ReportAddress",
	"UserAdmin":         "StatusEkonomis",
}

// deriveDetailRow menyusun satu baris berkas dari satu baris kueri — padanan langkah
// Property-Set per baris di `Activity/ExportDataDetailKlaim`.
func deriveDetailRow(
	raw detailRaw,
	fee inboxmanager.FeeScale,
	progress inboxmanager.ProgressNames,
) inboxmanager.DetailExportRow {
	row := inboxmanager.DetailExportRow{}

	for _, name := range detailCopied {
		row[name] = raw.text[name]
	}
	for _, name := range detailCommaToSpace {
		row[name] = strings.ReplaceAll(raw.text[name], ",", " ")
	}
	for _, name := range detailDecimalComma {
		row[name] = decimalComma(raw.text[name])
	}

	// Kelima kolom "… OR". Empat dikalikan; yang kelima — "Selisih … Adjuster OR" — adalah
	// faktornya sendiri, karena activity-nya menulis `OldAddress <= @replaceAll(.OldAddress)`
	// tanpa perkalian. Itu cacat Pega yang dibawa apa adanya.
	factor := raw.text["FaktorOR"]
	for target, source := range detailORProducts {
		row[target] = decimalComma(multiplyDecimal(raw.text[source], factor))
	}
	row["OldAddress"] = decimalComma(factor)

	row["ReporterName"] = decimalComma(adjusterFee(raw, fee))
	if progress != nil {
		row["AlasanTerlambat"] = progress.Positions(raw.text["ProgresJSON"])
	}

	// Tanggal — kueri Pega memformatnya dengan TO_CHAR; di sini dari tanggal yang sama.
	dol, hasDOL := raw.date["TglDateOfLoss"]
	row["StatusWork"] = formatDate(dol, hasDOL, "02/01/2006")
	row["ClaimNo"] = formatDate(dol, hasDOL, "01")
	row["CloseClaimNote"] = formatDate(dol, hasDOL, "2006")

	process, hasProcess := raw.date["TglProses"]
	row["UserTeknisEmail"] = formatDate(process, hasProcess, "02/01/2006")

	begin, hasBegin := raw.date["TglBegin"]
	row["RefNo"] = formatDate(begin, hasBegin, "2006")
	row["UserTeknisGroup"] = formatDate(begin, hasBegin, "02/01/2006")

	accept, hasAccept := raw.date["TglAksep"]
	row["DollarCurrencyVal"] = formatDate(accept, hasAccept, "02/01/2006")
	row["CompliancePosAuditByr"] = formatDate(accept, hasAccept, "01")
	row["ComplianceRemark"] = formatDate(accept, hasAccept, "2006")

	reject, hasReject := raw.date["TglReject"]
	row["UserTeknis"] = formatDate(reject, hasReject, "02/01/2006")
	row["Conveyance"] = formatDate(reject, hasReject, "01")
	row["Country"] = formatDate(reject, hasReject, "2006")

	end, hasEnd := raw.date["TglEnd"]
	row["RemarkRecommendation"] = formatDate(end, hasEnd, "02/01/2006")

	// TAT: `TO_DATE(TO_CHAR(tgl_proses)) - TO_DATE(TO_CHAR(tgl_aksep))` — selisih hari
	// kalender, jam diabaikan.
	if hasProcess && hasAccept {
		row["UserName"] = strconv.Itoa(calendarDays(accept, process))
	} else {
		row["UserName"] = ""
	}

	update, hasUpdate := raw.date["TglUpdateProgres"]
	row["ReceiverClaim"] = formatDate(update, hasUpdate, "02/01/06")
	next, hasNext := raw.date["TglNextFollowUp"]
	row["StatusReceiver"] = formatDate(next, hasNext, "02/01/2006")
	// Kueri Pega mengisi `view_email` dengan TGL_INPUT dan `view_ktp` dengan NEXT_FOLLOWUP
	// progres komite, sementara berkasnya menaruh `view_ktp` di bawah judul "Tgl Kirim" dan
	// `view_email` di bawah "Tgl Jatuh Tempo". Judulnya karena itu tampak tertukar di berkas
	// lama, dan itu dibawa apa adanya — hasilnya SAMA dengan Pega.
	sent, hasSent := raw.date["TglKirimKomite"]
	row["view_email"] = formatDate(sent, hasSent, "02/01/2006")
	due, hasDue := raw.date["TglJatuhTempoKomite"]
	row["view_ktp"] = formatDate(due, hasDue, "02/01/2006")

	for name := range inboxmanager.DetailExportEmptyColumns {
		row[name] = ""
	}
	return row
}

// adjusterFee menghitung "Adjuster Fee" — rumus yang sama dengan modul Report Klaim.
//
// Sumbernya menjumlahkan, atas setiap baris settlement: `grossvalue*currencyvalue` bila
// bertipe '4', dan `Get_InterpolasiPNC(SUM(TOTAL_CLAIM))` bila bukan — angka interpolasi yang
// sama ikut terjumlah SEKALI PER BARIS bukan-'4'. Kosong bila tidak ada baris settlement atau
// tangganya tidak dapat menjawab.
func adjusterFee(raw detailRaw, fee inboxmanager.FeeScale) string {
	rowsTotal, ok := readNumber(raw.text["FeeJumlahBaris"])
	if !ok || rowsTotal == 0 {
		return ""
	}
	direct, _ := readNumber(raw.text["FeeLangsung"])

	interpolated, _ := readNumber(raw.text["FeeJumlahBarisInterpolasi"])
	if interpolated == 0 {
		return strconv.FormatFloat(direct, 'f', -1, 64)
	}
	if fee == nil {
		return ""
	}
	base, ok := readNumber(raw.text["FeeDasarTotalClaim"])
	if !ok {
		return ""
	}
	unit, ok := fee.Fee(base)
	if !ok {
		return ""
	}
	return strconv.FormatFloat(direct+interpolated*unit, 'f', -1, 64)
}

// readNumber mengurai teks angka. Kosong menghasilkan (0, false).
func readNumber(text string) (float64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// decimalComma menulis pemisah desimal sebagai koma — `@replaceAll(.X, ".", ",")`.
func decimalComma(text string) string {
	return strings.ReplaceAll(text, ".", ",")
}

// formatDate memformat tanggal dengan jam dindingnya sendiri. Tanggal kosong menjadi teks
// kosong, sama dengan `TO_CHAR(NULL)`.
//
// Zona waktunya TIDAK diubah: nilai DATE Oracle tidak membawa zona, dan driver menyerahkannya
// dengan jam dinding yang sama dengan yang Pega format.
func formatDate(at time.Time, valid bool, layout string) string {
	if !valid {
		return ""
	}
	return at.Format(layout)
}

// calendarDays menghitung selisih hari kalender dari a ke b, mengabaikan jam — sama dengan
// pengurangan dua DATE di Oracle.
//
// # Dua jebakan, keduanya nyata di data
//
// Basis datanya memuat tanggal penanda 01/01/1000 pada "Tgl Aksep".
//
//   - `time.Time.Sub` tidak dipakai: Duration jenuh di sekitar 292 tahun, sehingga selisih
//     dengan tahun 1000 tertulis 106751.
//   - Kalender Go Gregorian sepanjang masa, sedangkan Oracle memakai kalender JULIAN untuk
//     tanggal sebelum 15/10/1582. Untuk 12/05/2020 − 01/01/1000 Oracle menjawab 372674
//     (diperiksa 2026-10-14); hitungan Gregorian murni menjawab 372679.
//
// Karena itu keduanya diubah ke nomor hari Julian dengan aturan kalender Oracle.
func calendarDays(a, b time.Time) int {
	return oracleDayNumber(b) - oracleDayNumber(a)
}

// oracleDayNumber mengubah tanggal menjadi nomor hari Julian (JDN) menurut kalender Oracle:
// Julian sebelum 15/10/1582, Gregorian sejak itu.
func oracleDayNumber(at time.Time) int {
	year, month, day := at.Year(), int(at.Month()), at.Day()

	a := (14 - month) / 12
	y := year + 4800 - a
	m := month + 12*a - 3
	base := day + (153*m+2)/5 + 365*y + y/4

	gregorian := year > 1582 ||
		(year == 1582 && (month > 10 || (month == 10 && day >= 15)))
	if gregorian {
		return base - y/100 + y/400 - 32045
	}
	return base - 32083
}

// multiplyDecimal mengalikan dua angka desimal berbentuk teks secara TEPAT.
//
// Ia padanan `@toDecimal(a) * @toDecimal(b)` Pega, yang bekerja dengan BigDecimal: skala
// hasil adalah jumlah skala kedua operan, dan tidak ada pembulatan. Float tidak dipakai —
// nilainya uang (`I-12`).
//
// Operan kosong dibaca NOL, seperti `@toDecimal("")`. Operan yang tidak terbaca membuat
// hasilnya kosong.
func multiplyDecimal(a, b string) string {
	ma, sa, okA := parseDecimal(a)
	mb, sb, okB := parseDecimal(b)
	if !okA || !okB {
		return ""
	}
	product := new(big.Int).Mul(ma, mb)
	return formatDecimal(product, sa+sb)
}

// parseDecimal mengurai teks desimal menjadi mantisa dan skala.
func parseDecimal(text string) (*big.Int, int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return big.NewInt(0), 0, true
	}

	negative := strings.HasPrefix(text, "-")
	text = strings.TrimPrefix(strings.TrimPrefix(text, "-"), "+")

	whole, fraction, _ := strings.Cut(text, ".")
	digits := whole + fraction
	if digits == "" {
		return nil, 0, false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return nil, 0, false
		}
	}

	mantissa, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, 0, false
	}
	if negative {
		mantissa.Neg(mantissa)
	}
	return mantissa, len(fraction), true
}

// formatDecimal menulis mantisa berskala sebagai teks desimal biasa, tanpa eksponen.
func formatDecimal(mantissa *big.Int, scale int) string {
	negative := mantissa.Sign() < 0
	digits := new(big.Int).Abs(mantissa).String()

	if scale > 0 {
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		digits = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}
