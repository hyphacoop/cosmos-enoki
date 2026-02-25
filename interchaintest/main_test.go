package e2e

import (
	"flag"
	"os"
	"testing"
)

var (
	flagPreUpgradeVersion  = flag.String("pre-upgrade-version", "v1.8.0", "Docker image version to use before the upgrade")
	flagPostUpgradeVersion = flag.String("post-upgrade-version", "v1.9.0", "Docker image version to use after the upgrade")
	flagUpgradeName        = flag.String("upgrade-name", "v1.9.0", "On-chain upgrade name used in the governance proposal")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
