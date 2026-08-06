package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/orchestrator"
)

// ==================== adventurers / executors ====================

func (s *Server) listAdventurers(w http.ResponseWriter, r *http.Request) {
	as, err := s.engine.ListAdventurers()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if as == nil {
		as = []*fsstore.AdventurerFile{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": as})
}

func (s *Server) getAdventurer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := s.engine.GetAdventurer(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": a})
}

func (s *Server) activateAdventurer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Name         string   `json:"name"`
		Agent        string   `json:"agent"`
		Model        *string  `json:"model"`
		Description  *string  `json:"description"`
		CustomPrompt *string  `json:"custom_prompt"`
		Tools        []string `json:"tools"`
	}
	if r.ContentLength > 0 {
		if err := readBodyLimited(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
			return
		}
	}

	if body.Model != nil {
		model := strings.TrimSpace(*body.Model)
		body.Model = &model
	}
	updates := &fsstore.AdventurerUpdate{
		Name:         body.Name,
		Agent:        body.Agent,
		Model:        body.Model,
		Description:  body.Description,
		CustomPrompt: body.CustomPrompt,
		Tools:        body.Tools,
	}
	if body.Agent != "" {
		if err := s.ensureSelectableAgent(body.Agent); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	a, err := s.engine.ActivateAdventurer(id, updates)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": a})
}

// createAdventurer 新建冒险者：POST /api/adventurers
// Body 字段与 AdventurerFile 对齐，class / name 必填。
func (s *Server) createAdventurer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID           string                 `json:"id"`
		Name         string                 `json:"name"`
		Class        model.AdventurerClass  `json:"class"`
		Agent        string                 `json:"agent"`
		Model        string                 `json:"model"`
		Description  string                 `json:"description"`
		CustomPrompt string                 `json:"custom_prompt"`
		Tools        []string               `json:"tools"`
		Level        int                    `json:"level"`
		Status       model.AdventurerStatus `json:"status"`
	}
	if r.ContentLength > 0 {
		if err := readBodyLimited(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
			return
		}
	}
	if strings.TrimSpace(body.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name 不能为空")
		return
	}
	if body.Class != model.ClassWarrior && body.Class != model.ClassMage {
		writeErr(w, http.StatusBadRequest, "class 必须是 warrior 或 mage")
		return
	}
	if body.Level <= 0 {
		body.Level = 1
	}
	if body.Status == model.AdventurerActive {
		if err := s.ensureSelectableAgent(body.Agent); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	a := &fsstore.AdventurerFile{
		ID:           body.ID,
		Name:         body.Name,
		Class:        body.Class,
		Agent:        body.Agent,
		Model:        strings.TrimSpace(body.Model),
		Description:  body.Description,
		CustomPrompt: body.CustomPrompt,
		Tools:        body.Tools,
		Level:        body.Level,
		Status:       body.Status,
	}
	saved, err := s.engine.SaveAdventurer(a)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "item": saved})
}

