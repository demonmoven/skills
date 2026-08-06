> [废弃资产清理平台](https://bytedance.larkoffice.com/wiki/SEEqwzIM2itMEYkfOwTceKYdngd)代码清理任务创建

{{ if gt ( len .TaskURL ) 0 }}
[view task]({{ .TaskURL }})
{{ end }}

# 
{{ if gt ( len .DeletedEndpoint )  0 }}
#### Deleted Endpoints:
{{ range .DeletedEndpoint }}
- [ ] {{ . }}
{{ end }}
_Please check if they can be deleted safely._
{{ end }}

{{ if gt ( len .VerifiedButNotLocatedEndpoint ) 0 }}
#### Not-Located Endpoints:
{{ range .VerifiedButNotLocatedEndpoint }}
- [ ] {{ . }}
{{ end }}
_These unused endpoints are not deleted successfully. Please delete them manually if necessary._
{{ end }}

#### Unused Codes:
- LoC: {{ .UnusedLoC }}

{{ if or (gt ( len .DownstreamRepoUnChecked ) 0) (gt ( len .DownstreamRepoChecked ) 0)  }}
#### Imported by:
{{ range $i, $v := .DownstreamRepoUnChecked }}
- [ ] {{ $i }}
{{ end }}
{{ range $i, $v := .DownstreamRepoChecked }}
- [x] {{ $i }}
{{ end }}
_{{ .Repo }} is imported by the repositories above. Please ensure they function properly after this MR._
{{ end }}

{{ if gt ( len .Reviewer ) 0 }}
#### Suggested Reviewers:
{{ range .Reviewer }}
- {{ . }}
{{ end }}
{{ end }}
