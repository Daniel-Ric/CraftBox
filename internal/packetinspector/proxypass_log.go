package packetinspector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

func (s *Store) importProxyPass(id string, in io.Reader, day time.Time) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 65536), maxRecord)
	count := 0
	var previous int64
	var current Packet
	var body strings.Builder
	flush := func() error {
		if current.Name == "" {
			return nil
		}
		payload, _ := json.Marshal(map[string]string{"text": body.String()})
		current.Payload = string(payload)
		if _, e := s.Append(id, current); e != nil {
			return e
		}
		count++
		body.Reset()
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		m := proxyLine.FindStringSubmatch(line)
		if m == nil {
			if current.Name == "" {
				if strings.TrimSpace(line) == "" {
					continue
				}
				return fmt.Errorf("Unrecognized ProxyPass log row %d", count+1)
			}
			if body.Len()+len(line)+1 > maxRecord-1024 {
				return errors.New("ProxyPass packet text exceeds 32 MiB")
			}
			body.WriteByte('\n')
			body.WriteString(line)
			continue
		}
		if e := flush(); e != nil {
			return e
		}
		stamp, e := time.ParseInLocation("2006-01-02 15:04:05.000", day.Format("2006-01-02")+" "+m[1][:8]+"."+m[1][9:], day.Location())
		if e != nil {
			return e
		}
		if previous > 0 && stamp.UnixMilli() < previous-12*3600000 {
			day = day.AddDate(0, 0, 1)
			stamp = stamp.AddDate(0, 0, 1)
		}
		previous = stamp.UnixMilli()
		direction := "clientbound"
		if m[2] == "SERVER BOUND" {
			direction = "serverbound"
		}
		current = Packet{Name: strings.TrimSuffix(m[3], "Packet"), Direction: direction, Time: stamp.UnixMilli(), Stage: "imported", Partial: true, Reason: "ProxyPass text log: inferred date, no original wire bytes or structured fields"}
		body.WriteString(m[3] + m[4])
	}
	if e := scanner.Err(); e != nil {
		return e
	}
	if e := flush(); e != nil {
		return e
	}
	if count == 0 {
		return errors.New("No ProxyPass packet rows found")
	}
	return nil
}
