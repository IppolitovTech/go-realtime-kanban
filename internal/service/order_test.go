package service

import (
	"math"
	"testing"
)

func float64Ptr(v float64) *float64 { return &v }

func TestOrderBetween(t *testing.T) {
	tests := []struct {
		name string
		prev *float64
		next *float64
		want float64
	}{
		{"empty list", nil, nil, orderStep},
		{"insert at start", nil, float64Ptr(2000), 1000},
		{"insert at end", float64Ptr(1000), nil, 1000 + orderStep},
		{"insert between", float64Ptr(1000), float64Ptr(2000), 1500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orderBetween(tt.prev, tt.next)
			if got != tt.want {
				t.Errorf("orderBetween(%v, %v) = %v, want %v", tt.prev, tt.next, got, tt.want)
			}
		})
	}
}

func TestOrderCollapsed(t *testing.T) {
	tests := []struct {
		name       string
		prev, next float64
		want       bool
	}{
		{"plenty of room", 1000, 2000, false},
		{"exactly equal", 1000, 1000, true},
		{"midpoint rounds back to prev", 1000, math.Nextafter(1000, 1001), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orderCollapsed(tt.prev, tt.next)
			if got != tt.want {
				t.Errorf("orderCollapsed(%v, %v) = %v, want %v", tt.prev, tt.next, got, tt.want)
			}
		})
	}
}

// TestOrderCollapse_RepeatedMidpointInsertion is the ADR 004-mandated
// edge-case test: repeatedly inserting exactly at the midpoint between the
// same two neighbors eventually collapses float64 precision to the point
// where (prev+next)/2 == prev. This proves orderCollapsed reliably catches
// that state before it would produce a duplicate/out-of-order order_num.
func TestOrderCollapse_RepeatedMidpointInsertion(t *testing.T) {
	prev, next := 1000.0, 1000.0+1e-9

	for i := 0; i < 100; i++ {
		if orderCollapsed(prev, next) {
			return
		}
		mid := orderBetween(&prev, &next)
		if mid <= prev || mid >= next {
			t.Fatalf("iteration %d: midpoint %v escaped (prev=%v, next=%v) without being flagged as collapsed", i, mid, prev, next)
		}
		next = mid
	}
	t.Fatal("expected orderCollapsed to trigger within 100 midpoint insertions")
}

func TestOrderNeedsRenumber(t *testing.T) {
	tests := []struct {
		name string
		prev *float64
		next *float64
		want bool
	}{
		{"empty list", nil, nil, false},
		{"append at end never collapses", float64Ptr(1000), nil, false},
		{"insert at start with room", nil, float64Ptr(1000), false},
		{"insert at start with no room", nil, float64Ptr(1e-9), true},
		{"two neighbors with room", float64Ptr(1000), float64Ptr(2000), false},
		{"two neighbors collapsed", float64Ptr(1000), float64Ptr(1000), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orderNeedsRenumber(tt.prev, tt.next)
			if got != tt.want {
				t.Errorf("orderNeedsRenumber(%v, %v) = %v, want %v", tt.prev, tt.next, got, tt.want)
			}
		})
	}
}

// TestOrderNeedsRenumber_RepeatedTopInsertion is the top-of-list analogue of
// TestOrderCollapse_RepeatedMidpointInsertion: repeatedly inserting at the
// very top of a list (prev == nil) halves order_num toward zero on every
// insert. Without a collapse check on this branch, that halving would run
// unchecked and could eventually erode order_num all the way to 0.0,
// colliding with the "empty list" sentinel used by appendOrder/MaxOrder.
func TestOrderNeedsRenumber_RepeatedTopInsertion(t *testing.T) {
	next := 1000.0

	for i := 0; i < 100; i++ {
		if orderNeedsRenumber(nil, &next) {
			return
		}
		mid := orderBetween(nil, &next)
		if mid <= 0 || mid >= next {
			t.Fatalf("iteration %d: midpoint %v escaped valid range (0, %v) without being flagged", i, mid, next)
		}
		next = mid
	}
	t.Fatal("expected orderNeedsRenumber to trigger within 100 top insertions")
}

func TestOrderSlotOccupied(t *testing.T) {
	tests := []struct {
		name   string
		orders []float64
		prev   *float64
		next   *float64
		want   bool
	}{
		{"empty column", nil, nil, nil, false},
		{"gap is free", []float64{1000, 2000}, float64Ptr(1000), float64Ptr(2000), false},
		{"another sibling already sits in the gap", []float64{1000, 1500, 2000}, float64Ptr(1000), float64Ptr(2000), true},
		{"free at the very top", []float64{1000, 2000}, nil, float64Ptr(1000), false},
		{"occupied at the very top", []float64{500, 1000, 2000}, nil, float64Ptr(1000), true},
		{"free at the very bottom", []float64{1000, 2000}, float64Ptr(2000), nil, false},
		{"occupied at the very bottom", []float64{1000, 2000, 3000}, float64Ptr(2000), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orderSlotOccupied(tt.orders, tt.prev, tt.next)
			if got != tt.want {
				t.Errorf("orderSlotOccupied(%v, %v, %v) = %v, want %v", tt.orders, tt.prev, tt.next, got, tt.want)
			}
		})
	}
}

func TestAppendOrder(t *testing.T) {
	tests := []struct {
		name     string
		maxOrder float64
		want     float64
	}{
		{"empty list", 0, orderStep},
		{"non-empty list", 1000, 1000 + orderStep},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendOrder(tt.maxOrder)
			if got != tt.want {
				t.Errorf("appendOrder(%v) = %v, want %v", tt.maxOrder, got, tt.want)
			}
		})
	}
}
