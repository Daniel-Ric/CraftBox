//go:build gui && !android

package desktop

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	mp "github.com/Daniel-Ric/CraftBox/internal/marketplace"
	"github.com/google/uuid"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) OpenMarketplaceItem(source, id, environment string) error {
	item, e := a.GetMarketplaceItem(source, id, environment)
	if e != nil {
		return e
	}
	if !mp.IsServerOffer(item) {
		return errors.New("Only 3PServerContent offers can be opened")
	}
	return a.OpenURL(item.Link)
}
func (a *App) JoinMarketplaceSource(id, environment string) error {
	session, e := a.marketplace(environment)
	if e != nil {
		return e
	}
	if _, e = a.ListMarketplaceSources(environment); e != nil {
		return e
	}
	source, e := session.service.Source(id)
	if e != nil {
		return e
	}
	if source.Kind == "experience" {
		id, e := uuid.Parse(source.ExperienceID)
		if e != nil || id == uuid.Nil {
			return errors.New("This discovery experience has no valid join ID")
		}
		return a.OpenURL("minecraft://joinExperience?" + url.Values{"experienceId": {id.String()}}.Encode())
	}
	host, port, e := net.SplitHostPort(source.Address)
	if e != nil {
		return e
	}
	return a.OpenURL("minecraft://connect?" + url.Values{"serverUrl": {host}, "serverPort": {port}}.Encode())
}
func (a *App) ExportMarketplaceItems(selection []mp.Favorite, format string) (string, error) {
	if len(selection) == 0 || len(selection) > 100 {
		return "", errors.New("Select between 1 and 100 offers")
	}
	if format != "json" && format != "csv" {
		return "", errors.New("Select JSON or CSV")
	}
	environment := firstMarketEnvironment(selection[0].Environment)
	for _, ref := range selection {
		if firstMarketEnvironment(ref.Environment) != environment {
			return "", errors.New("Cannot mix catalog environments in one export")
		}
	}
	session, e := a.marketplace(environment)
	if e != nil {
		return "", e
	}
	ctx, cancel := context.WithTimeout(session.ctx, 45*time.Second)
	defer cancel()
	items := []mp.Item{}
	for _, ref := range selection {
		item, e := session.service.Detail(ctx, ref.Source, ref.Item)
		if e != nil {
			return "", e
		}
		if !mp.IsServerOffer(item) {
			return "", errors.New("Only 3PServerContent offers can be exported")
		}
		items = append(items, item)
	}
	if e = session.check(); e != nil {
		return "", e
	}
	path, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export Marketplace metadata", DefaultFilename: marketExportName(format), Filters: []wruntime.FileFilter{{DisplayName: strings.ToUpper(format) + " metadata", Pattern: "*." + format}}})
	if e != nil || path == "" {
		return "", e
	}
	return path, writeMarketplaceExport(path, format, items)
}
func writeMarketplaceExport(path, format string, items []mp.Item) error {
	f, e := os.CreateTemp(filepath.Dir(path), "marketplace-export-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e != nil {
		f.Close()
		return e
	}
	if format == "json" {
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		e = enc.Encode(struct {
			Language string    `json:"language"`
			Items    []mp.Item `json:"items"`
		}{"en-US", items})
	} else {
		csvWriter := csv.NewWriter(f)
		e = csvWriter.Write([]string{"ID", "Title ID", "Title", "Creator", "Content type", "Prices", "Rating", "Rating count", "Start", "End", "Compatibility", "Tags", "Association"})
		for _, item := range items {
			if e != nil {
				break
			}
			prices, _ := json.Marshal(item.Prices)
			provenance, _ := json.Marshal(item.Provenance)
			rating, _ := json.Marshal(item.Rating)
			count, _ := json.Marshal(item.Ratings)
			cells := []string{item.ID, item.TitleID, item.Name, item.Creator, item.Type, string(prices), string(rating), string(count), item.Start, item.End, item.Compatibility, strings.Join(item.Tags, "; "), string(provenance)}
			for i, v := range cells {
				if strings.ContainsAny(strings.TrimLeft(v, " \t\r\n")[:min(1, len(strings.TrimLeft(v, " \t\r\n")))], "=+-@") {
					cells[i] = "'" + v
				}
			}
			e = csvWriter.Write(cells)
		}
		csvWriter.Flush()
		if e == nil {
			e = csvWriter.Error()
		}
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
