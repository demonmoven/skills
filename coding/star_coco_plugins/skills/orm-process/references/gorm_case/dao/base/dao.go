package base

import (
	"errors"
	"reflect"
	"strings"

	dbutil "code.byted.org/ad/star_dbutil"
	"code.byted.org/ad/star_dbutil/resolver"
	"code.byted.org/gopkg/context"

	"gorm.io/gorm"
)

type DAO struct {
	proxyDB   *gorm.DB
	tableName string
}

const (
	Deleted    = 1
	NotDeleted = 0
)

func NewDAO(tableName string) (*DAO, error) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return nil, errors.New("tableName cannot be blank")
	}
	return &DAO{
		tableName: tableName,
	}, nil
}

func NewDAOWithProxyDB(proxyDB *gorm.DB, tableName string) (*DAO, error) {
	if proxyDB == nil {
		return nil, errors.New("proxyDB cannot be nil")
	}
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return nil, errors.New("tableName cannot be blank")
	}
	return &DAO{
		proxyDB:   proxyDB,
		tableName: tableName,
	}, nil
}

func (dao *DAO) Table(ctx context.Context, opts ...DAOOption) *gorm.DB {
	var daoOpt daoOptions
	for _, opt := range opts {
		opt(&daoOpt)
	}
	db := dbutil.GetDB()
	if !daoOpt.readDB {
		db = db.Clauses(resolver.Write)
	}
	return db.WithContext(ctx).Table(dao.tableName)
}

func (dao *DAO) DoCreate(ctx context.Context, value interface{}, opts ...DAOOption) error {
	return dao.
		Table(ctx, opts...).
		Create(value).
		Error
}

func (dao *DAO) DoUpdateById(ctx context.Context, value interface{}, id int64, fields map[string]interface{},
	opts ...DAOOption) error {
	if len(fields) == 0 {
		return nil
	}
	return dao.
		Table(ctx, opts...).
		Model(value).
		Where("`id` = ?", id).
		Updates(fields).
		Error
}

func (dao *DAO) DoBatchUpdateById(ctx context.Context, value interface{}, ids []int64, fields map[string]interface{},
	opts ...DAOOption) error {
	if len(ids) == 0 || len(fields) == 0 {
		return nil
	}
	return dao.
		Table(ctx, opts...).
		Model(value).
		Where("`id` IN (?)", ids).
		Updates(fields).
		Error
}

func (dao *DAO) UpdateByWhere(ctx context.Context, filters map[string]interface{}, fields map[string]interface{}, opts ...DAOOption) error {
	db := dao.Table(ctx, opts...)
	db = DictToSQL(db, filters).Updates(fields)
	return db.Error
}

func (dao *DAO) DoGetById(ctx context.Context, value interface{}, id int64, opts ...DAOOption) error {
	db := dao.
		Table(ctx, opts...).
		Where("id = ?", id)
	db = WhereDeleted(value, db)
	return db.First(value).Error
}

func (dao *DAO) DoGetByIds(ctx context.Context, value interface{}, ids []int64, opts ...DAOOption) error {
	if len(ids) == 0 {
		return nil
	}
	db := dao.
		Table(ctx, opts...).
		Where("id in (?)", ids)
	db = WhereDeleted(value, db)
	return db.Find(value).Error
}

func (dao *DAO) GetOneByWhere(ctx context.Context, value interface{}, filters map[string]interface{}, opts ...DAOOption) error {
	db := dao.Table(ctx, opts...)
	db = DictToSQL(db, filters)
	db = WhereDeleted(value, db)
	return db.First(value).Error
}

func (dao *DAO) GetLastOneByWhere(ctx context.Context, value interface{}, filters map[string]interface{}, opts ...DAOOption) error {
	db := dao.Table(ctx, opts...)
	db = DictToSQL(db, filters)
	db = WhereDeleted(value, db)
	return db.Last(value).Error
}

func (dao *DAO) GetByWhere(ctx context.Context, value interface{}, filters map[string]interface{}, opts ...DAOOption) error {
	db := dao.Table(ctx, opts...)
	db = DictToSQL(db, filters)
	db = WhereDeleted(value, db)
	err := db.Find(value).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	return err
}

func (dao *DAO) GetByWhereWithDeleted(ctx context.Context, value interface{}, filters map[string]interface{}, opts ...DAOOption) error {
	db := dao.Table(ctx, opts...)
	db = DictToSQL(db, filters)
	err := db.Find(value).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	return err
}

func (dao *DAO) DeletedById(ctx context.Context, id int64, opts ...DAOOption) error {
	return dao.Table(ctx, opts...).
		Where("id = ?", id).
		Updates(map[string]interface{}{"deleted": Deleted}).
		Error
}

//以下基于proxyDB调用

func (dao *DAO) DoDeletedById(ctx context.Context, id int64, opts ...DAOOption) error {
	return dao.proxyDB.WithContext(ctx).Table(dao.tableName).
		Where("id = ?", id).Updates(map[string]interface{}{"deleted": Deleted}).Error
}

func (dao *DAO) QueryByIds(ctx context.Context, value interface{}, ids []int64) error {
	return dao.proxyDB.WithContext(ctx).Table(dao.tableName).Where("id in (?) and deleted = ? ", ids, NotDeleted).
		Find(value).Error
}

func (dao *DAO) DoUpdateByIdV1(ctx context.Context, id int64, fields map[string]interface{},
	opts ...DAOOption) error {
	if len(fields) == 0 {
		return nil
	}
	return dao.proxyDB.
		Table(dao.tableName).
		Where("`id` = ?", id).
		Updates(fields).
		Error
}

func (dao *DAO) DoCreateV1(ctx context.Context, value interface{}, opts ...DAOOption) error {
	return dao.proxyDB.
		Table(dao.tableName).
		Create(value).
		Error
}

func WhereDeleted(value interface{}, db *gorm.DB) *gorm.DB {

	reflectValue := reflect.ValueOf(value)
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	if kind := reflectValue.Kind(); kind == reflect.Slice {
		reflectType := reflectValue.Type().Elem()
		for reflectType.Kind() == reflect.Ptr {
			reflectType = reflectType.Elem()
		}
		if _, ok := reflectType.FieldByName("Deleted"); ok {
			return db.Where("deleted = 0")
		}
	} else if reflectValue.FieldByName("Deleted").IsValid() {
		return db.Where("deleted = 0")
	}
	return db
}

type daoOptions struct {
	readDB bool
}

type DAOOption func(*daoOptions)

// 默认是写库
func WithReadDB() DAOOption {
	return func(o *daoOptions) {
		o.readDB = true
	}
}

func DictToSQL(db *gorm.DB, filter map[string]interface{}) *gorm.DB {
	for k, v := range filter {
		if reflect.TypeOf(v).Kind() == reflect.Array && reflect.ValueOf(v).Len() > 0 {
			db = db.Where(k+" IN (?)", v)
		} else if reflect.TypeOf(v).Kind() == reflect.Slice && reflect.ValueOf(v).Len() > 0 {
			db = db.Where(k+" IN (?)", v)
		} else {
			db = db.Where(k+" = ?", v)
		}
	}
	return db
}
