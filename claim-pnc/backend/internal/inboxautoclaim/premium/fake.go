package premium

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/inboxautoclaim"
)

// Fake adalah pemeriksa premi tanpa jaringan, untuk pengujian dan mode memori.
//
// Bawaannya menjawab LUNAS (AgingAmount "0") untuk setiap polis. Polis tertentu dapat
// diberi jawaban lain, atau dibuat gagal seolah layanannya mati.
type Fake struct {
	lock   sync.Mutex
	answer map[string]string
	fail   map[string]bool
	calls  int

	// paid adalah jawaban total premi tab Cek Premi per "sumber|bisnis".
	paid        map[string]string
	totalFailed bool
}

// NewFake membentuk Fake yang menjawab lunas untuk semua polis.
func NewFake() *Fake {
	return &Fake{answer: map[string]string{}, fail: map[string]bool{}, paid: map[string]string{}}
}

// SetAging menetapkan AgingAmount untuk satu polis.
func (f *Fake) SetAging(policyNo, aging string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.answer[strings.ToUpper(strings.TrimSpace(policyNo))] = aging
}

// SetUnreachable membuat panggilan untuk satu polis gagal seolah layanannya mati.
func (f *Fake) SetUnreachable(policyNo string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.fail[strings.ToUpper(strings.TrimSpace(policyNo))] = true
}

// Calls menyebut berapa kali layanan dipanggil.
func (f *Fake) Calls() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.calls
}

// CheckPremium memenuhi inboxautoclaim.PremiumChecker.
func (f *Fake) CheckPremium(_ context.Context, _ string, query inboxautoclaim.PremiumQuery) (inboxautoclaim.PremiumAnswer, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.calls++

	key := strings.ToUpper(strings.TrimSpace(query.PolicyNo))
	if f.fail[key] {
		return inboxautoclaim.PremiumAnswer{}, errUnreachable
	}
	if aging, ok := f.answer[key]; ok {
		return inboxautoclaim.PremiumAnswer{AgingAmount: aging}, nil
	}
	return inboxautoclaim.PremiumAnswer{AgingAmount: "0"}, nil
}

// SetPremiumPaid menetapkan total premi terbayar untuk satu pasangan sumber bisnis + bisnis.
func (f *Fake) SetPremiumPaid(sourceOfBusiness, businessCode, total string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.paid[paidKey(sourceOfBusiness, businessCode)] = total
}

// SetTotalUnreachable membuat layanan total premi gagal seolah mati.
func (f *Fake) SetTotalUnreachable() {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.totalFailed = true
}

// PremiumPaidBySource memenuhi inboxautoclaim.PremiumChecker. Pasangan yang belum
// ditetapkan menjawab "0".
func (f *Fake) PremiumPaidBySource(_ context.Context, _ string, query inboxautoclaim.PremiumCheckQuery) (string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.calls++
	if f.totalFailed {
		return "", errUnreachable
	}
	if total, ok := f.paid[paidKey(query.SourceOfBusiness, query.BusinessCode)]; ok {
		return total, nil
	}
	return "0", nil
}

func paidKey(sourceOfBusiness, businessCode string) string {
	return strings.TrimSpace(sourceOfBusiness) + "|" + strings.TrimSpace(businessCode)
}

type fakeError string

func (e fakeError) Error() string { return string(e) }

const errUnreachable = fakeError("inboxautoclaim/premium: layanan tiruan dibuat tidak dapat dihubungi")

var _ inboxautoclaim.PremiumChecker = (*Fake)(nil)
