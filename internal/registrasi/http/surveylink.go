package registrasihttp

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// MountPublic mendaftarkan rute link email Komite survey. Rute ini TIDAK memerlukan sesi —
// anggota Komite membukanya dari email, seperti Service REST `KomiteAcceptSurvey` dan
// `ShowKomiteSurveyConfirmation` (paket layanannya diakses tanpa autentikasi). Pengamannya
// adalah parameter `key` yang diperiksa `PNCKomiteSurveyAcceptation_act`.
func MountPublic(r chi.Router, h *Handler) {
	r.Get("/publik/komite-survey/terima", h.SurveyAcceptLink)
	r.Get("/publik/komite-survey/konfirmasi", h.SurveyConfirmLink)
}

func linkCommand(r *http.Request) usecase.SurveyLinkCommand {
	q := r.URL.Query()
	return usecase.SurveyLinkCommand{
		CaseID: q.Get("caseid"), ObjectID: q.Get("object_id"), Surveyor: q.Get("surveyor"),
		Approved: q.Get("aproved"), Comment: q.Get("comment"), CommitteeID: q.Get("komiteid"),
		Count: q.Get("komitecount"), Key: q.Get("key"),
	}
}

// emailThankYou adalah HTML `EmailThankYou` — satu-satunya teks aktifnya.
var emailThankYou = template.Must(template.New("terima").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Komite Survey</title></head>
<body style="font-family:Arial,sans-serif"><center><h2>Terima Kasih atas Jawabannya</h2></center></body></html>`))

// SurveyAcceptLink menangani link YA dan kiriman halaman konfirmasi — `KomiteAcceptSurvey`.
// Jawabannya selalu `EmailThankYou`, tercatat atau tidak (Pega hanya punya respons default).
func (h *Handler) SurveyAcceptLink(w http.ResponseWriter, r *http.Request) {
	h.service.AcceptSurveyByLink(r.Context(), linkCommand(r), h.logger)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = emailThankYou.Execute(w, nil)
}

// komiteSurveyConfirmation adalah HTML `KomiteSurveyConfirmation`.
var komiteSurveyConfirmation = template.Must(template.New("konfirmasi").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Komite Survey</title>
<style>body{font-family:Arial,sans-serif;font-size:13px;margin:16px}table{border-collapse:collapse}td,th{border:1px solid #999;padding:4px 8px}
button{padding:6px 14px;margin-right:8px}textarea{width:320px;height:70px}</style></head>
<body>
{{if eq .Params.Approved "1"}}<p>Terima kasih telah menjawab, jawaban email anda telah di terima</p>{{else}}
<form onsubmit="return false">
<table>
<tr><th>ID Object</th><th>Nama Object</th><th>Lokasi Object</th><th>Nama Adjuster</th></tr>
<tr><td>{{.Params.ObjectID}}</td><td>{{.ObjectName}}</td><td>{{.Location}}</td><td>{{.Params.Surveyor}}</td></tr>
</table>
<p><select id="surveyor"><option value="">--Pilih--</option>{{range .Options}}<option value="{{.Login}}">{{.Name}}</option>{{end}}</select></p>
<p>Alasan :<br><textarea id="comment"></textarea></p>
<p><button type="button" onclick="yesKomite('1')">PILIH ADJUSTER</button><button type="button" onclick="yesKomite('2')">TIDAK PAKAI ADJUSTER</button></p>
</form>
<script>
var p = {caseid: {{.Params.CaseID}}, object_id: {{.Params.ObjectID}}, komiteid: {{.Params.CommitteeID}}, komitecount: {{.Params.Count}}, key: {{.Params.Key}}};
function yesKomite(aproved) {
  var surveyor = document.getElementById('surveyor').value;
  var comment = document.getElementById('comment').value;
  if (aproved == '1') {
    if (surveyor == '') { alert('harap isi surveyor'); return; }
    if (comment == '') { alert('harap isi comment'); return; }
  } else {
    if (comment == '') { alert('harap isi comment'); return; }
  }
  var q = 'caseid=' + encodeURIComponent(p.caseid) + '&object_id=' + encodeURIComponent(p.object_id) +
    '&surveyor=' + encodeURIComponent(surveyor) + '&comment=' + encodeURIComponent(comment) + '&aproved=' + aproved +
    '&komiteid=' + encodeURIComponent(p.komiteid) + '&komitecount=' + encodeURIComponent(p.komitecount) + '&key=' + encodeURIComponent(p.key);
  window.location = {{.AcceptPath}} + '?' + q;
}
</script>{{end}}
</body></html>`))

// SurveyConfirmLink menangani link TIDAK / PILIH ADJUSTER — `ShowKomiteSurveyConfirmation`.
func (h *Handler) SurveyConfirmLink(w http.ResponseWriter, r *http.Request) {
	cmd := linkCommand(r)
	page, err := h.service.SurveyConfirmationPage(r.Context(), cmd, r.URL.Query().Get("surveyorid"))
	if err != nil {
		h.logger.Warn("halaman konfirmasi Komite survey", "galat", err.Error())
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = komiteSurveyConfirmation.Execute(w, map[string]any{
		"Params": page.Params, "ObjectName": page.ObjectName, "Location": page.Location, "Options": page.Options,
		"AcceptPath": registrasi.SurveyAcceptPath,
	})
}
