package star_author_info

import (
	"code.byted.org/ad/star_goauthor/infrastructure/mysql/model"
	"gorm.io/gorm"
)

/*
import (
    "gorm.io/gorm"
)
*/

type Dao interface {
	// prop(name="handler", type="get")
	GetHandler() *gorm.DB

	BeginWithHandler(handler *gorm.DB) Dao

	/*
		sql(value = "select * from star_author_info limit 1", mode="placeholder")
	*/
	QueryAnyOne() (*model.StarAuthorInfo, error)
}
