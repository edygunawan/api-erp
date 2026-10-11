package settinggroup

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
func (u *usecase) Create(ctx context.Context, req CreateSettingGroupRequest) (*SettingGroupResponse, error) {
	if err := normalizeRequest(&req); err != nil {
		return nil, err
	}
	id, err := helper.NewUUID()
	if err != nil {
		return nil, err
	}
	item := &domain.SettingGroup{ID: id, Code: req.Code, Name: req.Name}
	if err := u.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return u.GetByID(ctx, id)
}
func (u *usecase) Update(ctx context.Context, id string, req UpdateSettingGroupRequest) (*SettingGroupResponse, error) {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "invalid setting group id")
	}
	if err := normalizeRequest(&req); err != nil {
		return nil, err
	}
	item, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New(errors.NotFound, "setting group not found")
	}
	item.Code = req.Code
	item.Name = req.Name
	if err := u.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return u.GetByID(ctx, id)
}
func (u *usecase) GetByID(ctx context.Context, id string) (*SettingGroupResponse, error) {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "invalid setting group id")
	}
	item, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New(errors.NotFound, "setting group not found")
	}
	res := SettingGroupResponse(*item)
	return &res, nil
}
func (u *usecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return errors.New(errors.Invalid, "invalid setting group id")
	}
	return u.repo.Delete(ctx, id)
}
func (u *usecase) List(ctx context.Context, options query.Options) (query.Page[SettingGroupResponse], error) {
	options, err := options.Normalize()
	if err != nil {
		return query.Page[SettingGroupResponse]{}, errors.Wrap(errors.Invalid, err.Error(), err)
	}
	if err := normalizeFilters(&options); err != nil {
		return query.Page[SettingGroupResponse]{}, err
	}
	items, total, err := u.repo.List(ctx, options)
	if err != nil {
		return query.Page[SettingGroupResponse]{}, err
	}
	res := make([]SettingGroupResponse, len(items))
	for i, item := range items {
		res[i] = SettingGroupResponse(item)
	}
	return query.NewPage(res, options, total), nil
}
