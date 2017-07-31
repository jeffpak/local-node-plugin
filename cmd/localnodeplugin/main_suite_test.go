package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
)

func TestNfsV3Driver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Node Plugin Main Suite")
}

var driverPath string

var _ = BeforeSuite(func() {
	var err error
	driverPath, err = Build("github.com/jeffpak/local-node-plugin/cmd/localnodeplugin")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	CleanupBuildArtifacts()
})
