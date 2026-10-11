package settinggroup

import (
	"context"
	"strings"
	"testing"
)

func TestSettingGroupValidation(t *testing.T) {
	for _, req := range []CreateSettingGroupRequest{
		{Code: " ", Name: "Name"}, {Code: "CODE", Name: " "},
		{Code: strings.Repeat("x", 51), Name: "Name"}, {Code: "CODE", Name: strings.Repeat("é", 101)},
	} {
		if _, err := New(nil).Create(context.Background(), req); err == nil {
			t.Fatal("invalid fields accepted")
		}
	}
	req := CreateSettingGroupRequest{Code: " appearance ", Name: " Appearance "}
	if err := normalizeRequest(&req); err != nil {
		t.Fatal(err)
	}
	if req.Code != "APPEARANCE" || req.Name != "Appearance" {
		t.Fatalf("request=%+v", req)
	}
	if _, err := New(nil).GetByID(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid UUID accepted")
	}
}
