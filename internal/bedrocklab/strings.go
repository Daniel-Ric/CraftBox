package bedrocklab

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
)

type StringMatch struct {
	Offset   int64  `json:"offset"`
	Text     string `json:"text"`
	Encoding string `json:"encoding"`
}

func SearchStrings(ctx context.Context, name, query string, minimum int) ([]StringMatch, error) {
	result := []StringMatch{}
	if len(query) < 2 || len(query) > 256 {
		return result, errors.New("enter between 2 and 256 search characters")
	}
	if minimum < 4 {
		minimum = 4
	}
	if minimum > 1024 {
		minimum = 1024
	}
	query = strings.ToLower(query)
	f, e := os.Open(name)
	if e != nil {
		return result, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return result, e
	}
	if !info.Mode().IsRegular() || info.Size() > 2<<30 {
		return result, errors.New("choose a regular executable up to 2 GiB")
	}
	type run struct {
		start int64
		b     []byte
	}
	runs := [3]run{}
	previous := byte(0)
	position := int64(0)
	flush := func(index int) {
		r := &runs[index]
		if len(r.b) >= minimum && strings.Contains(strings.ToLower(string(r.b)), query) && len(result) < 1000 {
			encoding := "ASCII"
			if index > 0 {
				encoding = "UTF-16LE"
			}
			result = append(result, StringMatch{r.start, string(r.b), encoding})
		}
		r.b = r.b[:0]
	}
	push := func(index int, b byte, at int64, valid bool) {
		r := &runs[index]
		if valid {
			if len(r.b) == 0 {
				r.start = at
			}
			if len(r.b) < 8192 {
				r.b = append(r.b, b)
			}
		} else {
			flush(index)
		}
	}
	buffer := make([]byte, 64<<10)
	for {
		if e = ctx.Err(); e != nil {
			return result, e
		}
		n, err := f.Read(buffer)
		for _, b := range buffer[:n] {
			push(0, b, position, b >= 32 && b <= 126)
			if position > 0 {
				index := 1 + int((position-1)%2)
				push(index, previous, position-1, b == 0 && previous >= 32 && previous <= 126)
			}
			previous = b
			position++
		}
		if len(result) >= 1000 {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, err
		}
	}
	for i := range runs {
		flush(i)
	}
	return result, nil
}
