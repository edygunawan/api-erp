package setting

import (
	"context"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/golang-pkgs/errors"
)

type usecase struct{ repo Repository }

func New(repo Repository) Usecase { return &usecase{repo: repo} }
func (u *usecase) Create(ctx context.Context, req CreateSettingRequest) (*SettingResponse, error) {
	if err := normalizeRequest(&req); err != nil {
		return nil, err
	}
	id, err := helper.NewUUID()
	if err != nil {
		return nil, err
	}
	item := &domain.Setting{ID: id, GroupID: req.GroupID, SettingKey: req.SettingKey, SettingValue: req.SettingValue, DataType: req.DataType, Description: req.Description}
	if err := u.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return u.GetByID(ctx, id)
}
func (u *usecase) Update(ctx context.Context, id string, req UpdateSettingRequest) (*SettingResponse, error) {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "invalid setting id")
	}
	if err := normalizeRequest(&req); err != nil {
		return nil, err
	}
	item, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New(errors.NotFound, "setting not found")
	}
	item.GroupID = req.GroupID
	item.SettingKey = req.SettingKey
	item.SettingValue = req.SettingValue
	item.DataType = req.DataType
	item.Description = req.Description
	if err := u.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return u.GetByID(ctx, id)
}
func (u *usecase) GetByID(ctx context.Context, id string) (*SettingResponse, error) {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "invalid setting id")
	}
	item, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New(errors.NotFound, "setting not found")
	}
	res := settingResponse(*item)
	return &res, nil
}
func (u *usecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return errors.New(errors.Invalid, "invalid setting id")
	}
	return u.repo.Delete(ctx, id)
}
func (u *usecase) List(ctx context.Context, options query.Options) (query.Page[SettingResponse], error) {
	options, err := options.Normalize()
	if err != nil {
		return query.Page[SettingResponse]{}, errors.Wrap(errors.Invalid, err.Error(), err)
	}
	if err := normalizeFilters(&options); err != nil {
		return query.Page[SettingResponse]{}, err
	}
	items, total, err := u.repo.List(ctx, options)
	if err != nil {
		return query.Page[SettingResponse]{}, err
	}
	res := make([]SettingResponse, len(items))
	for i, item := range items {
		res[i] = settingResponse(item)
	}
	return query.NewPage(res, options, total), nil
}
