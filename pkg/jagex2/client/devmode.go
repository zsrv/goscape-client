package client

import (
	"os"
	"strconv"
)

// developerMode mirrors the TS client's dev-client flag. When true, Examine
// menu options append the config-type id. Enabled with the DEVELOPER_MODE=true
// environment variable. Java: no equivalent — LostCityRS TS-client addition
// (commits 15260cc "feat: Dev client option" / b585f4c "fix: DEV_CLIENT env flag").
var developerMode = os.Getenv("DEVELOPER_MODE") == "true"

// examineIDSuffix returns the developer-mode id annotation appended to Examine
// menu text, e.g. " @whi@(@gre@1276@whi@)", or "" when developer mode is off.
// The id is the config type's .Index field (this port has no separate .Id).
func examineIDSuffix(id int) string {
	if !developerMode {
		return ""
	}
	return " @whi@(@gre@" + strconv.Itoa(id) + "@whi@)"
}
