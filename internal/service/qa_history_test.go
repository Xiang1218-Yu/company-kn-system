package service

import (
	"testing"

	"kn-system/internal/llm"
)

func TestBuildMessagesPreservesConversationHistoryOrder(t *testing.T) {
	svc := &QAService{}
	msgs := svc.buildMessages("第三个问题", "", [][2]string{
		{"第一个问题", "第一个回答"},
		{"第二个问题", "第二个回答"},
	})
	want := []llm.Message{
		{Role: "user", Content: "第一个问题"},
		{Role: "assistant", Content: "第一个回答"},
		{Role: "user", Content: "第二个问题"},
		{Role: "assistant", Content: "第二个回答"},
		{Role: "user", Content: "第三个问题"},
	}
	if len(msgs) < len(want) {
		t.Fatalf("buildMessages() returned %d messages, want at least %d", len(msgs), len(want))
	}
	got := msgs[len(msgs)-len(want):]
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}
