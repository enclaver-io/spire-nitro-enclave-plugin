package main

import (
	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/nodeattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"

	"github.com/enclaver-io/spire-nitro-enclave-plugin/pkg/nodeattestor"
)

func main() {
	plugin := new(nodeattestor.Plugin)
	// Serve the plugin. This function call will not return. If there is a
	// failure to serve, the process will exit with a non-zero exit code.
	pluginmain.Serve(
		nodeattestorv1.NodeAttestorPluginServer(plugin),
		// TODO: Remove if no configuration is required
		configv1.ConfigServiceServer(plugin),
	)
}
