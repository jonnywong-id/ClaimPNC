package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
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

// PLAStore memenuhi registrasi.PLASource.
type PLAStore struct {
	db *sql.DB
}

// NewPLAStore membentuk penyimpanan PLA.
func NewPLAStore(db *sql.DB) *PLAStore { return &PLAStore{db: db} }

var _ registrasi.PLASource = (*PLAStore)(nil)

// CoinsMembers membaca T_COINSLIST polis pada PRODKE-nya.
func (s *PLAStore) CoinsMembers(ctx context.Context, policyNumber, prodKe string) ([]registrasi.PLACoinsMember, error) {
	return coinsMembers(ctx, s.db, policyNumber, prodKe)
}

// coinsMembers dipakai bersama PLA dan DLA: keduanya membaca CoinsList yang sama.
func coinsMembers(ctx context.Context, db *sql.DB, policyNumber, prodKe string) ([]registrasi.PLACoinsMember, error) {
	rows, err := db.QueryContext(ctx, loadQuery("pla_koasuransi"), strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca CoinsList polis %q: %w", policyNumber, err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.PLACoinsMember
	for rows.Next() {
		var id, name, leader, share, deleted sql.NullString
		if err := rows.Scan(&id, &name, &leader, &share, &deleted); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris CoinsList: %w", err)
		}
		p, ok := parsePercent(share.String)
		out = append(out, registrasi.PLACoinsMember{
			ID: trimmed(id), Name: trimmed(name), Leader: trimmed(leader) == "true",
			Share: p, HasShare: ok, Deleted: trimmed(deleted) == "1",
		})
	}
	return out, rows.Err()
}

// FacOffers membaca Fac Offer polis untuk PLA FAC OUT. Baris ber-JSONDATA diurai seperti
// DLA FAC OUT (parseFacOfferRow); baris tanpa JSONDATA memakai kolom datar REINSURER_NAME
// dan PCT_SHAREREAS sebagai cadangan (FacOffer.FlatShare).
func (s *PLAStore) FacOffers(ctx context.Context, policyNumber, prodKe string) ([]registrasi.FacOffer, error) {
	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("pla_fac_offer"),
		strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca T_FACOFFER polis %q: %w", policyNumber, err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.FacOffer
	for rows.Next() {
		var id, name, pct, body sql.NullString
		if err := rows.Scan(&id, &name, &pct, &body); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris T_FACOFFER: %w", err)
		}
		f, ok, err := parseFacOfferRow(trimmed(id), []byte(body.String))
		if err != nil {
			return nil, err
		}
		if !ok {
			if trimmed(pct) == "" {
				continue
			}
			f = registrasi.FacOffer{ReinsurerID: trimmed(id), ReinsurerName: trimmed(name), FlatShare: trimmed(pct)}
		}
		if f.ReinsurerName == "" {
			f.ReinsurerName = trimmed(name)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SpreadingTSI membaca TSISPREADED T_SPREADINGLIST satu objek dan coverage polis: baris FAC
// OUT (10015) dan jumlah seluruh baris. nil bila tidak ada barisnya.
func (s *PLAStore) SpreadingTSI(ctx context.Context, q registrasi.SpreadingTSIQuery) (*big.Rat, *big.Rat, error) {
	number, prodKe := strings.TrimSpace(q.PolicyNumber), strings.TrimSpace(q.ProdKe)
	gp, object := strings.TrimSpace(q.GroupPanel), strings.TrimSpace(q.ObjectID)
	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("pla_spreading_tsi"),
		number, prodKe, strings.TrimSpace(q.Coverage), gp, object, gp, number, prodKe, object)
	if err != nil {
		return nil, nil, fmt.Errorf("registrasi/sqlstore: membaca T_SPREADINGLIST polis %q: %w", q.PolicyNumber, err)
	}
	defer func() { _ = rows.Close() }()
	var facOut, total *big.Rat
	for rows.Next() {
		var kind, tsi sql.NullString
		if err := rows.Scan(&kind, &tsi); err != nil {
			return nil, nil, fmt.Errorf("registrasi/sqlstore: membaca baris T_SPREADINGLIST: %w", err)
		}
		v := registrasi.DecimalOf(trimmed(tsi))
		if total == nil {
			total = new(big.Rat)
		}
		total.Add(total, v)
		if trimmed(kind) == registrasi.TreatyFacOut {
			if facOut == nil {
				facOut = new(big.Rat)
			}
			facOut.Add(facOut, v)
		}
	}
	return facOut, total, rows.Err()
}

// Recipient membaca login, negara, dan email penerima dari T_REINSURER.
func (s *PLAStore) Recipient(ctx context.Context, code, name string) (registrasi.PLARecipientInfo, error) {
	var login, country, email sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery("pla_penerima"), code, name).Scan(&login, &country, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.PLARecipientInfo{}, nil
	}
	if err != nil {
		return registrasi.PLARecipientInfo{}, fmt.Errorf("registrasi/sqlstore: membaca penerima PLA %q: %w", code, err)
	}
	return registrasi.PLARecipientInfo{Login: trimmed(login), Country: trimmed(country), Email: trimmed(email)}, nil
}

