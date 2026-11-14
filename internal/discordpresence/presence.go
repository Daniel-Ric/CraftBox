package discordpresence

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Settings struct {
	Enabled      bool   `json:"enabled"`
	Details      string `json:"details"`
	State        string `json:"state"`
	Automatic    bool   `json:"automatic"`
	ShowElapsed  bool   `json:"showElapsed"`
	LargeText    string `json:"largeText"`
	SmallImage   string `json:"smallImage"`
	SmallText    string `json:"smallText"`
	HideGitHub   bool   `json:"hideGitHub"`
	Button1Label string `json:"button1Label"`
	Button1URL   string `json:"button1Url"`
}

const ApplicationID = "1571194794569498714"

func (s Settings) Validate() error {
	if !s.Enabled {
		return nil
	}
	for _, v := range []string{s.Details, s.State, s.LargeText, s.SmallText} {
		if !utf8.ValidString(v) || utf8.RuneCountInString(v) > 128 {
			return errors.New("Discord status text must contain at most 128 characters")
		}
	}
	for _, v := range []string{s.SmallImage} {
		if !utf8.ValidString(v) || utf8.RuneCountInString(v) > 300 {
			return errors.New("Discord image keys or URLs must contain at most 300 characters")
		}
	}
	for _, b := range [][2]string{{s.Button1Label, s.Button1URL}} {
		label, link := strings.TrimSpace(b[0]), strings.TrimSpace(b[1])
		if label == "" && link == "" {
			continue
		}
		u, err := url.Parse(link)
		if label == "" || utf8.RuneCountInString(label) > 32 || len(link) > 512 || err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
			return errors.New("Each Discord button needs a label (up to 32 characters) and a valid http(s) URL (up to 512 characters)")
		}
	}
	return nil
}

type Activity struct {
	Details    string            `json:"details,omitempty"`
	State      string            `json:"state,omitempty"`
	Timestamps map[string]int64  `json:"timestamps,omitempty"`
	Assets     map[string]string `json:"assets,omitempty"`
	Buttons    []Button          `json:"buttons,omitempty"`
}
type Button struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func (s Settings) Activity(status string, started time.Time) Activity {
	a := Activity{Details: strings.TrimSpace(s.Details), State: strings.TrimSpace(s.State)}
	if s.Automatic {
		a.State = status
	}
	if s.ShowElapsed {
		a.Timestamps = map[string]int64{"start": started.Unix()}
	}
	a.Assets = map[string]string{"large_image": "https://raw.githubusercontent.com/Daniel-Ric/CraftBox/master/ui/desktop/frontend/public/craft-box-icon.png"}
	if text := strings.TrimSpace(s.LargeText); text != "" {
		a.Assets["large_text"] = text
	}
	if image := strings.TrimSpace(s.SmallImage); image != "" {
		a.Assets["small_image"] = image
		if text := strings.TrimSpace(s.SmallText); text != "" {
			a.Assets["small_text"] = text
		}
	}
	if !s.HideGitHub {
		a.Buttons = append(a.Buttons, Button{Label: "CraftBox on GitHub", URL: "https://github.com/Daniel-Ric/CraftBox"})
	}
	for _, b := range [][2]string{{s.Button1Label, s.Button1URL}} {
		if strings.TrimSpace(b[0]) != "" && strings.TrimSpace(b[1]) != "" {
			a.Buttons = append(a.Buttons, Button{Label: strings.TrimSpace(b[0]), URL: strings.TrimSpace(b[1])})
		}
	}
	return a
}

type stream interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
}
type client struct {
	conn  stream
	nonce uint64
}

func writeFrame(w io.Writer, op uint32, body []byte) error {
	b := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(b, op)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(body)))
	copy(b[8:], body)
	n, err := w.Write(b)
	if err == nil && n != len(b) {
		return io.ErrShortWrite
	}
	return err
}

func (c *client) receive(nonce string) error {
	for {
		var h [8]byte
		if _, err := io.ReadFull(c.conn, h[:]); err != nil {
			return err
		}
		op, size := binary.LittleEndian.Uint32(h[:]), binary.LittleEndian.Uint32(h[4:])
		if size > 1024*1024 {
			return errors.New("Discord IPC frame too large")
		}
		body := make([]byte, size)
		if _, err := io.ReadFull(c.conn, body); err != nil {
			return err
		}
		if op == 3 {
			if err := writeFrame(c.conn, 4, body); err != nil {
				return err
			}
			continue
		}
		if op != 1 {
			return errors.New("Discord IPC disconnected")
		}
		var reply struct {
			Event string `json:"evt"`
			Nonce string `json:"nonce"`
			Data  struct {
				Message string `json:"message"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &reply); err != nil {
			return err
		}
		if reply.Event == "ERROR" {
			return fmt.Errorf("Discord: %s", reply.Data.Message)
		}
		if nonce == "" && reply.Event == "READY" || nonce != "" && reply.Nonce == nonce {
			return nil
		}
	}
}

func handshake(conn stream) (*client, error) {
	c := &client{conn: conn}
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(map[string]any{"v": 1, "client_id": ApplicationID})
	if err := writeFrame(conn, 0, b); err != nil {
		return nil, err
	}
	if err := c.receive(""); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *client) SetActivity(a *Activity) error {
	if err := c.conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	c.nonce++
	nonce := strconv.FormatUint(c.nonce, 10)
	b, err := json.Marshal(map[string]any{"cmd": "SET_ACTIVITY", "args": map[string]any{"pid": os.Getpid(), "activity": a}, "nonce": nonce})
	if err != nil {
		return err
	}
	if err = writeFrame(c.conn, 1, b); err != nil {
		return err
	}
	return c.receive(nonce)
}
