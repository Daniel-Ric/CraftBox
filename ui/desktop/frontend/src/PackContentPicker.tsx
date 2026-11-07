import {
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import FeedbackFlyout from "./FeedbackFlyout";
import {
  Box,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Image,
  LoaderCircle,
  Search,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { api, type PackObject, type PackObjectPreview } from "./bridge";
import {
  isPackModel,
  packCategoryMatches,
  packObjectKind,
  packObjectLabel,
  packObjectMatches,
  packObjectTypes,
  readablePackName,
  sortPackObjects,
} from "./pack-content-model";
import "./pack-content-picker.css";

const rowHeight = 64;
export default function PackContentPicker({
  assetID,
  revision,
  encrypted,
  onBinding,
  onError,
  onLoading,
  manual,
  onChoose,
  initialSelection,
  onSelection,
}: {
  assetID: string;
  revision?: string;
  encrypted: boolean;
  onBinding: (value: PackObjectPreview | undefined) => void;
  onError: (message: string) => void;
  onLoading: (value: boolean) => void;
  manual: boolean;
  onChoose: () => void;
  initialSelection?: { object: string; variant: string };
  onSelection?: (value: { object: string; variant: string }) => void;
}) {
  const [catalog, setCatalog] = useState<{
    assetID: string;
    objects: PackObject[];
    loaded: boolean;
    error?: string;
  }>({ assetID: "", objects: [], loaded: false });
  const [selected, setSelected] = useState(""),
    [variant, setVariant] = useState(""),
    [category, setCategory] = useState("all"),
    [query, setQuery] = useState("");
  const [open, setOpen] = useState(false),
    [retry, setRetry] = useState(0),
    [focused, setFocused] = useState(0),
    [scrollTop, setScrollTop] = useState(0),
    [viewportHeight, setViewportHeight] = useState(320);
  const [listNode, setListNode] = useState<HTMLDivElement | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listID = useId();
  const callbacks = useRef({ onBinding, onError, onLoading });
  callbacks.current = { onBinding, onError, onLoading };
  const sequence = useRef(0),
    scope = useRef("");
  const objects = useMemo(
    () => (catalog.assetID === assetID ? catalog.objects : []),
    [catalog, assetID],
  );
  const sorted = useMemo(() => [...objects].sort(sortPackObjects), [objects]);
  const visible = useMemo(
    () =>
      sorted.filter(
        (v) => packCategoryMatches(v, category) && packObjectMatches(v, query),
      ),
    [sorted, category, query],
  );
  useEffect(() => {
    onSelection?.({ object: selected, variant });
  }, [selected, variant]);
  const current = objects.find((v) => v.id === selected);
  const currentIndex = visible.findIndex((v) => v.id === selected);
  const label = current ? packObjectLabel(current) : undefined;
  const activeIndex = Math.max(0, Math.min(focused, visible.length - 1));
  const firstRow = Math.max(0, Math.floor(scrollTop / rowHeight) - 4);
  const lastRow = Math.min(
    visible.length,
    firstRow + Math.ceil(viewportHeight / rowHeight) + 9,
  );
  const typeCounts = useMemo(
    () =>
      new Map(
        packObjectTypes.map(([id]) => [
          id,
          objects.filter((v) => packCategoryMatches(v, id)).length,
        ]),
      ),
    [objects],
  );
  const categories = packObjectTypes.filter(
    ([id]) => id === "all" || (typeCounts.get(id) ?? 0) > 0,
  );

  useEffect(() => {
    let cancelled = false;
    sequence.current++;
    const sameAsset = scope.current === assetID;
    scope.current = assetID;
    const previous = objects.find((v) => v.id === selected);
    setCatalog({ assetID, objects: [], loaded: false });
    if (!sameAsset) {
      setSelected(initialSelection?.object || "");
      setQuery("");
      setCategory("all");
      setOpen(false);
    }
    setVariant(!sameAsset ? initialSelection?.variant || "" : "");
    if (!manual) callbacks.current.onBinding(undefined);
    callbacks.current.onError("");
    if (encrypted) {
      callbacks.current.onLoading(false);
      return;
    }
    callbacks.current.onLoading(true);
    void api
      .GetAssetObjects(assetID)
      .then((items) => {
        if (cancelled) return;
        const defaultCategory = items.some((v) => isPackModel(v.kind))
          ? "models"
          : "all";
        const nextCategory =
          sameAsset && items.some((v) => packCategoryMatches(v, category))
            ? category
            : defaultCategory;
        setCategory(nextCategory);
        setCatalog({ assetID, objects: items, loaded: true });
        const matches = items
          .filter((v) => packCategoryMatches(v, nextCategory))
          .sort(sortPackObjects);
        const next =
          (!sameAsset
            ? items.find((v) => v.id === initialSelection?.object)
            : undefined) ??
          (sameAsset
            ? (items.find((v) => v.id === selected) ??
              items.find(
                (v) => v.name === previous?.name && v.kind === previous?.kind,
              ) ??
              items.find(
                (v) =>
                  !!previous?.geometryId &&
                  v.geometryId === previous.geometryId,
              ))
            : undefined) ??
          matches.find((v) => v.geometryFile || v.kind === "block") ??
          matches[0];
        setSelected(next?.id ?? "");
        if (!next) callbacks.current.onLoading(false);
      })
      .catch((error) => {
        if (!cancelled) {
          setCatalog({
            assetID,
            objects: [],
            loaded: true,
            error: String(error),
          });
          callbacks.current.onError(String(error));
          callbacks.current.onLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [assetID, encrypted, revision, retry]);

  useEffect(() => {
    const run = ++sequence.current;
    if (manual || encrypted) {
      callbacks.current.onLoading(false);
      return;
    }
    if (catalog.assetID !== assetID || !catalog.loaded) {
      callbacks.current.onLoading(true);
      return;
    }
    if (!selected || !objects.some((v) => v.id === selected)) {
      callbacks.current.onBinding(undefined);
      callbacks.current.onLoading(false);
      return;
    }
    callbacks.current.onLoading(true);
    callbacks.current.onBinding(undefined);
    void api
      .PreviewAssetObject(assetID, selected, variant)
      .then((value) => {
        if (run === sequence.current) {
          // Also canonicalize projects saved before variant/model linking.
          if (
            value.object.id !== selected &&
            objects.some((o) => o.id === value.object.id)
          )
            setSelected(value.object.id);
          callbacks.current.onBinding(value);
          callbacks.current.onError("");
          callbacks.current.onLoading(false);
        }
      })
      .catch((error) => {
        if (run === sequence.current) {
          callbacks.current.onError(String(error));
          callbacks.current.onLoading(false);
        }
      });
    return () => {
      sequence.current++;
    };
  }, [assetID, selected, variant, manual, catalog, encrypted]);

  useLayoutEffect(() => {
    if (!listNode) return;
    const resize = () => setViewportHeight(listNode.clientHeight);
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(listNode);
    return () => observer.disconnect();
  }, [listNode]);
  useLayoutEffect(() => {
    if (!listNode || !open) return;
    const top = activeIndex * rowHeight;
    if (top < listNode.scrollTop) listNode.scrollTop = top;
    else if (top + rowHeight > listNode.scrollTop + listNode.clientHeight)
      listNode.scrollTop = top + rowHeight - listNode.clientHeight;
    setScrollTop(listNode.scrollTop);
  }, [listNode, open, activeIndex, visible.length]);

  function changeOpen(next: boolean) {
    setOpen(next);
    if (next) setFocused(Math.max(0, currentIndex));
    else setQuery("");
  }
  function choose(object: PackObject) {
    onChoose();
    setSelected(object.id);
    setVariant("");
    changeOpen(false);
  }
  function step(direction: number) {
    const index =
      currentIndex < 0
        ? direction > 0
          ? 0
          : visible.length - 1
        : currentIndex + direction;
    if (visible[index]) choose(visible[index]);
  }
  function navigate(event: KeyboardEvent) {
    const keys: Record<string, number> = {
      ArrowDown: activeIndex + 1,
      ArrowUp: activeIndex - 1,
      Home: 0,
      End: visible.length - 1,
      PageDown:
        activeIndex + Math.max(1, Math.floor(viewportHeight / rowHeight)),
      PageUp: activeIndex - Math.max(1, Math.floor(viewportHeight / rowHeight)),
    };
    if (event.key in keys) {
      event.preventDefault();
      setFocused(Math.max(0, Math.min(visible.length - 1, keys[event.key])));
    }
    if (event.key === "Enter" && visible[activeIndex]) {
      event.preventDefault();
      choose(visible[activeIndex]);
    }
  }
  if (
    encrypted ||
    (catalog.assetID === assetID &&
      catalog.loaded &&
      !objects.length &&
      !catalog.error)
  )
    return null;
  return (
    <Dialog open={open} onOpenChange={changeOpen}>
      <div
        className="pack-content-picker"
        onKeyDown={(event) => {
          if (
            open ||
            event.altKey ||
            event.ctrlKey ||
            event.metaKey ||
            event.target instanceof HTMLSelectElement
          )
            return;
          if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
            event.preventDefault();
            step(event.key === "ArrowLeft" ? -1 : 1);
          }
        }}
      >
        <DialogTrigger
          render={
            <Button
              variant="outline"
              className="pack-current-object"
              disabled={!objects.length}
              aria-label={`Browse pack content${label ? `: ${label.name}` : ""}`}
              title={current?.name}
            >
              {catalog.loaded ? (
                <Box size={17} />
              ) : (
                <LoaderCircle className="animate-spin" size={16} />
              )}
              <span>
                <strong>
                  {label?.name ||
                    (catalog.error
                      ? "Unable to load objects"
                      : "Loading objects...")}
                </strong>
                <small>
                  {current
                    ? [packObjectKind(current.kind), label?.context]
                        .filter(Boolean)
                        .join(" - ")
                    : "Models, textures and item icons"}
                </small>
              </span>
              <ChevronDown size={15} />
            </Button>
          }
        />
        <div
          className="pack-object-navigation"
          role="group"
          aria-label="Object navigation"
        >
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Previous object"
            title="Previous object (Left arrow)"
            disabled={!visible.length || currentIndex === 0}
            onClick={() => step(-1)}
          >
            <ChevronLeft size={16} />
          </Button>
          <output aria-label="Object position">
            {currentIndex < 0
              ? `${visible.length} objects`
              : `${currentIndex + 1} / ${visible.length}`}
          </output>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Next object"
            title="Next object (Right arrow)"
            disabled={!visible.length || currentIndex === visible.length - 1}
            onClick={() => step(1)}
          >
            <ChevronRight size={16} />
          </Button>
        </div>
        {current && Object.keys(current.textureOptions).length > 1 && (
          <label className="pack-texture-variant">
            <span>Variant</span>
            <select
              aria-label="Object texture variant"
              value={variant}
              onChange={(e) => {
                onChoose();
                const target = current.textureVariantObjects?.[e.target.value];
                if (target && objects.some((o) => o.id === target))
                  setSelected(target);
                setVariant(e.target.value);
              }}
            >
              <option value="">Render controller</option>
              {Object.keys(current.textureOptions).map((key) => (
                <option key={key} value={key}>
                  {readablePackName(key)}
                </option>
              ))}
            </select>
          </label>
        )}
        {manual && (
          <Button variant="secondary" size="sm" onClick={onChoose}>
            Return to object
          </Button>
        )}
        {catalog.error && (
          <FeedbackFlyout title="Pack content">
          <p>{String(catalog.error)}</p>
          <Button variant="secondary" onClick={() => setRetry((v) => v + 1)}>
            Retry loading objects
          </Button>
          </FeedbackFlyout>
        )}
      </div>
      <DialogContent className="pack-object-dialog" initialFocus={searchRef}>
        <DialogHeader>
          <DialogTitle>Browse pack content</DialogTitle>
          <DialogDescription>
            Find a model or image, then select it to preview.
          </DialogDescription>
        </DialogHeader>
        <div className="studio-dialog-body pack-browser-body">
          <div className="pack-browser-filters">
            <label className="pack-browser-search">
              <Search size={15} />
              <input
                ref={searchRef}
                aria-label="Search pack objects"
                placeholder="Search names, groups or file paths"
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setFocused(0);
                }}
                onKeyDown={(event) => {
                  if (event.key === "ArrowDown") {
                    event.preventDefault();
                    listNode?.focus();
                  } else if (event.key === "Enter" && visible[activeIndex]) {
                    event.preventDefault();
                    choose(visible[activeIndex]);
                  }
                }}
              />
              {query && (
                <Button
                  size="icon-xs"
                  variant="ghost"
                  aria-label="Clear object search"
                  onClick={() => {
                    setQuery("");
                    setFocused(0);
                    searchRef.current?.focus();
                  }}
                >
                  <X size={13} />
                </Button>
              )}
            </label>
            <select
              aria-label="Object type"
              value={category}
              onChange={(e) => {
                setCategory(e.target.value);
                setFocused(0);
              }}
            >
              {categories.map(([id, name]) => (
                <option key={id} value={id}>
                  {name} ({typeCounts.get(id) ?? 0})
                </option>
              ))}
            </select>
          </div>
          <div className="pack-browser-count" role="status">
            {visible.length.toLocaleString("en-US")}{" "}
            {visible.length === 1 ? "object" : "objects"}
            {query ? ` matching "${query}"` : ""}
          </div>
          <div
            ref={setListNode}
            className="pack-browser-list"
            role="listbox"
            aria-label="Pack objects"
            tabIndex={0}
            aria-activedescendant={
              visible.length && activeIndex >= firstRow && activeIndex < lastRow
                ? `${listID}-${activeIndex}`
                : undefined
            }
            onFocus={() => {
              if (activeIndex < firstRow || activeIndex >= lastRow)
                setFocused(
                  Math.min(
                    visible.length - 1,
                    Math.ceil(scrollTop / rowHeight),
                  ),
                );
            }}
            onKeyDown={navigate}
            onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
          >
            {visible.length ? (
              <div
                style={{
                  paddingTop: firstRow * rowHeight,
                  paddingBottom: (visible.length - lastRow) * rowHeight,
                }}
              >
                {visible.slice(firstRow, lastRow).map((object, index) => {
                  const name = packObjectLabel(object),
                    at = firstRow + index,
                    Icon = isPackModel(object.kind) ? Box : Image;
                  return (
                    <button
                      type="button"
                      role="option"
                      aria-selected={object.id === selected}
                      aria-posinset={at + 1}
                      aria-setsize={visible.length}
                      tabIndex={-1}
                      id={`${listID}-${at}`}
                      key={object.id}
                      className="pack-browser-object"
                      data-focused={at === activeIndex || undefined}
                      onClick={() => choose(object)}
                      title={[object.name, object.geometryId, object.definition]
                        .filter(Boolean)
                        .join("\n")}
                    >
                      <Icon size={19} />
                      <span className="pack-browser-object-copy">
                        <strong>{name.name}</strong>
                        <small>
                          {name.context ||
                            object.definition.split("/").pop() ||
                            "Pack content"}
                        </small>
                      </span>
                      <span className="pack-browser-kind">
                        {packObjectKind(object.kind)}
                      </span>
                      {object.id === selected && <Check size={15} />}
                    </button>
                  );
                })}
              </div>
            ) : (
              <div className="pack-browser-empty">
                <Search size={28} />
                <strong>No matching objects</strong>
                <p>Try a shorter name or another content type.</p>
                <Button
                  variant="secondary"
                  onClick={() => {
                    setQuery("");
                    setCategory("all");
                    setFocused(0);
                  }}
                >
                  Clear filters
                </Button>
              </div>
            )}
          </div>
        </div>
        <DialogFooter>
          <small>Arrow keys to navigate. Enter to preview.</small>
          <Button variant="secondary" onClick={() => changeOpen(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
