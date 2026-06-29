// Package agent is the Tsugi write-plane gRPC server (PLAN §11): the localhost-only
// act-plane Yagura dials. Contract lives in proto/tsugi_agent.proto.
package agent

import "github.com/latoulicious/Tsugi/internal/agentpb"

// Server implements agentpb.TsugiAgentServer. Embedding the generated Unimplemented
// stub makes every RPC return codes.Unimplemented until wired (reads P5.2, deploy P5.4).
type Server struct {
	agentpb.UnimplementedTsugiAgentServer
}

func New() *Server { return &Server{} }

var _ agentpb.TsugiAgentServer = (*Server)(nil)
