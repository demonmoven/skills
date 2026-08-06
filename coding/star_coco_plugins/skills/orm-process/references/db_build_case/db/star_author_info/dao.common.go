package star_author_info

import (
	"code.byted.org/ad/star_goauthor/infrastructure/mysql/model"
	GoCommon "code.byted.org/ad/star_gocommon/common"
	GoCommonConsts "code.byted.org/ad/star_gocommon/consts"
	DalHelper "code.byted.org/ad/star_gocommon/utils/dal_helper"
	"code.byted.org/gopkg/logs"
	"database/sql"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"strings"
)

type CommonDao struct {
	handler *gorm.DB
	defaultContainDeleted bool
	defaultOrderBy string
}

func NewCommonDao(handler *gorm.DB) CommonDao{
	return CommonDao{
		handler: handler,
		defaultContainDeleted: false,
		defaultOrderBy: "-id",
	}
}

func (interstruct *CommonDao) Query(filter map[string]interface{}, options ...DalHelper.ModifyOptionFunc) ([]*model.StarAuthorInfo, error) {
	_result, _retErr := func() ([]*model.StarAuthorInfo, error) {
		_db := interstruct.handler

		queryOption := DalHelper.QueryOption{OrderBy: &interstruct.defaultOrderBy, ContainDeleted: &interstruct.defaultContainDeleted}
		for _, fn := range options {
			fn(&queryOption)
		}
		if !reflect.ValueOf(queryOption.Columns).IsNil() {
			_db = _db.Select(*queryOption.Columns)
		}

		emptyModel := model.StarAuthorInfo{}
		_db, needExc, err := DalHelper.DictToSQL(_db, filter, emptyModel)
		if err != nil{
			return nil, err
		}
		if !needExc {
			return make([]*model.StarAuthorInfo, 0), nil
		}

		if reflect.ValueOf(emptyModel).FieldByName("Deleted").IsValid() && !*queryOption.ContainDeleted {
			_db = _db.Where("deleted = 0")
		}
		if !reflect.ValueOf(queryOption.OrderByList).IsNil() {
			for _, _orderBy := range queryOption.OrderByList {
				if strings.HasPrefix(_orderBy, "-"){
					_db = _db.Order(strings.Trim(_orderBy, "-")+" desc")
				} else {
					_db = _db.Order(_orderBy)
				}
			}
		} else {
			if strings.HasPrefix(*queryOption.OrderBy, "-"){
				_db = _db.Order(strings.Trim(*queryOption.OrderBy, "-")+" desc")
			} else {
				_db = _db.Order(*queryOption.OrderBy)
			}
		}
		if !reflect.ValueOf(queryOption.Offset).IsNil() {
			_db = _db.Offset(int(*queryOption.Offset))
		}
		if !reflect.ValueOf(queryOption.Limit).IsNil() {
			_db = _db.Limit(int(*queryOption.Limit))
		}

		var _ret2 []model.StarAuthorInfo
		_db = _db.Find(&_ret2)
		if _db.Error != nil {
			if _db.Error != gorm.ErrRecordNotFound && _db.Error != sql.ErrNoRows {
				logs.Errorf("Scan occur error: %s", _db.Error.Error())
			}
			if _db.Error == gorm.ErrRecordNotFound || _db.Error == sql.ErrNoRows{
			    return nil, nil
			}
			return nil, _db.Error
		}
		var ret3 []*model.StarAuthorInfo
		for i := range _ret2 {
			ret3 = append(ret3, &_ret2[i])
		}
		return ret3, nil
	}()
	return _result, _retErr
}