// Previous membaca PLA terakhir kepada seorang penerima pada klaim yang sama.
func (s *PLAStore) Previous(ctx context.Context, claimID, code string) (registrasi.PLAPrevious, bool, error) {
	var number sql.NullString
	var date sql.NullTime
	err := executorFrom(ctx, s.db).QueryRowContext(ctx, loadQuery("pla_sebelumnya"), claimID, code).Scan(&number, &date)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.PLAPrevious{}, false, nil
	}
	if err != nil {
		return registrasi.PLAPrevious{}, false, fmt.Errorf("registrasi/sqlstore: membaca PLA sebelumnya: %w", err)
	}
	return registrasi.PLAPrevious{Number: trimmed(number), Date: wallWIB(date.Time)}, true, nil
}

// plaJSON adalah bentuk JSON_PLA — `ASM-FW-GCNMFW-Data-PLA`.
type plaJSON struct {
	ObjClass     string        `json:"pxObjClass"`
	EstimasiList []plaJSONLine `json:"EstimasiList"`
}

type plaJSONLine struct {
	ASMCount           string `json:"ASMCount"`
	Currency           string `json:"Currency"`
	CurrencyID         string `json:"CurrencyID"`
	EstimastionReserve string `json:"EstimastionReserve"`
	EstimationValue    string `json:"EstimationValue"`
	PercentPLA         string `json:"PercentPLA"`
	PremiShare         string `json:"PremiShare"`
	ObjClass           string `json:"pxObjClass"`
	ResultPLA          string `json:"ResultPLA"`
	// SharePLA hanya pada PLA FAC OUT: bagian reasuradur (nilai uang). Pada tipe itu
	// PercentPLA juga nilai uang — dasar pembagi IMProfit2 — bukan persen.
	SharePLA string `json:"SharePLA,omitempty"`
}

