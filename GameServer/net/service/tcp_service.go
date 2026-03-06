package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/handler"
	"github.com/pzqf/zMmoServer/GameServer/net/protolayer"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type TCPService struct {
	config         *config.Config
	connManager    *connection.ConnectionManager
	sessionManager *session.SessionManager
	playerHandler  *handler.PlayerHandler
	protocol       protolayer.Protocol
	listener       net.Listener
	isRunning      bool
	wg             sync.WaitGroup
}

func NewTCPService(cfg *config.Config, connManager *connection.ConnectionManager, sessionManager *session.SessionManager, playerHandler *handler.PlayerHandler, protocol protolayer.Protocol) *TCPService {
	return &TCPService{
		config:         cfg,
		connManager:    connManager,
		sessionManager: sessionManager,
		playerHandler:  playerHandler,
		protocol:       protocol,
		isRunning:      false,
	}
}

func (ts *TCPService) Name() string {
	return "TCPService"
}

func (ts *TCPService) Start(ctx context.Context) error {
	if ts.isRunning {
		return nil
	}

	zLog.Info("Starting TCP service...", zap.String("addr", ts.config.Server.ListenAddr))

	listener, err := net.Listen("tcp", ts.config.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", ts.config.Server.ListenAddr, err)
	}

	ts.listener = listener
	ts.isRunning = true

	ts.wg.Add(1)
	go ts.acceptConnections(ctx)

	zLog.Info("TCP service started successfully", zap.String("addr", ts.config.Server.ListenAddr))

	return nil
}

func (ts *TCPService) Stop(ctx context.Context) error {
	if !ts.isRunning {
		return nil
	}

	zLog.Info("Stopping TCP service...")

	if ts.listener != nil {
		ts.listener.Close()
	}

	ts.isRunning = false
	ts.wg.Wait()

	zLog.Info("TCP service stopped")

	return nil
}

func (ts *TCPService) acceptConnections(ctx context.Context) {
	defer ts.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := ts.listener.Accept()
			if err != nil {
				if ts.isRunning {
					zLog.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			ts.wg.Add(1)
			go ts.handleConnection(ctx, conn)
		}
	}
}

func (ts *TCPService) handleConnection(ctx context.Context, conn net.Conn) {
	defer ts.wg.Done()
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	sessionID := generateSessionID()

	zLog.Info("New connection", zap.String("session_id", sessionID), zap.String("client_addr", clientAddr))

	ts.sessionManager.CreateSession(sessionID, sessionID, clientAddr)

	conn.SetReadDeadline(time.Now().Add(time.Duration(ts.config.Server.ConnectionTimeout) * time.Second))
	conn.SetWriteDeadline(time.Now().Add(time.Duration(ts.config.Server.ConnectionTimeout) * time.Second))

	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := conn.Read(buffer)
			if err != nil {
				if ts.isRunning {
					zLog.Info("Connection closed", zap.String("session_id", sessionID), zap.Error(err))
				}
				ts.sessionManager.RemoveSession(sessionID)
				return
			}

			if n > 0 {
				conn.SetReadDeadline(time.Now().Add(time.Duration(ts.config.Server.ConnectionTimeout) * time.Second))

				ts.handleMessage(sessionID, conn, buffer[:n])
			}
		}
	}
}

func (ts *TCPService) handleMessage(sessionID string, conn net.Conn, data []byte) {
	session, exists := ts.sessionManager.GetSession(sessionID)
	if !exists {
		zLog.Warn("Session not found", zap.String("session_id", sessionID))
		return
	}

	ts.sessionManager.UpdateLastActive(sessionID)

	zLog.Info("Received message", zap.String("session_id", sessionID), zap.Int("data_len", len(data)))

	msgID, payload, err := ts.protocol.Decode(data)
	if err != nil {
		zLog.Error("Failed to decode message", zap.Error(err))
		return
	}

	ts.processMessage(session, conn, msgID, payload)
}

