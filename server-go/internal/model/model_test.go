package model

import "testing"

func TestTagsRoundtrip(t *testing.T) {
	in := []string{"a", "b", "我的散帅时刻"}
	j := TagsToJSON(in)
	out := ParseTags(j)
	if len(out) != 3 {
		t.Fatalf("标签数 got %d want 3", len(out))
	}
	if out[2] != "我的散帅时刻" {
		t.Fatalf("中文标签丢失: %q", out[2])
	}
	if len(ParseTags("")) != 0 {
		t.Fatal("空字符串应返回 []")
	}
	if len(ParseTags("not-json")) != 0 {
		t.Fatal("非法 JSON 应返回 []")
	}
}