// Issued membaca PLA yang sudah terbit untuk satu revisi CFS sebuah jaminan.
func (s *PLAStore) Issued(ctx context.Context, claimID, objectID string, coverageSeq, revision int) ([]registrasi.PLA, error) {
	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("pla_terbit"),
		claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(revision),
		registrasi.PLATypeCoins, registrasi.PLATypeFacOut, registrasi.PLATypeBPPDAN, registrasi.PLATypeEQPool)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca PLA terbit: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.PLA
	for rows.Next() {
		var number, recipient, code, note, currency, body, email, kind sql.NullString
		var date sql.NullTime
		if err := rows.Scan(&number, &recipient, &code, &date, &note, &currency, &body, &email, &kind); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris PLA: %w", err)
		}
		p := registrasi.PLA{
			ClaimID: claimID, ObjectID: objectID, CoverageSeq: coverageSeq, Revision: revision,
			Number: trimmed(number), Type: trimmed(kind), Recipient: trimmed(recipient),
			RecipientCode: trimmed(code), Date: wallWIB(date.Time), Note: note.String, PolicyCurrency: trimmed(currency),
			Info: registrasi.PLARecipientInfo{Email: trimmed(email)},
		}
		var doc plaJSON
		if body.Valid && json.Unmarshal([]byte(body.String), &doc) == nil {
			for _, l := range doc.EstimasiList {
				a := registrasi.PLAAmount{
					Currency: l.Currency, CurrencyID: l.CurrencyID,
					Reserve: parseMoney(l.EstimastionReserve), Base: parseMoney(l.EstimationValue),
					Result: parseMoney(l.ResultPLA), ASMCount: parseMoney(l.ASMCount),
				}
				switch p.Type {
				case registrasi.PLATypeFacOut:
					a.FacShare, a.FacBase = parseMoney(l.SharePLA), parseMoney(l.PercentPLA)
				case registrasi.PLATypeBPPDAN, registrasi.PLATypeEQPool:
					a.FacShare = parseMoney(l.SharePLA)
				default:
					a.Share, _ = parsePercent(l.PercentPLA)
				}
				p.Amount = append(p.Amount, a)
			}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// NextNumber menerbitkan nomor PLA seperti `PLA_DLA.prc` cabang PLA.
func (s *PLAStore) NextNumber(ctx context.Context, code string, year int) (string, error) {
	tx, ok := txFrom(ctx)
	if !ok {
		return "", errors.New("registrasi/sqlstore: nomor PLA hanya boleh diterbitkan di dalam transaksi")
	}
	var site sql.NullString
	if err := tx.QueryRowContext(ctx, loadQuery("pla_site")).Scan(&site); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca site aktif: %w", err)
	}
	var counter int64
	if err := tx.QueryRowContext(ctx, loadQuery("pla_urut")).Scan(&counter); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca PLA_SEQ: %w", err)
	}
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	yy := fmt.Sprintf("%02d", year%100)
	if _, err := tx.ExecContext(ctx, loadQuery("pla_nomor_sisip"),
		strings.ToUpper(hex.EncodeToString(key)), code, trimmed(site), yy, strconv.FormatInt(counter, 10)); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: mencatat nomor PLA: %w", err)
	}
	return code + yy + trimmed(site) + fmt.Sprintf("%015d", counter), nil
}

// Save menulis satu baris T_PLALIST.
func (s *PLAStore) Save(ctx context.Context, p registrasi.PLA) error {
	doc := plaJSON{ObjClass: "ASM-FW-GCNMFW-Data-PLA"}
	for _, a := range p.Amount {
		line := plaJSONLine{
			ASMCount: moneyText(a.ASMCount), Currency: a.Currency, CurrencyID: a.CurrencyID,
			EstimastionReserve: moneyText(a.Reserve), EstimationValue: moneyText(a.Base),
			PercentPLA: percentText(a.Share), PremiShare: moneyText(a.Result),
			ObjClass: "ASM-FW-GCNMFW-Data-Estimasi", ResultPLA: moneyText(a.Result),
		}
		switch p.Type {
		case registrasi.PLATypeFacOut:
			line.PercentPLA, line.SharePLA = moneyText(a.FacBase), moneyText(a.FacShare)
		case registrasi.PLATypeBPPDAN, registrasi.PLATypeEQPool:
			// PLABPPDAN_Act 3.9: SharePLA = persen bagian; PercentPLA tidak diisi.
			line.PercentPLA, line.SharePLA = "", moneyText(a.FacShare)
		}
		doc.EstimasiList = append(doc.EstimasiList, line)
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	// TGLPLA adalah DATE yang di sistem lama diisi SYSDATE server — jam dinding WIB. Baris
	// dari kedua sistem dibaca berdampingan, jadi ia ditulis dengan konvensi yang sama.
	wib := p.Date.In(clock.ZoneWIB)
	date := time.Date(wib.Year(), wib.Month(), wib.Day(), wib.Hour(), wib.Minute(), wib.Second(), 0, time.UTC)
	_, err = executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("pla_sisip"),
		p.ClaimID, p.ObjectID, strconv.Itoa(p.CoverageSeq), p.Number, p.Recipient, strconv.Itoa(p.Revision),
		p.Type, date, p.Note, p.RecipientCode, p.PolicyCurrency,
		emptyTextAsNil(p.Info.Email), emptyTextAsNil(p.Info.Login), emptyTextAsNil(p.Info.Country), string(body))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan PLA %s: %w", p.Number, err)
	}
	return nil
}

