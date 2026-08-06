package dao

import (
	"code.byted.org/ad/star_control/dal/dao/base"
	"code.byted.org/ad/star_control/dal/model"
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StarPlatformPermissionDao struct {
	*base.DAO
}

func (d *StarPlatformPermissionDao) CreateOrUpdate(ctx context.Context, db *gorm.DB, permission *model.StarPlatformPermission) error {
	return db.WithContext(ctx).
		Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"deleted", "modify_time"})}).
		Where(map[string]interface{}{
			"domain_type":           permission.DomainType,
			"domain_biz_identifier": permission.DomainBizIdentifier,
			"operation":             permission.Operation,
		}).
		FirstOrCreate(permission).Error
}

func (d *StarPlatformPermissionDao) QueryByIds(ctx context.Context, db *gorm.DB, ids []int64) ([]*model.StarPlatformPermission, error) {
	permissions := []*model.StarPlatformPermission{}
	d.DAO, _ = base.NewDAOWithProxyDB(db, (&model.StarPlatformPermission{}).TableName())
	err := d.DAO.QueryByIds(ctx, &permissions, ids)
	if err != nil {
		return permissions, err
	}
	return permissions, nil
}

func (d *StarPlatformPermissionDao) QueryByIdsAndDomain(ctx context.Context, db *gorm.DB, ids []int64, domainType int64, domainBizIdentifier string) ([]*model.StarPlatformPermission, error) {
	permissions := []*model.StarPlatformPermission{}
	err := db.WithContext(ctx).Where("id in (?) and deleted = ?", ids, base.NotDeleted).
		Where("domain_type = ? and domain_biz_identifier = ?", domainType, domainBizIdentifier).
		Find(&permissions).
		Error
	if err != nil {
		return permissions, err
	}
	return permissions, nil
}

func (d *StarPlatformPermissionDao) QueryByIdsAndDomains(ctx context.Context, db *gorm.DB, ids []int64, domainTypes []int64, domainBizIdentifiers []string) ([]*model.StarPlatformPermission, error) {
	permissions := []*model.StarPlatformPermission{}
	err := db.WithContext(ctx).Where("id in (?) and deleted = ?", ids, base.NotDeleted).
		Where("domain_type in (?) and domain_biz_identifier in (?)", domainTypes, domainBizIdentifiers).
		Find(&permissions).
		Error
	if err != nil {
		return permissions, err
	}
	return permissions, nil
}
