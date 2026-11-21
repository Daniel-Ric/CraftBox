package desktop

import (
	"context"

	"github.com/Daniel-Ric/CraftBox/locale"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func dialogFilters(filters []wruntime.FileFilter) []wruntime.FileFilter {
	if len(filters) == 0 {
		return filters
	}
	localized := make([]wruntime.FileFilter, len(filters))
	for i, filter := range filters {
		localized[i] = filter
		localized[i].DisplayName = locale.Text(filter.DisplayName)
	}
	return localized
}

func openFileDialog(ctx context.Context, options wruntime.OpenDialogOptions) (string, error) {
	options.Title = locale.Text(options.Title)
	options.Filters = dialogFilters(options.Filters)
	return wruntime.OpenFileDialog(ctx, options)
}

func openDirectoryDialog(ctx context.Context, options wruntime.OpenDialogOptions) (string, error) {
	options.Title = locale.Text(options.Title)
	return wruntime.OpenDirectoryDialog(ctx, options)
}

func openFilesDialog(ctx context.Context, options wruntime.OpenDialogOptions) ([]string, error) {
	options.Title = locale.Text(options.Title)
	options.Filters = dialogFilters(options.Filters)
	return wruntime.OpenMultipleFilesDialog(ctx, options)
}

func saveFileDialog(ctx context.Context, options wruntime.SaveDialogOptions) (string, error) {
	options.Title = locale.Text(options.Title)
	options.Filters = dialogFilters(options.Filters)
	return wruntime.SaveFileDialog(ctx, options)
}
