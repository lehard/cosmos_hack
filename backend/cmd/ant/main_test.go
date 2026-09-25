package main

import "testing"

func TestParseRoles(t *testing.T) {
	if r, err := parseRoles("api"); err != nil || len(r) != 1 {
		t.Fatalf("api: %v %v", r, err)
	}
	if r, err := parseRoles(" api , api "); err != nil || len(r) != 1 {
		t.Fatalf("дубликат: %v %v", r, err)
	}
	for _, bad := range []string{"", "nope", "api,migrate"} {
		if _, err := parseRoles(bad); err == nil {
			t.Errorf("%q: ждали ошибку", bad)
		}
	}
}
