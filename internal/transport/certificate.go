package transport

import (
	"html/template"
	"net/http"
	"strings"
)

func (s *Server) HandleCertificateAPI(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	code := strings.TrimPrefix(r.URL.Path, "/api/certificates/")
	certificate, err := s.app.GetCertificate(code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, certificate)
}

var certificateTemplate = template.Must(template.New("certificate").Parse(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>海岸带修复放行凭据</title><style>body{margin:0;background:#eef1ed;color:#17211d;font:16px Georgia,"Noto Serif SC",serif}.sheet{max-width:760px;margin:7vh auto;background:#fff;padding:56px;border:1px solid #aeb9b3;box-shadow:0 18px 60px #20372d22}.mark{color:#087f68;font:700 12px sans-serif;letter-spacing:0}.code{font:700 28px ui-monospace,monospace;color:#0c6658;word-break:break-all}.rule{height:4px;background:#e45b2a;margin:26px 0}.grid{display:grid;grid-template-columns:150px 1fr;gap:14px;margin-top:32px}.label{color:#5a6962;font:13px sans-serif}.digest{word-break:break-all;font:12px ui-monospace,monospace}@media(max-width:700px){.sheet{margin:0;padding:32px 20px;min-height:100vh}.grid{grid-template-columns:1fr;gap:5px 14px}}</style></head>
<body><main class="sheet"><div class="mark">COASTAL RESTORATION / VERIFIED RELEASE</div><h1>海岸带修复放行凭据</h1><div class="rule"></div><p>此凭据对应的证据链已完成独立复核并冻结，可通过下列摘要核对本地事件账本。</p><p class="code">{{.CredentialCode}}</p><div class="grid"><div class="label">案件编号</div><div>{{.CaseID}}</div><div class="label">独立复核员</div><div>{{.Reviewer}}</div><div class="label">复核决定</div><div>{{.Decision}}</div><div class="label">冻结时间</div><div>{{.FrozenAt}}</div><div class="label">复核说明</div><div>{{.ReviewNote}}</div><div class="label">证据摘要</div><div class="digest">{{.EvidenceDigest}}</div></div></main></body></html>`))

func (s *Server) HandleCertificatePage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	code := strings.TrimPrefix(r.URL.Path, "/certificates/")
	certificate, err := s.app.GetCertificate(code)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = certificateTemplate.Execute(w, certificate)
}
