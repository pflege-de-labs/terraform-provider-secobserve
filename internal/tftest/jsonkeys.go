package tftest

import (
	"encoding/json"
	"testing"
)

// JSONKeys returns the top-level keys a value serializes to.
//
// Used to assert that request types emit every managed field. SecObserve
// treats an absent key as "leave unchanged", so a dropped key silently changes
// behaviour instead of failing.
func JSONKeys(t *testing.T, value any) map[string]bool {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshalling %T: %v", value, err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshalling %T: %v", value, err)
	}

	keys := make(map[string]bool, len(decoded))
	for key := range decoded {
		keys[key] = true
	}
	return keys
}
