package server

import (
	"fmt"
	"net/http"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

// ==================== automations ====================

func (s *Server) listAutomations(w http.ResponseWriter, r *http.Request) {
	templatesOnly := r.URL.Query().Get("templates") == "1" || r.URL.Query().Get("template") == "1"
	var items []*fsstore.AutomationConfig
	var err error
	if templatesOnly {
		items, err = s.engine.ListAutomationTemplates()
	} else {
		items, err = s.engine.ListAutomations()
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*fsstore.AutomationConfig{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func isOfficialAutomation(cfg *fsstore.AutomationConfig) bool {
	return cfg != nil && cfg.Source == fsstore.SourceOfficial
}

func isContextAutomation(cfg *fsstore.AutomationConfig) bool {
	return cfg != nil && (cfg.ID == "auto_context_refresh" || cfg.HasTag("context"))
}

func (s *Server) rejectOfficialAutomation(w http.ResponseWriter, id string, action string) bool {
	cfg, err := s.engine.GetAutomation(id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "automation 不存在: "+err.Error())
		return true
	}
	if isOfficialAutomation(cfg) {
		writeErr(w, http.StatusForbidden, "官方自动化不能"+action+"；如需自定义请先创建副本")
		return true
	}
	return false
}

// createAutomation POST /api/automations — 新建自动化。
func (s *Server) createAutomation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Description      string `json:"description"`
		Query            string `json:"query"`
		QuestType        string `json:"quest_type"`
		Intensity        string `json:"intensity"`
		WorkingDir       string `json:"working_dir"`
		WorkspaceMode    string `json:"workspace_mode"`
		WarriorID        string `json:"warrior_id"`
		MageID           string `json:"mage_id"`
		Trigger          string `json:"trigger"`
		Cron             string `json:"cron"`
		Priority         int    `json:"priority"`
		AutoStart        *bool  `json:"auto_start"`
		TriageMode       string `json:"triage_mode"`
		AutoApply        *bool  `json:"auto_apply"`
		AllowHOTL        *bool  `json:"allow_hotl_auto_pass"`
		AllowL2          *bool  `json:"allow_l2"`
		WithDesignPhase  *bool  `json:"with_design_phase"`
		AutoSpawnExecute *bool  `json:"auto_spawn_execute"`
		TrustTier        string `json:"trust_tier"`
		TrustTierLocked  *bool  `json:"trust_tier_locked"`
		Enabled          *bool  `json:"enabled"`
	}
	if err := readBodyLimited(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	if body.Name == "" {
		writeErr(w, http.StatusBadRequest, "name 不能为空")
		return
	}
	if body.Query == "" {
		writeErr(w, http.StatusBadRequest, "query 不能为空")
		return
	}
	id := body.ID
	if id == "" {
		id = "auto_" + fsstore.NewIDShort()
	}
	trigger, err := parseAutomationTrigger(body.Trigger)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	workspaceMode, err := parseWorkspaceMode(body.WorkspaceMode)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.WorkspaceMode == "" {
		workspaceMode = model.WorkspaceAuto
	}
	questType, err := parseAutomationQuestType(body.QuestType)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	intensity, err := parseQuestIntensity(body.Intensity)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	cron := fsstore.NormalizeCronPreset(body.Cron)
	if trigger == fsstore.TriggerSchedule {
		if cron == "" {
			writeErr(w, http.StatusBadRequest, "schedule 自动化必须填写 cron")
			return
		}
		if _, err := orchestrator.ParseCron(cron); err != nil {
			writeErr(w, http.StatusBadRequest, "cron 表达式无效: "+err.Error())
			return
		}
	}
	autoStart := false
	if body.AutoStart != nil {
		autoStart = *body.AutoStart
	}
	autoApply := false
	if body.AutoApply != nil {
		autoApply = *body.AutoApply
	}
	allowL2 := false
	if body.AllowL2 != nil {
		allowL2 = *body.AllowL2
	}
	if body.AllowHOTL != nil {
	}
	if autoApply && allowL2 {
		writeErr(w, http.StatusBadRequest, "auto_apply 与 allow_l2 不能同时开启")
		return
	}
	trustTier := body.TrustTier
	if trustTier != "" && !fsstore.IsValidTrustTier(trustTier) {
		writeErr(w, http.StatusBadRequest, "trust_tier 必须是 tier_0 | tier_1 | tier_2 | tier_3")
		return
	}
	if err := s.ensureRunnableAdventurer(body.WarriorID, model.ClassWarrior); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.ensureRunnableAdventurer(body.MageID, model.ClassMage); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	cfg := &fsstore.AutomationConfig{
		ID:               id,
		Name:             body.Name,
		Description:      body.Description,
		Query:            body.Query,
		QuestType:        string(questType),
		Intensity:        string(intensity),
		WorkingDir:       body.WorkingDir,
		WorkspaceMode:    string(workspaceMode),
		WarriorID:        body.WarriorID,
		MageID:           body.MageID,
		Trigger:          trigger,
		Cron:             cron,
		Priority:         body.Priority,
		AutoStart:        autoStart,
		TriageMode:       body.TriageMode,
		AutoApply:        autoApply,
		AllowL2:          allowL2,
		WithDesignPhase:  body.WithDesignPhase != nil && *body.WithDesignPhase,
		AutoSpawnExecute: body.AutoSpawnExecute != nil && *body.AutoSpawnExecute,
		TrustTier:        trustTier,
		TrustTierLocked:  body.TrustTierLocked != nil && *body.TrustTierLocked,
		Enabled:          enabled,
	}
	if err := s.engine.SaveAutomation(cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	item, err := s.engine.GetAutomation(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "item": item})
}

func (s *Server) runAutomation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing automation id")
		return
	}
	qid, err := s.engine.RunAutomation(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":  true,
		"qid": qid,
	})
}

