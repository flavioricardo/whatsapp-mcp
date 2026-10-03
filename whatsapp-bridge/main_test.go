package main

import (
	"database/sql"
	"testing"

	"go.mau.fi/whatsmeow/types"
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
