package tempest

import "testing"

func TestAppIDFromToken(t *testing.T) {
	// "MTU0NTAwODQ2MjEyNjI1MjEyMg" is base64url for 1545008462126252122.
	id, ok := appIDFromToken("MTU0NTAwODQ2MjEyNjI1MjEyMg.GAGoi-.abcdefghijklmnopqrstuvwxyz")
	if !ok {
		t.Fatal("expected to parse app ID")
	}

	if id.String() != "1545008462126252122" {
		t.Errorf("app ID = %s, want 1545008462126252122", id.String())
	}

	if _, ok := appIDFromToken("not-a-token"); ok {
		t.Error("expected failure for malformed token")
	}
}
