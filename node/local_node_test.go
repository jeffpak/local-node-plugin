package node_test

import (
	"code.cloudfoundry.org/goshims/filepathshim/filepath_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/jeffpak/csi"
	"github.com/jeffpak/local-node-plugin/node"
	"os"
	"time"
)

var _ = Describe("Node Client", func() {
	var (
		nc           *node.LocalNode
		testLogger   lager.Logger
		context      context.Context
		fakeOs       *os_fake.FakeOs
		fakeFilepath *filepath_fake.FakeFilepath
		vc           *VolumeCapability
		volID        *VolumeID
		volumeName   string
		err          error
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("localdriver-local")
		context = &DummyContext{}

		fakeOs = &os_fake.FakeOs{}
		fakeFilepath = &filepath_fake.FakeFilepath{}
		fakeFilepath.AbsReturns("/path/to/mount", nil)

		nc = node.NewLocalNode(fakeOs, fakeFilepath, testLogger)
		volumeName = "test-volume-id"
		volID = &VolumeID{Values: map[string]string{"volume_name": volumeName}}
		vc = &VolumeCapability{Value: &VolumeCapability_Mount{}}
	})

	Describe("NodePublishVolume", func() {
		Context("when the volume has been created", func() {
			var (
				mount_path = "/path/to/mount/_mounts/test-volume-id"
			)

			JustBeforeEach(func() {
				mountSuccessful(context, nc, volID, vc, mount_path)
			})

			Context("when the volume exists", func() {
				AfterEach(func() {
					unmountSuccessful(context, nc, volID)
				})

				It("should mount the volume on the local filesystem", func() {
					Expect(fakeFilepath.AbsCallCount()).To(Equal(1))
					Expect(fakeOs.MkdirAllCallCount()).To(Equal(1))
					Expect(fakeOs.SymlinkCallCount()).To(Equal(1))
					from, to := fakeOs.SymlinkArgsForCall(0)
					Expect(from).To(Equal("/path/to/mount/_volumes/test-volume-id"))
					Expect(to).To(Equal(mount_path))
				})
			})

			Context("when the volume ID is missing", func() {
				BeforeEach(func() {
					fakeOs.StatReturns(nil, os.ErrNotExist)
				})
				AfterEach(func() {
					fakeOs.StatReturns(nil, nil)
				})

				It("returns an error", func() {
					var path string = ""
					resp, err := nc.NodePublishVolume(context, &NodePublishVolumeRequest{
						Version:          &Version{Major: 0, Minor: 0, Patch: 1},
						VolumeId:         &VolumeID{Values: map[string]string{}},
						TargetPath:       path,
						VolumeCapability: vc,
						Readonly:         false,
					})
					Expect(err.Error()).To(Equal("Volume name is missing in request"))
					Expect(resp.GetError().GetNodePublishVolumeError().GetErrorCode()).To(Equal(Error_NodePublishVolumeError_INVALID_VOLUME_ID))
					Expect(resp.GetError().GetNodePublishVolumeError().GetErrorDescription()).To(Equal("Volume name is missing in request"))
				})
			})
		})
	})

	Describe("NodeUnpublishVolume", func() {
		var (
			request          *NodeUnpublishVolumeRequest
			expectedResponse *NodeUnpublishVolumeResponse
		)
		Context("when NodeUnpublishVolume is called with a NodeUnpublishVolume", func() {
			BeforeEach(func() {
				request = &NodeUnpublishVolumeRequest{
					&Version{Major: 0, Minor: 0, Patch: 1},
					volID,
					&VolumeMetadata{},
					"unpublish-path",
				}
			})
			JustBeforeEach(func() {
				expectedResponse, err = nc.NodeUnpublishVolume(context, request)
			})
			It("should return a NodeUnpublishVolumeResponse", func() {
				Expect(expectedResponse).NotTo(BeNil())
			})
		})
	})

	Describe("GetNodeID", func() {
		var (
			request          *GetNodeIDRequest
			expectedResponse *GetNodeIDResponse
		)
		Context("when GetNodeID is called with a GetNodeIDRequest", func() {
			BeforeEach(func() {
				request = &GetNodeIDRequest{
					&Version{Major: 0, Minor: 0, Patch: 1},
				}
			})
			JustBeforeEach(func() {
				expectedResponse, err = nc.GetNodeID(context, request)
			})
			It("should return a GetNodeIDResponse that has a result with no node ID", func() {
				Expect(expectedResponse).NotTo(BeNil())
				Expect(expectedResponse.GetResult()).NotTo(BeNil())
				Expect(expectedResponse.GetResult().GetNodeId()).To(BeNil())
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("ProbeNode", func() {
		var (
			request          *ProbeNodeRequest
			expectedResponse *ProbeNodeResponse
		)
		Context("when ProbeNode is called with a ProbeNodeRequest", func() {
			BeforeEach(func() {
				request = &ProbeNodeRequest{
					&Version{Major: 0, Minor: 0, Patch: 1},
				}
			})
			JustBeforeEach(func() {
				expectedResponse, err = nc.ProbeNode(context, request)
			})
			It("should return a ProbeNodeResponse", func() {
				Expect(expectedResponse).NotTo(BeNil())
				Expect(expectedResponse.GetResult()).NotTo(BeNil())
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("NodeGetCapabilities", func() {
		var (
			request          *NodeGetCapabilitiesRequest
			expectedResponse *NodeGetCapabilitiesResponse
		)
		Context("when NodeGetCapabilities is called with a NodeGetCapabilitiesRequest", func() {
			BeforeEach(func() {
				request = &NodeGetCapabilitiesRequest{
					&Version{Major: 0, Minor: 0, Patch: 1},
				}
			})
			JustBeforeEach(func() {
				expectedResponse, err = nc.NodeGetCapabilities(context, request)
			})

			It("should return a ControllerGetCapabilitiesResponse with UNKNOWN specified", func() {
				Expect(expectedResponse).NotTo(BeNil())
				capabilities := expectedResponse.GetResult().GetCapabilities()
				Expect(capabilities).To(HaveLen(1))
				Expect(capabilities[0].GetRpc().GetType()).To(Equal(NodeServiceCapability_RPC_UNKNOWN))
				Expect(err).To(BeNil())
			})
		})
	})
})

func mountSuccessful(ctx context.Context, ns NodeServer, volID *VolumeID, volCapability *VolumeCapability, targetPath string) {
	var mountResponse *NodePublishVolumeResponse
	mountResponse, err := ns.NodePublishVolume(ctx, &NodePublishVolumeRequest{
		Version:          &Version{Major: 0, Minor: 0, Patch: 1},
		VolumeId:         volID,
		TargetPath:       targetPath,
		VolumeCapability: volCapability,
		Readonly:         false,
	})
	Expect(err).To(BeNil())
	Expect(mountResponse.GetError()).To(BeNil())
	Expect(mountResponse.GetResult()).NotTo(BeNil())
}

func unmountSuccessful(ctx context.Context, ns NodeServer, volID *VolumeID) {
	var path string = "/path/to/mount"
	unmountResponse, err := ns.NodeUnpublishVolume(ctx, &NodeUnpublishVolumeRequest{
		Version:    &Version{Major: 0, Minor: 0, Patch: 1},
		VolumeId:   volID,
		TargetPath: path,
	})
	Expect(unmountResponse.GetError()).To(BeNil())
	Expect(err).To(BeNil())
}

type DummyContext struct{}

func (*DummyContext) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }

func (*DummyContext) Done() <-chan struct{} { return nil }

func (*DummyContext) Err() error { return nil }

func (*DummyContext) Value(key interface{}) interface{} { return nil }
