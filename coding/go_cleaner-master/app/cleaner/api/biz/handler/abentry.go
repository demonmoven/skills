package handler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"code.byted.org/analyzers/go_cleaner/app/cleaner/api/biz/model"
	bdsso "code.byted.org/ucenter/bdsso_sessionlib"
)

func bindAbEntry(c *app.RequestContext) *model.ABEntry {
	body, _ := c.Body()
	var task model.ABEntry
	_ = json.Unmarshal(body, &task)
	return &task
}

func getAuthor(ctx context.Context, c *app.RequestContext) (userName string, err error) {
	session, err := bdsso.GetHertzSession(c)
	if err != nil {
		return
	}
	userName, err = session.UserName(ctx)
	if err != nil || userName == "" {
		err = errors.New("not login")
	}
	return
}

func MultiGetABEntry(ctx context.Context, c *app.RequestContext) {
	task := bindAbEntry(c)
	c.JSON(200, task.MultiGet())
}

func SaveABEntry(ctx context.Context, c *app.RequestContext) {
	task := bindAbEntry(c)
	var existEntry *model.ABEntry
	if task.ID > 0 {
		if v := (&model.ABEntry{ID: task.ID}).MultiGet(); len(v) > 0 {
			existEntry = v[0]
		}
	}
	if userName, err := getAuthor(ctx, c); err != nil || (existEntry != nil && !existEntry.Author.Has(userName)) {
		c.JSON(403, "Not Authorized")
		return
	} else if existEntry == nil {
		task.Author.Add(userName)
	}

	task.Save()
	c.JSON(200, (&model.ABEntry{}).MultiGet())
}

func DeleteABEntry(ctx context.Context, c *app.RequestContext) {
	task := bindAbEntry(c)
	var existEntry *model.ABEntry
	if task.ID > 0 {
		if v := (&model.ABEntry{ID: task.ID}).MultiGet(); len(v) > 0 {
			existEntry = v[0]
		}
	}
	if userName, err := getAuthor(ctx, c); err != nil || (existEntry != nil && !existEntry.Author.Has(userName)) {
		c.JSON(403, "Not Authorized")
		return
	}

	task.Delete()
	c.JSON(200, (&model.ABEntry{}).MultiGet())
}
