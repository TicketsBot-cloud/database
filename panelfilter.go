package database

import (
	"fmt"

	"github.com/jackc/pgtype"
)

// PanelFilter restricts analytics queries to a subset of panels. A nil
// *PanelFilter means no filtering (all panels, including unassigned).
type PanelFilter struct {
	PanelIds          []int
	IncludeUnassigned bool
}

// PanelPredicate renders a SQL predicate that filters on panel_id, prefixed
// with " AND ". The caller supplies the table alias and the two parameter
// numbers to bind against. Parameter numbers are stated explicitly rather than
// auto-incremented because several queries reference the predicate more than
// once and some bind parameters out of textual order.
//
// The clause consumes exactly two parameters:
//
//	$arrayParam      ::int4[]  panel IDs, or NULL for "no filter"
//	$unassignedParam ::bool    whether to include tickets with panel_id IS NULL
func PanelPredicate(alias string, arrayParam, unassignedParam int) string {
	return fmt.Sprintf(
		` AND ($%d::int4[] IS NULL OR %s.panel_id = ANY($%d::int4[]) OR ($%d::bool AND %s.panel_id IS NULL))`,
		arrayParam, alias, arrayParam, unassignedParam, alias,
	)
}

// Args returns the two bound values for PanelPredicate: the int4 array and the
// bool. Safe on a nil receiver, where it returns the "no filter" pair (NULL
// array, false).
//
// The empty-versus-NULL distinction matters and is the highest-probability
// silent bug in this change:
//
//	Set(nil)     -> Status: Null           -> $N IS NULL matches everything
//	Set([]int32{}) -> present, zero-dimension -> $N IS NULL is false, ANY($N) matches nothing
//
// Getting this backwards makes "unassigned only" silently mean "everything".
func (f *PanelFilter) Args() (pgtype.Int4Array, bool, error) {
	if f == nil {
		return pgtype.Int4Array{Status: pgtype.Null}, false, nil
	}

	if len(f.PanelIds) == 0 && !f.IncludeUnassigned {
		// Neither panels nor unassigned selected: no filter.
		return pgtype.Int4Array{Status: pgtype.Null}, false, nil
	}

	var arr pgtype.Int4Array

	if len(f.PanelIds) == 0 {
		// IncludeUnassigned is true but no panel IDs: "unassigned only".
		// Must be present-empty (not NULL) so the IS NULL check in the
		// predicate correctly excludes all panel-assigned tickets.
		if err := arr.Set([]int32{}); err != nil {
			return pgtype.Int4Array{}, false, fmt.Errorf("panel filter: setting empty array: %w", err)
		}
		return arr, true, nil
	}

	ids := make([]int32, len(f.PanelIds))
	for i, id := range f.PanelIds {
		ids[i] = int32(id)
	}

	if err := arr.Set(ids); err != nil {
		return pgtype.Int4Array{}, false, fmt.Errorf("panel filter: setting array: %w", err)
	}

	return arr, f.IncludeUnassigned, nil
}

// IsActive reports whether the filter narrows anything. A nil filter is not
// active (it matches all tickets).
func (f *PanelFilter) IsActive() bool {
	if f == nil {
		return false
	}
	return len(f.PanelIds) > 0 || f.IncludeUnassigned
}