func (interstruct *CommonDao) New(fieldMap map[string]interface{}) (*model.StarAuthorInfo, error) {

	_result, _retErr := func() (*model.StarAuthorInfo, error) {
		_db := interstruct.handler
		emptyModel := model.StarAuthorInfo{}

		var validFieldMap map[string]interface{}
		validFieldMap = make(map[string]interface{})
		for key, value := range(fieldMap){
			camelKey := DalHelper.ToCamel(key)
			if reflect.ValueOf(emptyModel).FieldByName(camelKey).IsValid() {
				validFieldMap[key] = value
			} else {
				return nil, GoCommonConsts.GenNonExistentColumnError(key)
			}
		}
		_, exists := validFieldMap["id"]
		if !exists {
		    //id := rand.Int63()
			id, err := GoCommon.GetGlobalId()
			if err != nil{
				return nil, err
			}
			validFieldMap["id"] = id
		}

		jsonBody, err := json.Marshal(validFieldMap)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(jsonBody, &emptyModel); err != nil {
			return nil, err
		}
		_tdb := _db.Create(&emptyModel)
		if _tdb.Error != nil {
			return nil, _tdb.Error
		}
		return &emptyModel, nil
	}()
	return _result, _retErr
}

// Update(updateModel *model.StarAuthorInfo, fieldMap map[string]interface{}) (*model.StarAuthorInfo, error)
func (interstruct *CommonDao) Update(updateModel *model.StarAuthorInfo, fieldMap map[string]interface{}) (*model.StarAuthorInfo, error) {
	_result, _retErr := func() (*model.StarAuthorInfo, error) {
		_db := interstruct.handler
		var validFieldMap map[string]interface{}
		validFieldMap = make(map[string]interface{})
		for key, value := range fieldMap{
			camelKey := DalHelper.ToCamel(key)
			if reflect.ValueOf(*updateModel).FieldByName(camelKey).IsValid() {
				validFieldMap[key] = value
			} else {
				return nil, GoCommonConsts.GenNonExistentColumnError(key)
			}
		}
		_tdb := _db.Model(*updateModel).Updates(validFieldMap)
		if _tdb.Error != nil {
			return nil, _tdb.Error
		}
		return updateModel, nil
	}()
	return _result, _retErr
}


func (interstruct *CommonDao) Count(filter map[string]interface{}, options ...DalHelper.ModifyOptionFunc) (int64, error){
	_db := interstruct.handler
	_db = _db.Table("star_author_info")
	_db, needExc, err := DalHelper.DictToSQL(_db, filter, model.StarAuthorInfo{})
	if err != nil{
		return 0, err
	}
	if !needExc {
		return 0, nil
	}

    emptyModel := model.StarAuthorInfo{}
	queryOption := DalHelper.QueryOption{ContainDeleted: &interstruct.defaultContainDeleted}
	for _, fn := range options {
		fn(&queryOption)
	}
	if reflect.ValueOf(emptyModel).FieldByName("Deleted").IsValid() && !*queryOption.ContainDeleted {
		_db = _db.Where("deleted = 0")
	}

	var totalCnt int64
	_db = _db.Count(&totalCnt)
	if _db.Error != nil{
		logs.Errorf("Scan occur error: %s", _db.Error.Error())
		return 0, _db.Error
	}
	return totalCnt, nil
}


func (interstruct *CommonDao) Sum(filter map[string]interface{}, sumField string, options ...DalHelper.ModifyOptionFunc) (int64, error){
	emptyModel := model.StarAuthorInfo{}
	sumFieldCamelKey := DalHelper.ToCamel(sumField)
	if !reflect.ValueOf(emptyModel).FieldByName(sumFieldCamelKey).IsValid() {
		return 0, GoCommonConsts.GenNonExistentColumnError(sumField)
	}
	_db := interstruct.handler
	_db = _db.Table("star_author_info")
	_db = _db.Select(fmt.Sprintf("sum(%s) as total", sumField))
	_db, needExc, err := DalHelper.DictToSQL(_db, filter, emptyModel)
	if err != nil{
		return 0, err
	}
	if !needExc {
		return 0, nil
	}

	queryOption := DalHelper.QueryOption{ContainDeleted: &interstruct.defaultContainDeleted}
	for _, fn := range options {
		fn(&queryOption)
	}
	if reflect.ValueOf(emptyModel).FieldByName("Deleted").IsValid() && !*queryOption.ContainDeleted {
		_db = _db.Where("deleted = 0")
	}

	type SumRes struct {
		Total int64
	}
	var sum SumRes
	 _db = _db.Find(&sum)
	if _db.Error != nil{
        logs.Errorf("Scan occur error: %s", _db.Error.Error())
		return 0, _db.Error
	}
	return sum.Total, nil
}

