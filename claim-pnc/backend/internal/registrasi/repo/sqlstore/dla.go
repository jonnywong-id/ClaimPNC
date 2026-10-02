package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// DLAStore memenuhi registrasi.DLASource.
type DLAStore struct {
	db *sql.DB
}

// NewDLAStore membentuk penyimpanan DLA.
func NewDLAStore(db *sql.DB) *DLAStore { return &DLAStore{db: db} }

var _ registrasi.DLASource = (*DLAStore)(nil)

// Issued membaca DLA revisi 0 satu adjustment.
func (s *DLAStore) Issued(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]registrasi.DLA, error) {
	return s.rows(ctx, "dla_terbit", claimID, objectID, coverageSeq, adjustmentSeq)
}

// PreDLA membaca Pre DLA satu adjustment.
func (s *DLAStore) PreDLA(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]registrasi.DLA, error) {
	return s.rows(ctx, "dla_predla", claimID, objectID, coverageSeq, adjustmentSeq)
}

func (s *DLAStore) rows(ctx context.Context, query, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]registrasi.DLA, error) {
	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery(query),
		claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: %s: %w", query, err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.DLA
	for rows.Next() {
		var number, recipient, code, kind, note, value, percent, share, claimAmount sql.NullString
		var currency, policyCurrency, qs, qsri, accepted, email, login, country, printed, sent sql.NullString
		var date, acceptedDate sql.NullTime
		if err := rows.Scan(&number, &recipient, &code, &kind, &date, &note, &value, &percent, &share,
			&claimAmount, &currency, &policyCurrency, &qs, &qsri, &accepted, &acceptedDate, &email,
			&login, &country, &printed, &sent); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris DLA: %w", err)
		}
		out = append(out, registrasi.DLA{
			ClaimID: claimID, ObjectID: objectID, CoverageSeq: coverageSeq, AdjustmentSeq: adjustmentSeq,
			Number: trimmed(number), Type: trimmed(kind), Recipient: trimmed(recipient), RecipientCode: trimmed(code),
			Date: wallWIB(date.Time), Note: note.String,
			Value: trimmed(value), Percent: trimmed(percent), ShareSpread: trimmed(share), ClaimAmount: trimmed(claimAmount),
			Currency: trimmed(currency), PolicyCurrency: trimmed(policyCurrency), QSPQS: trimmed(qs), QSRI: trimmed(qsri),
			AcceptedNo: trimmed(accepted), AcceptedDate: wallWIB(acceptedDate.Time),
			Info:    registrasi.PLARecipientInfo{Email: trimmed(email), Login: trimmed(login), Country: trimmed(country)},
			Printed: trimmed(printed) == "1", Sent: trimmed(sent) == "1",
		})
	}
	return out, rows.Err()
}

// Previous membaca DLA terakhir kepada seorang penerima pada klaim yang sama.
func (s *DLAStore) Previous(ctx context.Context, claimID, code string) (registrasi.DLAPrevious, bool, error) {
	var number sql.NullString
	var date sql.NullTime
	err := executorFrom(ctx, s.db).QueryRowContext(ctx, loadQuery("dla_sebelumnya"), claimID, code).Scan(&number, &date)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.DLAPrevious{}, false, nil
	}
	if err != nil {
		return registrasi.DLAPrevious{}, false, fmt.Errorf("registrasi/sqlstore: membaca DLA sebelumnya: %w", err)
	}
	return registrasi.DLAPrevious{Number: trimmed(number), Date: wallWIB(date.Time)}, true, nil
}

// ReinsuranceCase membaca REINSURANCETYPE.TYPE per kode treaty.
func (s *DLAStore) ReinsuranceCase(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, loadQuery("dla_jenis_reas"))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca jenis reasuransi: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var id, kind sql.NullString
		if err := rows.Scan(&id, &kind); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris jenis reasuransi: %w", err)
		}
		out[trimmed(id)] = trimmed(kind)
	}
	return out, rows.Err()
}

