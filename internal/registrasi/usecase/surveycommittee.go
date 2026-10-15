package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// Persetujuan Komite survey lewat link email — Service REST `KomiteAcceptSurvey`
// (`PNCKomiteSurveyAcceptation_act`, jawaban HTML `EmailThankYou`) dan
// `ShowKomiteSurveyConfirmation` (`ShowKomiteSurveyor`, HTML `KomiteSurveyConfirmation`).
//
// Ditiru PERSIS Pega (keputusan Work Owner 2026-10-11): anggota dikenali dari indeks
// `komitecount`, bukan dari login; `key` dibandingkan dengan kunci kasus; link anggota
// berikutnya dikirim dengan `key` KOSONG (`tempCvg.pyLabel` tidak pernah diisi), sehingga hanya
// anggota pertama yang dapat memutuskan lewat link; persetujuan terakhir lewat link TIDAK menulis
// T_SURVEYORLIST maupun progres. Setiap hasil — tercatat atau tidak — dijawab halaman yang sama.

// SurveyLinkCommand adalah parameter link KomiteAcceptSurvey.
type SurveyLinkCommand struct {
	CaseID      string // caseid
	ObjectID    string // object_id
	Surveyor    string // surveyor — LOGIN_APLIKASI
	Approved    string // aproved — 1 setuju, 2 tolak
	Comment     string // comment
	CommitteeID string // komiteid
	Count       string // komitecount
	Key         string // key
}

// SurveyConfirmation adalah isi halaman KomiteSurveyConfirmation.
type SurveyConfirmation struct {
	Params     SurveyLinkCommand
	ObjectName string
	Location   string
	Options    []registrasi.SurveyorOption // TempResultSurveyor
	Marine     []registrasi.SurveyorOption // TempResultmarine
}

func newCommitteeKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// sendSurveyCommitteeMail mengirim email SendApproveCommittee ke satu anggota. Galat dikembalikan
// sebagai teks untuk ditampilkan, bukan menggagalkan proses yang sudah tersimpan.
func (l *Service) sendSurveyCommitteeMail(ctx context.Context, claim registrasi.Claim, o registrasi.SurveyObject,
	committeeID, to string, count int, key string, note registrasi.SurveyCommitteeNote) string {
	if l.surveyMailer == nil {
		return "pengirim email Komite survey tidak terpasang di server ini"
	}
	if strings.TrimSpace(l.surveyLinkBase) == "" {
		return "ALAMAT_PUBLIK belum diisi, link persetujuan email Komite tidak dapat dibentuk"
	}
	if strings.TrimSpace(to) == "" {
		return "email anggota Komite kosong di POOLDATA.EMAILKOMITE"
	}
	cause := ""
	for _, it := range claim.InsuredItem {
		if strings.TrimSpace(it.ID) == o.ObjectID && len(it.Coverage) > 0 {
			cause = it.Coverage[len(it.Coverage)-1].CauseOfLoss
		}
	}
	html, err := registrasi.RenderSurveyCommitteeMail(registrasi.SurveyCommitteeMail{
		Claim: claim, Object: o, Note: note, Count: count, CauseName: cause, At: l.clock.Now(),
		Links: registrasi.BuildSurveyCommitteeLinks(l.surveyLinkBase, registrasi.SurveyLinkParams{
			CaseID: claim.ID, ObjectID: o.ObjectID, SurveyorType: o.SurveyorType, Login: o.SurveyorLogin,
			Name: o.SurveyorName, CommitteeID: committeeID, Count: count, Key: key,
		}),
	})
	if err != nil {
		return err.Error()
	}
	if err := l.surveyMailer.Send(ctx, splitEmails(to), registrasi.SurveyCommitteeSubject(claim.Number), html); err != nil {
		return err.Error()
	}
	return ""
}

