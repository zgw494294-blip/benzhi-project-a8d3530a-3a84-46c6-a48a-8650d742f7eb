package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/transport"
)

func runSelfcheck(address string, logger *slog.Logger) error {
	dataDir, err := os.MkdirTemp("", "coastal-evidence-selfcheck-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dataDir)
	handler, err := buildHandler(dataDir, logger)
	if err != nil {
		return fmt.Errorf("初始化账本: %w", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("监听配置地址 %s: %w", address, err)
	}
	server := transport.NewHTTPServer(address, handler)
	serveResult := make(chan error, 1)
	go func() {
		serveErr := server.Serve(listener)
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		serveResult <- serveErr
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		<-serveResult
	}()

	baseURL := "http://" + listener.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	if _, err := selfcheckGet(client, baseURL+"/healthz"); err != nil {
		return fmt.Errorf("健康检查: %w", err)
	}
	indexPage, err := selfcheckGet(client, baseURL+"/")
	if err != nil || !strings.Contains(string(indexPage), "<body>") || !strings.Contains(string(indexPage), "海岸带修复证据验收台") {
		return fmt.Errorf("浏览器工作台首页验证失败: %w", err)
	}
	styleAsset, err := selfcheckGet(client, baseURL+"/assets/app.css")
	if err != nil || !strings.Contains(string(styleAsset), ".workspace") {
		return fmt.Errorf("工作台样式资源验证失败: %w", err)
	}
	scriptAsset, err := selfcheckGet(client, baseURL+"/assets/app.js")
	if err != nil || !strings.Contains(string(scriptAsset), "submitAcceptance") {
		return fmt.Errorf("工作台脚本资源验证失败: %w", err)
	}
	createBody := selfcheckMeta("监测员-自检", domain.RoleMonitor, 0, "selfcheck-create")
	createBody["name"] = "自检红树林修复单元"
	createBody["siteCode"] = "SELF-MG-01"
	createBody["habitatType"] = "红树林"
	createBody["baseline"] = []map[string]any{
		{"indicator": "植被覆盖率", "minimum": 70, "maximum": 100, "unit": "%"},
	}
	created, err := selfcheckPost(client, baseURL, "/api/cases", createBody)
	if err != nil {
		return fmt.Errorf("建档: %w", err)
	}
	if created.Case == nil || created.Case.Version != 1 {
		return fmt.Errorf("建档响应缺少案件或版本错误")
	}
	caseID := created.Case.ID

	monitorBody := selfcheckMeta("监测员-自检", domain.RoleMonitor, created.Case.Version, "selfcheck-monitor")
	monitorBody["indicator"] = "植被覆盖率"
	monitorBody["observedValue"] = 52
	monitorBody["unit"] = "%"
	monitorBody["evidenceNote"] = "自检样方 A-01，影像证据 IMG-SELF-001"
	monitorBody["capturedBy"] = "监测员-自检"
	monitorBody["capturedAt"] = time.Now().UTC()
	monitorBody["remediationOwner"] = "整改员-自检"
	monitorBody["remediationDueAt"] = time.Now().UTC().Add(24 * time.Hour)
	monitored, err := selfcheckPost(client, baseURL, "/api/cases/"+caseID+"/monitoring", monitorBody)
	if err != nil {
		return fmt.Errorf("监测与整改生成: %w", err)
	}
	if monitored.Action == nil || monitored.Action.Status != domain.RemediationOpen {
		return fmt.Errorf("超基线监测未生成整改任务")
	}

	retestBody := selfcheckMeta("整改员-自检", domain.RoleRemediator, monitored.Case.Version, "selfcheck-retest")
	retestBody["owner"] = "整改员-自检"
	retestBody["observedValue"] = 78
	retestBody["evidenceNote"] = "补植完成，复测影像 IMG-SELF-002"
	retested, err := selfcheckPost(client, baseURL, "/api/cases/"+caseID+"/remediations/"+monitored.Action.ID+"/retest", retestBody)
	if err != nil {
		return fmt.Errorf("整改复测: %w", err)
	}
	if retested.Action == nil || retested.Action.Status != domain.RemediationResolved {
		return fmt.Errorf("合格复测未关闭整改")
	}

	acceptBody := selfcheckMeta("复核员-自检", domain.RoleReviewer, retested.Case.Version, "selfcheck-accept")
	acceptBody["reviewer"] = "复核员-自检"
	acceptBody["decision"] = domain.DecisionAccepted
	acceptBody["reviewNote"] = "证据顺序完整，整改和复测闭环清晰，同意冻结放行"
	accepted, err := selfcheckPost(client, baseURL, "/api/cases/"+caseID+"/acceptance", acceptBody)
	if err != nil {
		return fmt.Errorf("独立验收: %w", err)
	}
	if accepted.Certificate == nil || accepted.Case.Status != domain.CaseAcceptanceFrozen {
		return fmt.Errorf("验收未冻结或未签发凭据")
	}
	code := accepted.Certificate.CredentialCode
	certificateJSON, err := selfcheckGet(client, baseURL+"/api/certificates/"+code)
	if err != nil || !strings.Contains(string(certificateJSON), code) {
		return fmt.Errorf("凭据 JSON 验证失败: %w", err)
	}
	certificatePage, err := selfcheckGet(client, baseURL+"/certificates/"+code)
	if err != nil || !strings.Contains(string(certificatePage), "海岸带修复放行凭据") {
		return fmt.Errorf("凭据页面验证失败: %w", err)
	}
	if _, err := buildHandler(dataDir, logger); err != nil {
		return fmt.Errorf("账本重启恢复: %w", err)
	}
	return nil
}
