package fsstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// QuestArtifactSchemaVersion 是产出物存储格式的版本号。
const QuestArtifactSchemaVersion = "quest_artifact.v1"

// QuestArtifact 描述一个 quest 的产出物元信息（二进制文件或声明型链接）。
type QuestArtifact struct {
	ID            string `json:"id"`
	SchemaVersion string `json:"schema_version,omitempty"`
	Kind          string `json:"kind"` // image | file | log | document | archive | unknown
	MIME          string `json:"mime,omitempty"`
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
	StoragePath   string `json:"storage_path"`
	Source        string `json:"source,omitempty"` // upload | api | automation
	CreatedAtMs   int64  `json:"created_at_ms"`
}

// ArtifactInput 是保存产出物时传入的输入参数（二进制型产出物）。
type ArtifactInput struct {
	Name   string
	MIME   string
	Source string
	Reader io.Reader
}

func (qs *QuestStore) artifactsDir(qid string) string {
	return filepath.Join(qs.dir(qid), "artifacts")
}

// SaveArtifact 保存一个二进制产出物到 quest 的 artifacts 目录，返回元信息。
// 空 qid 或 nil reader 会报错；空 name 会回退为 "artifact"。
func (qs *QuestStore) SaveArtifact(qid string, input ArtifactInput) (QuestArtifact, error) {
	if strings.TrimSpace(qid) == "" {
		return QuestArtifact{}, fmt.Errorf("quest id 不能为空")
	}
	if input.Reader == nil {
		return QuestArtifact{}, fmt.Errorf("artifact reader 不能为空")
	}
	name := sanitizeArtifactName(input.Name)
	if name == "" {
		name = "artifact"
	}
	if input.Source == "" {
		input.Source = "upload"
	}
	dir := qs.artifactsDir(qid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return QuestArtifact{}, err
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return QuestArtifact{}, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(tmp, hash), input.Reader)
	closeErr := tmp.Close()
	if copyErr != nil {
		return QuestArtifact{}, copyErr
	}
	if closeErr != nil {
		return QuestArtifact{}, closeErr
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	mimeType := strings.TrimSpace(input.MIME)
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = mime.TypeByExtension(filepath.Ext(name))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	kind := artifactKind(mimeType, name)
	id := "art_" + sum[:12]
	filename := id + "_" + name
	dst := filepath.Join(dir, filename)
	if _, err := os.Stat(dst); err == nil {
		_ = os.Remove(tmpName)
	} else if err := os.Rename(tmpName, dst); err != nil {
		return QuestArtifact{}, err
	}
	return QuestArtifact{
		ID:            id,
		SchemaVersion: QuestArtifactSchemaVersion,
		Kind:          kind,
		MIME:          mimeType,
		Name:          name,
		Size:          size,
		SHA256:        sum,
		StoragePath:   filepath.ToSlash(filepath.Join("artifacts", filename)),
		Source:        input.Source,
		CreatedAtMs:   NowMs(),
	}, nil
}

// OpenArtifact 按 ID 打开一个产出物文件。若产出物是外部链接则返回错误。
// 查找范围覆盖 inputs 和 outputs。
func (qs *QuestStore) OpenArtifact(qid, artifactID string) (*os.File, QuestArtifact, error) {
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return nil, QuestArtifact{}, err
	}
	allArtifacts := make([]QuestArtifact, 0, len(q.Inputs)+len(q.Outputs))
	allArtifacts = append(allArtifacts, q.Inputs...)
	allArtifacts = append(allArtifacts, q.Outputs...)
	for _, artifact := range allArtifacts {
		if artifact.ID != artifactID {
			continue
		}
		// 声明型产出物没有本地文件
		if artifact.StoragePath == "" || strings.HasPrefix(artifact.StoragePath, "http://") || strings.HasPrefix(artifact.StoragePath, "https://") {
			return nil, QuestArtifact{}, fmt.Errorf("artifact 是外部链接，无法本地打开: %s", artifactID)
		}
		// 支持绝对路径（worktree 模式 workspace 在 quest 目录外时，deliverable path 归一化为绝对路径）
		p := filepath.FromSlash(artifact.StoragePath)
		if !filepath.IsAbs(p) {
			p = filepath.Join(qs.dir(qid), p)
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, QuestArtifact{}, err
		}
		return f, artifact, nil
	}
	return nil, QuestArtifact{}, fmt.Errorf("artifact 不存在: %s", artifactID)
}

// ArtifactPath 返回产出物的本地绝对路径；外部链接型产出物返回空串。
// 支持绝对路径归一化（worktree 模式 deliverable 已经是绝对路径）。
func (qs *QuestStore) ArtifactPath(qid string, artifact QuestArtifact) string {
	if artifact.StoragePath == "" {
		return ""
	}
	p := filepath.FromSlash(artifact.StoragePath)
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(qs.dir(qid), p)
}