// Treaty membaca pengaturan treaty satu kode pada tahun polis.
func (s *DLAStore) Treaty(ctx context.Context, businessCode string, year int, treatyType string) (registrasi.TreatyArrangement, error) {
	yy := strconv.Itoa(year)
	var group sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery("dla_treaty_grup"), businessCode, yy, treatyType).Scan(&group)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return registrasi.TreatyArrangement{}, fmt.Errorf("registrasi/sqlstore: membaca grup treaty: %w", err)
	}
	g := trimmed(group)
	var out registrasi.TreatyArrangement

	rows, err := s.db.QueryContext(ctx, loadQuery("dla_treaty_reas"), yy, g, treatyType)
	if err != nil {
		return out, fmt.Errorf("registrasi/sqlstore: membaca reasuradur treaty: %w", err)
	}
	for rows.Next() {
		var pct, name, id, typeName sql.NullString
		if err := rows.Scan(&pct, &name, &id, &typeName); err != nil {
			_ = rows.Close()
			return out, fmt.Errorf("registrasi/sqlstore: membaca baris reasuradur treaty: %w", err)
		}
		out.Reinsurers = append(out.Reinsurers, registrasi.TreatyReinsurer{
			PctShare: trimmed(pct), Name: trimmed(name), ID: trimmed(id), TypeName: trimmed(typeName)})
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	var limit sql.NullString
	err = s.db.QueryRowContext(ctx, loadQuery("dla_treaty_limit"), treatyType, yy, g).Scan(&limit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, fmt.Errorf("registrasi/sqlstore: membaca limit treaty: %w", err)
	}
	out.Limit = trimmed(limit)

	if registrasi.TreatyUsesQS(treatyType) {
		qs, err := s.db.QueryContext(ctx, loadQuery("dla_treaty_qs"), yy, treatyType, g)
		if err != nil {
			return out, fmt.Errorf("registrasi/sqlstore: membaca QS treaty: %w", err)
		}
		for qs.Next() {
			var pct, name sql.NullString
			if err := qs.Scan(&pct, &name); err != nil {
				_ = qs.Close()
				return out, err
			}
			out.QSPct = trimmed(pct) // baris terakhir yang dipakai
			out.QSParts = append(out.QSParts, registrasi.TreatyQSPart{Name: trimmed(name), Pct: trimmed(pct)})
		}
		_ = qs.Close()
		if err := qs.Err(); err != nil {
			return out, err
		}
	}
	return out, nil
}

// NextNumber menerbitkan nomor DLA seperti `PLA_DLA.prc` cabang DLA.
func (s *DLAStore) NextNumber(ctx context.Context, code string, year int) (string, error) {
	tx, ok := txFrom(ctx)
	if !ok {
		return "", errors.New("registrasi/sqlstore: nomor DLA hanya boleh diterbitkan di dalam transaksi")
	}
	var site sql.NullString
	if err := tx.QueryRowContext(ctx, loadQuery("pla_site")).Scan(&site); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca site aktif: %w", err)
	}
	var counter int64
	if err := tx.QueryRowContext(ctx, loadQuery("dla_urut")).Scan(&counter); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca DLA_SEQ: %w", err)
	}
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	yy := fmt.Sprintf("%02d", year%100)
	if _, err := tx.ExecContext(ctx, loadQuery("dla_nomor_sisip"),
		strings.ToUpper(hex.EncodeToString(key)), code, trimmed(site), yy, strconv.FormatInt(counter, 10)); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: mencatat nomor DLA: %w", err)
	}
	return code + yy + trimmed(site) + fmt.Sprintf("%015d", counter), nil
}

// Save menulis satu baris T_DLALIST.
func (s *DLAStore) Save(ctx context.Context, d registrasi.DLA) error {
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("dla_sisip"),
		d.ClaimID, d.ObjectID, strconv.Itoa(d.CoverageSeq), strconv.Itoa(d.AdjustmentSeq), d.Number, d.Recipient,
		emptyTextAsNil(d.AcceptedNo), d.Type, wallDate(d.Date), d.Value, emptyTextAsNil(d.Note), d.RecipientCode,
		d.Currency, d.PolicyCurrency, emptyTextAsNil(d.ShareSpread), d.Percent, d.ClaimAmount,
		emptyTextAsNil(d.QSPQS), emptyTextAsNil(d.QSRI),
		emptyTextAsNil(d.Info.Email), emptyTextAsNil(d.Info.Login), emptyTextAsNil(d.Info.Country), nullWallDate(d.AcceptedDate))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan DLA %s: %w", d.Number, err)
	}
	return nil
}