func (s *Server) archiveAutomationNoFinding(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing automation id")
		return
	}
	var req struct {
		Reason  string `json:"reason"`
		Summary string `json:"summary"`
	}
	if bodyErr := readBodyLimited(r, &req); bodyErr != nil {
		req.Reason = "no finding"
	}
	item := &fsstore.AutomationDiscoveryArchive{
		AutomationID: id,
		Outcome:      "no_finding",
		Reason:       req.Reason,
		Summary:      req.Summary,
	}
	if err := s.root.ArchiveAutomationDiscovery(item); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

func (s *Server) listAutomationDiscoveryArchive(w http.ResponseWriter, r *http.Request) {
	items, err := s.root.LoadAutomationDiscoveryArchive()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (s *Server) setAutomationEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing automation id")
		return
	}
	item, err := s.engine.SetAutomationEnabled(id, enabled)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"id":      id,
		"enabled": enabled,
		"item":    item,
	})
}

func (s *Server) enableAutomation(w http.ResponseWriter, r *http.Request) {
	s.setAutomationEnabled(w, r, true)
}

func (s *Server) disableAutomation(w http.ResponseWriter, r *http.Request) {
	s.setAutomationEnabled(w, r, false)
}

func parseAutomationTrigger(raw string) (fsstore.AutomationTrigger, error) {
	switch fsstore.AutomationTrigger(raw) {
	case "":
		return fsstore.TriggerManual, nil
	case fsstore.TriggerManual:
		return fsstore.TriggerManual, nil
	case fsstore.TriggerSchedule:
		return fsstore.TriggerSchedule, nil
	case fsstore.TriggerEvent:
		return fsstore.TriggerEvent, nil
	default:
		return "", fmt.Errorf("trigger 必须是 manual | schedule | event")
	}
}

func parseAutomationQuestType(raw string) (model.QuestType, error) {
	switch model.QuestType(raw) {
	case "":
		return model.QuestTypeExecute, nil
	case model.QuestTypeExecute:
		return model.QuestTypeExecute, nil
	case model.QuestTypeDesign:
		return model.QuestTypeDesign, nil
	default:
		return "", fmt.Errorf("quest_type 必须是 execute 或 design")
	}
}

