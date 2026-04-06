package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OdooReportTool publishes a self-contained HTML report to the Odoo server
// and returns a shareable URL. The LLM calls this tool whenever it wants to
// present a visual report instead of (or in addition to) plain-text output.
type OdooReportTool struct {
	client   *http.Client
	channel  string
	chatID   string
	senderID string
	metadata map[string]string
}

func NewOdooReportTool() *OdooReportTool {
	return &OdooReportTool{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// --- Tool interface ---

func (t *OdooReportTool) Name() string {
	return "create_visual_report"
}

func (t *OdooReportTool) Description() string {
	return "Publish a self-contained HTML report to the Odoo server and get back a URL. " +
		"Use this whenever you want to present data visually (tables, charts, summaries) " +
		"instead of plain Markdown. Generate complete, standalone HTML with embedded CSS " +
		"and optional CDN JS (e.g. Chart.js). Return the URL to the user so they can open " +
		"the report in a new browser tab."
}

func (t *OdooReportTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Short descriptive title for the report (e.g. 'Sales Report – March 2026')",
			},
			"html_content": map[string]any{
				"type": "string",
				"description": "Complete, self-contained HTML document. Must include <!DOCTYPE html>, " +
					"<html>, <head> (with <meta charset> and embedded <style>), and <body>. " +
					"CDN scripts (Chart.js, etc.) are allowed. No external stylesheets that " +
					"require authentication.",
			},
		},
		"required": []string{"title", "html_content"},
	}
}

// --- MessageContextualTool interface ---

func (t *OdooReportTool) SetMessageContext(channel, chatID, senderID string, metadata map[string]string) {
	t.channel = channel
	t.chatID = chatID
	t.senderID = senderID
	t.metadata = metadata
}

// --- Execute ---

func (t *OdooReportTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return ErrorResult("title is required")
	}

	htmlContent, ok := args["html_content"].(string)
	if !ok || strings.TrimSpace(htmlContent) == "" {
		return ErrorResult("html_content is required")
	}

	odooURL := os.Getenv("ODOO_URL")
	if odooURL == "" {
		return ErrorResult("ODOO_URL env var is not set; cannot publish report to Odoo")
	}

	// Build the payload – pull res_id / model from message metadata when available.
	payload := map[string]any{
		"title":        title,
		"html_content": htmlContent,
	}
	if t.metadata != nil {
		if resID, ok := t.metadata["res_id"]; ok {
			payload["res_id"] = resID
		}
		if model, ok := t.metadata["model"]; ok {
			payload["model"] = model
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal payload: %v", err))
	}

	endpoint := fmt.Sprintf("%s/odooclaw/save_report", strings.TrimSuffix(odooURL, "/"))
	if odooDB := os.Getenv("ODOO_DB"); odooDB != "" {
		endpoint = fmt.Sprintf("%s?db=%s", endpoint, odooDB)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request to Odoo failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read Odoo response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return ErrorResult(fmt.Sprintf("Odoo returned status %d: %s", resp.StatusCode, string(body)))
	}

	var result struct {
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse Odoo response: %v", err))
	}
	if result.Error != "" {
		return ErrorResult(fmt.Sprintf("Odoo error: %s", result.Error))
	}

	// Build the absolute URL so the LLM can include it in its reply.
	baseURL := strings.TrimSuffix(odooURL, "/")
	absoluteURL := fmt.Sprintf("%s%s", baseURL, result.URL)

	forLLM := fmt.Sprintf(
		`Report published successfully.
Title: %s
URL:   %s

Include this link in your reply so the user can open the report:
[%s](%s)`,
		title, absoluteURL, title, absoluteURL,
	)

	return &ToolResult{
		ForLLM:  forLLM,
		Silent:  true, // The LLM will mention the link in its own reply
		IsError: false,
	}
}