// MarkPrinted mengisi NOTES dan ISDLA = '1'.
func (s *DLAStore) MarkPrinted(ctx context.Context, claimID, number, note string) error {
	if _, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("dla_cetak"), note, claimID, number); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menandai DLA %s tercetak: %w", number, err)
	}
	return nil
}

// wallDate menulis waktu sebagai DATE jam dinding WIB — konvensi SYSDATE Pega.
func wallDate(t time.Time) time.Time {
	w := t.In(clock.ZoneWIB)
	return time.Date(w.Year(), w.Month(), w.Day(), w.Hour(), w.Minute(), w.Second(), 0, time.UTC)
}

func nullWallDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return wallDate(t)
}

// ---- dokumen polis ---------------------------------------------------------------------

// Policy membaca bahan DLA polis. Kepala polis dan SpreadingList masih dari dokumen polis
// terbaru (belum ada tabel padanannya); CoinsList dari T_COINSLIST dan FacOfferList dari
// T_FACOFFER, keduanya pada PRODKE snapshot klaim.
func (s *DLAStore) Policy(ctx context.Context, policyNumber, prodKe string) (registrasi.DLAPolicy, error) {
	number, prodKe := strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe)
	var text sql.NullString
	p := registrasi.DLAPolicy{TSISpreaded: map[string]*big.Rat{}}
	err := s.db.QueryRowContext(ctx, loadQuery("dla_polis_dokumen"), number).Scan(&text)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return registrasi.DLAPolicy{}, fmt.Errorf("registrasi/sqlstore: membaca dokumen polis %q: %w", policyNumber, err)
	case text.String != "":
		if p, err = parseDLAPolicy([]byte(text.String)); err != nil {
			return registrasi.DLAPolicy{}, err
		}
	}
	if err := s.policyHeader(ctx, &p, number, prodKe); err != nil {
		return registrasi.DLAPolicy{}, err
	}
	if err := s.policySpreading(ctx, &p, number, prodKe); err != nil {
		return registrasi.DLAPolicy{}, err
	}
	if p.Coins, err = coinsMembers(ctx, s.db, policyNumber, prodKe); err != nil {
		return registrasi.DLAPolicy{}, err
	}
	if p.FacOffer, err = s.facOffers(ctx, policyNumber, prodKe); err != nil {
		return registrasi.DLAPolicy{}, err
	}
	return p, nil
}

// policyHeader menimpa kepala polis dokumen dengan kolom T_GENERAL dan T_OFFERFACIN yang
// terisi — tabel lebih dulu, dokumen hanya cadangan (Work Owner, 2026-10-01).
func (s *DLAStore) policyHeader(ctx context.Context, p *registrasi.DLAPolicy, number, prodKe string) error {
	var caseID, sumTSI, coins, syariah, business, status sql.NullString
	var start sql.NullTime
	err := s.db.QueryRowContext(ctx, loadQuery("dla_polis_kepala"), number, prodKe).
		Scan(&caseID, &sumTSI, &coins, &syariah, &business, &status, &start)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return fmt.Errorf("registrasi/sqlstore: membaca T_GENERAL polis %q: %w", number, err)
	default:
		if v := trimmed(caseID); v != "" {
			p.CaseID = v
		}
		if v := trimmed(sumTSI); v != "" {
			p.SumOfTSI = registrasi.DecimalOf(v)
		}
		if v := trimmed(coins); v != "" {
			p.TypeOfCoins = v
		}
		if v := trimmed(syariah); v != "" {
			p.Syariah = v == "1"
		}
		if v := trimmed(business); v != "" {
			p.BusinessCode = v
		}
		if v := trimmed(status); v != "" {
			p.StatusBusiness = v
		}
		// Tahun mulai: dokumen lebih dulu (T_GENERAL tidak mengikuti endorsemen).
		if t := wibDate(start); p.StartYear == 0 && !t.IsZero() {
			p.StartYear = t.In(clock.ZoneWIB).Year()
		}
	}

	var share sql.NullString
	err = s.db.QueryRowContext(ctx, loadQuery("dla_offer_facin"), number, prodKe).Scan(&share)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return fmt.Errorf("registrasi/sqlstore: membaca T_OFFERFACIN polis %q: %w", number, err)
	default:
		if v := trimmed(share); v != "" {
			p.OfferFacInShare = registrasi.DecimalOf(v)
		}
	}
	return nil
}

