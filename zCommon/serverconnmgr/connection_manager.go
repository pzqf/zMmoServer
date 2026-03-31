package serverconnmgr

import (
	"fmt"
	"net"
	"time"

	"github.com/pzqf/zUtil/zMap"
)

type ConnectionManager struct {
	connections   *zMap.TypedMap[string, net.Conn]
	maxRetries    int
	retryInterval time.Duration
}

func NewConnectionManager(maxRetries int, retryInterval time.Duration) *ConnectionManager {
	return &ConnectionManager{
		connections:   zMap.NewTypedMap[string, net.Conn](),
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

func (cm *ConnectionManager) Connect(address string, port int) (net.Conn, error) {
	return cm.ConnectWithRetry(address, port, cm.maxRetries, cm.retryInterval)
}

func (cm *ConnectionManager) ConnectWithRetry(address string, port int, maxRetries int, retryInterval time.Duration) (net.Conn, error) {
	var lastError error
	for i := 0; i < maxRetries; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, port), 5*time.Second)
		if err == nil {
			return conn, nil
		}

		lastError = err
		time.Sleep(retryInterval * time.Duration(i+1))
	}

	return nil, fmt.Errorf("connect failed after %d retries: %w", maxRetries, lastError)
}

func (cm *ConnectionManager) IsConnected(id string) bool {
	_, exists := cm.connections.Load(id)
	return exists
}

func (cm *ConnectionManager) GetConnection(id string) (net.Conn, error) {
	conn, exists := cm.connections.Load(id)
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	return conn, nil
}

func (cm *ConnectionManager) AddConnection(id string, conn net.Conn) error {
	cm.connections.Store(id, conn)
	return nil
}

func (cm *ConnectionManager) RemoveConnection(id string) error {
	conn, exists := cm.connections.LoadAndDelete(id)
	if !exists {
		return fmt.Errorf("connection not found: %s", id)
	}

	conn.Close()
	return nil
}

func (cm *ConnectionManager) Close(id string) error {
	return cm.RemoveConnection(id)
}

func (cm *ConnectionManager) CloseAll() error {
	cm.connections.Range(func(id string, conn net.Conn) bool {
		conn.Close()
		return true
	})
	cm.connections.Clear()
	return nil
}

func (cm *ConnectionManager) GetConnectionCount() int {
	return int(cm.connections.Len())
}
