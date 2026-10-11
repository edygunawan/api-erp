package setting

import (
	"context"
	"strings"
	"testing"
)

const groupID = "00000000-0000-4000-8003-000000000001"

func text(value string) *string { return &value }
func TestSettingValueValidation(t *testing.T) {
	for _, test := range []struct {
		name, kind string
		value      *string
		valid      bool
	}{
		{"string whitespace", "STRING", text("  text  "), true},
		{"nullable value", "NUMBER", nil, true},
		{"number", "NUMBER", text(" -1.25e2 "), true},
		{"infinite number", "NUMBER", text("1e400"), false},
		{"non-number", "NUMBER", text("true"), false},
		{"invalid number syntax", "NUMBER", text("01"), false},
		{"nan", "NUMBER", text("NaN"), false},
		{"boolean", "BOOLEAN", text(" true "), true},
		{"false", "BOOLEAN", text("false"), true},
		{"boolean coercion", "BOOLEAN", text("1"), false},
		{"JSON object", "JSON", text(`{"enabled":true}`), true},
		{"JSON array", "JSON", text(`[1,null,"a"]`), true},
		{"broken JSON", "JSON", text(`{"a":}`), false},
		{"multiple JSON values", "JSON", text(`{} {}`), false},
		{"unknown data type", "DATE", text("today"), false},
		{"TEXT byte limit", "STRING", text(strings.Repeat("é", 32768)), false},
		{"invalid UTF8", "STRING", text(string([]byte{0xff})), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := CreateSettingRequest{GroupID: groupID, SettingKey: "example", DataType: test.kind, SettingValue: test.value}
			err := normalizeRequest(&req)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v err=%v", test.valid, err)
			}
			if test.valid && test.kind == "STRING" && *req.SettingValue != *test.value {
				t.Fatal("string whitespace changed")
			}
		})
	}
}
func TestSettingRequestNormalization(t *testing.T) {
	req := CreateSettingRequest{GroupID: " " + groupID + " ", SettingKey: " Color_Theme ", DataType: " string ", SettingValue: text("mono"), Description: text("  ")}
	if err := normalizeRequest(&req); err != nil {
		t.Fatal(err)
	}
	if req.GroupID != groupID || req.SettingKey != "color_theme" || req.DataType != "STRING" || req.Description != nil {
		t.Fatalf("request=%+v", req)
	}
	defaultReq := CreateSettingRequest{GroupID: groupID, SettingKey: "example"}
	if err := normalizeRequest(&defaultReq); err != nil || defaultReq.DataType != "STRING" {
		t.Fatalf("default=%+v err=%v", defaultReq, err)
	}
}
func TestInvalidSettingRequestStopsBeforeRepository(t *testing.T) {
	for _, req := range []CreateSettingRequest{
		{GroupID: "invalid", SettingKey: "key"},
		{GroupID: groupID, SettingKey: " "},
		{GroupID: groupID, SettingKey: strings.Repeat("a", 101)},
		{GroupID: groupID, SettingKey: "key", Description: text(strings.Repeat("a", 256))},
	} {
		if _, err := New(nil).Create(context.Background(), req); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
	if _, err := New(nil).GetByID(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid UUID accepted")
	}
	if err := New(nil).Delete(context.Background(), "invalid"); err == nil {
		t.Fatal("invalid UUID accepted")
	}
}
