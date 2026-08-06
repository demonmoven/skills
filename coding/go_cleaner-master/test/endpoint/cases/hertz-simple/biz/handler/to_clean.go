package handler

import (
	"context"

	"code.byted.org/middleware/hertz/pkg/app"
	"code.byted.org/middleware/hertz/pkg/common/utils"
)

func ToClean1(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, utils.H{
		"message": "a method to clean 1",
	})
}

func ToClean2(ctx context.Context, c *app.RequestContext) {
	c.JSON(200, utils.H{
		"message": "a method to clean 2",
	})
}