func splitEmails(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == ',' }) {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// AcceptSurveyByLink adalah `PNCKomiteSurveyAcceptation_act`. Tidak mengembalikan galat ke
// pemanggil link: seperti Pega, setiap hasil dijawab "Terima Kasih atas Jawabannya". Galat
// teknis dicatat ke log.
func (l *Service) AcceptSurveyByLink(ctx context.Context, p SurveyLinkCommand, logger *slog.Logger) {
	if err := l.acceptSurveyByLink(ctx, p); err != nil && logger != nil {
		logger.Warn("link persetujuan Komite survey tidak tercatat",
			slog.String("komite", p.CommitteeID), slog.String("komitecount", p.Count), slog.String("galat", err.Error()))
	}
}

func (l *Service) acceptSurveyByLink(ctx context.Context, p SurveyLinkCommand) error {
	store, err := l.surveyStore()
	if err != nil {
		return err
	}
	// Langkah 7: SearchSurveyorName — surveyor menurut login (boleh tidak ditemukan).
	surveyor, _, err := store.SurveyorByLogin(ctx, p.Surveyor)
	if err != nil {
		return err
	}
	// Langkah 8–11: kasus komite, key, komitecount.
	committee, err := l.committees.Get(ctx, strings.TrimSpace(p.CommitteeID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(committee.TransferType) != registrasi.SurveyCommitteeTransfer {
		return fmt.Errorf("kasus %s bukan komite survey", committee.ID)
	}
	note, err := store.CommitteeNote(ctx, committee.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(p.Key) != note.Key || note.Key == "" {
		return fmt.Errorf("key tidak cocok (langkah 9)")
	}
	current, open := committee.Current()
	count, _ := strconv.Atoi(strings.TrimSpace(p.Count))
	if !open || count != current.Level {
		return fmt.Errorf("komitecount %d bukan jenjang yang ditunggu (langkah 11)", count)
	}
	claim, err := l.claim.GetByNumber(ctx, committee.ClaimNumber)
	if err != nil {
		return err
	}
	objects, err := store.Objects(ctx, claim.ID)
	if err != nil {
		return err
	}
	now := l.clock.Now().UTC()
	approved := strings.TrimSpace(p.Approved)
	final := approved == registrasi.DecisionApprove && current.Level == len(committee.Members)

	// Langkah 30: putusan anggota pada kasus komite.
	switch approved {
	case registrasi.DecisionApprove, registrasi.DecisionReject:
		if err := committee.Decide(current.Operator, approved, p.Comment, now); err != nil {
			return err
		}
	default:
		// aproved lain (mis. 0): tercatat, IsKomiteLoop salah, alur komite selesai.
		for i := range committee.Members {
			if committee.Members[i].Level == current.Level {
				committee.Members[i].Note, committee.Members[i].DecidedAt = strings.TrimSpace(p.Comment), now
			}
			committee.Members[i].CaseStatus = registrasi.CommitteeCaseClosed
		}
	}
	if committee.Outcome() != "" || approved != registrasi.DecisionApprove {
		committee.DecidedAt = now
	}
	committee.UpdatedAt, committee.UpdatedBy = now, current.Operator

	// Langkah 15 dan 24: objek yang diputuskan.
	var decided *registrasi.SurveyObject
	for i := range objects {
		if objects[i].ObjectID == strings.TrimSpace(p.ObjectID) {
			decided = &objects[i]
		}
	}
	if decided != nil {
		if approved == registrasi.DecisionApprove {
			decided.SurveyorName = surveyor.Name
			decided.SurveyorLogin = strings.TrimSpace(p.Surveyor)
			decided.SurveyorAddr = surveyor.Address
		}
		if final {
			// Langkah 20–22: kasus survey — di aplikasi ini hanya nomornya (tanpa baris
			// T_SURVEYORLIST, seperti jalur link Pega).
			id, err := store.NextSurveyID(ctx, now)
			if err != nil {
				return err
			}
			decided.SurveyID, decided.Status, decided.Selected = id, registrasi.SurveyInProgress, false
		}
		if approved == registrasi.DecisionReject {
			decided.Status, decided.Selected = registrasi.SurveyRejected, false
		}
	}
	claim.ClaimStatus = registrasi.StatusClaimSurvey // langkah 15.4 (selalu)
	claim.UpdatedBy, claim.UpdatedAt = current.Operator, now

	if err := l.unit.Run(ctx, func(ctx context.Context) error {
		if err := store.SaveObjects(ctx, claim.ID, objects, current.Operator, now); err != nil {
			return err
		}
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.committees.Save(ctx, committee); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		event := "KOMITE_SURVEY_SETUJU"
		if approved != registrasi.DecisionApprove {
			event = "KOMITE_SURVEY_TOLAK"
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: event, Actor: current.Operator, At: now,
			Note: fmt.Sprintf("%s jenjang %d dari %d lewat link email — %s", committee.ID, current.Level, len(committee.Members), strings.TrimSpace(p.Comment)),
		})
	}); err != nil {
		return err
	}

	// Langkah 14: email ke anggota berikutnya — dengan key KOSONG, seperti Pega.
	if approved == registrasi.DecisionApprove && !final && decided != nil && current.Level < len(committee.Members) {
		next := committee.Members[current.Level]
		emails, err := store.CommitteeMembers(ctx, committee.Line, false)
		if err == nil {
			to := ""
			for _, m := range emails {
				if strings.EqualFold(m.OperatorID, next.Operator) {
					to = m.Email
				}
			}
			if msg := l.sendSurveyCommitteeMail(ctx, claim, *decided, committee.ID, to, current.Level+1, "", note); msg != "" {
				return fmt.Errorf("email anggota berikutnya: %s", msg)
			}
		}
	}
	return nil
}

// SurveyConfirmationPage adalah `ShowKomiteSurveyor`: objek yang diputuskan dan pilihan adjuster
// menurut surveyorid (2/4 Loss Adjuster + "Others" untuk 2, 3 Expert, 4/2 Survey Agent ke daftar
// marine). Tidak memeriksa key, komitecount, maupun status jawaban — seperti Pega.
func (l *Service) SurveyConfirmationPage(ctx context.Context, p SurveyLinkCommand, surveyorType string) (SurveyConfirmation, error) {
	out := SurveyConfirmation{Params: p}
	store, err := l.surveyStore()
	if err != nil {
		return out, err
	}
	if claim, err := l.claim.Get(ctx, strings.TrimSpace(p.CaseID)); err == nil {
		if objects, err := store.Objects(ctx, claim.ID); err == nil {
			for _, o := range objects {
				if o.ObjectID == strings.TrimSpace(p.ObjectID) {
					out.ObjectName, out.Location = o.Name, o.Location
				}
			}
		}
	}
	t := strings.TrimSpace(surveyorType)
	switch t {
	case registrasi.SurveyorLossAdjuster, registrasi.SurveyorSurveyAgent:
		if out.Options, err = store.Surveyors(ctx, registrasi.SurveyorLossAdjuster); err != nil {
			return out, err
		}
		if t == registrasi.SurveyorLossAdjuster {
			out.Options = append(out.Options, registrasi.SurveyorOption{Name: "Others"})
		}
		if out.Marine, err = store.Surveyors(ctx, registrasi.SurveyorSurveyAgent); err != nil {
			return out, err
		}
	case registrasi.SurveyorExpert:
		if out.Options, err = store.Surveyors(ctx, registrasi.SurveyorExpert); err != nil {
			return out, err
		}
	}
	return out, nil
}

// WithSurveyMail memasang pengirim email dan pangkal alamat link Komite survey.
func (l *Service) WithSurveyMail(m registrasi.SurveyMailer, linkBase string) {
	l.surveyMailer, l.surveyLinkBase = m, strings.TrimSpace(linkBase)
}
