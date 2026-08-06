package comment_retained

import (
	"path/filepath"
	"strconv"
	"strings"

	"code.byted.org/analyzers/go_cleaner/pkg/comment_retained/internal"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
)

type option struct {
}

type Option func(*option)

//func WithHandlerPath(handlerPath string) Option {
//	return func(o *option) {
//		o.handlerPath = handlerPath
//	}
//}

func Run(annotateRetainedCode string, opts ...Option) {
	opt := option{}
	for _, o := range opts {
		o(&opt)
	}
	annotatedFiles := strings.Split(annotateRetainedCode, "||")
	repoRoot, _ := util.GetWorkingRepositoryRoot()
	for _, annotateInfo := range annotatedFiles {
		splits := strings.Split(annotateInfo, ":")
		if len(splits) != 2 {
			continue
		}
		fileName := filepath.Join(repoRoot, splits[0])

		var lineRangeList []*internal.LineRange
		lineRangeInfo := strings.Split(splits[1], ";")
		for _, lineRange := range lineRangeInfo {
			lineRangeSplits := strings.Split(lineRange, ",")
			if len(lineRangeSplits) != 2 {
				continue
			}
			start, _ := strconv.ParseInt(lineRangeSplits[0], 10, 32)
			end, _ := strconv.ParseInt(lineRangeSplits[1], 10, 32)
			lineRangeList = append(lineRangeList, &internal.LineRange{
				LineStart: int(start),
				LineEnd:   int(end),
			})
		}
		if len(lineRangeList) == 0 {
			continue
		}
		internal.InsertComments(fileName, lineRangeList)
	}
}
