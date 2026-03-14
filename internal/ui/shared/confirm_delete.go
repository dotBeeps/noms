package shared

// ConfirmDeleteResult holds the outcome of a two-press delete check.
type ConfirmDeleteResult struct {
	ConfirmDelete int    // updated confirmDelete state
	URI           string // non-empty when deletion should proceed
	Confirmed     bool   // true when the second press was detected
}

// CheckConfirmDelete implements the two-press delete pattern:
//   - First press on a user's own post sets confirmDelete to selectedIndex.
//   - Second press on the same index confirms the delete.
//   - Returns unchanged state if the post doesn't belong to ownDID.
func CheckConfirmDelete(confirmDelete, selectedIndex int, postAuthorDID, ownDID, postURI string) ConfirmDeleteResult {
	if postAuthorDID != ownDID {
		return ConfirmDeleteResult{ConfirmDelete: confirmDelete}
	}
	if confirmDelete == selectedIndex {
		return ConfirmDeleteResult{ConfirmDelete: -1, URI: postURI, Confirmed: true}
	}
	return ConfirmDeleteResult{ConfirmDelete: selectedIndex}
}
