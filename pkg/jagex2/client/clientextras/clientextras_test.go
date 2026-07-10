package clientextras

import (
	"os"
	"reflect"
	"testing"
)

// The default must remain a direct os.Exit so the stock client's shutdown
// behavior is unchanged; only embedders (e.g. goscape-singleplayer) replace it.
func TestExitFuncDefaultsToOSExit(t *testing.T) {
	if reflect.ValueOf(ExitFunc).Pointer() != reflect.ValueOf(os.Exit).Pointer() {
		t.Fatal("clientextras.ExitFunc default is not os.Exit")
	}
}
