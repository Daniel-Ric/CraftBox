package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/creachadair/jrpc2"
	"github.com/df-mc/go-nethernet"
	"github.com/google/uuid"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling/internal"
)


type Conn struct {

	conn   *websocket.Conn
	client *jrpc2.Client

	d Dialer

	pmid uuid.UUID


	once sync.Once


	ctx    context.Context
	cancel context.CancelCauseFunc

	notifier *internal.Notifier

	pending *internal.PendingMap



	credentials *nethernet.Credentials

	credentialsExpiry time.Time

	credentialsMu sync.Mutex
}


func (conn *Conn) Signal(ctx context.Context, signal *nethernet.Signal) error {
	messagingID, err := uuid.Parse(signal.NetworkID)
	if err != nil {
		return fmt.Errorf("signaling/messaging: recipient ID must be a UUID: %q", signal.NetworkID)
	}

	id := uuid.New()
	ch := conn.pending.Add(id)
	defer conn.pending.Remove(id)

	if err := conn.send(ctx, id, map[string]any{
		"params": map[string]any{
			"netherNetId": conn.d.NetworkID,
			"message":     signal.String(),
		},
		"jsonrpc": "2.0",
		"method":  MethodSignalingWebRTC,
	}, messagingID); err != nil {
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


func (conn *Conn) send(ctx context.Context, id uuid.UUID, inner any, messagingID uuid.UUID) error {
	data, err := json.Marshal(inner)
	if err != nil {
		return fmt.Errorf("encode inner message: %w", err)
	}
	_, err = conn.client.Call(ctx, MethodSignalingSendMessage, map[string]any{
		"toPlayerId": messagingID,
		"messageId":  id,
		"message":    string(data),
	})
	if err != nil {
		return fmt.Errorf("call %q: %w", MethodSignalingSendMessage, err)
	}
	return nil
}





func (conn *Conn) Notify(notifier nethernet.Notifier) (stop func()) {
	return conn.notifier.Register(notifier)
}




func (conn *Conn) Credentials(ctx context.Context) (*nethernet.Credentials, error) {
	conn.credentialsMu.Lock()
	defer conn.credentialsMu.Unlock()

	if conn.credentials != nil && time.Now().Before(conn.credentialsExpiry) {
		return conn.credentials, nil
	}

	var credentials nethernet.Credentials
	if err := conn.client.CallResult(ctx, MethodSignalingCredentials, map[string]any{}, &credentials); err != nil {
		return nil, fmt.Errorf("call %q: %w", MethodSignalingCredentials, err)
	}

	conn.credentials = &credentials
	conn.credentialsExpiry = time.Now().Add(time.Duration(credentials.ExpirationInSeconds) * time.Second)

	return conn.credentials, nil
}


func (conn *Conn) PongData(b []byte) {
}




func (conn *Conn) NetworkID() string {
	return conn.pmid.String()
}


func (conn *Conn) PlayerMessagingID() uuid.UUID {
	return conn.pmid
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
		err = conn.client.Close()
	})
	return err
}



func (conn *Conn) handleCallback(ctx context.Context, request *jrpc2.Request) (any, error) {
	switch request.Method() {
	case MethodSystemPong:
		return nil, nil
	case MethodSignalingReceiveMessage:
		return conn.handleMessage(ctx, request)
	default:
		return nil, fmt.Errorf("received message for unknown method: %q", request.Method())
	}
}




func (conn *Conn) handleMessage(ctx context.Context, request *jrpc2.Request) (v any, err error) {
	var param envelope
	if err := request.UnmarshalParams(&param); err != nil {
		return nil, fmt.Errorf("decode parameters: %w", err)
	}
	if err2 := conn.handleInnerMessage(ctx, &param); err2 != nil {
		err = errors.Join(err, err2)
	}
	return nil, err
}


type envelope struct {

	From uuid.UUID

	Message *jrpc2.ParsedRequest

	RawMessage string `json:"-"`


	ID uuid.UUID `json:"Id"`
}



func (m *envelope) UnmarshalJSON(b []byte) error {
	type Alias envelope
	data := struct {
		*Alias
		Message string
	}{Alias: (*Alias)(m)}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(data.Message), &m.Message); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	m.RawMessage = data.Message
	return nil
}


func (conn *Conn) handleInnerMessage(ctx context.Context, envelope *envelope) error {
	switch envelope.Message.Method {
	case MethodSignalingDeliveryNotification:
		var params struct {
			MessageID uuid.UUID `json:"messageId"`
		}
		if err := json.Unmarshal(envelope.Message.Params, &params); err != nil {
			return fmt.Errorf("decode request parameters: %w", err)
		}
		if params.MessageID == uuid.Nil {
			return errors.New("invalid request parameters")
		}
		if !conn.pending.Done(params.MessageID, nil) {
			return fmt.Errorf("unexpected message ID: %s", params.MessageID)
		}
		return nil
	case MethodSignalingWebRTC:
		var params struct {
			NetherNetID string `json:"netherNetId"`
			Message     string `json:"message"`
		}
		if err := json.Unmarshal(envelope.Message.Params, &params); err != nil {
			return fmt.Errorf("decode request parameters: %w", err)
		}
		signal := &nethernet.Signal{NetworkID: envelope.From.String()}
		if err := signal.UnmarshalText([]byte(params.Message)); err != nil {
			return fmt.Errorf("decode signal: %w", err)
		}
		conn.notifier.Signal(signal)

		if err := conn.send(ctx, uuid.New(), map[string]any{
			"params": map[string]any{
				"messageId": envelope.ID,
			},
			"jsonrpc": "2.0",
			"method":  MethodSignalingDeliveryNotification,
		}, envelope.From); err != nil {
			return fmt.Errorf("acknowledge message: %w", err)
		}
		return nil
	default:
		if envelope.RawMessage != "" {
			dec := json.NewDecoder(strings.NewReader(envelope.RawMessage))
			dec.DisallowUnknownFields()
			var e signaling.Error
			if err := dec.Decode(&e); err == nil {
				if !conn.pending.Done(envelope.ID, &e) {
					return fmt.Errorf("unexpected message ID: %s", envelope.ID)
				}
				return nil
			}
		}
		return fmt.Errorf("unknown inner request method: %q", envelope.Message.Method)
	}
}



func (conn *Conn) ping() {
	ticker := time.NewTicker(time.Second * 50)
	defer ticker.Stop()

	for {
		select {
		case <-conn.ctx.Done():
			return
		case <-ticker.C:
			if _, err := conn.client.Call(conn.ctx, MethodSystemPing, map[string]any{}); err != nil {
				conn.d.Log.Error("error pinging", slog.Any("error", err))
				_ = conn.close(fmt.Errorf("call %q: %w", MethodSystemPing, err))
				return
			}
		}
	}
}

const (


	MethodSystemPing = "System_Ping_v1_0"


	MethodSystemPong = "System_Pong_v1_0"




	MethodSignalingCredentials = "Signaling_TurnAuth_v1_0"


	MethodSignalingReceiveMessage = "Signaling_ReceiveMessage_v1_0"


	MethodSignalingSendMessage = "Signaling_SendClientMessage_v1_0"




	MethodSignalingDeliveryNotification = "Signaling_DeliveryNotification_V1_0"


	MethodSignalingWebRTC = "Signaling_WebRtc_v1_0"
)
