package internal

import (
	_ "embed"
	"fmt"
	"go/constant"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/atotto/clipboard"
	"golang.org/x/tools/go/ssa"

	"code.byted.org/analyzers/go_cleaner/pkg/ab/internal/model"
)

type ABEntryParamSig struct {
	KeyPos    []int       `json:"key_index"`
	DftPos    int         `json:"default_index"`
	KeyPrefix string      `json:"key_prefix"`
	Dft       interface{} `json:"default"`
}

type Entry struct {
	PackageName string          `json:"pkg"`
	RecvName    string          `json:"recv"`
	FuncName    []string        `json:"func"`
	Sig         ABEntryParamSig `json:"sig"`
}

type EntryFinder struct {
	entries   map[string]*ABEntryParamSig
	entryPkgs map[string]bool
}

func NewEntryFinder(entry []*model.ABEntry) *EntryFinder {
	var entries = map[string]*ABEntryParamSig{}
	var entryPkgs = map[string]bool{}
	for _, v := range GetEntry(entry) {
		pkg := v.PackageName
		entryPkgs[pkg] = true
		recv := v.RecvName
		sig := v.Sig
		for _, fn := range v.FuncName {
			if recv == "" {
				entries[fmt.Sprintf("%s.%s", pkg, fn)] = &sig
			} else {
				entries[fmt.Sprintf("(*%s.%s).%s", pkg, recv, fn)] = &sig
				entries[fmt.Sprintf("(%s.%s).%s", pkg, recv, fn)] = &sig
			}
		}
	}
	return &EntryFinder{entries: entries, entryPkgs: entryPkgs}
}

func (e *EntryFinder) IsEntry(fn *ssa.Function) (sigType *ABEntryParamSig, ok bool) {
	if fn.Package() == nil || fn.Package().Pkg == nil || !e.entryPkgs[fn.Package().Pkg.Path()] {
		return
	}
	sigType, ok = e.entries[fn.String()]
	return
}

func GetEntry(tasks []*model.ABEntry) []Entry {
	var entry []Entry
	if len(tasks) == 0 {
		tasks = (&model.ABEntry{}).MultiGet()
	}
	for i := len(tasks) - 1; i >= 0; i-- {
		t := tasks[i]
		sig := ABEntryParamSig{
			KeyPos:    t.KeyIndex,
			DftPos:    t.DefaultIndex,
			KeyPrefix: t.KeyPrefix,
			Dft:       nil,
		}
		switch t.DefaultTyp {
		case "int", "int32", "int63", "uint", "unit32", "uint64":
			dft, err := strconv.ParseInt(t.DefaultVal, 10, 64)
			if err != nil {
				continue
			}
			sig.Dft = dft
		case "float", "float32", "float64":
			dft, err := strconv.ParseFloat(t.DefaultVal, 10)
			if err != nil {
				continue
			}
			sig.Dft = dft
		case "bool", "boolean":
			switch t.DefaultVal {
			case "true", "True", "1":
				sig.Dft = true
			case "false", "False", "0":
				sig.Dft = false
			default:
				continue
			}
		case "string":
			sig.Dft = t.DefaultVal
		}

		entry = append(entry, Entry{
			PackageName: t.Package,
			RecvName:    t.Recv,
			FuncName:    t.Func,
			Sig:         sig,
		})
	}
	return entry
}

func (d *Detector) collectExpiredABCall() {
	posKey := map[token.Pos][]constant.Value{}
	keys := map[string]bool{}
	for call, sigType := range d.abEntryCallSite {
		k, _ := d.keyDftParse(call.Common(), sigType)
		posKey[call.Pos()] = k
		for _, v := range k {
			if isString(v) {
				keys[constant.StringVal(v)] = true
			}
		}
	}
	var keyList []string
	for key := range keys {
		keyList = append(keyList, key)
	}
	d.KeyChecker.MustMultiGet(keyList)

	for pos, k := range posKey {
		var effective KeyEffectiveness = KeyExpired
		for _, v := range k {
			if !isString(v) || v == VariantValue {
				effective = KeyUncertain
				break
			}
			sv := d.KeyChecker.Effectiveness(constant.StringVal(v))
			// 有效key和最近个月新增的都认为有效
			if sv == KeyEffective || d.blame.Within(d.position(pos), d.position(pos), d.opt.Before) {
				// 一个有效就是有效，不能删除分支
				effective = KeyEffective
				break
			} else if sv == KeyUncertain {
				// 不确定
				effective = KeyUncertain
			}
		}
		d.abEntryCallPosEffective[pos] = effective
	}
}

func (d *Detector) collectABCall() {
	for fn := range d.AllFunction {
		if sigType, ok := d.EntryFinder.IsEntry(fn); ok {
			if node, ok := d.cg.Nodes[fn]; ok {
				for _, inE := range node.In {
					_, e := d.EntryFinder.IsEntry(inE.Caller.Func)
					if d.IsProjectFunc(inE.Caller.Func) && !e {
						d.abEntryCallSite[inE.Site] = sigType
						d.abEntryCallPos[inE.Site.Value().Pos()] = sigType
					}
				}
			}
		}
	}
}

