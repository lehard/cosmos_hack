package main

import (
	"testing"
	"time"
)

// Эпик 37: смены шаблонов нормативного слоя для проверки «по графику должен быть».
func TestPatternShiftsAt(t *testing.T) {
	msk := time.FixedZone("MSK", 3*3600)
	got := patternShiftsAt(time.Date(2026, 9, 26, 15, 55, 0, 0, msk))
	if len(got) != 1 || got[0].ID != "SHIFT-1" || got[0].Start.Hour() != 8 {
		t.Fatalf("первая смена: %+v", got)
	}
	night := patternShiftsAt(time.Date(2026, 9, 27, 0, 30, 0, 0, msk))
	if len(night) != 1 || night[0].ID != "SHIFT-2" || night[0].Start.Day() != 26 {
		t.Fatalf("вторая смена через полночь: %+v", night)
	}
	if n := patternShiftsAt(time.Date(2026, 9, 27, 3, 0, 0, 0, msk)); len(n) != 0 {
		t.Fatalf("вне смен: %+v", n)
	}
}
