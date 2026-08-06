package model

import "testing"

func TestValidateQuestionText(t *testing.T) {
	tests := []struct {
		name    string
		q       string
		wantErr bool
	}{
		// 合法提问
		{"正常中文疑问", "这个修复需要改 public API，确认可以改吗？", false},
		{"正常英文疑问", "Should I also update the CLI flag name to match?", false},
		{"短但带问号", "继续?", false},
		{"短但带中文问号", "用 v2？", false},
		{"含代码的提问", "go vet 报 unused import，这个文件确实没用，可以直接删吗", false},

		// 空或纯空白
		{"空字符串", "", true},
		{"纯空格", "   ", true},
		{"纯tab", "\t\t", true},

		// 占位符黑名单
		{"need input", "need input", true},
		{"NEED INPUT 大写", "NEED INPUT", true},
		{"需要输入", "需要输入", true},
		{"需要用户输入", "需要用户输入", true},
		{"等待输入", "等待输入", true},
		{"问用户", "问用户", true},
		{"单问号", "?", true},
		{"三问号", "???", true},

		// 过短且非疑问
		{"过短无问号", "继续", true},
		{"过短英文", "go on", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuestionText(tt.q)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQuestionText(%q) err=%v, wantErr=%v", tt.q, err, tt.wantErr)
			}
		})
	}
}
