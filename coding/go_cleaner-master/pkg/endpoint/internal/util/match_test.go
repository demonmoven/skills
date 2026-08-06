package util

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

var TestCodeContext = `package main

type RouterGroup struct {}

func (r *RouterGroup) Group(s string) *RouterGroup {
	return r
}

func (r *RouterGroup) POST(s string, args ... string) *RouterGroup {
	return r
}

func (r *RouterGroup) GET(s string, args ... string) *RouterGroup {
	return r
}

func (r *RouterGroup) Use(fn func()) {}

func main() {
	r := &RouterGroup{}
	%s
}
`

func parse(t *testing.T, code string) *ast.File {
	src := fmt.Sprintf(TestCodeContext, code)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", []byte(src), parser.AllErrors|parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

type TestCase struct {
	Endpoint    []string
	Code        string
	ShouldMatch bool
}

func TestBackwardMatchAndRewrite(t *testing.T) {
	tests := []TestCase{
		{
			[]string{
				"/api/admin/authority/data_auth/add",
			},
			`root := r.Group("/api/admin")
			authorityGroup := root.Group("/authority")
			{
				{
					dataAuthGroup := authorityGroup.Group("/data_auth")
					dataAuthGroup.POST("/add", interceptor.MustHaveAuth("data_auth_edit"), api.WrapData(auth.IouDataAuth(true)))
				}
			}`,
			true,
		},
		{
			[]string{
				"/announce/add",
			},
			`{
				announceGroup := r.Group("/announce")
				announceGroup.Use(authMw.DefaultApiMW(admin.GetUserName))
				announceListGroup := announceGroup.Group("/list")
				announceGroup.POST("/add", interceptor.MustHaveAuth("announce_manage_add"), api.WrapData(announce.Add))
				r.OPTIONS("/api/admin/announce/list", interceptor.AllowCors)
			}`,
			true,
		},
		{
			[]string{
				"/api/home/user/team_member",
			},
			`homeGroup := r.Group("/api/home")
			{
				homeGroup.Use(ginlog.LogReqMiddleware())
				homeGroup.Use(admin.LoginCheck)
				homeGroup.Use(auth.FromSourceInject)
				userGroup := homeGroup.Group("/user")
				userGroup.GET("/team_member", api.WrapData(auth.TeamMembers))
			}`,
			true,
		},
		{
			[]string{
				"/api/home/user/team_member",
			},
			`homeGroup := r.Group("/api/home")
			{
				homeGroup.Use(ginlog.LogReqMiddleware())
				homeGroup.Use(admin.LoginCheck)
				homeGroup.Use(auth.FromSourceInject)
				userGroup := homeGroup.Group("/user")
				userGroup = homeGroup.Group("/foo")
				userGroup.GET("/team_member", api.WrapData(auth.TeamMembers))
			}`,
			false,
		},
	}

	for _, test := range tests {
		matched := BackwardMatchAndRewrite(parse(t, test.Code), test.Endpoint)

		if !matched && test.ShouldMatch {
			t.Fatalf("Expect match %v but failed", test.Endpoint)
		}

		if matched && !test.ShouldMatch {
			t.Fatalf("Expect not match %v but matched", test.Endpoint)
		}
	}
}
