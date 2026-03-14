package shared

import "testing"

func TestCheckConfirmDelete(t *testing.T) {
	t.Parallel()

	t.Run("not own post", func(t *testing.T) {
		t.Parallel()
		res := CheckConfirmDelete(-1, 0, "did:other", "did:me", "at://post/1")
		if res.Confirmed {
			t.Fatal("should not confirm delete for other user's post")
		}
		if res.ConfirmDelete != -1 {
			t.Fatalf("expected confirmDelete -1, got %d", res.ConfirmDelete)
		}
	})

	t.Run("first press sets confirm", func(t *testing.T) {
		t.Parallel()
		res := CheckConfirmDelete(-1, 3, "did:me", "did:me", "at://post/1")
		if res.Confirmed {
			t.Fatal("first press should not confirm")
		}
		if res.ConfirmDelete != 3 {
			t.Fatalf("expected confirmDelete 3, got %d", res.ConfirmDelete)
		}
	})

	t.Run("second press confirms", func(t *testing.T) {
		t.Parallel()
		res := CheckConfirmDelete(3, 3, "did:me", "did:me", "at://post/1")
		if !res.Confirmed {
			t.Fatal("second press should confirm")
		}
		if res.URI != "at://post/1" {
			t.Fatalf("expected URI at://post/1, got %s", res.URI)
		}
		if res.ConfirmDelete != -1 {
			t.Fatalf("expected confirmDelete -1 after confirm, got %d", res.ConfirmDelete)
		}
	})

	t.Run("different index resets", func(t *testing.T) {
		t.Parallel()
		res := CheckConfirmDelete(2, 5, "did:me", "did:me", "at://post/1")
		if res.Confirmed {
			t.Fatal("different index should not confirm")
		}
		if res.ConfirmDelete != 5 {
			t.Fatalf("expected confirmDelete 5, got %d", res.ConfirmDelete)
		}
	})
}
