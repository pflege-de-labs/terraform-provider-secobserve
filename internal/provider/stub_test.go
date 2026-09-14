package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// stubSecObserve is a minimal SecObserve stand-in that reproduces the
// server-side behaviours this provider has to absorb:
//
//   - the security gate fill-and-clear logic, including its truthiness test
//     which makes a submitted threshold of 0 unsettable
//   - the branch housekeeping fill-and-clear logic
//   - normalizing an empty propagate_branches list to null
//   - requiring the exact name as a delete confirmation
//   - the GitHub issue tracker base URL default
//
// It lets the drift-sensitive paths be tested in CI without a container. It is
// not a substitute for the acceptance tests against a real instance, which is
// why the assertions here stay focused on those behaviours.
type stubSecObserve struct {
	server *httptest.Server

	mu       sync.Mutex
	nextID   int64
	products map[int64]map[string]any

	// instance-wide defaults the fill-in logic draws on
	thresholdDefaults map[string]int64
	housekeepingDays  int64
}

func newStubSecObserve(t *testing.T) *stubSecObserve {
	t.Helper()

	stub := &stubSecObserve{
		nextID:   1,
		products: map[int64]map[string]any{},
		thresholdDefaults: map[string]int64{
			"security_gate_threshold_critical": 0,
			"security_gate_threshold_high":     0,
			"security_gate_threshold_medium":   99999,
			"security_gate_threshold_low":      99999,
			"security_gate_threshold_none":     99999,
			"security_gate_threshold_unknown":  99999,
		},
		housekeepingDays: 30,
	}

	stub.server = httptest.NewServer(http.HandlerFunc(stub.route))
	t.Cleanup(stub.server.Close)

	t.Setenv("SECOBSERVE_BASE_URL", stub.server.URL)
	t.Setenv("SECOBSERVE_API_TOKEN", "stub-token")
	// The acceptance test framework refuses to run without this, and these
	// tests need no external system.
	t.Setenv("TF_ACC", "1")

	return stub
}

func (s *stubSecObserve) route(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "APIToken stub-token" {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "Invalid token."})
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	switch {
	case path == "api/users/me":
		writeJSON(w, http.StatusOK, map[string]any{"id": 1, "username": "admin", "is_superuser": true})
	case path == "api/status/version":
		writeJSON(w, http.StatusOK, map[string]any{"version": "1.58.0"})
	case path == "api/products" && r.Method == http.MethodPost:
		s.createProduct(w, r)
	case path == "api/products" && r.Method == http.MethodGet:
		s.listProducts(w, r)
	case strings.HasPrefix(path, "api/products/"):
		s.productByID(w, r, strings.TrimPrefix(path, "api/products/"))
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
	}
}

func (s *stubSecObserve) createProduct(w http.ResponseWriter, r *http.Request) {
	body, err := decodeBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++

	product := body
	product["id"] = id
	product["is_product_group"] = false
	s.applyServerLogic(product)
	s.products[id] = product

	writeJSON(w, http.StatusCreated, product)
}

func (s *stubSecObserve) listProducts(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wanted := r.URL.Query().Get("name")
	results := []map[string]any{}
	for _, product := range s.products {
		// SecObserve filters names with icontains.
		if wanted == "" || strings.Contains(asString(product["name"]), wanted) {
			results = append(results, product)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(results), "next": nil, "previous": nil, "results": results,
	})
}

func (s *stubSecObserve) productByID(w http.ResponseWriter, r *http.Request, suffix string) {
	id, err := strconv.ParseInt(strings.Trim(suffix, "/"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	product, exists := s.products[id]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, product)

	case http.MethodPatch:
		body, err := decodeBody(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
			return
		}
		// An absent key means "leave unchanged".
		for key, value := range body {
			product[key] = value
		}
		s.applyServerLogic(product)
		writeJSON(w, http.StatusOK, product)

	case http.MethodDelete:
		// The name is a mandatory, exactly-matching confirmation.
		confirmations := r.URL.Query()["name"]
		if len(confirmations) != 1 || confirmations[0] != asString(product["name"]) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"name": "Confirmation name must match the product name.",
			})
			return
		}
		delete(s.products, id)
		w.WriteHeader(http.StatusNoContent)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "Method not allowed."})
	}
}

// applyServerLogic mirrors ProductCoreSerializer.validate.
func (s *stubSecObserve) applyServerLogic(product map[string]any) {
	// Null leaves the thresholds alone: that is what makes inheritance work.
	if gate, present := product["security_gate_active"].(bool); present {
		if gate {
			for threshold, fallback := range s.thresholdDefaults {
				// Truthiness, not presence: a submitted 0 is replaced too.
				if !isTruthyNumber(product[threshold]) {
					product[threshold] = fallback
				}
			}
		} else {
			for threshold := range s.thresholdDefaults {
				product[threshold] = nil
			}
		}
	}

	if housekeeping, present := product["repository_branch_housekeeping_active"].(bool); present {
		if housekeeping {
			if !isTruthyNumber(product["repository_branch_housekeeping_keep_inactive_days"]) {
				product["repository_branch_housekeeping_keep_inactive_days"] = s.housekeepingDays
			}
		} else {
			product["repository_branch_housekeeping_keep_inactive_days"] = nil
			product["repository_branch_housekeeping_exempt_branches"] = ""
		}
	}

	// An empty or effectively-empty propagation list becomes null.
	if rules, ok := product["propagate_branches"].([]any); ok && len(rules) == 0 {
		product["propagate_branches"] = nil
	}

	if asString(product["issue_tracker_type"]) == "GitHub" && asString(product["issue_tracker_base_url"]) == "" {
		product["issue_tracker_base_url"] = "https://api.github.com"
	}

	// Fields the API always reports but never accepts.
	for _, computed := range []string{"security_gate_passed", "product_group_name", "repository_default_branch",
		"repository_default_branch_name"} {
		if _, exists := product[computed]; !exists {
			product[computed] = nil
		}
	}
}

func decodeBody(r *http.Request) (map[string]any, error) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("could not decode request body: %w", err)
	}
	return body, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func isTruthyNumber(value any) bool {
	number, ok := value.(float64)
	return ok && number != 0
}