// policySpreading mengganti TSISpreaded dokumen dengan T_SPREADINGLIST bila tabel itu punya
// baris untuk PRODKE-nya: TSISpreaded pertama per TreatyType, FlagDelete = '1' dilewati —
// aturan yang sama dengan collectSpreading.
func (s *DLAStore) policySpreading(ctx context.Context, p *registrasi.DLAPolicy, number, prodKe string) error {
	rows, err := s.db.QueryContext(ctx, loadQuery("dla_spreading"), number, prodKe)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca T_SPREADINGLIST polis %q: %w", number, err)
	}
	defer func() { _ = rows.Close() }()
	table := map[string]*big.Rat{}
	any := false
	for rows.Next() {
		var kind, tsi, deleted sql.NullString
		if err := rows.Scan(&kind, &tsi, &deleted); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris T_SPREADINGLIST: %w", err)
		}
		any = true
		k := trimmed(kind)
		if _, seen := table[k]; k != "" && !seen && trimmed(deleted) != "1" {
			table[k] = registrasi.DecimalOf(trimmed(tsi))
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if any {
		p.TSISpreaded = table
	}
	return nil
}

// facOffers membaca T_FACOFFER: satu baris per reasuradur, JSONDATA memuat seluruh
// FacOfferList, sehingga yang diambil hanya entri milik REINSURER_ID baris itu.
func (s *DLAStore) facOffers(ctx context.Context, policyNumber, prodKe string) ([]registrasi.FacOffer, error) {
	rows, err := s.db.QueryContext(ctx, loadQuery("dla_fac_offer"), strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca T_FACOFFER polis %q: %w", policyNumber, err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.FacOffer
	for rows.Next() {
		var id, body sql.NullString
		if err := rows.Scan(&id, &body); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris T_FACOFFER: %w", err)
		}
		f, ok, err := parseFacOfferRow(trimmed(id), []byte(body.String))
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, f)
		}
	}
	return out, rows.Err()
}

// parseFacOfferRow mengambil entri FacOfferList yang ReinsurerID-nya sama dengan baris.
func parseFacOfferRow(reinsurerID string, body []byte) (registrasi.FacOffer, bool, error) {
	if len(body) == 0 {
		return registrasi.FacOffer{}, false, nil
	}
	var doc jsonMap
	if err := json.Unmarshal(body, &doc); err != nil {
		return registrasi.FacOffer{}, false, fmt.Errorf("registrasi/sqlstore: JSONDATA T_FACOFFER %s tidak terbaca: %w", reinsurerID, err)
	}
	for _, o := range list(doc, "FacOfferList") {
		if field(o, "ReinsurerID") == reinsurerID {
			return parseFacOffer(o), true, nil
		}
	}
	return registrasi.FacOffer{}, false, nil
}

type jsonMap = map[string]any

func parseDLAPolicy(body []byte) (registrasi.DLAPolicy, error) {
	var doc jsonMap
	if len(body) == 0 {
		return registrasi.DLAPolicy{}, nil
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return registrasi.DLAPolicy{}, fmt.Errorf("registrasi/sqlstore: dokumen polis tidak terbaca: %w", err)
	}
	p := registrasi.DLAPolicy{
		CaseID:         field(doc, "CaseID"),
		SumOfTSI:       registrasi.DecimalOf(field(doc, "SumOfTSI")),
		TypeOfCoins:    field(doc, "TypeOfCoins"),
		Syariah:        field(doc, "SyariahStatus") == "1",
		BusinessCode:   field(object(doc, "Quotation"), "BusinessCode"),
		StatusBusiness: field(object(doc, "Quotation"), "StatusBusiness"),
		TSISpreaded:    map[string]*big.Rat{},
	}
	if v := field(object(doc, "OfferFacIn"), "PercentShare"); v != "" {
		p.OfferFacInShare = registrasi.DecimalOf(v)
	}
	if t := parsePegaDateTime(field(doc, "StartDateTime")); !t.IsZero() {
		p.StartYear = t.In(clock.ZoneWIB).Year()
	}
	// CoinsList dan FacOfferList tidak dibaca dari sini — lihat Policy.
	collectSpreading(doc, p.TSISpreaded, 0)
	return p, nil
}

