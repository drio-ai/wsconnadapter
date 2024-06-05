// an adapter for representing WebSocket connection as a net.Conn
// some caveats apply: https://github.com/gorilla/websocket/issues/441

// Picked up just the adapter from https://github.com/function61/holepunch-server
// Revision when file was picked up is 8f5e8775e813bbec97c09e7988c0c780486a9df2

// The same file is present in a different repository here https://github.com/xandout/soxy
// I am not sure if the holepunch-server and soxy are related or not. They are both distributed
// with an Apache 2.0 license and are public.

package wsconnadapter

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Adapter struct {
	conn       *websocket.Conn
	readMutex  sync.Mutex
	writeMutex sync.Mutex
	reader     io.Reader
}

func NewAdapter(conn *websocket.Conn) *Adapter {
	return &Adapter{
		conn: conn,
	}
}

func (a *Adapter) Read(b []byte) (int, error) {
	// Read() can be called concurrently, and we mutate some internal state here
	a.readMutex.Lock()
	defer a.readMutex.Unlock()

	if a.reader == nil {
		_, reader, err := a.conn.NextReader()
		if err != nil {
			return 0, err
		}

		a.reader = reader
	}

	// Removed messageType check to make sure it is binary. Our use for this adapter
	// is to integrate it with WebSocket and then send STOMP traffic over it. It will
	// be a text message. We don't want to restrict this to text only either.

	bytesRead, err := a.reader.Read(b)
	if err != nil {
		a.reader = nil

		// EOF for the current Websocket frame, more will probably come so..
		if err == io.EOF {
			// .. we must hide this from the caller since our semantics are a
			// stream of bytes across many frames
			err = nil
		}
	}

	return bytesRead, err
}

func (a *Adapter) Write(b []byte) (int, error) {
	a.writeMutex.Lock()
	defer a.writeMutex.Unlock()

	nextWriter, err := a.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}

	bytesWritten, err := nextWriter.Write(b)
	nextWriter.Close()

	return bytesWritten, err
}

func (a *Adapter) Close() error {
	return a.conn.Close()
}

func (a *Adapter) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *Adapter) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *Adapter) SetDeadline(t time.Time) error {
	if err := a.SetReadDeadline(t); err != nil {
		return err
	}

	return a.SetWriteDeadline(t)
}

func (a *Adapter) SetReadDeadline(t time.Time) error {
	return a.conn.SetReadDeadline(t)
}

func (a *Adapter) SetWriteDeadline(t time.Time) error {
	return a.conn.SetWriteDeadline(t)
}
