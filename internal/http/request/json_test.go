package request

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeRejectsInvalidBodies(t *testing.T) {
	cases := []string{"", `null`, `[]`, `{"name":"x","unknown":1}`, `{"name":"x"} {}`, `{"name":`, `{"name":"` + strings.Repeat("x", MaxBodyBytes) + `"}`}
	for i, body := range cases {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var dst struct {
			Name string `json:"name"`
		}
		if err := Decode(r, &dst); err == nil {
			t.Fatalf("invalid body %d accepted", i)
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"valid"}`))
	var dst struct {
		Name string `json:"name"`
	}
	if err := Decode(r, &dst); err != nil || dst.Name != "valid" {
		t.Fatalf("valid decode: %v %v", dst, err)
	}
}
