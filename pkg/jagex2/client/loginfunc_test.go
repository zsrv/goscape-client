package client

import (
	"testing"

	"github.com/zsrv/goscape-client/pkg/jagex2/datastruct/jstring"
)

// TestLoginServerComputation verifies the loginServer index derivation
// introduced in rev-244 (Client.java:2602-2603):
//
//	long username37 = JString.toBase37(username);
//	int loginServer = (int)(username37 >> 16 & 0x1FL);
//
// The result must always be in [0, 31].
func TestLoginServerComputation(t *testing.T) {
	names := []string{"zezima", "admin", "a", "zzzzzzzzzzzz", ""}
	for _, name := range names {
		username37 := jstring.ToBase37(name)
		loginServer := int(username37 >> 16 & 0x1F)
		if loginServer < 0 || loginServer > 31 {
			t.Errorf("ToBase37(%q) >> 16 & 0x1F = %d; want in [0, 31]", name, loginServer)
		}
	}
}
