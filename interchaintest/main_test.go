package e2e

import (
	"flag"
	"os"
	"testing"
)

var (
	flagPreUpgradeVersion  = flag.String("pre-upgrade-version", "v2.1.0", "Docker image version to use before the upgrade")
	flagPostUpgradeVersion = flag.String("post-upgrade-version", "main", "Docker image version to use after the upgrade")
	flagUpgradeName        = flag.String("upgrade-name", "v3.0.0", "On-chain upgrade name used in the governance proposal")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
