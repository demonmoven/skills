package idl

import (
	"encoding/json"
	"testing"
)

func TestBuildBundle_Contract(t *testing.T) {
	b := BuildBundle()
	if b.IDLVersion != ExportVersion {
		t.Fatalf("idl version = %s, want %s", b.IDLVersion, ExportVersion)
	}
	if len(b.Skills) == 0 {
		t.Fatal("expected skills")
	}
	if len(b.Classes["warrior"].Skills) == 0 || len(b.Classes["mage"].Skills) == 0 {
		t.Fatalf("expected per-class skill exports: %+v", b.Classes)
	}

	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("bundle should marshal: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("empty marshal")
	}
	// 验证 skill body 不会泄露到 IDL 中（Manifest 应该不含 body）
	var rawBundle map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawBundle); err != nil {
		t.Fatalf("unmarshal bundle failed: %v", err)
	}
	if _, hasTools := rawBundle["tools"]; hasTools {
		t.Fatal("v0.3.4+ bundle should not carry tools field")
	}
	if _, hasAgents := rawBundle["agent_schemas"]; hasAgents {
		t.Fatal("v0.3.4+ bundle should not carry agent_schemas field")
	}
	var skillsJSON []json.RawMessage
	if err := json.Unmarshal(rawBundle["skills"], &skillsJSON); err != nil {
		t.Fatalf("unmarshal skills failed: %v", err)
	}
	for i, sk := range skillsJSON {
		if hasTopLevelJSONKey(sk, "body") {
			t.Fatalf("skill[%d] 不应包含 body 字段，原始输出: %s", i, string(sk))
		}
	}
}

// hasTopLevelJSONKey 检查 JSON 对象顶层是否包含指定 key（不递归）
func hasTopLevelJSONKey(raw json.RawMessage, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
