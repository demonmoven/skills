package orchestrator

import (
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// ==================== 用户上下文（Engine 层包装） ====================
//
// 薄包装：直接透传 fsstore 层的能力，方便 ToolEngine 接口调用。

func (e *Engine) ListContextDims() ([]fsstore.ContextDimInfo, error) {
	return e.root.ListContextDims()
}

func (e *Engine) ReadContextDim(name string) (*fsstore.ContextDim, error) {
	return e.root.ReadContextDim(name)
}

func (e *Engine) ReadContextSummary() (string, error) {
	return e.root.ReadContextSummary()
}

// WriteContextDim 写入（创建或覆盖）一个上下文维度。
// 供 automation 刷新上下文时调用。
func (e *Engine) WriteContextDim(name, body string) (*fsstore.ContextDimInfo, error) {
	dim, err := e.root.WriteContextDim(name, body)
	if err != nil {
		return nil, err
	}
	e.publishContextUpdated("dim", map[string]any{
		"name":          dim.Name,
		"updated_at_ms": dim.UpdatedAtMs,
		"size_bytes":    dim.SizeBytes,
	})
	return dim, nil
}

// WriteContextSummary 写入总览摘要。
func (e *Engine) WriteContextSummary(body string) error {
	if err := e.root.WriteContextSummary(body); err != nil {
		return err
	}
	e.publishContextUpdated("summary", nil)
	return nil
}

// GetContextMeta 读取上下文元信息。
func (e *Engine) GetContextMeta() (*fsstore.ContextMeta, error) {
	return e.root.GetContextMeta()
}

func (e *Engine) BuildKnowledgeExport() (*fsstore.KnowledgeExport, error) {
	return e.root.BuildKnowledgeExport()
}

func (e *Engine) WriteKnowledgeExport(outDir string) (*fsstore.KnowledgeExport, error) {
	return e.root.WriteKnowledgeExport(outDir)
}

func (e *Engine) KnowledgeExportPreview() (*fsstore.KnowledgeExport, *fsstore.KnowledgeExportMeta, error) {
	return e.root.KnowledgeExportPreview()
}

func (e *Engine) WriteKnowledgeExportMeta(filename string, bundle *fsstore.KnowledgeExport) error {
	return e.root.WriteKnowledgeExportMeta(filename, bundle)
}

// SetContextGeneratedBy 标记上下文生成来源。
func (e *Engine) SetContextGeneratedBy(by string) error {
	if err := e.root.SetContextGeneratedBy(by); err != nil {
		return err
	}
	e.publishContextUpdated("meta", map[string]any{"generated_by": by})
	return nil
}

func (e *Engine) publishContextUpdated(kind string, extra map[string]any) {
	payload := map[string]any{
		"kind": kind,
	}
	for k, v := range extra {
		payload[k] = v
	}
	e.publishSystem(events.EvtContextUpdated, payload)
}
