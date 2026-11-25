package websocketadapter

import (
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

type WsReadWriteCloser struct {
	*websocket.Conn
	reader io.Reader
	mtx    sync.Mutex
}

func NewWsReadWriteCloser(wsConn *websocket.Conn) io.ReadWriteCloser {
	return &WsReadWriteCloser{Conn: wsConn}
}

func (w *WsReadWriteCloser) Read(data []byte) (int, error) {
	for {
		if w.reader == nil {

			t, r, err := w.NextReader()
			if err != nil {
				return 0, err
			}
			if t != websocket.BinaryMessage {
				continue
			}

			w.reader = r
		}

		n, err := w.reader.Read(data)
		if err == io.EOF {
			w.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}

		return n, err
	}

}

func (w *WsReadWriteCloser) Write(data []byte) (int, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	err := w.Conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (w *WsReadWriteCloser) Close() error {
	return w.Close()
}
