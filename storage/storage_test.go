package storage_test

import (
	"testing"

	"github.com/JGpGH/golfu/storage"
)

func TestNewTrashableLifecycle(t *testing.T) {
	item := storage.NewIndexed("alpha", 42)
	trashable := storage.NewTrashable(item, false)
	if trashable == nil {
		t.Fatal("expected trashable instance")
	}
	if trashable.CanBeTrashed() {
		t.Fatal("trashable should respect initial flag")
	}
	if got := trashable.Value(); got != item {
		t.Fatalf("unexpected value: %#v", got)
	}
	if trashable.Index() != item.Index() {
		t.Fatalf("unexpected index: %s", trashable.Index())
	}
	trashable.SetCanBeTrashed()
	if !trashable.CanBeTrashed() {
		t.Fatal("trashable should enable trashing after SetCanBeTrashed")
	}
}

func TestNewTrashablesProducesReadySlice(t *testing.T) {
	input := []storage.Indexed[int]{
		storage.NewIndexed("a", 1),
		storage.NewIndexed("b", 2),
		storage.NewIndexed("c", 3),
	}
	trashables := storage.NewTrashables(input, true)
	if len(trashables) != len(input) {
		t.Fatalf("expected %d trashables, got %d", len(input), len(trashables))
	}
	for i, trashable := range trashables {
		if trashable == nil {
			t.Fatalf("trashable %d is nil", i)
		}
		if trashable.Value() != input[i] {
			t.Fatalf("trashable %d holds unexpected value", i)
		}
		if trashable.Index() != input[i].Index() {
			t.Fatalf("trashable %d returned unexpected index", i)
		}
		if !trashable.CanBeTrashed() {
			t.Fatalf("trashable %d should allow trashing", i)
		}
	}
}

func TestNewTrashablesRespectsFlag(t *testing.T) {
	input := []storage.Indexed[int]{storage.NewIndexed("x", 5)}
	trashables := storage.NewTrashables(input, false)
	if len(trashables) != 1 {
		t.Fatalf("expected 1 trashable, got %d", len(trashables))
	}
	if trashables[0].CanBeTrashed() {
		t.Fatal("trashable should inherit false flag")
	}
}
