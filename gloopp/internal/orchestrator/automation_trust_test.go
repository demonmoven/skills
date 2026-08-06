package orchestrator

import (
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// TestTrustVerificationTypeForFailure 覆盖失败原因区分：
// human_rejected / mage_rejected / 操作性失败。
// 回归：此前所有 EvtQuestFailed 都落 "quest.failed"，mage 质量拒绝与
// agent 崩溃无法区分；显式 recordAutomationTrustOutcome("human_rejected")
// 又被同步事件 dedup 吞掉。
func TestTrustVerificationTypeForFailure(t *testing.T) {
	cases := []struct {
		name string
		q    *fsstore.QuestMeta
		want string
	}{
		{
			name: "human reject",
			q:    &fsstore.QuestMeta{FinalizedBy: "user", FinalVerdict: model.VerdictReject, FinalComment: "不要了"},
			want: "human_rejected",
		},
		{
			name: "mage reject",
			q:    &fsstore.QuestMeta{FinalComment: "法师评审拒绝：剑士产出不达标"},
			want: "mage_rejected",
		},
		{
			name: "agent error with attribution",
			q: &fsstore.QuestMeta{
				FinalComment:       "agent 连续错误达到阈值：3",
				FailureAttribution: &fsstore.FailureAttribution{Reason: "agent_error"},
			},
			want: "quest_failed:agent_error",
		},
		{
			name: "operational fallback",
			q:    &fsstore.QuestMeta{FinalComment: "something broke"},
			want: "quest_failed",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := trustVerificationTypeForFailure(c.q); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestTrustVerificationTypeForSuccess(t *testing.T) {
	cases := []struct {
		name string
		q    *fsstore.QuestMeta
		want string
	}{
		{name: "human approved", q: &fsstore.QuestMeta{FinalizedBy: "user"}, want: "human_approved"},
		{name: "policy auto pass", q: &fsstore.QuestMeta{FinalizedBy: "policy", AutoPassedByPolicy: "context_store_automation_auto_pass"}, want: "policy_auto_pass"},
		{name: "policy auto complete", q: &fsstore.QuestMeta{FinalizedBy: "policy"}, want: "policy_auto_complete"},
		{name: "default quest success", q: &fsstore.QuestMeta{}, want: "quest_success"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := trustVerificationTypeForSuccess(c.q); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
