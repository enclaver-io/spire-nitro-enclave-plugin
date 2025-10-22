# SPIRE Nitro Enclave Plugin

## Overview

This plugin works with [Enclaver](https://enclaver.io) to give workloads a SPIFFE identity based on the attestation document and the PCR0 value.
The established identity (via SVID-x.509) can be used with mTLS to create a trusted connection. Alternatively, a SVID-JWT can be used as an authentication token.

## Theory of Operation

In a typical SPIRE setup, a server is paired with a SPIRE node agent.
The node agent authenticates with the server via some mechanism to prove its identity (e.g. a token or a AWS instance metadata document).
The node agent is then responsible for validating individual workload identities.

A Nitro Enclave is a (stripped down) VM and is treated as a node for the purposes of the SPIRE server.
Therefore a node and a workload are one and the same in the case of the Nitro enclave.
The Enclaver implements the node agent functionality and attests with the server using the attestation document.
The enclave gets its identity via the PCR0 value.

## Getting Started
0. Make sure Go is installed.

1. Install the SPIRE server and this plugin.
```bash
git clone https://github.com/enclaver-io/spire-nitro-enclave-plugin
cd spire-nitro-enclave-plugin
go build -o spire-nitor-enclave-plugin cmd/spire-nitro-enclave-plugin/main.go
```

2. Configure the SPIRE server to use the plugin. Locate SPIRE server's `server.conf` and add the following to the `plugins` section:
```hcl
    NodeAttestor "nitro_enclave" {
        plugin_cmd = "path/to/spire-nitor-enclave-plugin"
        plugin_data {}
    }
```

3. In enclaver.yaml, configure SPIFFE and specify the SPIRE server address.

4. Use Enclaver to build the enclave and note its PCR0 value.

5. Register the agent/workload with the server (replace PCR0-value with the actual value from the previous step):
```bash
bin/spire-server entry create \
   -parentID spiffe://example.org/spire/agent/nitro-enclave/PCR0-value \
   -spiffeID spiffe://example.org/myservice \
   -selector nitro_enclave:*
```

6. Use Enclaver to run the enclave image.

7. The Enclaver will write out the private key and the SVID to `/var/run/spiffe`
