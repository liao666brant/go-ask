package data

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOfflineLookup(t *testing.T) {
	e, ok := Lookup("  HELLO ")
	if !ok || e.Word != "hello" || e.Phonetic == "" || !strings.Contains(e.Translation, "喂") {
		t.Fatalf("hello: %+v", e)
	}
	e, ok = Lookup("translate")
	if !ok || !strings.Contains(e.Translation, "翻译") || !strings.Contains(e.POS, "vt.") || strings.Contains(e.Translation, `\n`) {
		t.Fatalf("translate: %+v", e)
	}
	if _, ok = Lookup("not-a-real-goask-word"); ok {
		t.Fatal("unexpected dictionary hit")
	}
}

func TestConversationLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.NewConversation("What is  ss?")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.AddMessage(id, "user", "What is ss?"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.AddMessage(id, "assistant", "端口列表 100%_"); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, term := range []string{"端口", "%_", "WHAT", ""} {
		entries, err := store.Conversations(term)
		if err != nil || len(entries) != 1 || entries[0].Title != "What is ss?" {
			t.Fatalf("find %q: %+v %v", term, entries, err)
		}
	}
	msgs, err := store.Messages(id)
	if err != nil || len(msgs) != 2 || msgs[0].Role != "user" || msgs[1].Content != "端口列表 100%_" {
		t.Fatalf("restore: %+v %v", msgs, err)
	}
	if err = store.DeleteConversation(id); err != nil {
		t.Fatal(err)
	}
	msgs, err = store.Messages(id)
	if err != nil || len(msgs) != 0 {
		t.Fatalf("delete left messages: %+v %v", msgs, err)
	}
	if _, err = store.AddMessage(id, "assistant", "stale"); err == nil {
		t.Fatal("accepted a message for a deleted conversation")
	}
}