// ========== 产出物（Outputs）管理 ==========

// SaveOutputArtifact 保存一个文件型产出物（有二进制内容），并写入 quest meta 的 Outputs。
func (qs *QuestStore) SaveOutputArtifact(qid string, input ArtifactInput) (QuestArtifact, error) {
	art, err := qs.SaveArtifact(qid, input)
	if err != nil {
		return QuestArtifact{}, err
	}
	meta, err := qs.LoadQuest(qid)
	if err != nil {
		return QuestArtifact{}, err
	}
	meta.Outputs = append(meta.Outputs, art)
	if err := qs.SaveQuest(meta); err != nil {
		return QuestArtifact{}, err
	}
	return art, nil
}

// DeclaredOutput 描述一个声明型产出物（如飞书文档链接），没有二进制内容。
type DeclaredOutput struct {
	Name        string // 显示名，如 "设计方案文档"
	Kind        string // image | file | log | document | archive | link | unknown
	URL         string // 外部链接（可选）
	Description string // 描述（可选）
	Source      string // 来源，如 warrior_phase / mage_review
}

// AddDeclaredOutput 添加一个声明型产出物（无二进制，如飞书文档链接）到 quest meta 的 Outputs。
func (qs *QuestStore) AddDeclaredOutput(qid string, decl DeclaredOutput) (QuestArtifact, error) {
	if strings.TrimSpace(qid) == "" {
		return QuestArtifact{}, fmt.Errorf("quest id 不能为空")
	}
	if strings.TrimSpace(decl.Name) == "" {
		return QuestArtifact{}, fmt.Errorf("产出物名称不能为空")
	}
	kind := strings.ToLower(strings.TrimSpace(decl.Kind))
	if kind == "" {
		kind = "unknown"
	}
	source := decl.Source
	if source == "" {
		source = "declared"
	}

	// 用 name + url 做一个稳定的 id
	idSeed := decl.Name + "|" + decl.URL
	sum := sha256.Sum256([]byte(idSeed))
	id := "out_" + hex.EncodeToString(sum[:6])

	art := QuestArtifact{
		ID:            id,
		SchemaVersion: QuestArtifactSchemaVersion,
		Kind:          kind,
		MIME:          "text/uri-list",
		Name:          decl.Name,
		Size:          0,
		SHA256:        hex.EncodeToString(sum[:]),
		StoragePath:   decl.URL, // 声明型产出物用 URL 作为 storage_path
		Source:        source,
		CreatedAtMs:   NowMs(),
	}

	meta, err := qs.LoadQuest(qid)
	if err != nil {
		return QuestArtifact{}, err
	}
	// 去重：同名同 URL 视为同一产出物
	for _, existing := range meta.Outputs {
		if existing.Name == art.Name && existing.StoragePath == art.StoragePath {
			return existing, nil
		}
	}
	meta.Outputs = append(meta.Outputs, art)
	if err := qs.SaveQuest(meta); err != nil {
		return QuestArtifact{}, err
	}
	return art, nil
}

// ClearOutputs 清空指定来源的产出物（返工/重新执行时使用）。
// source 为空时清空所有产出物。
func (qs *QuestStore) ClearOutputs(qid, source string) error {
	meta, err := qs.LoadQuest(qid)
	if err != nil {
		return err
	}
	if source == "" {
		meta.Outputs = nil
	} else {
		filtered := make([]QuestArtifact, 0, len(meta.Outputs))
		for _, art := range meta.Outputs {
			if art.Source != source {
				filtered = append(filtered, art)
			}
		}
		meta.Outputs = filtered
	}
	return qs.SaveQuest(meta)
}

func sanitizeArtifactName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	if len(name) > 160 {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(ext) > 24 {
			ext = ext[:24]
		}
		limit := 160 - len(ext)
		if limit < 1 {
			limit = 1
		}
		if len(base) > limit {
			base = base[:limit]
		}
		name = base + ext
	}
	return name
}

func artifactKind(mimeType, name string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "text/") || ext == ".log":
		return "log"
	case mimeType == "application/pdf", ext == ".pdf", ext == ".doc", ext == ".docx", ext == ".md":
		return "document"
	case strings.Contains(mimeType, "zip"), ext == ".zip", ext == ".tar", ext == ".gz", ext == ".tgz":
		return "archive"
	case mimeType != "" && mimeType != "application/octet-stream":
		return "file"
	default:
		return "unknown"
	}
}

var artifactNamePattern = regexp.MustCompile(`^[A-Za-z0-9._ -]+$`)

// IsLikelySafeArtifactName 判断文件名是否只包含安全字符（字母、数字、点、下划线、空格、短横线）。
// 用于 mage 评审阶段快速拒绝可疑的 deliverable 文件名。
func IsLikelySafeArtifactName(name string) bool {
	return name != "" && artifactNamePattern.MatchString(name)
}
