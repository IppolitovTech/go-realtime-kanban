package service

import (
	"math"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/domain"
)

// orderStep is the gap left between siblings on first insertion/append and
// after a renumbering pass — see ADR 004.
const orderStep = 1000.0

// orderCollapseEpsilon is the minimum gap between neighboring order_num
// values below which we treat float precision as exhausted and trigger a
// local renumbering instead of returning a midpoint indistinguishable from
// one of its neighbors — see ADR 004.
const orderCollapseEpsilon = 1e-6

// orderBetween returns the order_num for an item inserted between prev and
// next. A nil prev means "insert at the start of the list", a nil next
// means "insert at the end"; both nil means the list is empty.
func orderBetween(prev, next *float64) float64 {
	switch {
	case prev == nil && next == nil:
		return orderStep
	case prev == nil:
		return *next / 2
	case next == nil:
		return *prev + orderStep
	default:
		return (*prev + *next) / 2
	}
}

// orderCollapsed reports whether the gap between two neighboring order_num
// values has shrunk below the point where a distinct midpoint can still be
// represented — see ADR 004.
func orderCollapsed(prev, next float64) bool {
	return math.Abs(next-prev) < orderCollapseEpsilon
}

// orderNeedsRenumber reports whether inserting between prev and next
// (either nil meaning "no neighbor on that side") needs a renumbering pass
// first, per ADR 004. Appending past the last item (next == nil) grows
// order_num without bound, so it never collapses. Inserting before the
// first item (prev == nil) halves next toward zero on every repeated
// top-insert, so it is checked against an implied zero floor the same way
// two real neighbors are checked against each other — this closes the gap
// where repeated top-inserts could otherwise erode precision all the way to
// 0, colliding with the "empty list" sentinel used by appendOrder/MaxOrder.
func orderNeedsRenumber(prev, next *float64) bool {
	switch {
	case prev == nil && next == nil:
		return false
	case prev == nil:
		return orderCollapsed(0, *next)
	case next == nil:
		return false
	default:
		return orderCollapsed(*prev, *next)
	}
}

// orderSlotOccupied reports whether some sibling's order_num already lies
// strictly between prev and next (either nil meaning "no bound on that
// side"). A true result means another insert has already claimed the gap
// the caller believes is free — e.g. two Move calls racing on the same
// stale (prev, next) pair — so recomputing the midpoint would silently
// collide with it instead of landing in a genuinely free slot.
func orderSlotOccupied(siblingOrders []float64, prev, next *float64) bool {
	for _, o := range siblingOrders {
		if prev != nil && o <= *prev {
			continue
		}
		if next != nil && o >= *next {
			continue
		}
		return true
	}
	return false
}

// renumberAndInsert reassigns order_num (step orderStep) to every sibling
// in ids except movingID, with movingID's insertion slot reserved right
// after prevID (nil means "at the start"), and returns the order_num
// movingID itself should get. updateOrder persists one sibling's new
// order_num and must run inside the same transaction/lock the caller used
// to read ids — see ADR 004. Shared by CardService and ColumnService, whose
// renumbering algorithm is otherwise identical bar which repository they
// call.
func renumberAndInsert(ids []uuid.UUID, movingID uuid.UUID, prevID *uuid.UUID, updateOrder func(id uuid.UUID, order float64) error) (float64, error) {
	rest := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != movingID {
			rest = append(rest, id)
		}
	}

	insertAt := len(rest)
	if prevID == nil {
		insertAt = 0
	} else {
		for i, id := range rest {
			if id == *prevID {
				insertAt = i + 1
				break
			}
		}
	}
	rest = append(rest[:insertAt:insertAt], append([]uuid.UUID{movingID}, rest[insertAt:]...)...)

	var movedOrder float64
	for i, id := range rest {
		newOrder := float64(i+1) * orderStep
		if id == movingID {
			movedOrder = newOrder
			continue
		}
		if err := updateOrder(id, newOrder); err != nil {
			return 0, err
		}
	}
	return movedOrder, nil
}

// orderedSibling is the minimal shape resolveMoveOrder needs from a card or
// column: its identity and current position among its siblings.
type orderedSibling struct {
	ID       uuid.UUID
	OrderNum float64
}

// moveOrderRequest is resolveMoveOrder's input, named so call sites read as
// self-documenting field assignments instead of a run of positional
// same-typed arguments (PrevOrder/NextOrder are both *float64, MovingID/
// PrevID are both *uuid.UUID-shaped) that are easy to mix up.
type moveOrderRequest struct {
	MovingID uuid.UUID
	// PrevID is the sibling to insert MovingID right after during a
	// renumbering pass; nil means "at the start".
	PrevID *uuid.UUID
	// PrevOrder/NextOrder are the neighbors MovingID should end up between;
	// either nil means "no neighbor on that side".
	PrevOrder, NextOrder *float64
	// PrevField names the caller's request field to attribute a
	// prev-after-next validation error to (e.g. "prev_card_id").
	PrevField string
	// Siblings must be every sibling in the target list; MovingID's own
	// entry, if present, is filtered out automatically.
	Siblings []orderedSibling
	// UpdateOrder persists one sibling's new order_num during a renumbering
	// pass and must run inside the same transaction/lock the caller used to
	// read Siblings.
	UpdateOrder func(id uuid.UUID, order float64) error
}

// resolveMoveOrder returns the order_num req.MovingID should get to sit
// between req.PrevOrder and req.NextOrder among req.Siblings, renumbering
// all of them first if the requested slot has collapsed or is already
// occupied by another sibling — see ADR 004. Shared by CardService.Move and
// ColumnService.Move, whose reordering flow is otherwise identical bar which
// repository they call.
func resolveMoveOrder(req moveOrderRequest) (float64, error) {
	if req.PrevOrder != nil && req.NextOrder != nil && *req.PrevOrder >= *req.NextOrder {
		return 0, domain.NewValidationError(req.PrevField, "must be positioned before the next item in the current order")
	}

	if !orderNeedsRenumber(req.PrevOrder, req.NextOrder) {
		orders := make([]float64, 0, len(req.Siblings))
		for _, s := range req.Siblings {
			if s.ID != req.MovingID {
				orders = append(orders, s.OrderNum)
			}
		}
		if !orderSlotOccupied(orders, req.PrevOrder, req.NextOrder) {
			return orderBetween(req.PrevOrder, req.NextOrder), nil
		}
	}

	ids := make([]uuid.UUID, len(req.Siblings))
	for i, s := range req.Siblings {
		ids[i] = s.ID
	}
	return renumberAndInsert(ids, req.MovingID, req.PrevID, req.UpdateOrder)
}

// appendOrder returns the order_num for an item appended to the end of a
// list whose current maximum order_num is maxOrder. 0 stands for "the list
// is empty" (see CardRepository/ColumnRepository.MaxOrder), not a real
// order_num — every real order_num starts at orderStep.
func appendOrder(maxOrder float64) float64 {
	if maxOrder == 0 {
		return orderStep
	}
	return maxOrder + orderStep
}