func (ts *TCPService) processMessage(sess *session.Session, conn net.Conn, msgID uint32, payload []byte) {
	zLog.Info("Processing message", zap.String("session_id", sess.SessionID), zap.Uint32("msg_id", msgID))

	var response proto.Message
	var err error

	switch msgID {
	case 1003:
		response, err = ts.handlePlayerLogin(sess, payload)
	case 1004:
		response, err = ts.handlePlayerCreate(sess, payload)
	case 1005:
		response, err = ts.handlePlayerSelect(sess, payload)
	case 1006:
		response, err = ts.handlePlayerLogout(sess)
	default:
		zLog.Warn("Unknown message ID", zap.String("session_id", sess.SessionID), zap.Uint32("msg_id", msgID))
		return
	}

	if err != nil {
		zLog.Error("Failed to process message", zap.Error(err))
		return
	}

	if response != nil {
		ts.sendResponse(conn, msgID, response)
	}
}

func (ts *TCPService) handlePlayerLogin(sess *session.Session, payload []byte) (*protocol.PlayerLoginResponse, error) {
	zLog.Info("Handling player login", zap.String("session_id", sess.SessionID))

	var req protocol.PlayerLoginRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal player login request", zap.Error(err))
		return nil, err
	}

	sess.AccountID = id.AccountIdType(req.PlayerId)
	sess.Status = session.SessionStatusLoggedIn

	response, err := ts.playerHandler.HandlePlayerLogin(sess.SessionID, id.AccountIdType(req.PlayerId))
	if err != nil {
		zLog.Error("Failed to handle player login", zap.Error(err))
		return nil, err
	}

	zLog.Info("Player login handled", zap.Int64("account_id", int64(req.PlayerId)))
	return response, nil
}

func (ts *TCPService) handlePlayerCreate(sess *session.Session, payload []byte) (*protocol.PlayerCreateResponse, error) {
	zLog.Info("Handling player create", zap.String("session_id", sess.SessionID))

	var req protocol.PlayerCreateRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal player create request", zap.Error(err))
		return nil, err
	}

	response, err := ts.playerHandler.HandlePlayerCreate(sess.SessionID, id.AccountIdType(sess.AccountID), req.Name, req.Sex, req.Age)
	if err != nil {
		zLog.Error("Failed to handle player create", zap.Error(err))
		return nil, err
	}

	zLog.Info("Player create handled", zap.String("player_name", req.Name))
	return response, nil
}

func (ts *TCPService) handlePlayerSelect(sess *session.Session, payload []byte) (*protocol.Response, error) {
	zLog.Info("Handling player select", zap.String("session_id", sess.SessionID))

	var req protocol.PlayerLoginRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal player select request", zap.Error(err))
		return nil, err
	}

	response, err := ts.playerHandler.HandlePlayerSelect(sess.SessionID, id.PlayerIdType(req.PlayerId))
	if err != nil {
		zLog.Error("Failed to handle player select", zap.Error(err))
		return nil, err
	}

	sess.Status = session.SessionStatusInGame

	zLog.Info("Player select handled", zap.Int64("player_id", int64(req.PlayerId)))
	return response, nil
}

func (ts *TCPService) handlePlayerLogout(sess *session.Session) (*protocol.Response, error) {
	zLog.Info("Handling player logout", zap.String("session_id", sess.SessionID))

	response, err := ts.playerHandler.HandlePlayerLogout(sess.SessionID)
	if err != nil {
		zLog.Error("Failed to handle player logout", zap.Error(err))
		return nil, err
	}

	return response, nil
}

// SendResponse 发送响应消息
func (ts *TCPService) SendResponse(conn net.Conn, msgID uint32, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return err
	}

	packet, err := ts.protocol.Encode(msgID, data)
	if err != nil {
		zLog.Error("Failed to encode response", zap.Error(err))
		return err
	}

	_, err = conn.Write(packet)
	if err != nil {
		zLog.Error("Failed to send response", zap.Error(err))
		return err
	}

	zLog.Info("Response sent", zap.Uint32("msg_id", msgID), zap.Int("data_len", len(packet)))
	return nil
}

func (ts *TCPService) sendResponse(conn net.Conn, msgID uint32, msg proto.Message) error {
	return ts.SendResponse(conn, msgID, msg)
}

func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
