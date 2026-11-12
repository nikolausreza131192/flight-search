package usecase

import (
	"testing"

	"bookcabin/internal/entity"
)

func TestComputeFavoursCheaperAndFaster(t *testing.T) {
	flights := []entity.Flight{
		{Price: entity.Price{Amount: 500000}, Duration: entity.Duration{TotalMinutes: 100}, Stops: 0},
		{Price: entity.Price{Amount: 1500000}, Duration: entity.Duration{TotalMinutes: 200}, Stops: 1},
	}
	scored := applyScore(flights)

	if scored[0].BestValueScore <= scored[1].BestValueScore {
		t.Fatalf("cheaper/shorter flight should score higher: %v vs %v",
			scored[0].BestValueScore, scored[1].BestValueScore)
	}
	if scored[0].BestValueScore > 100 || scored[0].BestValueScore < 0 {
		t.Fatalf("score out of range: %v", scored[0].BestValueScore)
	}
}

func TestComputeSingleResult(t *testing.T) {
	flights := []entity.Flight{{Price: entity.Price{Amount: 500000}, Duration: entity.Duration{TotalMinutes: 100}}}
	scored := applyScore(flights)
	if scored[0].BestValueScore != 100 {
		t.Fatalf("single result should score 100, got %v", scored[0].BestValueScore)
	}
}

func TestComputeEmpty(t *testing.T) {
	if scored := applyScore(nil); scored != nil {
		t.Fatalf("empty input should score to nil, got %v", scored)
	}
}
