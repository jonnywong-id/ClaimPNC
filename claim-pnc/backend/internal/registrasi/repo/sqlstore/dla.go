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

// Policy membaca bahan DLA dari dokumen polis terbaru.
func (s *DLAStore) Policy(ctx context.Context, policyNumber string) (registrasi.DLAPolicy, error) {
	var text sql.NullString
	var blob []byte
	err := s.db.QueryRowContext(ctx, loadQuery("dla_polis_dokumen"), strings.TrimSpace(policyNumber)).Scan(&text, &blob)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.DLAPolicy{}, nil
	}
	if err != nil {
		return registrasi.DLAPolicy{}, fmt.Errorf("registrasi/sqlstore: membaca dokumen polis %q: %w", policyNumber, err)
	}
	body := []byte(text.String)
	if !text.Valid || text.String == "" {
		body = blob
	}
	return parseDLAPolicy(body)
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
	for _, c := range list(doc, "CoinsList") {
		share, ok := parsePercent(field(c, "PercentShare"))
		p.Coins = append(p.Coins, registrasi.PLACoinsMember{
			ID: field(c, "CoinsID"), Name: field(c, "CoinsName"), Leader: field(c, "Leader") == "true",
			Share: share, HasShare: ok, Deleted: field(c, "FlagDelete") == "1",
		})
	}
	for _, o := range list(doc, "FacOfferList") {
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
		p.FacOffer = append(p.FacOffer, f)
	}
	collectSpreading(doc, p.TSISpreaded, 0)
	return p, nil
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
