import {
  FolderOpen,
  Save,
  Radio,
  PanelRight,
  SlidersHorizontal,
  Sun,
  Palette,
  Paintbrush,
  View,
  Plus,
  ScanSearch,
  RefreshCw,
  MoreHorizontal,
  Crosshair,
} from "lucide-react";
export default function HeaderActionIcon({ label }: { label: string }) {
  const Icon = /pick model/i.test(label)
    ? Crosshair
    : /capture|record/i.test(label)
      ? Radio
      : /properties/i.test(label)
        ? PanelRight
        : /open|folder/i.test(label)
          ? FolderOpen
          : /save/i.test(label)
            ? Save
            : /advanced|setting/i.test(label)
              ? SlidersHorizontal
              : /lighting/i.test(label)
                ? Sun
                : /color/i.test(label)
                  ? Palette
                  : /looks/i.test(label)
                    ? Paintbrush
                    : /panorama/i.test(label)
                      ? View
                      : /new/i.test(label)
                        ? Plus
                        : /inspect/i.test(label)
                          ? ScanSearch
                          : /refresh/i.test(label)
                            ? RefreshCw
                            : MoreHorizontal;
  return <Icon size={15} aria-hidden="true" />;
}
