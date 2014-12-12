package server

import (
	. "github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/packet"
	"net"
	"sync/atomic"
)

/*
   Conn acts like a MySQL server connection, you can use MySQL client to communicate with it.
*/
type Conn struct {
	*packet.Conn

	capability uint32

	connectionID uint32

	status uint16

	user string

	salt []byte

	h Handler

	stmts  map[uint32]*Stmt
	stmtID uint32
}

var baseConnID uint32 = 10000

func NewConn(conn net.Conn, user string, password string, h Handler) (*Conn, error) {
	c := new(Conn)

	c.h = h

	c.user = user
	c.Conn = packet.NewConn(conn)

	c.connectionID = atomic.AddUint32(&baseConnID, 1)

	c.stmts = make(map[uint32]*Stmt)

	c.salt, _ = RandomBuf(20)

	if err := c.handshake(password); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Conn) handshake(password string) error {
	if err := c.writeInitialHandshake(); err != nil {
		return err
	}

	if err := c.readHandshakeResponse(password); err != nil {
		c.writeError(err)

		return err
	}

	if err := c.writeOK(nil); err != nil {
		return err
	}

	c.ResetSequence()

	return nil
}

func (c *Conn) Close() {
	if c.Conn != nil {
		c.Conn.Close()
		c.Conn = nil
	}
}

func (c *Conn) Closed() bool {
	return c.Conn == nil
}

func (c *Conn) GetUser() string {
	return c.user
}

func (c *Conn) ConnectionID() uint32 {
	return c.connectionID
}

func (c *Conn) IsAutoCommit() bool {
	return c.status&SERVER_STATUS_AUTOCOMMIT > 0
}

func (c *Conn) IsInTransaction() bool {
	return c.status&SERVER_STATUS_IN_TRANS > 0
}

func (c *Conn) SetInTransaction() {
	c.status |= SERVER_STATUS_IN_TRANS
}

func (c *Conn) ClearInTransaction() {
	c.status &= ^SERVER_STATUS_IN_TRANS
}