package marketplace

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func contains(s, part string) bool { return strings.Contains(s, part) }
func decode(raw []byte) (map[string]any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var m map[string]any
	e := d.Decode(&m)
	if e == nil && m == nil {
		e = errors.New("Catalog item is not an object")
	}
	return m, e
}
func field(m map[string]any, key string) any {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}
func text(v any) string           { s, _ := v.(string); return strings.TrimSpace(s) }
func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func list(v any) []any            { a, _ := v.([]any); return a }
func integer(v any) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, e := n.Int64()
	return i, e == nil
}
func str(m map[string]any, key string) string { return text(field(m, key)) }
func stringsOf(v any) []string {
	out := []string{}
	for _, v := range list(v) {
		if s := text(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}
func english(v any) string {
	m := object(v)
	for _, k := range []string{"en-US", "en-GB", "en", "NEUTRAL"} {
		if s := str(m, k); s != "" {
			return s
		}
	}
	return text(v)
}
func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
func SafeImage(value string) string {
	u, e := url.Parse(value)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Host == "" || len(value) > 8192 {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return ""
	}
	return u.String()
}
func sanitize(v any, parent string, depth int) any {
	if depth > 32 {
		return "[depth limit]"
	}
	switch n := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, x := range n {
			low := strings.ToLower(k)
			if strings.Contains(low, "token") || strings.Contains(low, "secret") || strings.Contains(low, "password") || strings.Contains(low, "contentkey") || low == "key" || strings.Contains(low, "authorization") {
				out[k] = "[hidden]"
				continue
			}
			if (strings.EqualFold(parent, "Contents") || strings.EqualFold(parent, "Content")) && (low == "url" || low == "downloadurl") {
				out[k] = "[binary URL omitted]"
				continue
			}
			out[k] = sanitize(x, k, depth+1)
		}
		return out
	case []any:
		out := []any{}
		for _, x := range n {
			out = append(out, sanitize(x, parent, depth+1))
		}
		return out
	case string:
		if u, e := url.Parse(n); e == nil && u.IsAbs() {
			q := u.Query()
			for k := range q {
				low := strings.ToLower(k)
				if strings.Contains(low, "sig") || strings.Contains(low, "token") || low == "key" {
					return "[signed URL omitted]"
				}
			}
		}
		return n
	default:
		return v
	}
}
func Normalize(raw json.RawMessage, title string) (Item, error) {
	m, e := decode(raw)
	// Some catalog search deployments wrap the catalog item in a search hit.
	// Normalize the actual item so IDs, tags and prices all use the same scope.
	if e == nil && str(m, "Id") == "" && object(field(m, "Item")) != nil {
		m = object(field(m, "Item"))
		raw, e = json.Marshal(m)
	}
	if e != nil {
		return Item{}, e
	}
	dp := object(field(m, "DisplayProperties"))
	id := str(m, "Id")
	if id == "" {
		return Item{}, errors.New("Catalog item has no ID")
	}
	item := Item{ID: id, TitleID: title, Name: first(english(field(m, "Title")), id), Description: english(field(m, "Description")), Creator: str(dp, "creatorName"), CreatorID: creatorID(m), Type: str(m, "ContentType"), Kind: str(m, "Type"), Images: []Image{}, Tags: rawTags(field(m, "Tags")), Platforms: stringsOf(field(m, "Platforms")), IDs: []string{id}, Prices: []Price{}, References: []Reference{}, Compatibility: "unknown", Provenance: []Provenance{}}
	for _, entry := range list(field(m, "Images")) {
		im := object(entry)
		if u := SafeImage(str(im, "Url")); u != "" {
			item.Images = append(item.Images, Image{u, str(im, "Type"), str(im, "Tag")})
		}
	}
	sort.SliceStable(item.Images, func(i, j int) bool {
		return strings.Contains(strings.ToLower(item.Images[i].Type+item.Images[i].Tag), "thumbnail") && !strings.Contains(strings.ToLower(item.Images[j].Type+item.Images[j].Tag), "thumbnail")
	})
	for _, a := range list(field(m, "AlternateIds")) {
		if s := str(object(a), "Value"); s != "" {
			item.IDs = append(item.IDs, s)
		}
	}
	for _, r := range list(field(m, "ItemReferences")) {
		ref := object(r)
		if s := str(ref, "Id"); s != "" {
			item.References = append(item.References, Reference{s, str(ref, "Type")})
		}
	}
	for _, r := range list(field(object(field(m, "PriceOptions")), "Prices")) {
		p := object(r)
		price := Price{Amounts: []Amount{}}
		price.Units, _ = integer(field(p, "UnitAmount"))
		price.Duration, _ = integer(field(p, "UnitDurationInSeconds"))
		for _, a := range list(field(p, "Amounts")) {
			money := object(a)
			value, ok := integer(field(money, "Amount"))
			currency := first(str(money, "ItemId"), str(money, "CurrencyId"), str(money, "Id"))
			if ok && value >= 0 && currency != "" {
				price.Amounts = append(price.Amounts, Amount{Currency: currency, Value: value})
			}
		}
		if len(price.Amounts) > 0 {
			item.Prices = append(item.Prices, price)
		}
	}
	if len(item.Prices) == 0 {
		if value, ok := integer(field(dp, "price")); ok && value >= 0 {
			currency := first(str(dp, "currencyId"), str(dp, "currency"))
			if currency == "" && (strings.EqualFold(title, "20CA2") || strings.EqualFold(title, "EE7B")) {
				currency = "Minecoins"
			}
			if currency != "" {
				item.Prices = append(item.Prices, Price{Amounts: []Amount{{Currency: currency, Value: value}}})
			}
		}
	}
	for _, p := range item.Prices {
		free := true
		for _, a := range p.Amounts {
			if a.Value != 0 {
				free = false
			}
		}
		if free {
			item.Free = true
		}
	}
	if b, ok := field(dp, "onSale").(bool); ok {
		item.Sale = b
	}
	if discount, ok := integer(field(dp, "discountPercent")); ok && discount > 0 {
		item.Sale = true
	}
	rating := object(field(m, "Rating"))
	if number, ok := field(rating, "Average").(json.Number); ok {
		if value, e := number.Float64(); e == nil && value >= 0 && value <= 5 {
			item.Rating = &value
		}
	}
	item.Ratings, _ = integer(field(rating, "TotalCount"))
	item.MinVersion = str(dp, "minClientVersion")
	item.MaxVersion = str(dp, "maxClientVersion")
	for _, c := range list(field(m, "Contents")) {
		content := object(c)
		if item.MinVersion == "" {
			item.MinVersion = str(content, "MinClientVersion")
		}
		if item.MaxVersion == "" {
			item.MaxVersion = str(content, "MaxClientVersion")
		}
	}
	item.Start = str(m, "StartDate")
	item.End = str(m, "EndDate")
	item.Modified = str(m, "LastModifiedDate")
	item.Link = "minecraft://showStoreOffer?" + url.Values{"offerId": {id}}.Encode()
	item.Technical, _ = json.MarshalIndent(sanitize(m, "", 0), "", "  ")
	preview := object(field(dp, "preview"))
	geometry := field(preview, "geometry")
	if gm := object(geometry); gm != nil && field(gm, "minecraft:geometry") != nil {
		if texture := SafeImage(str(preview, "texture")); texture != "" {
			raw, _ := json.Marshal(geometry)
			if len(raw) <= 1<<20 {
				item.Preview = &ModelPreview{Geometry: raw, Texture: texture}
			}
		}
	}
	return item, nil
}
func creatorID(m map[string]any) string {
	if s := str(m, "CreatorId"); s != "" {
		return s
	}
	for _, key := range []string{"CreatorEntity", "CreatorEntityKey"} {
		key := object(field(m, key))
		if id := str(key, "Id"); id != "" {
			return first(str(key, "Type"), "title_player_account") + "!" + id
		}
	}
	dp := object(field(m, "DisplayProperties"))
	if id := str(dp, "originalCreatorId"); id != "" {
		if strings.Contains(id, "!") {
			return id
		}
		return "master_player_account!" + id
	}
	return ""
}
func version(s string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return out, false
	}
	for i := 0; i < 3 && i < len(parts); i++ {
		n, e := strconv.Atoi(parts[i])
		if e != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
func compareVersion(a, b [3]int) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
func Compatible(item *Item, current string) {
	v, ok := version(current)
	if !ok {
		return
	}
	known := false
	for _, bound := range []struct {
		s   string
		min bool
	}{{item.MinVersion, true}, {item.MaxVersion, false}} {
		if bound.s == "" {
			continue
		}
		b, ok := version(bound.s)
		if !ok {
			continue
		}
		known = true
		c := compareVersion(v, b)
		if bound.min && c < 0 || !bound.min && c > 0 {
			item.Compatibility = "incompatible"
			return
		}
	}
	if len(item.Platforms) > 0 {
		platform := false
		for _, p := range item.Platforms {
			if strings.Contains(strings.ToLower(p), "windows") || strings.Contains(strings.ToLower(p), "uwp") || strings.EqualFold(p, "Win10") {
				platform = true
			}
		}
		if !platform {
			item.Compatibility = "incompatible"
			return
		}
		known = true
	}
	if known {
		item.Compatibility = "compatible"
	}
}

func rawTags(v any) []string {
	out := []string{}
	for _, entry := range list(v) {
		if s, ok := entry.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