func (d *Detector) determineABCall(call *ssa.Call) (v []constant.Value, ok bool) {
	if sigType, ok := d.abEntryCallPos[call.Pos()]; ok {
		ks, dfts := d.keyDftParse(call.Common(), sigType)
		for _, k := range ks {
			isString(k)
			if !isString(k) || !d.KeyChecker.Effectiveness(constant.StringVal(k)).Expired() {
				return []constant.Value{VariantValue}, true
			}
		}
		return dfts, true
	}
	return nil, false
}

func (d *Detector) keyDftParse(call *ssa.CallCommon, sig *ABEntryParamSig) (keyV, dftV []constant.Value) {
	args := call.Args

	var ks [][]constant.Value
	for _, i := range sig.KeyPos {
		if i >= len(args) {
			keyV = []constant.Value{VariantValue}
			continue
		}
		ks = append(ks, d.Value(args[i]))
	}
	if len(ks) == 0 {
		keyV = []constant.Value{VariantValue}
	}

	if keyV == nil {
		keyV, _ = combine(ks[0], ks[1:]...)
		if sig.KeyPrefix != "" {
			var keyVV []constant.Value
			for _, key := range keyV {
				if key.Kind() == constant.String {
					keyVV = append(keyVV, constant.MakeString(sig.KeyPrefix+"."+constant.StringVal(key)))
				} else {
					keyVV = append(keyVV, key)
				}
			}
			keyV = keyVV
		}
	}

	if sig.Dft != nil {
		switch v := sig.Dft.(type) {
		case int:
			dftV = []constant.Value{constant.MakeInt64(int64(v))}
		case int64:
			dftV = []constant.Value{constant.MakeInt64(v)}
		case int32:
			dftV = []constant.Value{constant.MakeInt64(int64(v))}
		case float32:
			dftV = []constant.Value{constant.MakeFloat64(float64(v))}
		case float64:
			dftV = []constant.Value{constant.MakeFloat64(v)}
		case string:
			dftV = []constant.Value{constant.MakeString(v)}
		case bool:
			dftV = []constant.Value{constant.MakeBool(v)}
		default:
			dftV = []constant.Value{VariantValue}
		}
	} else if sig.DftPos >= len(args) {
		dftV = []constant.Value{VariantValue}
	} else {
		dftV = d.Value(args[sig.DftPos])
	}
	return
}

type ABOutput struct {
	Pos token.Position
	Key string
}

func (d *Detector) DebugABEntryArg() (output []*ABOutput) {
	var effectiveLog []string
	var unusedLog []string
	var uncertainLog []string

	rawLogs := map[token.Pos][][]constant.Value{}
	keys := map[string]bool{}
	for call, sigType := range d.abEntryCallSite {
		k, dft := d.keyDftParse(call.Common(), sigType)
		rawLogs[call.Pos()] = [][]constant.Value{k, dft}
		for _, v := range k {
			if isString(v) {
				keys[constant.StringVal(v)] = true
			}
		}
	}

	relPath := func(path string) string {
		base, _ := filepath.Abs(".")
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return "-"
		}
		return rel
	}

	for pos, kv := range rawLogs {
		blameInfo := d.blame.Search(d.position(pos))
		keyStr := fmt.Sprintf("%v", kv[0])
		if len(keyStr) > 50 {
			keyStr = keyStr[:50]
		}
		position := d.position(pos)
		position.Filename = relPath(position.Filename)
		path := position.String()
		switch d.abEntryCallPosEffective[pos] {
		case KeyUncertain:
			uncertainLog = append(uncertainLog, strings.Join([]string{path, keyStr, fmt.Sprintf("%v", kv[1]), "uncertain", blameInfo.author, blameInfo.modifyAt}, "\t"))
		case KeyEffective:
			effectiveLog = append(effectiveLog, strings.Join([]string{path, keyStr, fmt.Sprintf("%v", kv[1]), "true", blameInfo.author, blameInfo.modifyAt}, "\t"))
		case KeyExpired:
			unusedLog = append(unusedLog, strings.Join([]string{path, keyStr, fmt.Sprintf("%v", kv[1]), "false", blameInfo.author, blameInfo.modifyAt}, "\t"))
			if !d.blame.Within(position, position, d.opt.Before) {
				output = append(output, &ABOutput{
					Pos: position,
					Key: keyStr,
				})
			}
		default:
		}
	}

	head := strings.Join([]string{"pos", "key", "default", "effective", "update_by", "update_at"}, "\t")
	sort.Strings(effectiveLog)
	sort.Strings(unusedLog)
	sort.Strings(uncertainLog)
	s := []string{head}
	s = append(s, unusedLog...)
	s = append(s, uncertainLog...)
	s = append(s, effectiveLog...)
	table := strings.Join(s, "\n")
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.StripEscape)
	_, _ = fmt.Fprintln(writer, table)
	_ = writer.Flush()
	_ = clipboard.WriteAll(table)

	effectiveKeyN := 0
	expiredKeyN := 0
	uncertainKeyN := 0
	for k := range keys {
		effective := d.KeyChecker.Effectiveness(k)
		if effective == KeyEffective {
			effectiveKeyN += 1
		} else if effective == KeyExpired {
			expiredKeyN += 1
		} else {
			uncertainKeyN += 1
		}
	}
	fmt.Printf("ABParam Key Number: Unused/Uncertain/Effective = %d/%d/%d\n", expiredKeyN, uncertainKeyN, effectiveKeyN)

	return
}
