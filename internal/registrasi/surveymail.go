package registrasi

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"
)

// # Email dan link persetujuan Komite survey
//
// Correspondence `SendApproveCommittee` — email ke anggota Komite survey beserta link YA / TIDAK
// / PILIH ADJUSTER. Link aslinya menuju Service REST `KomiteAcceptSurvey` dan
// `ShowKomiteSurveyConfirmation`; di sini menuju rute publik aplikasi dengan parameter yang
// sama (caseid, object_id, surveyor, surveyorid, aproved, komiteid, komitecount, key).

// SurveyCommitteeLinks adalah alamat link email Komite survey.
type SurveyCommitteeLinks struct {
	Accept  string // KomiteAcceptSurvey — YA (aproved=1)
	Confirm string // ShowKomiteSurveyConfirmation — TIDAK / PILIH ADJUSTER (aproved=0)
}

// SurveyLinkParams adalah parameter link — `SendApproveCommittee` baris 362-375.
type SurveyLinkParams struct {
	CaseID       string // caseid — klaim
	ObjectID     string // object_id
	SurveyorType string // surveyorid — `$THIS.SurveyorType`
	Login        string // surveyor pada link YA — `ObjectSurveyorLogin`
	Name         string // surveyor pada link TIDAK — `ObjectSurveyor` (nama)
	CommitteeID  string // komiteid
	Count        int    // komitecount — `tempCvg.ObjectID`
	Key          string // key — `tempCvg.pyLabel`
}

// BuildSurveyCommitteeLinks menyusun kedua link dari pangkal alamat publik aplikasi.
func BuildSurveyCommitteeLinks(base string, p SurveyLinkParams) SurveyCommitteeLinks {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	q := func(approved, surveyor string) string {
		v := url.Values{}
		v.Set("caseid", p.CaseID)
		v.Set("object_id", p.ObjectID)
		v.Set("surveyor", surveyor)
		v.Set("surveyorid", p.SurveyorType)
		v.Set("aproved", approved)
		v.Set("komiteid", p.CommitteeID)
		v.Set("komitecount", fmt.Sprint(p.Count))
		v.Set("key", p.Key)
		return v.Encode()
	}
	return SurveyCommitteeLinks{
		Accept:  base + SurveyAcceptPath + "?" + q("1", p.Login),
		Confirm: base + SurveyConfirmPath + "?" + q("0", p.Name),
	}
}

// Jalur publik link Komite survey.
const (
	SurveyAcceptPath  = "/api/publik/komite-survey/terima"
	SurveyConfirmPath = "/api/publik/komite-survey/konfirmasi"
)

// SurveyCommitteeSubject adalah subjek email — "AKSEPTASI KOMITE SURVEYOR " + pyID.
func SurveyCommitteeSubject(claimNumber string) string {
	return "AKSEPTASI KOMITE SURVEYOR " + claimNumber
}

// SurveyTypeTitle adalah judul per tipe surveyor pada SendApproveCommittee / AdjusterAppointed_CC.
var SurveyTypeTitle = map[string]string{
	SurveyorInternal:     "Surveyor",
	SurveyorLossAdjuster: "Loss Adjuster",
	SurveyorExpert:       "Expert / Consultant",
	SurveyorSurveyAgent:  "Survey Agent",
}

// SurveyCommitteeMail adalah bahan email Komite survey.
type SurveyCommitteeMail struct {
	Claim     Claim
	Object    SurveyObject
	Note      SurveyCommitteeNote
	Count     int // komitecount link — 2 berarti mode "Pilih Adjuster"
	Links     SurveyCommitteeLinks
	CauseName string
	At        time.Time
}

