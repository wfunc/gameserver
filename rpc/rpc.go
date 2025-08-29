package rpc

import (
	"net"
	"net/rpc"

	"github.com/wfunc/gameserver/logger"
	"github.com/wfunc/gameserver/services"
)

// Server manages the RPC listener.
type Server struct {
	listener net.Listener
	address  string
}

// NewServer creates a new RPC server.
func NewServer(addr string) (*Server, error) {
	// Register the GameService with the rpc package so it knows how to handle it.
	// We can do this here or in the main server startup.
	// For simplicity, we assume any service using this RPC server is registered elsewhere.

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		listener: listener,
		address:  addr,
	}, nil
}

// Start begins listening for RPC requests.
func (s *Server) Start() {
	logger.Log.Infof("RPC server listening on %s", s.address)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed.
			if _, ok := err.(*net.OpError); ok {
				logger.Log.Info("RPC server listener closed.")
				return
			}
			logger.Log.Errorf("RPC server accept error: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

// Stop closes the RPC listener.
func (s *Server) Stop() {
	if s.listener != nil {
		logger.Log.Info("Stopping RPC server.")
		s.listener.Close()
	}
}

// GameService is the struct that exposes RPC methods.
type GameService struct {
	playerService *services.PlayerService
}

// NewGameService creates a new GameService.
func NewGameService(ps *services.PlayerService) *GameService {
	return &GameService{playerService: ps}
}

// GetPlayerWithStats is an RPC method to get player data.
// It must follow the net/rpc signature: exported method, exported arguments,
// second argument is a pointer, return type is error.
type GetPlayerArgs struct {
	UserID int64
}

type GetPlayerReply struct {
	Data map[string]interface{}
}

func (gs *GameService) GetPlayerWithStats(args *GetPlayerArgs, reply *GetPlayerReply) error {
	data, err := gs.playerService.GetPlayerWithStats(args.UserID)
	if err != nil {
		return err
	}
	reply.Data = data
	return nil
}