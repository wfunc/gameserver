// network/connection.go
package network

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Packet struct {
	MsgID  uint16
	Data   []byte
	Length uint16
}

type Connection interface {
	Send(msgID uint16, data []byte) error
	Close() error
	RemoteAddr() net.Addr
	SetHeartbeat(interval time.Duration)
	ReadPacket() (*Packet, error)
}

type WSConnection struct {
	conn      *websocket.Conn
	sendMutex sync.Mutex
	heartbeat time.Duration
}

func NewWSConnection(conn *websocket.Conn) *WSConnection {
	return &WSConnection{conn: conn}
}

func (c *WSConnection) Send(msgID uint16, data []byte) error {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	// 封包: 2字节消息ID + 2字节数据长度 + 数据
	packet := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(packet[0:2], msgID)
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(data)))
	copy(packet[4:], data)

	return c.conn.WriteMessage(websocket.BinaryMessage, packet)
}

func (c *WSConnection) ReadPacket() (*Packet, error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if len(data) < 4 {
		return nil, io.ErrShortBuffer
	}

	msgID := binary.BigEndian.Uint16(data[0:2])
	length := binary.BigEndian.Uint16(data[2:4])

	if len(data) < int(4+length) {
		return nil, io.ErrShortBuffer
	}

	return &Packet{
		MsgID:  msgID,
		Length: length,
		Data:   data[4 : 4+length],
	}, nil
}

func (c *WSConnection) SetHeartbeat(interval time.Duration) {
	c.heartbeat = interval
	c.conn.SetReadDeadline(time.Now().Add(interval * 2))
}

func (c *WSConnection) Close() error {
	return c.conn.Close()
}

func (c *WSConnection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}