// parseFacOffer mengurai satu entri FacOfferList; bentuknya berbeda per Group Panel
// (PropertyList, AnekaList, CargoList).
func parseFacOffer(o jsonMap) registrasi.FacOffer {
	offered := field(o, "TotalOffered")
	f := registrasi.FacOffer{
		ReinsurerName: field(o, "ReinsurerName"), ReinsurerID: field(o, "ReinsurerID"),
		Deleted: field(o, "FlagDelete") == "1", OfferedMissing: offered == "" || offered == "0",
	}
	for _, x := range list(o, "PropertyList") {
		f.Property = append(f.Property, registrasi.FacObject{ObjectNo: field(x, "ObjectNo"), Coverage: facCoverages(x)})
	}
	for _, x := range list(o, "AnekaList") {
		f.Aneka = append(f.Aneka, registrasi.FacObject{
			ObjectName: field(x, "ObjectName"), SumTSIAneka: field(x, "SumTSIObjectAneka"), Coverage: facCoverages(x)})
	}
	for _, x := range list(o, "CargoList") {
		f.Cargo = append(f.Cargo, registrasi.FacObject{
			GoodID: field(x, "GoodID"), GoodNote: field(x, "GoodNote"), IndexObject: field(x, "IndexObject"),
			Coverage: facCoverages(x)})
	}
	return f
}

func facCoverages(x jsonMap) []registrasi.FacCoverage {
	var out []registrasi.FacCoverage
	for _, c := range list(x, "CoverageList") {
		cv := registrasi.FacCoverage{
			Code: field(c, "Coverage"), TSI: field(c, "TSI"), SumTSI: field(c, "SumTSI"), TSISublimit: field(c, "TSISublimit"),
		}
		if fo := list(c, "FacOutObjectList"); len(fo) > 0 {
			cv.ShareOffered = field(fo[0], "ShareOffered")
			cv.Percent = field(fo[0], "Percent")
		}
		out = append(out, cv)
	}
	return out
}

// collectSpreading mengumpulkan TSISpreaded pertama per TreatyType dari setiap SpreadingList
// dokumen. FacOfferList (salinan histori) dilewati.
func collectSpreading(v any, out map[string]*big.Rat, depth int) {
	if depth > 12 {
		return
	}
	switch t := v.(type) {
	case jsonMap:
		for k, x := range t {
			if k == "FacOfferList" || k == "OldFacOffer" {
				continue
			}
			if k == "SpreadingList" {
				for _, r := range asList(x) {
					kind := field(r, "TreatyType")
					if _, seen := out[kind]; kind != "" && !seen && field(r, "FlagDelete") != "1" {
						out[kind] = registrasi.DecimalOf(field(r, "TSISpreaded"))
					}
				}
				continue
			}
			collectSpreading(x, out, depth+1)
		}
	case []any:
		for _, x := range t {
			collectSpreading(x, out, depth+1)
		}
	}
}

func object(m jsonMap, key string) jsonMap {
	if v, ok := m[key].(jsonMap); ok {
		return v
	}
	return jsonMap{}
}

func list(m jsonMap, key string) []jsonMap { return asList(m[key]) }

func asList(v any) []jsonMap {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]jsonMap, 0, len(arr))
	for _, x := range arr {
		if m, ok := x.(jsonMap); ok {
			out = append(out, m)
		}
	}
	return out
}

func field(m jsonMap, key string) string { return docText(m[key]) }
