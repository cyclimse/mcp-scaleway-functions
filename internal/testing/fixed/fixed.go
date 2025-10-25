package fixed

import "time"

const (
	// Note: all uuids here were generated with uuidgen.
	SomeFunctionID   = "40247e59-0cb5-4f75-abdd-85077f069c6d"
	SomeFunctionName = "my-function"

	SomeNamespaceID   = "7cd330f8-c9da-463f-a5d5-1c5e8591d5b4"
	SomeNamespaceName = "my-namespace"

	SomeProjectID   = "761e00a5-9881-4dd4-b73e-23a51bd068d5"
	SomeProjectName = "my-project"

	SomeRegion = "fr-par"

	SomeDockerContainerID = "d9b100f2f636ffddd6ae1e4ae015f1a4"

	SomeCodeArchiveDigest = "sha256:400ef282bdb172f85ddc65dc1c452bfcaa8dbe1732cae89d54afcad81a2e283e"
)

//nolint:gochecknoglobals
var (
	SomeTimestampA = time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	SomeTimestampB = time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)
)
