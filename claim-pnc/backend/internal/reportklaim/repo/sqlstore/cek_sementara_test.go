package sqlstore

import (
	"testing"

	"claim-pnc/internal/reportklaim"
)

func TestSementaraDaftarSiapTanpaKueri(t *testing.T) {
	for _, r := range reportklaim.Catalog() {
		if !r.Availability.Ready {
			t.Logf("TERHALANG  %-24s %s", r.Code, r.Title)
			continue
		}
		p, ada := plans[r.Code]
		if !ada {
			t.Errorf("TANPA RENCANA  %-24s %s", r.Code, r.Title)
			continue
		}
		name := p.query(reportklaim.Filter{})
		if !hasQuery(name) {
			t.Errorf("KUERI HILANG   %-24s -> %s", r.Code, name)
		}
	}
}
