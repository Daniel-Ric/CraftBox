package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/df-mc/go-nethernet"
	"github.com/google/uuid"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling/internal"
)


type Conn struct {

	conn *websocket.Conn

	d Dialer



	credentials atomic.Pointer[nethernet.Credentials]

	credentialsReceived chan struct{}


	once sync.Once


	ctx    context.Context
	cancel context.CancelCauseFunc

	notifier *internal.Notifier

	pending *internal.PendingMap
}


func (conn *Conn) Signal(ctx context.Context, signal *nethernet.Signal) error {
	id := uuid.New()
	ch := conn.pending.Add(id)
	defer conn.pending.Remove(id)

	if err := conn.write(Message{
		Type: MessageTypeSignal,
		To:   signal.NetworkID,
		Data: signal.String(),
		ID:   id,
	}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.ctx.Done():
		return context.Cause(conn.ctx)
	case err := <-ch:
		return err
	}
}





func (conn *Conn) Notify(notifier nethernet.Notifier) (stop func()) {
	return conn.notifier.Register(notifier)
}





func (conn *Conn) complete(id uuid.UUID, err error) {
	if !conn.pending.Done(id, err) {
		conn.d.Log.Warn("unexpected message ID", slog.String("id", id.String()), slog.Any("result", err))
	}
}




func (conn *Conn) Credentials(ctx context.Context) (*nethernet.Credentials, error) {
	select {
	case <-conn.ctx.Done():
		return nil, context.Cause(conn.ctx)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-conn.credentialsReceived:
		return conn.credentials.Load(), nil
	}
}

func (conn *Conn) PongData(b []byte) {
}




func (conn *Conn) NetworkID() string {
	return conn.d.NetworkID
}



func (conn *Conn) Close() (err error) {
	return conn.close(net.ErrClosed)
}



func (conn *Conn) Context() context.Context {
	return conn.ctx
}

func (conn *Conn) close(cause error) (err error) {
	conn.once.Do(func() {
		conn.d.Log.Debug("closing connection", slog.Any("cause", cause))

		_ = conn.notifier.Close()

		conn.cancel(cause)
		err = conn.conn.Close(websocket.StatusNormalClosure, "")
	})
	return err
}






func (conn *Conn) read() {
	go func() {
		ticker := time.NewTicker(time.Second * 15)
		defer ticker.Stop()

		for {
			select {
			case <-conn.ctx.Done():
				return
			case <-ticker.C:
				if err := conn.write(Message{
					Type: MessageTypePing,
				}); err != nil {
					_ = conn.close(fmt.Errorf("background: write ping: %w", err))
					return
				}
			}
		}
	}()

	for {
		var message Message
		if err := wsjson.Read(context.Background(), conn.conn, &message); err != nil {
			_ = conn.close(err)
			return
		}
		log := conn.d.Log.With(slog.Any("message", message))
		if message.ID == uuid.Nil {
			log.Warn("received message without an ID", slog.Any("message", message))
			continue
		}
		switch message.Type {
		case MessageTypeCredentials:
			if message.From != "Server" {
				log.Warn("received credentials from non-Server", slog.Any("message", message))
				continue
			}
			var credentials nethernet.Credentials
			if err := json.Unmarshal([]byte(message.Data), &credentials); err != nil {
				log.Error("error decoding credentials", slog.Any("error", err))
				continue
			}
			closeChan := conn.credentials.Load() == nil
			conn.credentials.Store(&credentials)
			if closeChan {
				close(conn.credentialsReceived)
			}
		case MessageTypeSignal:
			signal := &nethernet.Signal{}
			if err := signal.UnmarshalText([]byte(message.Data)); err != nil {
				log.Error("error decoding signal", slog.Any("error", err))
				continue
			}
			signal.NetworkID = message.From

			conn.notifier.Signal(signal)
		case MessageTypeError:
			err := &Error{}
			if err2 := json.Unmarshal([]byte(message.Data), err); err2 != nil {
				log.Error("error decoding error", slog.Any("error", err2))
				continue
			}
			log.Debug("received error", slog.Any("message", message))
			conn.complete(message.ID, err)
		case MessageTypeDelivered:
			var status MessageStatus
			if err := json.Unmarshal([]byte(message.Data), &status); err != nil {
				log.Error("error decoding message status", slog.Any("message", message), slog.Any("error", err))
				continue
			}
			if status.DeliveredOn.IsZero() {
				log.Warn("delivery time is not included in message data")
			}
			conn.complete(message.ID, nil)
		case MessageTypeAccepted:
			continue
		default:
			log.Warn("received message for unknown type")
		}
	}
}




func (conn *Conn) write(message Message) error {
	return wsjson.Write(context.Background(), conn.conn, message)
}
