package star_author_info

import (
	"code.byted.org/ad/creative_core/exception"
	dbutil "code.byted.org/ad/star_dbutil"
	"code.byted.org/ad/star_dbutil/resolver"
	"code.byted.org/ad/star_goauthor/infrastructure/mysql/db/star_author_info"
	"code.byted.org/ad/star_goauthor/infrastructure/mysql/model"
	"code.byted.org/ad/star_gocommon/utils"
	DalHelper "code.byted.org/ad/star_gocommon/utils/dal_helper"
	PaginationGen "code.byted.org/ad/star_kitex_gen/kitex_gen/model/pagination"
	"code.byted.org/gopkg/logs"
	"context"
	"reflect"
	"time"
)

type StarAuthorInfoDAL struct {
	model     *model.StarAuthorInfo
	commonDao star_author_info.CommonDao
	ctx       context.Context
}

func NewStarAuthorInfoDAL(ctx context.Context) StarAuthorInfoDAL {
	_model := model.NewStarAuthorInfo()
	db := dbutil.GetSession(ctx)
	commonDao := star_author_info.NewCommonDao(db)
	return StarAuthorInfoDAL{ctx: ctx, model: _model, commonDao: commonDao}
}

func NewStarAuthorInfoDALWithWriteDB(ctx context.Context) StarAuthorInfoDAL {
	_model := model.NewStarAuthorInfo()
	db := dbutil.GetSession(ctx).Clauses(resolver.Write)
	commonDao := star_author_info.NewCommonDao(db)
	return StarAuthorInfoDAL{ctx: ctx, model: _model, commonDao: commonDao}
}

func (b *StarAuthorInfoDAL) Query(filter map[string]interface{}, options ...DalHelper.ModifyOptionFunc) ([]*model.StarAuthorInfo, error) {
	_db, err := b.commonDao.Query(filter, options...)
	return _db, exception.NewSystemError(b.ctx, err)
}

func (b *StarAuthorInfoDAL) QueryById(id int64) (*model.StarAuthorInfo, error) {
	ret, err := b.QueryOne(map[string]interface{}{
		"id": id,
	}, DalHelper.WithDeleted())
	if err != nil {
		return nil, exception.NewSystemError(b.ctx, err)
	}
	return ret, nil
}

func (b *StarAuthorInfoDAL) QueryByIds(ids []int64) ([]*model.StarAuthorInfo, error) {
	if len(ids) == 0 {
		return make([]*model.StarAuthorInfo, 0), nil
	}

	ret, err := b.Query(map[string]interface{}{
		"id": ids,
	}, DalHelper.WithDeleted())
	if err != nil {
		return []*model.StarAuthorInfo{}, exception.NewSystemError(b.ctx, err)
	}
	return ret, nil
}

func (b *StarAuthorInfoDAL) QueryOne(filter map[string]interface{}, options ...DalHelper.ModifyOptionFunc) (*model.StarAuthorInfo, error) {
	options = append(options, DalHelper.WithLimit(1))
	ret, err := b.Query(filter, options...)
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, exception.NewSystemError(b.ctx, err)
	}
	return ret[0], nil
}

func (b *StarAuthorInfoDAL) _UpdateFieldWithValue(fieldMap map[string]interface{}, key string, value interface{}) bool {
	if _, ok := fieldMap[key]; ok {
		return true
	}

	fieldName := utils.ParseFieldNameToUpper(key)
	logs.CtxTrace(b.ctx, "检查StarAuthorInfo的字段[%s]->[%s]是否存在", key, fieldName)
	if reflect.ValueOf(b.model).Elem().FieldByName(fieldName).IsValid() {
		fieldMap[key] = value
		return true
	} else {
		logs.CtxWarn(b.ctx, "StarAuthorInfo不存在字段[%s]->[%s]", key, fieldName)
		return false
	}
}

func (b *StarAuthorInfoDAL) _UpdateFieldType(fieldMap map[string]interface{}) {
	for key, value := range fieldMap {
		var newV interface{}
		switch value.(type) {
		case []byte:
			newV = string(value.([]byte))
			break
		default:
			continue
		}
		fieldMap[key] = newV
	}
}

func (b *StarAuthorInfoDAL) New(fieldMap map[string]interface{}) (*model.StarAuthorInfo, error) {
	b._UpdateFieldWithValue(fieldMap, "create_time", time.Now())
	b._UpdateFieldWithValue(fieldMap, "modify_time", time.Now())
	b._UpdateFieldType(fieldMap)
	_model, err := b.commonDao.New(fieldMap)
	if err != nil {
		return nil, exception.NewSystemError(b.ctx, err)
	}
	return _model, nil
}

func (b *StarAuthorInfoDAL) Update(updateModel *model.StarAuthorInfo, fieldMap map[string]interface{}) (*model.StarAuthorInfo, error) {
	b._UpdateFieldWithValue(fieldMap, "modify_time", time.Now())
	b._UpdateFieldType(fieldMap)
	err := utils.SetValueFromMap(b.ctx, updateModel, fieldMap)
	if err != nil {
		return nil, err
	}

	_model, err := b.commonDao.Update(updateModel, fieldMap)
	if err != nil {
		return nil, exception.NewSystemError(b.ctx, err)
	}
	return _model, nil
}

func (b *StarAuthorInfoDAL) Count(filter map[string]interface{}, options ...DalHelper.ModifyOptionFunc) (int64, error) {
	totalCnt, err := b.commonDao.Count(filter, options...)
	if err != nil {
		return 0, exception.NewSystemError(b.ctx, err)
	}
	return totalCnt, nil
}

func (b *StarAuthorInfoDAL) QueryWithPage(filter map[string]interface{}, page, limit int32, options ...DalHelper.ModifyOptionFunc) ([]*model.StarAuthorInfo, *PaginationGen.Pagination, error) {
	options = append(options, []DalHelper.ModifyOptionFunc{DalHelper.WithLimit(limit), DalHelper.WithOffset((page - 1) * limit)}...)
	ret, err := b.commonDao.Query(filter, options...)
	if err != nil {
		return nil, nil, exception.NewSystemError(b.ctx, err)
	}
	totalCnt, err := b.commonDao.Count(filter, options...)
	if err != nil {
		return nil, nil, exception.NewSystemError(b.ctx, err)
	}
	hasMore := page*limit < int32(totalCnt)
	pagination := &PaginationGen.Pagination{
		Limit:      limit,
		Page:       page,
		TotalCount: int32(totalCnt),
		HasMore:    &hasMore,
	}
	return ret, pagination, nil
}

func (b *StarAuthorInfoDAL) Exist(filter map[string]interface{}) (bool, error) {
	ret, err := b.QueryOne(filter, DalHelper.WithColumns([]string{"id"}))
	if err != nil || ret == nil {
		return false, exception.NewSystemError(b.ctx, err)
	}
	return true, nil
}

func (b *StarAuthorInfoDAL) Delete(deleteModel *model.StarAuthorInfo) (*model.StarAuthorInfo, error) {
	ret, err := b.Update(deleteModel, map[string]interface{}{"deleted": int8(1)})
	if err != nil || ret == nil {
		return nil, exception.NewSystemError(b.ctx, err)
	}
	return ret, nil
}

func (b *StarAuthorInfoDAL) Sum(filter map[string]interface{}, sumField string, options ...DalHelper.ModifyOptionFunc) (int64, error) {
	ret, err := b.commonDao.Sum(filter, sumField, options...)
	if err != nil {
		return 0, exception.NewSystemError(b.ctx, err)
	}
	return ret, nil
}
