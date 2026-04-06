package biz

import (
	"testing"
	"time"

	"subscription-service/internal/constants"
)

func TestFirstPeriodEndUTC_Month_Jan1(t *testing.T) {
	purchase := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	end, err := FirstPeriodEndUTC(purchase, constants.PeriodTypeMonth, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
}

func TestFirstPeriodEndUTC_Month_Jan31_shortFebruary(t *testing.T) {
	purchase := time.Date(2025, 1, 31, 9, 0, 0, 0, time.UTC)
	end, err := FirstPeriodEndUTC(purchase, constants.PeriodTypeMonth, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
}

func TestNextPeriodEndUTC_Month_anchor31_afterFebEnd(t *testing.T) {
	currentEnd := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	end, err := NextPeriodEndUTC(currentEnd, 31, constants.PeriodTypeMonth, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
}

func TestFirstPeriodEndUTC_Day(t *testing.T) {
	purchase := time.Date(2025, 1, 1, 15, 0, 0, 0, time.UTC)
	end, err := FirstPeriodEndUTC(purchase, constants.PeriodTypeDay, 7)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC)
	if !end.Equal(want) {
		t.Fatalf("got %v want %v", end, want)
	}
}

func TestFirstPeriodEndUTC_Forever(t *testing.T) {
	end, err := FirstPeriodEndUTC(time.Now().UTC(), constants.PeriodTypeForever, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !end.Equal(ForeverEndUTC()) {
		t.Fatalf("got %v", end)
	}
}
