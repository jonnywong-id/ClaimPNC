package memory

import (
	"context"
	"sync"
	"time"

	"claim-pnc/internal/registrasi"
)

// Cashier adalah penyimpanan dan layanan Transfer Kasir di memori, untuk pengujian dan
// pengembangan lokal. Layanannya TIDAK pernah menjawab berhasil kecuali Reply diisi.
type Cashier struct {
	mu sync.Mutex

	Banks map[string]string // nama bank → LBG_ID
	Logs  []registrasi.CashierLog
	// Unregistered adalah rekening yang TIDAK terdaftar di Kasir; selebihnya terdaftar.
	Unregistered map[string]bool
	// ServiceLogs merekam LogService; ServiceLogErr membuatnya gagal.
	ServiceLogs   []registrasi.CashierServiceLog
	ServiceLogErr error
	Marked        []CashierMark

	Reply    registrasi.CashierReply
	Err      error
	Payloads []registrasi.CashierPayload
	Services []string
}

// CashierMark merekam MarkTransferred.
type CashierMark struct {
	ClaimID, ObjectID          string
	CoverageSeq, AdjustmentSeq int
	At                         time.Time
	CaseID                     string
}

// NewCashier membentuk Transfer Kasir kosong.
func NewCashier() *Cashier { return &Cashier{Banks: map[string]string{}} }

var (
	_ registrasi.CashierStore   = (*Cashier)(nil)
	_ registrasi.CashierGateway = (*Cashier)(nil)
)

func (c *Cashier) BankGroupID(_ context.Context, bankName, bankID string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.Banks[bankName]
	return id, ok && id == bankID, nil
}

func (c *Cashier) Log(_ context.Context, e registrasi.CashierLog) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Logs = append(c.Logs, e)
	return nil
}

func (c *Cashier) AccountRegistered(_ context.Context, accountNo, _ string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.Unregistered[accountNo], nil
}

func (c *Cashier) LogService(_ context.Context, e registrasi.CashierServiceLog) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ServiceLogErr != nil {
		return c.ServiceLogErr
	}
	c.ServiceLogs = append(c.ServiceLogs, e)
	return nil
}

func (c *Cashier) MarkTransferred(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, at time.Time, caseID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Marked = append(c.Marked, CashierMark{claimID, objectID, coverageSeq, adjustmentSeq, at, caseID})
	return nil
}

func (c *Cashier) Transfer(_ context.Context, _, service string, payload registrasi.CashierPayload) (registrasi.CashierReply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Payloads = append(c.Payloads, payload)
	c.Services = append(c.Services, service)
	return c.Reply, c.Err
}
