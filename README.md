# 海岸带修复证据验收台

海岸带修复证据验收台是面向生态修复现场监测员、整改责任人和独立验收复核员的本地浏览器工作台。它把基线锁定、现场监测、偏差整改、复测和独立验收串成一个不可跳步的案件流程，并在验收冻结后签发可校验的放行凭据。

系统只使用 Go 标准库和原生 HTML、CSS、JavaScript。业务事件以 JSON Lines 写入本地账本，每条事件包含连续序号和 SHA-256 前序摘要；服务启动时会校验整条链、重放案件投影并原子更新快照。

## 构建

    go build .

## 运行

默认只监听高位回环地址 127.0.0.1:19081，数据写入 data/：

    go run .

可通过参数指定监听地址和数据目录：

    go run . -addr=127.0.0.1:19120 -data=./local-data

也可设置 PORT 为端口号，服务会监听 127.0.0.1:PORT。启动后访问 http://127.0.0.1:19081/。

## 测试与自检

运行全部回归测试：

    go test ./...

运行会自行结束的 HTTP 全链路自检：

    go run . -selfcheck

自检会在临时账本上真实启动 HTTP 服务，依次完成案件建档、偏差监测、整改复测、独立验收、凭据读取和重启恢复，完成后关闭监听并删除临时数据。

## 主要接口

- POST /api/cases：建立案件并锁定基线。
- GET /api/cases 与 GET /api/cases/{id}：读取案件列表、完整投影和事件时间线。
- POST /api/cases/{id}/monitoring：提交监测证据并自动计算风险、按需创建整改。
- POST /api/cases/{id}/remediations/{actionId}/retest：提交复测并关闭或保留整改。
- POST /api/cases/{id}/acceptance：提交独立复核；通过时冻结案件并签发凭据。
- GET /api/certificates/{credentialCode}：读取凭据 JSON。
- GET /certificates/{credentialCode}：查看公开凭据页面。
- GET /healthz：检查服务及事件账本摘要。

所有写入命令都要求 actor、role、expectedVersion 和 idempotencyKey。expectedVersion 用于乐观并发控制；相同 idempotencyKey 的同一命令可安全重试。角色取值为 monitor、remediator 和 reviewer。

## 本地数据

- data/events.jsonl：只追加的哈希事件账本。
- data/snapshot.json：可由账本重新构建的查询投影快照。

不要手工修改事件账本。摘要或前序链不一致时，服务会拒绝启动，避免使用无法追溯的数据。
