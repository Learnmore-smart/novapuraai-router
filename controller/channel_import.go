package controller

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ChannelImportRow is one credential row for bulk import (CSV/JSON).
// Never log Key values.
type ChannelImportRow struct {
	Name         string `json:"name"`
	Type         int    `json:"type"`
	Key          string `json:"key"`
	BaseURL      string `json:"base_url"`
	Models       string `json:"models"`
	Group        string `json:"group"`
	Weight       *uint  `json:"weight"`
	Priority     *int64 `json:"priority"`
	ModelMapping string `json:"model_mapping"`
	Remark       string `json:"remark"`
}

type ChannelImportRequest struct {
	// Format: "json" (default) or "csv" when body is text/csv / multipart
	Format string             `json:"format"`
	Rows   []ChannelImportRow `json:"rows"`
	// CSV raw body when using JSON wrapper
	CSV string `json:"csv"`
	// Skip rows whose key already exists on any channel (substring match per key line)
	Dedupe bool `json:"dedupe"`
	// Default type if row.type is 0
	DefaultType int `json:"default_type"`
}

type ChannelImportResult struct {
	Success   int      `json:"success"`
	Duplicate int      `json:"duplicate"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// ImportChannels imports channels row-by-row with partial success (MVP §11.3).
// POST /api/channel/import
// Accepts:
//   - application/json: { "rows": [...], "dedupe": true }
//   - application/json: { "csv": "name,type,key,...\n...", "dedupe": true }
//   - text/csv or multipart file field "file"
func ImportChannels(c *gin.Context) {
	ct := c.ContentType()
	var rows []ChannelImportRow
	dedupe := true
	defaultType := 1

	if strings.Contains(ct, "multipart/form-data") {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "file required"})
			return
		}
		f, err := file.Open()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		defer f.Close()
		rows, err = parseChannelCSV(f)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		if v := c.PostForm("dedupe"); v == "false" || v == "0" {
			dedupe = false
		}
	} else if strings.Contains(ct, "text/csv") {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		parsed, err := parseChannelCSV(strings.NewReader(string(body)))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		rows = parsed
	} else {
		var req ChannelImportRequest
		if err := common.DecodeJson(c.Request.Body, &req); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json body"})
			return
		}
		if req.CSV == "" && len(req.Rows) == 0 {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "rows or csv required"})
			return
		}
		// Default dedupe=true; set "dedupe": false in JSON to disable
		dedupe = true
		if c.Query("dedupe") == "false" {
			dedupe = false
		}
		// If client sent explicit false via form we can't know; for JSON use DefaultType only.
		// Optional: when Format == "no-dedupe"
		if req.Format == "no-dedupe" {
			dedupe = false
		}
		if req.DefaultType > 0 {
			defaultType = req.DefaultType
		}
		if req.CSV != "" {
			parsed, err := parseChannelCSV(strings.NewReader(req.CSV))
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
				return
			}
			rows = parsed
		} else {
			rows = req.Rows
		}
	}

	// existing keys for dedupe (expensive but ok for MVP admin import)
	existingKeys := map[string]struct{}{}
	if dedupe {
		var channels []model.Channel
		if err := model.DB.Select("id", "key").Find(&channels).Error; err == nil {
			for _, ch := range channels {
				plain, _ := common.DecryptChannelKey(ch.Key)
				for _, line := range strings.Split(plain, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						existingKeys[line] = struct{}{}
					}
				}
			}
		}
	}

	result := ChannelImportResult{Errors: make([]string, 0)}
	toInsert := make([]model.Channel, 0)

	for i, row := range rows {
		rowNum := i + 1
		key := strings.TrimSpace(row.Key)
		if key == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: empty key", rowNum))
			continue
		}
		if dedupe {
			if _, ok := existingKeys[key]; ok {
				result.Duplicate++
				continue
			}
		}
		chType := row.Type
		if chType == 0 {
			chType = defaultType
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			prefix := key
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
			name = fmt.Sprintf("import-%s", prefix)
		}
		group := strings.TrimSpace(row.Group)
		if group == "" {
			group = "default"
		}
		models := strings.TrimSpace(row.Models)
		if models == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: models required", rowNum))
			continue
		}
		weight := uint(0)
		if row.Weight != nil {
			weight = *row.Weight
		}
		priority := int64(0)
		if row.Priority != nil {
			priority = *row.Priority
		}
		baseURL := strings.TrimSpace(row.BaseURL)
		remark := strings.TrimSpace(row.Remark)
		ch := model.Channel{
			Type:        chType,
			Key:         key,
			Name:        name,
			Models:      models,
			Group:       group,
			Weight:      &weight,
			Priority:    &priority,
			CreatedTime: common.GetTimestamp(),
			Status:      common.ChannelStatusEnabled,
		}
		if baseURL != "" {
			ch.BaseURL = &baseURL
		}
		if row.ModelMapping != "" {
			m := row.ModelMapping
			ch.ModelMapping = &m
		}
		if remark != "" {
			ch.Remark = &remark
		}
		toInsert = append(toInsert, ch)
		if dedupe {
			existingKeys[key] = struct{}{}
		}
	}

	// Insert one-by-one so partial failures don't roll back successes
	for _, ch := range toInsert {
		local := ch
		if err := local.Insert(); err != nil {
			result.Failed++
			// Do not include key in error text
			result.Errors = append(result.Errors, fmt.Sprintf("insert %s: %v", local.Name, err))
			continue
		}
		result.Success++
	}

	if result.Success > 0 {
		service.ResetProxyClientCache()
		model.InitChannelCache()
		recordManageAudit(c, "channel.import", map[string]interface{}{
			"success":   result.Success,
			"duplicate": result.Duplicate,
			"failed":    result.Failed,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("imported %d, duplicate %d, failed %d", result.Success, result.Duplicate, result.Failed),
		"data":    result,
	})
}

func parseChannelCSV(r io.Reader) ([]ChannelImportRow, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv parse: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty csv")
	}
	// header map
	header := map[string]int{}
	for i, h := range records[0] {
		header[strings.ToLower(strings.TrimSpace(h))] = i
	}
	// require key column
	if _, ok := header["key"]; !ok {
		// no header — treat as: key only or name,type,key,models,base_url,group
		if len(records[0]) >= 1 && !strings.EqualFold(records[0][0], "key") && !strings.EqualFold(records[0][0], "name") {
			// bare keys one per line
			rows := make([]ChannelImportRow, 0, len(records))
			for _, rec := range records {
				if len(rec) == 0 || strings.TrimSpace(rec[0]) == "" {
					continue
				}
				rows = append(rows, ChannelImportRow{Key: strings.TrimSpace(rec[0]), Models: "gpt-4o-mini"})
			}
			return rows, nil
		}
		return nil, fmt.Errorf("csv must have a key column header")
	}

	get := func(rec []string, name string) string {
		idx, ok := header[name]
		if !ok || idx >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[idx])
	}

	rows := make([]ChannelImportRow, 0, len(records)-1)
	for _, rec := range records[1:] {
		key := get(rec, "key")
		if key == "" {
			continue
		}
		row := ChannelImportRow{
			Name:         get(rec, "name"),
			Key:          key,
			BaseURL:      firstNonEmpty(get(rec, "base_url"), get(rec, "baseurl")),
			Models:       get(rec, "models"),
			Group:        get(rec, "group"),
			ModelMapping: get(rec, "model_mapping"),
			Remark:       get(rec, "remark"),
		}
		if t := get(rec, "type"); t != "" {
			var ti int
			_, _ = fmt.Sscanf(t, "%d", &ti)
			row.Type = ti
		}
		if w := get(rec, "weight"); w != "" {
			var wi uint
			_, _ = fmt.Sscanf(w, "%d", &wi)
			row.Weight = &wi
		}
		if p := get(rec, "priority"); p != "" {
			var pi int64
			_, _ = fmt.Sscanf(p, "%d", &pi)
			row.Priority = &pi
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