// UpdateNote mengganti catatan PLA yang sudah terbit — isian REMARKS layar PrintPLA_dtl.
func (s *PLAStore) UpdateNote(ctx context.Context, claimID, number string, revision int, note string) error {
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("pla_catatan"), note, claimID, number, strconv.Itoa(revision))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan catatan PLA %s: %w", number, err)
	}
	return nil
}

// Signature membaca nama dan gambar tanda tangan (JSONDATA.TTDWeb, PNG base64).
func (s *PLAStore) Signature(ctx context.Context, id string) (string, []byte, error) {
	var name, body sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery("pla_ttd"), id).Scan(&name, &body)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("registrasi/sqlstore: membaca tanda tangan: %w", err)
	}
	var doc struct {
		TTDWeb string `json:"TTDWeb"`
	}
	var png []byte
	if json.Unmarshal([]byte(body.String), &doc) == nil && doc.TTDWeb != "" {
		png, _ = base64.StdEncoding.DecodeString(doc.TTDWeb)
	}
	return trimmed(name), png, nil
}

// PASignature membaca satu tanda tangan POOLDATA.M_SIGNATURE1. Nama kunci JSON dipangkas
// spasinya — kunci gambar tersimpan sebagai "TTD " (terukur 2026-10-07). Dokumen atau gambar
// yang tidak dapat diurai dikembalikan kosong, bukan galat: Draft Persetujuan tetap tercetak.
func (s *PLAStore) PASignature(ctx context.Context, id string) (string, []byte, error) {
	var body sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery("pa_ttd"), id).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("registrasi/sqlstore: membaca tanda tangan PA: %w", err)
	}
	var doc map[string]any
	if json.Unmarshal([]byte(body.String), &doc) != nil {
		return "", nil, nil
	}
	field := map[string]string{}
	for k, v := range doc {
		if text, ok := v.(string); ok {
			field[strings.ToUpper(strings.TrimSpace(k))] = strings.TrimSpace(text)
		}
	}
	image := field["TTD"]
	if i := strings.Index(image, "base64,"); i >= 0 {
		image = image[i+len("base64,"):]
	}
	png, _ := base64.StdEncoding.DecodeString(image)
	return field["SIGNATURE_NAME"], png, nil
}

// LODEmails membaca email tertanggung (pengkinian data) dan email PIC teknik.
func (s *PLAStore) LODEmails(ctx context.Context, claimNumber, technicalPIC string) (string, string, error) {
	insured, err := s.firstText(ctx, "lod_email_tertanggung", claimNumber)
	if err != nil {
		return "", "", err
	}
	pic := ""
	if strings.TrimSpace(technicalPIC) != "" {
		if pic, err = s.firstText(ctx, "lod_email_pic", technicalPIC); err != nil {
			return "", "", err
		}
	}
	return insured, pic, nil
}

func (s *PLAStore) firstText(ctx context.Context, query, key string) (string, error) {
	var v sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery(query), key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: %s: %w", query, err)
	}
	return trimmed(v), nil
}

// wallWIB menafsirkan DATE jam dinding WIB dari basis data sebagai waktu WIB.
func wallWIB(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, clock.ZoneWIB)
}

// moneyText menulis sen sebagai rupiah dengan dua desimal — `135000000.00`.
func moneyText(m registrasi.Money) string {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// percentText menulis persentase dengan dua desimal — `7.50`.
func percentText(p registrasi.Percent) string {
	return new(big.Rat).SetFrac64(int64(p), 10_000).FloatString(2)
}
