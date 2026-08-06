package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

const threadPostsFile = "thread_posts.jsonl"

type ThreadArtifactRef struct {
	ID           string               `json:"id"`
	ArtifactType model.ArtifactType   `json:"artifact_type"`
	Ref          string               `json:"ref,omitempty"`
	Role         model.PostAuthorRole `json:"role,omitempty"`
}

type ThreadPost struct {
	PostID         string               `json:"post_id"`
	ThreadID       string               `json:"thread_id"`
	ParentReplyID  string               `json:"parent_reply_id,omitempty"`
	RootPostID     string               `json:"root_post_id"`
	CausalRefs     []string             `json:"causal_refs,omitempty"`
	AuthorIdentity string               `json:"author_identity,omitempty"`
	AuthorRole     model.PostAuthorRole `json:"author_role"`
	ArtifactRefs   []ThreadArtifactRef  `json:"artifact_refs,omitempty"`
	SourceEventID  int64                `json:"source_event_id,omitempty"`
	Kind           string               `json:"kind,omitempty"`
	Content        string               `json:"content,omitempty"`
	CreatedAtMs    int64                `json:"created_at_ms"`
}

type AppendThreadPostOptions struct {
	PostID         string
	ThreadID       string
	ParentReplyID  string
	RootPostID     string
	CausalRefs     []string
	AuthorIdentity string
	AuthorRole     model.PostAuthorRole
	ArtifactRefs   []ThreadArtifactRef
	SourceEventID  int64
	Kind           string
	Content        string
	CreatedAtMs    int64
}

func (qs *QuestStore) threadPostsPath(qid string) string {
	return filepath.Join(qs.dir(qid), threadPostsFile)
}

func (qs *QuestStore) AppendThreadPost(qid string, opts AppendThreadPostOptions) (*ThreadPost, error) {
	if strings.TrimSpace(qid) == "" {
		return nil, fmt.Errorf("quest id 为空")
	}
	if strings.TrimSpace(opts.Content) == "" {
		return nil, fmt.Errorf("post content 不能为空")
	}
	post := &ThreadPost{
		PostID:         strings.TrimSpace(opts.PostID),
		ThreadID:       firstNonEmpty(opts.ThreadID, qid),
		ParentReplyID:  strings.TrimSpace(opts.ParentReplyID),
		RootPostID:     strings.TrimSpace(opts.RootPostID),
		CausalRefs:     append([]string(nil), opts.CausalRefs...),
		AuthorIdentity: strings.TrimSpace(opts.AuthorIdentity),
		AuthorRole:     opts.AuthorRole,
		ArtifactRefs:   append([]ThreadArtifactRef(nil), opts.ArtifactRefs...),
		SourceEventID:  opts.SourceEventID,
		Kind:           firstNonEmpty(strings.TrimSpace(opts.Kind), "post"),
		Content:        opts.Content,
		CreatedAtMs:    opts.CreatedAtMs,
	}
	if post.PostID == "" {
		post.PostID = "post_" + NewIDShort()
	}
	if post.RootPostID == "" {
		post.RootPostID = RootPostID(qid)
	}
	if post.AuthorRole == "" {
		post.AuthorRole = model.PostRoleSystem
	}
	if post.CreatedAtMs == 0 {
		post.CreatedAtMs = NowMs()
	}
	if qs.threadPostExists(qid, post.PostID) {
		return post, nil
	}
	if err := validateThreadPostAuthority(post); err != nil {
		return nil, err
	}
	if err := AppendJSONL(qs.threadPostsPath(qid), post); err != nil {
		return nil, err
	}
	return post, nil
}

func (qs *QuestStore) threadPostExists(qid, postID string) bool {
	if strings.TrimSpace(qid) == "" || strings.TrimSpace(postID) == "" {
		return false
	}
	posts, err := qs.LoadThreadPosts(qid)
	if err != nil {
		return false
	}
	for _, post := range posts {
		if post.PostID == postID {
			return true
		}
	}
	return false
}

func (qs *QuestStore) EnsureRootThreadPost(q *QuestMeta) (*ThreadPost, error) {
	if q == nil || strings.TrimSpace(q.ID) == "" {
		return nil, fmt.Errorf("quest 为空")
	}
	posts, err := qs.LoadThreadPosts(q.ID)
	if err != nil {
		return nil, err
	}
	rootID := RootPostID(q.ID)
	for i := range posts {
		if posts[i].PostID == rootID {
			return &posts[i], nil
		}
	}
	content := q.OriginalRequest
	if strings.TrimSpace(content) == "" {
		content = q.Query
	}
	if strings.TrimSpace(content) == "" {
		content = "Legacy quest " + q.ID
	}
	authorRole := model.PostRoleHuman
	if q.Source() == model.SourceAutomation {
		authorRole = model.PostRoleAutomation
	}
	return qs.AppendThreadPost(q.ID, AppendThreadPostOptions{
		PostID:         rootID,
		ThreadID:       q.ID,
		RootPostID:     rootID,
		AuthorIdentity: q.CreatedBy,
		AuthorRole:     authorRole,
		Kind:           "root",
		Content:        content,
		CreatedAtMs:    q.CreatedAtMs,
	})
}

func (qs *QuestStore) AppendReportThreadPost(qid string, q *QuestMeta, opts AppendThreadPostOptions) error {
	if q != nil {
		if _, err := qs.EnsureRootThreadPost(q); err != nil {
			return err
		}
	}
	if strings.TrimSpace(opts.Content) == "" {
		opts.Content = strings.TrimSpace(string(opts.AuthorRole)) + " report submitted"
	}
	_, err := qs.AppendThreadPost(qid, opts)
	return err
}

func (qs *QuestStore) LoadThreadPosts(qid string) ([]ThreadPost, error) {
	posts, err := ReadJSONL[ThreadPost](qs.threadPostsPath(qid), 0)
	if err != nil {
		if os.IsNotExist(err) {
			return []ThreadPost{}, nil
		}
		return nil, err
	}
	return posts, nil
}

func RootPostID(qid string) string {
	return "root_" + strings.TrimSpace(qid)
}

func authorIdentityForRole(q *QuestMeta, role model.PostAuthorRole) string {
	if q == nil {
		return ""
	}
	switch role {
	case model.PostRoleMaker:
		return q.WarriorID
	case model.PostRoleChecker:
		return q.MageID
	case model.PostRoleAutomation:
		return q.CreatedBy
	case model.PostRoleHuman:
		return q.CreatedBy
	default:
		return ""
	}
}

func validateThreadPostAuthority(post *ThreadPost) error {
	for _, ref := range post.ArtifactRefs {
		switch ref.ArtifactType {
		case model.ArtifactTypeDelivery, model.ArtifactTypeMakerReport:
			if post.AuthorRole != model.PostRoleMaker {
				return fmt.Errorf("%s must be attached by maker, got %s", ref.ArtifactType, post.AuthorRole)
			}
		case model.ArtifactTypeReviewReport, model.ArtifactTypeEvidence, model.ArtifactTypeVerificationArtifact:
			if post.AuthorRole != model.PostRoleChecker {
				return fmt.Errorf("%s must be attached by checker, got %s", ref.ArtifactType, post.AuthorRole)
			}
		case model.ArtifactTypeDecisionNote:
			if post.AuthorRole != model.PostRoleSystem && post.AuthorRole != model.PostRoleHuman {
				return fmt.Errorf("%s must be attached by system or human, got %s", ref.ArtifactType, post.AuthorRole)
			}
		}
	}
	return nil
}