var surveyCommitteeTemplate = template.Must(template.New("komite").Funcs(template.FuncMap{
	"date": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("02/01/2006")
	},
	"pct": func(p Percent) string {
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", float64(p)/10000), "0"), ".")
	},
	"money": func(m Money) string {
		digits := fmt.Sprintf("%d", int64(m)/100)
		var out []byte
		for i, d := range []byte(digits) {
			if i > 0 && (len(digits)-i)%3 == 0 {
				out = append(out, '.')
			}
			out = append(out, d)
		}
		return string(out)
	},
}).Parse(`<html><body style="font-family:Arial,sans-serif;font-size:12px">
<p style="text-align:right;font-style:italic">Persetujuan Komite {{.Title}}</p>
<h3 style="text-align:center;margin:0">CLAIM COMMITEE</h3>
<p style="text-align:center;margin:0 0 8px 0">SHEET NO.01</p>
<table cellpadding="3" style="border-collapse:collapse;font-size:12px">
<tr><td>INSURED</td><td>:</td><td>{{.M.Claim.Policy.InsuredName}}</td></tr>
<tr><td>CLASS OF INSURANCE</td><td>:</td><td>{{.M.Claim.Policy.BusinessName}}</td></tr>
<tr><td>POLICY NO.</td><td>:</td><td>{{.M.Claim.Policy.Number}}</td></tr>
<tr><td>PERIOD OF INSURANCE</td><td>:</td><td>{{date .M.Claim.Policy.CoverageStart}} s/d {{date .M.Claim.Policy.CoverageEnd}}</td></tr>
<tr><td>INTEREST INSURED</td><td>:</td><td>{{.Interest}}</td></tr>
<tr><td>CLAIM NO.</td><td>:</td><td>{{.M.Claim.Number}}</td></tr>
<tr><td>DATE OF ACCIDENT</td><td>:</td><td>{{date .M.Claim.DateOfLoss}}</td></tr>
<tr><td>NATURE OF LOSS</td><td>:</td><td>{{.M.CauseName}}</td></tr>
<tr><td>RISK LOCATION</td><td>:</td><td>{{.M.Object.SurveyLocation}}</td></tr>
<tr><td>DATE</td><td>:</td><td>{{.When}}</td></tr>
<tr><td>INITIAL</td><td>:</td><td>{{.M.Note.Initial}}</td></tr>
</table>
<p><b>COMMENT AND ANALYSIS</b></p>
<p>I. Circumtances/Cause Of Loss :<br>{{.M.Note.Circumstances}}</p>
<p>II. Spreading Risk :</p>
<table border="1" cellpadding="3" style="border-collapse:collapse;font-size:12px">
<tr><th>SPREADING RISK</th><th>%</th></tr>
{{range .Spreading}}<tr><td>{{.Name}}</td><td>{{pct .Share}}</td></tr>{{end}}
<tr><td><b>TOTAL SPREADING</b></td><td><b>100</b></td></tr>
</table>
<p>III. Nominated Adjuster :<br>{{.M.Note.Nominated}}</p>
<p>IV. Remarks :<br>{{.M.Note.Remarks}}</p>
<p>Object List :</p>
<table border="1" cellpadding="3" style="border-collapse:collapse;font-size:12px">
<tr><th>Nama Object</th><th>Lokasi Object</th><th>Nama {{.Title}}</th></tr>
<tr><td>{{.M.Object.Name}}</td><td>{{.M.Object.Location}}</td><td>{{.M.Object.SurveyorName}}</td></tr>
</table>
<p>Coverage List :</p>
<table border="1" cellpadding="3" style="border-collapse:collapse;font-size:12px">
<tr><th>Coverage</th><th>Cause of Loss</th><th>Sum of TSI</th></tr>
{{range .Coverage}}<tr><td>{{.Name}}</td><td>{{.CauseOfLoss}}</td><td>{{money .TSI}}</td></tr>{{end}}
</table>
{{if eq .M.Count 2}}<p>Pilih Adjuster untuk case ini ?</p>
<p><a href="{{.M.Links.Confirm}}" style="padding:6px 14px;background:#1d4ed8;color:#fff;text-decoration:none">PILIH ADJUSTER</a></p>
{{else}}<p>Apakah anda menyetujui pengajuan Adjuster ini?</p>
<p><a href="{{.M.Links.Accept}}" style="padding:6px 14px;background:#15803d;color:#fff;text-decoration:none">YA</a>
&nbsp; <a href="{{.M.Links.Confirm}}" style="padding:6px 14px;background:#b91c1c;color:#fff;text-decoration:none">TIDAK</a></p>{{end}}
</body></html>`))

// RenderSurveyCommitteeMail menyusun badan email `SendApproveCommittee` untuk satu objek.
func RenderSurveyCommitteeMail(m SurveyCommitteeMail) (string, error) {
	var item InsuredItem
	for _, it := range m.Claim.InsuredItem {
		if strings.TrimSpace(it.ID) == strings.TrimSpace(m.Object.ObjectID) {
			item = it
		}
	}
	var names []string
	var spreading []Spreading
	for i, c := range item.Coverage {
		names = append(names, c.Name)
		if i == 0 {
			for _, s := range c.Spreading {
				if !s.Removed {
					spreading = append(spreading, s)
				}
			}
		}
	}
	title := SurveyTypeTitle[m.Object.SurveyorType]
	if title == "" {
		title = "Surveyor"
	}
	var buf bytes.Buffer
	err := surveyCommitteeTemplate.Execute(&buf, map[string]any{
		"M": m, "Title": title, "Interest": strings.Join(names, ", "), "Spreading": spreading,
		"Coverage": item.Coverage, "When": m.At.In(wib).Format("02 Jan 2006 15:04:05"),
	})
	return buf.String(), err
}

var wib = time.FixedZone("WIB", 7*3600)
