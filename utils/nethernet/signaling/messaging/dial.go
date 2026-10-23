package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/creachadair/jrpc2"
	"github.com/sandertv/gophertunnel/minecraft/service"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling/internal"
)


type Dialer struct {


	Environment *signaling.Environment

	HTTPClient *http.Client

	Log *slog.Logger




	NetworkID string
}


func (d Dialer) Dial(src service.TokenSource) (*Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	return d.DialContext(ctx, src)
}


func (d Dialer) DialContext(ctx context.Context, src service.TokenSource) (*Conn, error) {
	if d.Environment == nil {
		discovery, err := service.Default(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover network services: %w", err)
		}
		d.Environment = new(signaling.Environment)
		if err := discovery.Environment(d.Environment); err != nil {
			return nil, fmt.Errorf("resolve environment for %q: %w", d.Environment.ServiceName(), err)
		}
	}
	if d.HTTPClient == nil {
		d.HTTPClient = http.DefaultClient
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.NetworkID == "" {
		d.NetworkID = strconv.FormatUint(rand.Uint64(), 10)
	}

	token, err := src.ServiceToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("request service token: %w", err)
	}

	opts := &websocket.DialOptions{
		HTTPHeader: make(http.Header),
		HTTPClient: d.HTTPClient,
	}
	opts.HTTPHeader.Set("Authorization", token.AuthorizationHeader)
	requestURL := d.Environment.ServiceURI.JoinPath("/ws/v1.0/messaging/connect")
	c, _, err := websocket.Dial(ctx, requestURL.String(), opts)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		conn: c,
		d:    d,
		pmid: token.Claims.PlayerMessagingID,

		notifier: internal.NewNotifier(d.Log),
		pending:  internal.NewPendingMap(),
	}
	conn.ctx, conn.cancel = context.WithCancelCause(context.Background())
	conn.client = jrpc2.NewClient(&websocketChannel{c}, &jrpc2.ClientOptions{
		Logger: func(text string) {
			d.Log.Debug(text, slog.String("src", "jrpc2"))
		},
		OnNotify: func(request *jrpc2.Request) {
			d.Log.Warn("notification received", slog.Group("request",
				slog.String("id", request.ID()),
				slog.String("params", request.ParamString()),
			))
		},
		OnCallback: func(ctx context.Context, request *jrpc2.Request) (v any, err error) {
			defer func() {
				if err2 := recover(); err2 != nil {
					d.Log.Error("callback handler panicked", slog.Any("error", err2))
					v, err = nil, nil
				}
			}()
			v, err = conn.handleCallback(ctx, request)
			if err != nil {


				conn.d.Log.Error("error handling server message",
					slog.GroupAttrs("request",
						slog.String("id", request.ID()),
						slog.String("method", request.Method()),
						slog.String("params", request.ParamString()),
					),
					slog.Any("error", err),
				)
				return nil, nil
			}
			return v, nil
		},
	})
	go conn.ping()
	return conn, nil
}



type websocketChannel struct{ *websocket.Conn }


func (ch *websocketChannel) Send(b []byte) error {
	return ch.Write(context.Background(), websocket.MessageText, b)
}



func (ch *websocketChannel) Recv() ([]byte, error) {
	_, msg, err := ch.Read(context.Background())
	return msg, err
}


func (ch *websocketChannel) Close() error {
	return ch.Conn.Close(websocket.StatusNormalClosure, "")
}