// updateAdventurer 全量/部分更新：POST/PATCH /api/adventurers/{id}
// 任何非零/非空字段都会被合并写回。
func (s *Server) updateAdventurer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := s.engine.GetAdventurer(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		Name         *string                 `json:"name"`
		Class        *model.AdventurerClass  `json:"class"`
		Agent        *string                 `json:"agent"`
		Model        *string                 `json:"model"`
		Description  *string                 `json:"description"`
		CustomPrompt *string                 `json:"custom_prompt"`
		Tools        *[]string               `json:"tools"`
		Level        *int                    `json:"level"`
		Status       *model.AdventurerStatus `json:"status"`
		Exp          *int64                  `json:"exp"`
		WinCount     *int                    `json:"win_count"`
		LoseCount    *int                    `json:"lose_count"`
	}
	if r.ContentLength > 0 {
		if err := readBodyLimited(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
			return
		}
	}
	if body.Name != nil {
		existing.Name = *body.Name
	}
	if body.Class != nil {
		existing.Class = *body.Class
	}
	if body.Agent != nil {
		existing.Agent = *body.Agent
	}
	if body.Model != nil {
		existing.Model = strings.TrimSpace(*body.Model)
	}
	if body.Description != nil {
		existing.Description = *body.Description
	}
	if body.CustomPrompt != nil {
		existing.CustomPrompt = *body.CustomPrompt
	}
	if body.Tools != nil {
		existing.Tools = *body.Tools
	}
	if body.Level != nil {
		existing.Level = *body.Level
	}
	if body.Status != nil {
		existing.Status = *body.Status
	}
	if body.Exp != nil {
		existing.Exp = *body.Exp
	}
	if body.WinCount != nil {
		existing.WinCount = *body.WinCount
	}
	if body.LoseCount != nil {
		existing.LoseCount = *body.LoseCount
	}

	if existing.Status == model.AdventurerPendingSetup && existing.Agent != "" {
		existing.Status = model.AdventurerActive
	}
	if existing.Status == model.AdventurerActive {
		if err := s.ensureSelectableAgent(existing.Agent); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	saved, err := s.engine.SaveAdventurer(existing)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": saved})
}

// retireAdventurer POST /api/adventurers/{id}/retire — 软删除
func (s *Server) retireAdventurer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.engine.RetireAdventurer(id); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) ensureSelectableAgent(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("agent 不能为空")
	}
	if s.root == nil {
		return nil
	}
	agent, err := s.root.GetAgent(name)
	if err != nil {
		return fmt.Errorf("agent %s 不存在", name)
	}
	if !agent.Enabled {
		return fmt.Errorf("agent %s 未启用", name)
	}
	if agent.Type == model.AgentTypeMock || name == "mock" {
		return fmt.Errorf("agent %s 是 Mock，仅用于测试，不能绑定冒险者", name)
	}
	if s.engine != nil && !s.engine.HasExecutor(name) {
		if _, err := s.engine.RegisterAgentExecutor(agent); err != nil {
			return fmt.Errorf("agent %s 无法注册执行器: %w", name, err)
		}
	}
	return nil
}

func (s *Server) ensureRunnableAdventurer(id string, class model.AdventurerClass) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if s.engine == nil {
		return nil
	}
	adv, err := s.engine.GetAdventurer(id)
	if err != nil {
		return err
	}
	if adv.Class != class {
		return fmt.Errorf("冒险者 %s 职业是 %s，不是 %s", id, adv.Class, class)
	}
	if adv.Status != model.AdventurerActive {
		return fmt.Errorf("冒险者 %s 状态是 %s，未激活", id, adv.Status)
	}
	if err := s.ensureSelectableAgent(adv.Agent); err != nil {
		return fmt.Errorf("冒险者 %s 绑定的 %w", id, err)
	}
	return nil
}

