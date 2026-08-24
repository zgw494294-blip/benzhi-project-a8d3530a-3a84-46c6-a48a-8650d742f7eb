package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type selfcheckEnvelope struct {
	Data  json.RawMessage
	Error *struct {
		Code    string
		Message string
	}
}

func selfcheckPost(client *http.Client, baseURL, path string, body any) (application.CommandResult, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return application.CommandResult{}, err
	}
	request, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return application.CommandResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return application.CommandResult{}, err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return application.CommandResult{}, err
	}
	var envelope selfcheckEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return application.CommandResult{}, fmt.Errorf("解析 %s 响应: %w", path, err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if envelope.Error != nil {
			return application.CommandResult{}, fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return application.CommandResult{}, fmt.Errorf("%s 返回 HTTP %d", path, response.StatusCode)
	}
	var result application.CommandResult
	if err := json.Unmarshal(envelope.Data, &result); err != nil {
		return application.CommandResult{}, err
	}
	return result, nil
}

func selfcheckGet(client *http.Client, url string) ([]byte, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s 返回 HTTP %d", url, response.StatusCode)
	}
	return body, nil
}

func selfcheckMeta(actor string, role domain.Role, version int64, key string) map[string]any {
	return map[string]any{
		"actor": actor, "role": role, "expectedVersion": version, "idempotencyKey": key,
	}
}
