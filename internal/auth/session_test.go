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

	sessionID, err := store.Create(userID, ttl, SessionMeta{})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if sessionID == "" {
		t.Error("expected session ID to be non-empty")
	}

	// Verify that the session ID has sufficient length (approximately 44 characters in Base64)
	if len(sessionID) < 40 {
		t.Errorf("session ID too short: %d characters", len(sessionID))
	}

	// Verify that the created session can be retrieved
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

	sessionID, err := store.Create(userID, ttl, SessionMeta{})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Get the session
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

	// Verify that the expiry time is set correctly
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

	// Retrieve with a non-existent session ID
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
	ttl := 100 * time.Millisecond // Expires in 100 milliseconds

	sessionID, err := store.Create(userID, ttl, SessionMeta{})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify that the session exists
	session, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session == nil {
		t.Fatal("expected session to exist")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get the expired session (should be automatically deleted)
	session, err = store.Get(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session != nil {
		t.Error("expected expired session to be nil")
	}

	// Verify it still doesn't exist when retrieved again (deleted)
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

	sessionID, err := store.Create(userID, ttl, SessionMeta{})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify that the session exists
	session, err := store.Get(sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if session == nil {
		t.Fatal("expected session to exist")
	}

	// Delete the session
	err = store.Delete(sessionID)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify that the session cannot be retrieved after deletion
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

	// Create 3 sessions
	// Session 1: Expires immediately
	sessionID1, _ := store.Create(1, 50*time.Millisecond, SessionMeta{})

	// Session 2: Valid
	sessionID2, _ := store.Create(2, 10*time.Hour, SessionMeta{})

	// Session 3: Expires immediately
	sessionID3, _ := store.Create(3, 50*time.Millisecond, SessionMeta{})

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Run cleanup
	store.CleanupExpired()

	// Sessions 1 and 3 should be deleted
	session1, _ := store.Get(sessionID1)
	if session1 != nil {
		t.Error("expected session 1 to be cleaned up")
	}

	session3, _ := store.Get(sessionID3)
	if session3 != nil {
		t.Error("expected session 3 to be cleaned up")
	}

	// Session 2 should remain
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

	// Test concurrent access
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 100 goroutines create sessions simultaneously
	sessionIDs := make([]string, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			sessionID, err := store.Create(int64(index), 1*time.Hour, SessionMeta{})
			if err != nil {
				t.Errorf("goroutine %d: failed to create session: %v", index, err)
				return
			}
			sessionIDs[index] = sessionID
		}(i)
	}

	wg.Wait()

	// Verify that all sessions can be retrieved
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

	// Create an initial session
	sessionID, _ := store.Create(999, 1*time.Hour, SessionMeta{})

	var wg sync.WaitGroup
	const numReaders = 50
	const numWriters = 10

	// 50 goroutines read simultaneously
	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = store.Get(sessionID)
			}
		}()
	}

	// 10 goroutines write simultaneously
	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(index int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = store.Create(int64(index*100+j), 1*time.Hour, SessionMeta{})
			}
		}(i)
	}

	// Verify that no data races or deadlocks occur
	wg.Wait()
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple session IDs and verify no duplicates
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

func TestSessionStore_ListTouchAndDelete(t *testing.T) {
	store := NewInMemorySessionStore()

	idA, err := store.Create(1, time.Hour, SessionMeta{UserAgent: "UA-A", IP: "10.0.0.1"})
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	idB, err := store.Create(1, time.Hour, SessionMeta{UserAgent: "UA-B", IP: "10.0.0.2", RememberSelector: "sel-b"})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	if _, err := store.Create(2, time.Hour, SessionMeta{}); err != nil {
		t.Fatalf("Create other: %v", err)
	}

	infos := store.ListByUser(1)
	if len(infos) != 2 {
		t.Fatalf("expected 2 sessions for user 1, got %d", len(infos))
	}
	for _, in := range infos {
		if in.Handle == "" || in.Handle == idA || in.Handle == idB {
			t.Fatalf("bad handle %q", in.Handle)
		}
	}

	later := time.Now().Add(time.Minute)
	store.Touch(idA, later)
	for _, in := range store.ListByUser(1) {
		if in.Handle == HashSessionID(idA) && !in.LastUsedAt.Equal(later) {
			t.Fatalf("Touch did not update LastUsedAt: %v", in.LastUsedAt)
		}
	}

	store.SetRememberSelector(idA, "sel-a")
	if s, _ := store.Get(idA); s == nil || s.RememberSelector != "sel-a" {
		t.Fatalf("SetRememberSelector failed: %+v", s)
	}

	if ok, _ := store.DeleteByHandleForUser(2, HashSessionID(idA)); ok {
		t.Fatalf("must not delete another user's session")
	}
	if ok, _ := store.DeleteByHandleForUser(1, HashSessionID(idA)); !ok {
		t.Fatalf("expected to delete session A for user 1")
	}
	if s, _ := store.Get(idA); s != nil {
		t.Fatalf("session A should be gone")
	}

	store.DeleteBySelector("sel-b")
	if s, _ := store.Get(idB); s != nil {
		t.Fatalf("session B should be gone after DeleteBySelector")
	}
}

func TestSessionStore_DeleteByUserExcept(t *testing.T) {
	store := NewInMemorySessionStore()
	keep, _ := store.Create(1, time.Hour, SessionMeta{})
	drop, _ := store.Create(1, time.Hour, SessionMeta{})

	store.DeleteByUserExcept(1, keep)
	if s, _ := store.Get(keep); s == nil {
		t.Fatalf("kept session must survive")
	}
	if s, _ := store.Get(drop); s != nil {
		t.Fatalf("other session must be deleted")
	}
}