type updateAutomationReq struct {
	Name             *string   `json:"name"`
	Description      *string   `json:"description"`
	Query            *string   `json:"query"`
	QuestType        *string   `json:"quest_type"`
	Intensity        *string   `json:"intensity"`
	WorkingDir       *string   `json:"working_dir"`
	WorkspaceMode    *string   `json:"workspace_mode"`
	WarriorID        *string   `json:"warrior_id"`
	MageID           *string   `json:"mage_id"`
	Trigger          *string   `json:"trigger"`
	Cron             *string   `json:"cron"`
	Priority         *int      `json:"priority"`
	AutoStart        *bool     `json:"auto_start"`
	TriageMode       *string   `json:"triage_mode"`
	AutoApply        *bool     `json:"auto_apply"`
	AllowHOTL        *bool     `json:"allow_hotl_auto_pass"`
	AllowL2          *bool     `json:"allow_l2"`
	WithDesignPhase  *bool     `json:"with_design_phase"`
	AutoSpawnExecute *bool     `json:"auto_spawn_execute"`
	TrustTier        *string   `json:"trust_tier"`
	TrustTierLocked  *bool     `json:"trust_tier_locked"`
	Connectors       *[]string `json:"connectors"`
}

func (s *Server) updateAutomation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing automation id")
		return
	}
	if s.rejectOfficialAutomation(w, id, "修改") {
		return
	}
	var req updateAutomationReq
	if err := readBodyLimited(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "body 解析失败: "+err.Error())
		return
	}
	var trigger *fsstore.AutomationTrigger
	if req.Trigger != nil {
		v, err := parseAutomationTrigger(*req.Trigger)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		trigger = &v
	}
	var cron *string
	if req.Cron != nil {
		v := fsstore.NormalizeCronPreset(*req.Cron)
		cron = &v
	}
	existing, existingErr := s.engine.GetAutomation(id)
	if existingErr != nil {
		writeErr(w, http.StatusBadRequest, "automation 不存在: "+existingErr.Error())
		return
	}
	triggerForValidation := existing.Trigger
	if trigger != nil {
		triggerForValidation = *trigger
	}
	cronForValidation := existing.Cron
	if cron != nil {
		cronForValidation = *cron
	}
	if triggerForValidation == fsstore.TriggerSchedule {
		if cronForValidation == "" {
			writeErr(w, http.StatusBadRequest, "schedule 自动化必须填写 cron")
			return
		}
		if _, err := orchestrator.ParseCron(cronForValidation); err != nil {
			writeErr(w, http.StatusBadRequest, "cron 表达式无效: "+err.Error())
			return
		}
	}
	var workspaceMode *model.WorkspaceMode
	if req.WorkspaceMode != nil {
		v, err := parseWorkspaceMode(*req.WorkspaceMode)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		workspaceMode = &v
	}
	if req.WarriorID != nil {
		if err := s.ensureRunnableAdventurer(*req.WarriorID, model.ClassWarrior); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.MageID != nil {
		if err := s.ensureRunnableAdventurer(*req.MageID, model.ClassMage); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.TrustTier != nil {
		if !fsstore.IsValidTrustTier(*req.TrustTier) {
			writeErr(w, http.StatusBadRequest, "trust_tier 必须是 tier_0 | tier_1 | tier_2 | tier_3")
			return
		}
	}
	item, err := s.engine.UpdateAutomation(id, orchestrator.AutomationUpdate{
		Name:             req.Name,
		Description:      req.Description,
		Query:            req.Query,
		QuestType:        req.QuestType,
		Intensity:        req.Intensity,
		WorkingDir:       req.WorkingDir,
		WorkspaceMode:    workspaceMode,
		WarriorID:        req.WarriorID,
		MageID:           req.MageID,
		Trigger:          trigger,
		Cron:             cron,
		Priority:         req.Priority,
		AutoStart:        req.AutoStart,
		TriageMode:       req.TriageMode,
		AutoApply:        req.AutoApply,
		AllowHOTL:        req.AllowHOTL,
		AllowL2:          req.AllowL2,
		WithDesignPhase:  req.WithDesignPhase,
		AutoSpawnExecute: req.AutoSpawnExecute,
		TrustTier:        req.TrustTier,
		TrustTierLocked:  req.TrustTierLocked,
		Connectors:       req.Connectors,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

func (s *Server) deleteAutomation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "missing automation id")
		return
	}
	if s.rejectOfficialAutomation(w, id, "删除") {
		return
	}
	if err := s.engine.DeleteAutomation(id); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}
