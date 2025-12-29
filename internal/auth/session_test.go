package auth

import (
	"sync"
	"testing"
	"time"
)

func TestSessionStore_Create(t *testing.T) {
	store := NewInMemorySessionStore()

	userID := int64(123)
	ttl := 1 * time.Hour

	sessionID, err := store.Create(userID, ttl)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if sessionID == "" {
		t.Error("expected session ID to be non-empty")
	}

	// セッションIDが十分な長さであることを確認（Base64で44文字程度）
	if len(sessionID) < 40 {
		t.Errorf("session ID too short: %d characters", len(sessionID))
	}

	// 作成したセッションを取得できることを確認
	session, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if session == nil {
		t.Fatal("expected session to exist")
	}

	if session.UserID != userID {
		t.Errorf("expected user ID %d, got %d", userID, session.UserID)
	}
}

func TestSessionStore_Get(t *testing.T) {
	store := NewInMemorySessionStore()

	userID := int64(456)
	ttl := 1 * time.Hour

	sessionID, err := store.Create(userID, ttl)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// セッションを取得
	session, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if session == nil {
		t.Fatal("expected session to exist")
	}

	if session.UserID != userID {
		t.Errorf("expected user ID %d, got %d", userID, session.UserID)
	}

	// 有効期限が正しく設定されていることを確認
	expectedExpiry := time.Now().Add(ttl)
	if session.ExpiresAt.Before(expectedExpiry.Add(-5 * time.Second)) {
		t.Error("expiry time is too early")
	}
	if session.ExpiresAt.After(expectedExpiry.Add(5 * time.Second)) {
		t.Error("expiry time is too late")
	}
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	store := NewInMemorySessionStore()

	// 存在しないセッションIDで取得
	session, err := store.Get("nonexistent-session-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session != nil {
		t.Error("expected session to be nil for nonexistent ID")
	}
}

func TestSessionStore_Get_Expired(t *testing.T) {
	store := NewInMemorySessionStore()

	userID := int64(789)
	ttl := 100 * time.Millisecond // 100ミリ秒で期限切れ

	sessionID, err := store.Create(userID, ttl)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// セッションが存在することを確認
	session, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session == nil {
		t.Fatal("expected session to exist")
	}

	// 期限切れまで待つ
	time.Sleep(150 * time.Millisecond)

	// 期限切れセッションを取得（自動削除されるはず）
	session, err = store.Get(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session != nil {
		t.Error("expected expired session to be nil")
	}

	// 再度取得しても存在しないことを確認（削除されている）
	session, err = store.Get(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session != nil {
		t.Error("expected session to remain deleted")
	}
}

func TestSessionStore_Delete(t *testing.T) {
	store := NewInMemorySessionStore()

	userID := int64(111)
	ttl := 1 * time.Hour

	sessionID, err := store.Create(userID, ttl)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// セッションが存在することを確認
	session, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session == nil {
		t.Fatal("expected session to exist")
	}

	// セッションを削除
	err = store.Delete(sessionID)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// 削除後に取得できないことを確認
	session, err = store.Get(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session != nil {
		t.Error("expected session to be deleted")
	}
}

func TestSessionStore_CleanupExpired(t *testing.T) {
	store := NewInMemorySessionStore()

	// 3つのセッションを作成
	// セッション1: すぐに期限切れ
	sessionID1, _ := store.Create(1, 50*time.Millisecond)

	// セッション2: 有効
	sessionID2, _ := store.Create(2, 10*time.Hour)

	// セッション3: すぐに期限切れ
	sessionID3, _ := store.Create(3, 50*time.Millisecond)

	// 期限切れまで待つ
	time.Sleep(100 * time.Millisecond)

	// クリーンアップ実行
	store.CleanupExpired()

	// セッション1と3は削除されているはず
	session1, _ := store.Get(sessionID1)
	if session1 != nil {
		t.Error("expected session 1 to be cleaned up")
	}

	session3, _ := store.Get(sessionID3)
	if session3 != nil {
		t.Error("expected session 3 to be cleaned up")
	}

	// セッション2は残っているはず
	session2, _ := store.Get(sessionID2)
	if session2 == nil {
		t.Error("expected session 2 to remain")
	}
	if session2 != nil && session2.UserID != 2 {
		t.Errorf("expected user ID 2, got %d", session2.UserID)
	}
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemorySessionStore()

	// 並行アクセスのテスト
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 100個のgoroutineが同時にセッションを作成
	sessionIDs := make([]string, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			sessionID, err := store.Create(int64(index), 1*time.Hour)
			if err != nil {
				t.Errorf("goroutine %d: failed to create session: %v", index, err)
				return
			}
			sessionIDs[index] = sessionID
		}(i)
	}

	wg.Wait()

	// 全てのセッションが取得できることを確認
	for i := 0; i < numGoroutines; i++ {
		session, err := store.Get(sessionIDs[i])
		if err != nil {
			t.Errorf("failed to get session %d: %v", i, err)
		}
		if session == nil {
			t.Errorf("session %d not found", i)
		}
		if session != nil && session.UserID != int64(i) {
			t.Errorf("session %d: expected user ID %d, got %d", i, i, session.UserID)
		}
	}
}

func TestSessionStore_ConcurrentReadWrite(t *testing.T) {
	store := NewInMemorySessionStore()

	// 初期セッションを作成
	sessionID, _ := store.Create(999, 1*time.Hour)

	var wg sync.WaitGroup
	const numReaders = 50
	const numWriters = 10

	// 50個のgoroutineが同時に読み取り
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = store.Get(sessionID)
			}
		}()
	}

	// 10個のgoroutineが同時に書き込み
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(index int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = store.Create(int64(index*100+j), 1*time.Hour)
			}
		}(i)
	}

	// データ競合やデッドロックが発生しないことを確認
	wg.Wait()
}

func TestGenerateSessionID(t *testing.T) {
	// セッションIDを複数生成して、重複がないことを確認
	ids := make(map[string]bool)
	const numIDs = 1000

	for i := 0; i < numIDs; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("failed to generate session ID: %v", err)
		}

		if ids[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}
		ids[id] = true

		if len(id) < 40 {
			t.Errorf("session ID too short: %d characters", len(id))
		}
	}

	if len(ids) != numIDs {
		t.Errorf("expected %d unique IDs, got %d", numIDs, len(ids))
	}
}
