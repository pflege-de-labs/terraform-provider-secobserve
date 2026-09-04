package client

import (
	"context"
	"fmt"
	"net/url"
)

// ErrNotFound is returned by the lookup helpers when nothing matched. It is
// distinct from a 404 APIError: the request succeeded, the object does not
// exist.
type ErrNotFound struct {
	Kind  string
	Name  string
	Field string
}

func (e *ErrNotFound) Error() string {
	field := e.Field
	if field == "" {
		field = "name"
	}
	return fmt.Sprintf("no %s found with %s %q", e.Kind, field, e.Name)
}

// ErrAmbiguous is returned when more than one object matched an exact-name
// lookup. SecObserve does not enforce uniqueness on every name (general rule
// names in particular), so guessing would silently manage the wrong object.
type ErrAmbiguous struct {
	Kind string
	Name string
	IDs  []int64
}

func (e *ErrAmbiguous) Error() string {
	return fmt.Sprintf("%d %s objects are named %q (ids %v); names are not unique here, reference the object by id",
		len(e.IDs), e.Kind, e.Name, e.IDs)
}

// Named is implemented by every list item the name lookups operate on.
type Named interface {
	GetID() int64
	GetName() string
}

// FindByExactName resolves a name to a single object.
//
// SecObserve's list filters match names with icontains, not equality
// (core/api/filters.py), so the server-side filter is only a prefetch: the
// exact match has to be applied client-side, and multiple hits have to be
// reported rather than resolved arbitrarily.
func FindByExactName[T Named](ctx context.Context, c *Client, kind, path, name string, extra url.Values) (T, error) {
	var zero T

	query := url.Values{}
	for key, values := range extra {
		query[key] = values
	}
	query.Set("name", name)

	candidates, err := List[T](ctx, c, path, query)
	if err != nil {
		return zero, err
	}

	var matches []T
	for _, candidate := range candidates {
		if candidate.GetName() == name {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return zero, &ErrNotFound{Kind: kind, Name: name}
	case 1:
		return matches[0], nil
	default:
		ids := make([]int64, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.GetID())
		}
		return zero, &ErrAmbiguous{Kind: kind, Name: name, IDs: ids}
	}
}
