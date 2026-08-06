package internal

import "testing"

func TestInsertComments(t *testing.T) {
	var err error
	err = InsertComments(
		"/Users/bytedance/go/src/code.byted.org/webcast/im_manager/badge_util/util.go",
		[]*LineRange{
			{LineStart: 21, LineEnd: 93},
		},
	)
	if err != nil {
		t.Logf("insertComments error: %+v", err)
	}
	//err = InsertComments(
	//	"/Users/bytedance/go/src/code.byted.org/webcast/im_manager/handler.go",
	//	[]*LineRange{
	//		{LineStart: 88, LineEnd: 134},
	//	},
	//)
	//if err != nil {
	//	t.Logf("insertComments error: %+v", err)
	//}
}
