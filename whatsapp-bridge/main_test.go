package main

import (
	"database/sql"
	"strings"
	"testing"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestMigrateLIDs(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // keep the single in-memory DB
	if _, err := db.Exec(messagesSchema); err != nil {
		t.Fatal(err)
	}

	pnByLID := map[string]string{"111": "5511999990001", "222": "5511999990002"}
	toPN := func(j types.JID) types.JID {
		if pn, ok := pnByLID[j.User]; ok && j.Server == types.HiddenUserServer {
			return types.NewJID(pn, types.DefaultUserServer)
		}
		return j
	}
	nameFor := func(j types.JID) string { return "Name " + j.User }

	mustExec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	// 111@lid: only LID row exists. 222@lid: PN row already exists too. 333@lid: no mapping.
	mustExec(`INSERT INTO chats VALUES ('111@lid','111','2026-10-01 10:00:00'),
		('222@lid','222','2026-10-02 10:00:00'), ('5511999990002@s.whatsapp.net','Existing','2026-10-01 09:00:00'),
		('333@lid','333','2026-10-01 10:00:00'), ('g1@g.us','Group','2026-10-01 10:00:00')`)
	// 111@lid's chat label must follow it; 444@lid only exists in chat_labels.
	pnByLID["444"] = "5511999990004"
	mustExec(`INSERT INTO chat_labels VALUES ('111@lid','L1'), ('444@lid','L1'), ('333@lid','L1')`)
	mustExec(`INSERT INTO messages (id, chat_jid, sender, content) VALUES
		('m1','111@lid','111','a'), ('m2','222@lid','222','b'), ('m3','5511999990002@s.whatsapp.net','222@lid','c'),
		('m4','333@lid','333','d'), ('m5','g1@g.us','111','e'), ('m6','g1@g.us','5511888880000','f')`)

	if err := migrateLIDs(db, toPN, nameFor); err != nil {
		t.Fatal(err)
	}

	chats := map[string]string{}
	rows, _ := db.Query("SELECT jid, name || '|' || last_message_time FROM chats")
	for rows.Next() {
		var j, v string
		rows.Scan(&j, &v)
		chats[j] = v
	}
	rows.Close()
	want := map[string]string{
		"5511999990001@s.whatsapp.net": "Name 5511999990001|2026-10-01 10:00:00", // moved, renamed
		"5511999990002@s.whatsapp.net": "Existing|2026-10-02 10:00:00",           // merged, keeps name, newest time
		"333@lid":                      "333|2026-10-01 10:00:00",                // unmapped, untouched
		"g1@g.us":                      "Group|2026-10-01 10:00:00",
	}
	if len(chats) != len(want) {
		t.Fatalf("chats = %v, want %v", chats, want)
	}
	for j, v := range want {
		if chats[j] != v {
			t.Errorf("chat %s = %q, want %q", j, chats[j], v)
		}
	}

	var labeled []string
	rows, _ = db.Query("SELECT chat_jid FROM chat_labels ORDER BY chat_jid")
	for rows.Next() {
		var j string
		rows.Scan(&j)
		labeled = append(labeled, j)
	}
	rows.Close()
	if got, want := strings.Join(labeled, ","), "333@lid,5511999990001@s.whatsapp.net,5511999990004@s.whatsapp.net"; got != want {
		t.Errorf("chat_labels = %s, want %s", got, want)
	}

	msgs := map[string]string{}
	rows, _ = db.Query("SELECT id, chat_jid || '|' || sender FROM messages")
	for rows.Next() {
		var id, v string
		rows.Scan(&id, &v)
		msgs[id] = v
	}
	rows.Close()
	wantMsgs := map[string]string{
		"m1": "5511999990001@s.whatsapp.net|5511999990001",
		"m2": "5511999990002@s.whatsapp.net|5511999990002",
		"m3": "5511999990002@s.whatsapp.net|5511999990002@s.whatsapp.net",
		"m4": "333@lid|333",
		"m5": "g1@g.us|5511999990001",
		"m6": "g1@g.us|5511888880000",
	}
	for id, v := range wantMsgs {
		if msgs[id] != v {
			t.Errorf("message %s = %q, want %q", id, msgs[id], v)
		}
	}
}

func TestExtractTextContentCaptions(t *testing.T) {
	cases := map[string]*waProto.Message{
		"img": {ImageMessage: &waProto.ImageMessage{Caption: proto.String("img")}},
		"vid": {VideoMessage: &waProto.VideoMessage{Caption: proto.String("vid")}},
		"doc": {DocumentMessage: &waProto.DocumentMessage{Caption: proto.String("doc")}},
		"":    {AudioMessage: &waProto.AudioMessage{}},
	}
	for want, msg := range cases {
		if got := extractTextContent(msg); got != want {
			t.Errorf("extractTextContent = %q, want %q", got, want)
		}
	}
}

func TestLabels(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(messagesSchema); err != nil {
		t.Fatal(err)
	}
	s := &MessageStore{db: db}
	count := func(q string) (n int) { db.QueryRow(q).Scan(&n); return }

	s.StoreLabel("1", "Cardápio Zap", 3, "CUSTOM", false)
	s.StoreLabel("2", "GrupoZap", 5, "CUSTOM", false)
	s.StoreChatLabel("a@s.whatsapp.net", "1", true)
	s.StoreChatLabel("a@s.whatsapp.net", "1", true) // duplicate is a no-op
	s.StoreChatLabel("b@s.whatsapp.net", "2", true)
	if n := count("SELECT COUNT(*) FROM chat_labels"); n != 2 {
		t.Fatalf("chat_labels = %d, want 2", n)
	}
	s.StoreChatLabel("a@s.whatsapp.net", "1", false)
	if n := count("SELECT COUNT(*) FROM chat_labels WHERE label_id = '1'"); n != 0 {
		t.Errorf("unlabel left %d rows", n)
	}
	s.StoreLabel("2", "", 0, "", true)
	if n := count("SELECT COUNT(*) FROM labels") + count("SELECT COUNT(*) FROM chat_labels"); n != 1 {
		t.Errorf("after delete labels+chat_labels = %d, want 1 (label 1 only)", n)
	}
}

func TestExtractDirectPathFromURL(t *testing.T) {
	got := extractDirectPathFromURL("https://mmg.whatsapp.net/v/t62.7117-24/1_n.enc?ccb=11-4&oh=X&oe=Y&_nc_sid=5e03e0&mms3=true")
	if want := "/v/t62.7117-24/1_n.enc?ccb=11-4&oh=X&oe=Y&_nc_sid=5e03e0"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenameSelfNamedChats(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(messagesSchema); err != nil {
		t.Fatal(err)
	}
	// "me" is our own number: a, b were misnamed; self chat, group and a real name stay.
	if _, err := db.Exec(`INSERT INTO chats (jid, name) VALUES ('a@s.whatsapp.net','me'), ('b@s.whatsapp.net','me'),
		('me@s.whatsapp.net','me'), ('g@g.us','me'), ('c@s.whatsapp.net','Carol')`); err != nil {
		t.Fatal(err)
	}
	nameFor := func(j types.JID) string {
		if j.User == "a" {
			return "Alice"
		}
		return j.User
	}
	if err := renameSelfNamedChats(db, "me", nameFor); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"a@s.whatsapp.net": "Alice", "b@s.whatsapp.net": "b", "me@s.whatsapp.net": "me", "g@g.us": "me", "c@s.whatsapp.net": "Carol"}
	for j, w := range want {
		var got string
		db.QueryRow("SELECT name FROM chats WHERE jid = ?", j).Scan(&got)
		if got != w {
			t.Errorf("%s = %q, want %q", j, got, w)
		}
	}
}

func TestHistoryUnwrap(t *testing.T) {
	raw := &waProto.Message{EphemeralMessage: &waProto.FutureProofMessage{Message: &waProto.Message{
		DocumentWithCaptionMessage: &waProto.FutureProofMessage{Message: &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{Caption: proto.String("cap"), FileName: proto.String("f.pdf")}}}}}}
	m := (&events.Message{RawMessage: raw}).UnwrapRaw().Message
	if got := extractTextContent(m); got != "cap" {
		t.Errorf("caption = %q", got)
	}
	if mt, fn, _, _, _, _, _ := extractMediaInfo(m); mt != "document" || fn != "f.pdf" {
		t.Errorf("media = %q %q", mt, fn)
	}
	if (&events.Message{}).UnwrapRaw().Message != nil {
		t.Error("nil raw message should unwrap to nil")
	}
}
