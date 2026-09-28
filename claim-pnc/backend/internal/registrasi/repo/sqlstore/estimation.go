package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// saveItems menuliskan item objek dan estimasi satu coverage.
//
// Item memakai UPDATE-lalu-INSERT dan item berlebih ditandai hapus (TC_PNC_OBJECTITEM
// punya penandanya). Estimasi hanya diperbarui atau disisipkan: T_CLAIM_ESTIMASI tidak
// punya penanda hapus, dan penghapusan fisik dilarang (`D-66`). Layanan karena itu
// menolak permintaan yang mengurangi estimasi yang sudah tersimpan.
func (r *ClaimStore) saveItems(
	ctx context.Context,
	exec executor,
	k registrasi.Claim,
	objectID string,
	coverageSeq int,
	items []registrasi.ObjectItem,
	now time.Time,
) error {
	coverageID := strconv.Itoa(coverageSeq)
	actor := k.UpdatedBy

	for n, item := range items {
		itemID := strconv.Itoa(n + 1)
		sum := int64(item.SumConverted())
		if err := upsert(ctx, exec,
			"item_perbarui", []any{
				emptyTextAsNil(item.Name), emptyTextAsNil(item.Description), sum, actor, now,
				emptyTextAsNil(item.Group), k.ID, objectID, coverageID, itemID},
			"item_sisip", []any{
				emptyTextAsNil(item.Name), emptyTextAsNil(item.Description), sum, actor, now,
				k.ID, objectID, coverageID, itemID, actor, now, emptyTextAsNil(item.Group)},
		); err != nil {
			return fmt.Errorf("item %s: %w", itemID, err)
		}

		for m, e := range item.Estimation {
			estimateID := strconv.Itoa(m + 1)
			var date, faceSheetDate any
			if !e.Date.IsZero() {
				date = e.Date.UTC()
			}
			if !e.FaceSheetDate.IsZero() {
				faceSheetDate = e.FaceSheetDate.UTC()
			}
			printed := 0
			if e.FaceSheet {
				printed = 1
			}
			if err := upsert(ctx, exec,
				"estimasi_perbarui", []any{
					e.Type, e.Currency, int64(e.Value), int64(e.Rate), int64(e.Converted), date,
					printed, faceSheetDate, k.ID, objectID, coverageID, itemID, estimateID},
				"estimasi_sisip", []any{
					e.Type, e.Currency, int64(e.Value), int64(e.Rate), int64(e.Converted), date,
					k.ID, objectID, coverageID, itemID, estimateID, actor, now, printed, faceSheetDate},
			); err != nil {
				return fmt.Errorf("estimasi %s.%s: %w", itemID, estimateID, err)
			}
		}
	}

	_, err := exec.ExecContext(ctx, loadQuery("item_tandai_sisa"),
		actor, now, k.ID, objectID, coverageID, len(items))
	return err
}

// loadItems membaca item dan estimasi seluruh coverage klaim.
//
// Penjodohannya ke pohon memakai OBJECTID dan urutan coverage, sama seperti spreading.
func loadItems(
	ctx context.Context,
	exec executor,
	k *registrasi.Claim,
	coverageAt func(objectID, coverageID string) *registrasi.Coverage,
) error {
	rows, err := exec.QueryContext(ctx, loadQuery("item_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca item objek: %w", err)
	}
	type key struct{ object, coverage, item string }
	itemAt := map[key][2]int{} // posisi item: indeks coverage di peta, indeks item
	var coverages []*registrasi.Coverage
	coverageIndex := map[*registrasi.Coverage]int{}
	for rows.Next() {
		var objectID, coverageID, itemID, name, description, group sql.NullString
		if err := rows.Scan(&objectID, &coverageID, &itemID, &name, &description, &group); err != nil {
			_ = rows.Close()
			return fmt.Errorf("registrasi/sqlstore: membaca baris item objek: %w", err)
		}
		c := coverageAt(strings.TrimSpace(objectID.String), strings.TrimSpace(coverageID.String))
		if c == nil {
			continue
		}
		ci, ok := coverageIndex[c]
		if !ok {
			ci = len(coverages)
			coverages = append(coverages, c)
			coverageIndex[c] = ci
		}
		itemAt[key{strings.TrimSpace(objectID.String), strings.TrimSpace(coverageID.String), strings.TrimSpace(itemID.String)}] = [2]int{ci, len(c.Item)}
		c.Item = append(c.Item, registrasi.ObjectItem{Name: name.String, Description: description.String, Group: group.String})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("registrasi/sqlstore: menelusuri item objek: %w", err)
	}
	_ = rows.Close()

	rows, err = exec.QueryContext(ctx, loadQuery("estimasi_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca estimasi: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			objectID, coverageID, itemID, estimateID, kind, currency sql.NullString
			value, rate, converted, printed                          sql.NullInt64
			date, faceSheetDate                                      sql.NullTime
		)
		if err := rows.Scan(&objectID, &coverageID, &itemID, &estimateID, &kind, &currency,
			&value, &rate, &converted, &date, &printed, &faceSheetDate); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris estimasi: %w", err)
		}
		at, ok := itemAt[key{strings.TrimSpace(objectID.String), strings.TrimSpace(coverageID.String), strings.TrimSpace(itemID.String)}]
		if !ok {
			// Estimasi milik item yang ditandai hapus, atau baris Pega yang tidak punya
			// item di TC_PNC_OBJECTITEM.
			continue
		}
		e := registrasi.Estimation{
			Type:      strings.TrimSpace(kind.String),
			Currency:  strings.TrimSpace(currency.String),
			Value:     registrasi.Money(value.Int64),
			Rate:      registrasi.ExchangeRate(rate.Int64),
			Converted: registrasi.Money(converted.Int64),
			FaceSheet: printed.Valid && printed.Int64 == 1,
		}
		if date.Valid {
			e.Date = date.Time.UTC()
		}
		if faceSheetDate.Valid {
			e.FaceSheetDate = faceSheetDate.Time.UTC()
		}
		item := &coverages[at[0]].Item[at[1]]
		item.Estimation = append(item.Estimation, e)
	}
	return rows.Err()
}

// CurrencyDirectory membaca master mata uang.
type CurrencyDirectory struct {
	db *sql.DB
}

// NewCurrencyDirectory membentuk pembaca master mata uang.
func NewCurrencyDirectory(db *sql.DB) *CurrencyDirectory { return &CurrencyDirectory{db: db} }

// Currencies mengembalikan seluruh mata uang, diurutkan menurut kodenya.
func (d *CurrencyDirectory) Currencies(ctx context.Context) ([]registrasi.CurrencyOption, error) {
	rows, err := d.db.QueryContext(ctx, loadQuery("mata_uang_daftar"))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca mata uang: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.CurrencyOption
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris mata uang: %w", err)
		}
		if strings.TrimSpace(id.String) == "" {
			continue
		}
		result = append(result, registrasi.CurrencyOption{ID: strings.TrimSpace(id.String), Name: strings.TrimSpace(name.String)})
	}
	return result, rows.Err()
}
