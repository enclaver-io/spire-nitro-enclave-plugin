package nodeattestor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl"
	"github.com/hf/nitrite"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/spire-plugin-sdk/pluginsdk"
	nodeattestorv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/server/nodeattestor/v1"
	configv1 "github.com/spiffe/spire-plugin-sdk/proto/spire/service/common/config/v1"
	"github.com/spiffe/spire/pkg/common/idutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// This compile-time assertion ensures the plugin conforms properly to the
	// pluginsdk.NeedsLogger interface.
	// TODO: Remove if the plugin does not need the logger.
	_ pluginsdk.NeedsLogger = (*Plugin)(nil)

	// This compile-time assertion ensures the plugin conforms properly to the
	// pluginsdk.NeedsHostServices interface.
	// TODO: Remove if the plugin does not need host services.
	_ pluginsdk.NeedsHostServices = (*Plugin)(nil)
)

// Config defines the configuration for the plugin.
// TODO: Add relevant configurables or remove if no configuration is required.
type Config struct {
	trustDomain spiffeid.TrustDomain
}

// Plugin implements the NodeAttestor plugin
type Plugin struct {
	// UnimplementedNodeAttestorServer is embedded to satisfy gRPC
	nodeattestorv1.UnimplementedNodeAttestorServer

	// UnimplementedConfigServer is embedded to satisfy gRPC
	configv1.UnimplementedConfigServer

	// Configuration should be set atomically
	configMtx sync.RWMutex
	config    *Config

	// The logger received from the framework via the SetLogger method
	logger hclog.Logger
}

// Attest implements the NodeAttestor Attest RPC. Attest attests attestation payload received from the agent and
// optionally participates in challenge/response attestation mechanics. This RPC uses a bidirectional stream for
// communication.
func (p *Plugin) Attest(stream nodeattestorv1.NodeAttestor_AttestServer) error {
	config, err := p.getConfig()
	if err != nil {
		return err
	}

	req, err := stream.Recv()
	if err != nil {
		return err
	}

	attestationDoc := req.GetPayload()
	verified, err := nitrite.Verify(attestationDoc, nitrite.VerifyOptions{
		CurrentTime: time.Now(),
	})
	if err != nil {
		return err
	}

	agentID, err := makeAgentId(config.trustDomain, verified.Document.PCRs[0])
	if err != nil {
		return err
	}

	return stream.Send(&nodeattestorv1.AttestResponse{
		Response: &nodeattestorv1.AttestResponse_AgentAttributes{
			AgentAttributes: &nodeattestorv1.AgentAttributes{
				SpiffeId:    agentID.String(),
				CanReattest: false,
			},
		},
	})
}

// Configure configures the plugin. This is invoked by SPIRE when the plugin is
// first loaded. In the future, it may be invoked to reconfigure the plugin.
// As such, it should replace the previous configuration atomically.
// TODO: Remove if no configuration is required
func (p *Plugin) Configure(ctx context.Context, req *configv1.ConfigureRequest) (*configv1.ConfigureResponse, error) {
	td, err := spiffeid.TrustDomainFromString(req.CoreConfiguration.TrustDomain)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse trust domain: %v", err)
	}
	config := Config{}
	if err := hcl.Decode(&config, req.HclConfiguration); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode configuration: %v", err)
	}

	config.trustDomain = td

	// TODO: Validate configuration before setting/replacing existing
	// configuration

	p.setConfig(&config)
	return &configv1.ConfigureResponse{}, nil
}

// BrokerHostServices is called by the framework when the plugin is loaded to
// give the plugin a chance to obtain clients to SPIRE host services.
// TODO: Remove if the plugin does not need host services.
func (p *Plugin) BrokerHostServices(broker pluginsdk.ServiceBroker) error {
	// TODO: Use the broker to obtain host service clients
	return nil
}

// SetLogger is called by the framework when the plugin is loaded and provides
// the plugin with a logger wired up to SPIRE's logging facilities.
// TODO: Remove if the plugin does not need the logger.
func (p *Plugin) SetLogger(logger hclog.Logger) {
	p.logger = logger
}

// setConfig replaces the configuration atomically under a write lock.
// TODO: Remove if no configuration is required
func (p *Plugin) setConfig(config *Config) {
	p.configMtx.Lock()
	p.config = config
	p.configMtx.Unlock()
}

// getConfig gets the configuration under a read lock.
// TODO: Remove if no configuration is required
func (p *Plugin) getConfig() (*Config, error) {
	p.configMtx.RLock()
	defer p.configMtx.RUnlock()
	if p.config == nil {
		return nil, status.Error(codes.FailedPrecondition, "not configured")
	}
	return p.config, nil
}

func makeAgentId(td spiffeid.TrustDomain, pcr0 []byte) (spiffeid.ID, error) {
	suffix := fmt.Sprintf("/nitro-enclave/%x", pcr0)
	return idutil.AgentID(td, suffix)
}