// executorInfo 是给前端看的 executors 摘要
type executorInfo struct {
	ID             string            `json:"id"`
	Agent          string            `json:"agent"`
	Name           string            `json:"name"`
	Type           string            `json:"type"` // acp | cli | mock
	CapabilityTier string            `json:"capability_tier,omitempty"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	EnvFiles       []string          `json:"env_files,omitempty"`
	DefaultModel   string            `json:"default_model"`
	Enabled        bool              `json:"enabled"`
	Official       bool              `json:"official,omitempty"`
	Capabilities   []string          `json:"capabilities"`
}

func (s *Server) listExecutors(w http.ResponseWriter, r *http.Request) {
	items := []executorInfo{}
	runtimeExecs := map[string]executor.Executor{}
	if s.engine != nil {
		runtimeExecs = s.engine.Executors()
	}
	if s.root != nil {
		if ps, err := s.root.ListAgents(); err == nil {
			for _, p := range ps {
				items = append(items, executorInfoForAgent(p, runtimeExecs[p.Name]))
			}
		}
	}
	// 兜底：永远有 mock
	haveMock := false
	for _, it := range items {
		if it.Type == "mock" {
			haveMock = true
			break
		}
	}
	if !haveMock {
		items = append(items, executorInfo{
			ID:             "exe_mock",
			Agent:          "mock",
			Name:           "Mock Executor",
			Type:           "mock",
			DefaultModel:   "mock-fast",
			Enabled:        true,
			CapabilityTier: string(executor.CapabilityTierTest),
			Capabilities:   []string{"tool_use", "streaming", "context_export"},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func executorInfoForAgent(p *fsstore.AgentConfig, ex executor.Executor) executorInfo {
	id := "exe_" + p.Name
	name := p.Name + " executor"
	typ := string(p.Type)
	defModel := p.DefaultModel
	caps := defaultCapabilities(p.Type)
	tier := defaultCapabilityTier(p)
	if ex != nil {
		if ex.ID() != "" {
			id = ex.ID()
		}
		if ex.Name() != "" {
			name = ex.Name()
		}
		typ = string(ex.Type())
		if ex.DefaultModel() != "" && defModel == "" {
			defModel = ex.DefaultModel()
		}
		caps = capabilitiesToStrings(ex.Capabilities())
		tier = string(ex.CapabilityTier())
	}
	return executorInfo{
		ID:             id,
		Agent:          p.Name,
		Name:           name,
		Type:           typ,
		CapabilityTier: tier,
		Command:        p.Command,
		Args:           append([]string(nil), p.Args...),
		Env:            p.Env,
		EnvFiles:       append([]string(nil), p.EnvFiles...),
		DefaultModel:   defModel,
		Enabled:        p.Enabled,
		Official:       p.Official || fsstore.IsBuiltinAgentName(p.Name),
		Capabilities:   caps,
	}
}

func defaultCapabilityTier(p *fsstore.AgentConfig) string {
	if p == nil {
		return ""
	}
	switch p.Type {
	case model.AgentTypeACP:
		return string(executor.CapabilityTierB)
	case model.AgentTypeMock:
		return string(executor.CapabilityTierTest)
	case model.AgentTypeCLI:
		name := strings.TrimSuffix(filepath.Base(p.Command), filepath.Ext(p.Command))
		switch name {
		case "relay", "claude", "codex":
			return string(executor.CapabilityTierB)
		case "aiden":
			switch orchestrator.DetectAidenVariantForAgent(p.Name, p.Args) {
			case executor.AidenVariantXClaude, executor.AidenVariantXCodex:
				return string(executor.CapabilityTierB)
			default:
				return string(executor.CapabilityTierC)
			}
		default:
			return string(executor.CapabilityTierC)
		}
	default:
		return string(executor.CapabilityTierC)
	}
}

func settingsAgentInfo(p *fsstore.AgentConfig) executorInfo {
	item := executorInfoForAgent(p, nil)
	item.Name = p.Name
	return item
}

func defaultCapabilities(t model.AgentType) []string {
	caps := []string{"tool_use", "streaming"}
	switch t {
	case model.AgentTypeACP, model.AgentTypeCLI:
		caps = append(caps, "code_exec", "context_export")
	case model.AgentTypeMock:
		caps = append(caps, "context_export")
	}
	return caps
}

func capabilitiesToStrings(caps []executor.Capability) []string {
	out := make([]string, 0, len(caps))
	for _, cap := range caps {
		out = append(out, string(cap))
	}
	return out
}

// exeIDToAgent 把前端 exe_<name> 或 <name> 格式还原成 agent name
func exeIDToAgent(id string) string {
	return strings.TrimPrefix(id, "exe_")
}

func (s *Server) setExecutorEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	id := exeIDToAgent(r.PathValue("id"))
	if s.root == nil {
		writeErr(w, http.StatusInternalServerError, "root 未初始化")
		return
	}
	p, err := s.engine.SetAgentEnabled(id, enabled)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var ex executor.Executor
	if s.engine != nil {
		ex = s.engine.Executors()[p.Name]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"id":      "exe_" + p.Name,
		"enabled": p.Enabled,
		"item":    executorInfoForAgent(p, ex),
	})
}

func (s *Server) enableExecutor(w http.ResponseWriter, r *http.Request) {
	s.setExecutorEnabled(w, r, true)
}

func (s *Server) disableExecutor(w http.ResponseWriter, r *http.Request) {
	s.setExecutorEnabled(w, r, false)
}

func (s *Server) deleteExecutor(w http.ResponseWriter, r *http.Request) {
	id := exeIDToAgent(r.PathValue("id"))
	if s.root == nil {
		writeErr(w, http.StatusInternalServerError, "root 未初始化")
		return
	}
	if fsstore.IsBuiltinAgentName(id) {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("官方内置 Agent %s 不能删除；如不使用请禁用", id))
		return
	}
	if err := s.engine.DeleteAgentConfig(id); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": "exe_" + id, "agent": id})
}

// upsertExecutor POST /api/executors — 新建或全量更新一个 Agent
func (s *Server) upsertExecutor(w http.ResponseWriter, r *http.Request) {
	if s.root == nil {
		writeErr(w, http.StatusInternalServerError, "root 未初始化")
		return
	}
	var body struct {
		ID           string            `json:"id"`
		Name         string            `json:"name"`
		Type         model.AgentType   `json:"type"`
		DefaultModel string            `json:"default_model"`
		Enabled      *bool             `json:"enabled"`
		Command      string            `json:"command"`
		Args         []string          `json:"args"`
		Env          map[string]string `json:"env"`
		EnvFiles     []string          `json:"env_files"`
	}
	if err := readBodyLimited(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = exeIDToAgent(body.ID)
	}
	if name == "" {
		writeErr(w, http.StatusBadRequest, "name 不能为空")
		return
	}
	if body.Type == "" {
		body.Type = model.AgentTypeCLI
	}
	officialType, isOfficial := fsstore.BuiltinAgentType(name)
	if isOfficial && body.Type != officialType {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("官方内置 Agent %s 的类型固定为 %s，不能改为 %s", name, officialType, body.Type))
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	p := &fsstore.AgentConfig{
		Name:         name,
		Type:         body.Type,
		Command:      body.Command,
		Args:         body.Args,
		Env:          body.Env,
		EnvFiles:     body.EnvFiles,
		DefaultModel: body.DefaultModel,
		Enabled:      enabled,
		Official:     isOfficial,
	}
	if _, err := s.engine.UpsertAgentConfig(p); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	saved, err := s.root.GetAgent(name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"item": executorInfoForAgent(saved, nil),
	})
}

// getSettings GET /api/settings — 返回 GlobalConfig + agents
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	cfg := s.engine.Config()
	agents, _ := s.engine.ListAgentConfigs()
	items := []executorInfo{}
	if agents != nil {
		for _, agent := range agents {
			items = append(items, settingsAgentInfo(agent))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": cfg,
		"agents": items,
	})
}

func (s *Server) recommendMageCommands(w http.ResponseWriter, r *http.Request) {
	cfg := s.engine.Config()
	projectDir := strings.TrimSpace(r.URL.Query().Get("project_dir"))
	if projectDir == "" && cfg != nil {
		projectDir = cfg.DefaultWorkingDir
	}
	if projectDir == "" && s.engine != nil {
		projectDir = s.engine.WorkDir()
	}
	items, err := fsstore.RecommendMageCommands(projectDir, cfg.CommandAllowlist)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"project_dir": projectDir,
		"items":       items,
	})
}

// saveSettings PATCH/POST /api/settings — 局部更新全局配置
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) {
	var patch fsstore.GlobalConfigPatch
	if r.ContentLength > 0 {
		if err := readBodyLimited(r, &patch); err != nil {
			writeErr(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
			return
		}
	}
	cfg, err := s.engine.SaveConfig(&patch)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	agents, _ := s.engine.ListAgentConfigs()
	items := []executorInfo{}
	if agents != nil {
		for _, agent := range agents {
			items = append(items, settingsAgentInfo(agent))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": cfg,
		"agents": items,
	})
}
