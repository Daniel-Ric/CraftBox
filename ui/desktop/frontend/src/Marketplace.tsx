import FeedbackFlyout from "./FeedbackFlyout";
import {useWorkspaceActions} from "./workspace-actions";
import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Copy,
  Download,
  ExternalLink,
  Gamepad2,
  Globe,
  Image,
  LayoutGrid,
  List,
  LoaderCircle,
  RefreshCw,
  Search,
  SlidersHorizontal,
  Star,
  Store,
  Tags,
  TriangleAlert,
  X,
} from "lucide-react";
import { api } from "./bridge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  marketDate,
  discoveryBanner,
  marketKey,
  marketPrice,
  marketCardPrice,
  marketOfferSummary,
  hasMarketPreview,
  currencyLabel,
  marketRef,
  takeMarketplaceNavigation,
  type MarketDiscovery,
  type MarketFavorite,
  type MarketItem,
  type MarketPage,
  type MarketQuery,
} from "./marketplace-model";
import "./marketplace.css";
const MarketplaceModel = lazy(() => import("./MarketplaceModel"));
type View =
  "offers" | "collections" | "creator" | "favorites";
const emptyPage: MarketPage = {
  items: [],
  cursor: "",
  warnings: [],
  checkedAt: "",
  relation: "offers",
};
const message = (e: unknown) => (e instanceof Error ? e.message : String(e));
function Thumbnail({
  item,
  preferredImage,
}: {
  item: MarketItem;
  preferredImage?: string;
}) {
  const [failedURLs, setFailedURLs] = useState<string[]>([]);
  const fallback = item.images[0]?.url;
  const url = [preferredImage, fallback].find(
    (candidate) => candidate && !failedURLs.includes(candidate),
  );
  useEffect(() => setFailedURLs([]), [preferredImage, fallback]);
  return url ? (
    <img
      src={url}
      loading="lazy"
      decoding="async"
      referrerPolicy="no-referrer"
      alt={item.name}
      style={url === preferredImage ? { objectFit: "contain" } : undefined}
      onError={() => setFailedURLs((old) => [...old, url])}
    />
  ) : (
    <div className="market-image-placeholder">
      <Image size={24} />
      <span>No preview image</span>


    </div>
  );
}
function isOffer(item: MarketItem) {
  return item.tags.includes("3PServerContent");
}
function favoriteKey(ref: MarketFavorite) {
  return `${ref.source}/${ref.item}`;
}
function sortLoadedOffers(
  a: MarketItem,
  b: MarketItem,
  sort: string,
  currency: string,
) {
  if (sort === "name") return a.name.localeCompare(b.name);
  if (sort === "recent") return b.start.localeCompare(a.start);
  if (sort === "rating") return (b.rating ?? -1) - (a.rating ?? -1);
  if (sort === "popular") return b.ratings - a.ratings;
  if (sort === "price-asc" || sort === "price-desc") {
    const value = (item: MarketItem) =>
      Math.min(
        ...item.prices
          .filter(
            (p) => p.amounts.length === 1 && p.amounts[0].currency === currency,
          )
          .map((p) => p.amounts[0].value),
      );
    const av = value(a),
      bv = value(b);
    if (!Number.isFinite(av) || !Number.isFinite(bv))
      return Number.isFinite(av) ? -1 : Number.isFinite(bv) ? 1 : 0;
    return sort === "price-desc" ? bv - av : av - bv;
  }
  return 0;
}
export default function Marketplace({
  loggedIn,
  account,
  onLogin,
}: {
  loggedIn: boolean;
  account: string;
  onLogin: () => void;
}) {
  const [creator, setCreator] = useState("");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [playerCounts, setPlayerCounts] = useState<Record<string, number>>({});
  const [creatorImages, setCreatorImages] = useState<
    Record<string, { url: string; collectionId: string }>
  >({});
  const [navigationRevision, setNavigationRevision] = useState(0);
  const [environment, setEnvironment] = useState<"production" | "staging">(
    "production",
  );
  const reference = (item: MarketItem): MarketFavorite => ({
    ...marketRef(item),
    environment,
  });
  const [online, setOnline] = useState(navigator.onLine),
    [discovery, setDiscovery] = useState<MarketDiscovery>(),
    [sourceID, setSourceID] = useState("all"),
    [sourceSearch, setSourceSearch] = useState(""),
    [sourceKind, setSourceKind] = useState("all"),
    [view, setView] = useState<View>("offers"),
    [layout, setLayout] = useState<"grid" | "list">("grid"),
    [search, setSearch] = useState(""),
    [submitted, setSubmitted] = useState(""),
    [sort, setSort] = useState("recent"),
    [type, setType] = useState(""),
    [tag, setTag] = useState(""),
    [price, setPrice] = useState(""),
    [currency, setCurrency] = useState(""),
    [maxPrice, setMaxPrice] = useState(""),
    [compatible, setCompatible] = useState(false);
  const [page, setPage] = useState<MarketPage>(emptyPage),
    [favorites, setFavorites] = useState<MarketFavorite[]>([]),
    [active, setActive] = useState<MarketItem>(),
    [detail, setDetail] = useState<MarketItem>(),
    [detailTab, setDetailTab] = useState("overview"),
    [imageIndex, setImageIndex] = useState(0),
    [busy, setBusy] = useState(""),
    [loading, setLoading] = useState(false),
    [loadingSources, setLoadingSources] = useState(false),
    [loadingDetail, setLoadingDetail] = useState(false),
    [error, setError] = useState(""),
    [detailError, setDetailError] = useState(""),
    [notice, setNotice] = useState(""),
    [facets, setFacets] = useState<{ types: string[]; currencies: string[] }>({
      types: [],
      currencies: [],
    });
  const sourceSequence = useRef(0),
    querySequence = useRef(0),
    detailSequence = useRef(0),
    mounted = useRef(true),
    pendingNavigation = useRef<MarketFavorite | undefined>(undefined);
  useEffect(() => {
    mounted.current = true;
    const changed = () => setOnline(navigator.onLine);
    window.addEventListener("online", changed);
    window.addEventListener("offline", changed);
    return () => {
      mounted.current = false;
      sourceSequence.current++;
      querySequence.current++;
      detailSequence.current++;

      window.removeEventListener("online", changed);
      window.removeEventListener("offline", changed);
    };
  }, []);
  const available = online && loggedIn;
  useEffect(() => {
    let current = true;
    setPlayerCounts({});
    if (available && discovery) void api.GetMarketplacePlayerCounts(environment)
      .then(counts => { if (current) setPlayerCounts(counts); })
      .catch(() => { /* Unavailable counts remain unknown, never zero. */ });
    return () => { current = false; };
  }, [available, account, environment, discovery]);
  useEffect(() => {
    let current = true;
    setCreatorImages({});
    if (available && discovery) {
      void api
        .GetMarketplaceCreatorImages(environment)
        .then((images) => {
          if (current) setCreatorImages(images);
        })
        .catch(() => {
          /* Discovery previews remain available if optional creator metadata fails. */
        });
    }
    return () => {
      current = false;
    };
  }, [available, account, environment, discovery]);
  function clearContent() {
    querySequence.current++;
    detailSequence.current++;

    setPage(emptyPage);
    setActive(undefined);
    setDetail(undefined);


    setLoading(false);
    setLoadingDetail(false);
    setDetailError("");
    setBusy("");
  }
  useWorkspaceActions("marketplace", (action) => {
    if (action === "refresh") void refresh();
    else if (action === "search" || action === "settings") setFiltersOpen(true);
    else if (action === "collections") setView("collections");
    else if (action === "favorites") setView("favorites");
    else if (action === "list") setLayout("list");
    else if (action === "grid") setLayout("grid");
  });
  async function refresh() {
    if (!available) return;
    const sequence = ++sourceSequence.current;

    clearContent();
    setDiscovery(undefined);
    setLoadingSources(true);
    setError("");
    try {
      const [d, f] = await Promise.all([
        api.ListMarketplaceSources(environment),
        api.GetMarketplaceFavorites(environment),
      ]);
      if (!mounted.current || sequence !== sourceSequence.current) return;
      setDiscovery(d);
      setFavorites(f);
      setSourceID((old) =>
        old === "all" || d.sources.some((s) => s.id === old) ? old : "all",
      );
      const pending = takeMarketplaceNavigation();
      if (pending) {
        pendingNavigation.current = pending;
        setSourceID(pending.source);
        if (pending.environment && pending.environment !== environment)
          setEnvironment(pending.environment);
        setView("offers");
      }
    } catch (e) {
      if (sequence === sourceSequence.current) {
        setDiscovery(undefined);
        setError(message(e));
      }
    } finally {
      if (sequence === sourceSequence.current) setLoadingSources(false);
    }
  }
  useEffect(() => {
    clearContent();

    setFacets({ types: [], currencies: [] });
    sourceSequence.current++;
    if (available) void refresh();
    else {
      setDiscovery(undefined);
      setLoadingSources(false);
    }
  }, [online, loggedIn, account, environment]);
  useEffect(() => {
    const navigate = () => {
      const ref = takeMarketplaceNavigation();
      if (!ref) return;
      pendingNavigation.current = ref;
      setSourceID(ref.source);
      setView("offers");
      if (ref.environment && ref.environment !== environment)
        setEnvironment(ref.environment);
      else setNavigationRevision((v) => v + 1);
    };
    window.addEventListener("craftbox:open-marketplace", navigate);
    return () =>
      window.removeEventListener("craftbox:open-marketplace", navigate);
  }, [environment]);
  useEffect(() => {
    const timer = setTimeout(() => setSubmitted(search.trim()), 300);
    return () => clearTimeout(timer);
  }, [search]);
  const source = discovery?.sources.find((s) => s.id === sourceID);
  const relation = view;
  const query = useMemo<MarketQuery>(
    () => ({
      environment,
      creator,
      source: sourceID,
      relation,
      search: submitted,
      cursor: "",
      count: 24,
      sort,
      type,
      tag,
      price,
      currency,
      maxPrice:
        currency &&
        maxPrice.trim() &&
        Number.isFinite(Number(maxPrice)) &&
        Number(maxPrice) >= 0
          ? Number(maxPrice)
          : undefined,
      compatible,
    }),
    [
      environment,
      creator,
      sourceID,
      relation,
      submitted,
      sort,
      type,
      tag,
      price,
      currency,
      maxPrice,
      compatible,
    ],
  );
  const queryKey = JSON.stringify(query);
  useEffect(() => {
    setFacets({ types: [], currencies: [] });
    if (sourceID !== "all") setCreator("");
  }, [sourceID]);
  async function loadOffers(append = false) {
    if (!available || !discovery) return;
    const sequence = ++querySequence.current;
    setLoading(true);
    setError("");
    if (!append) {
      setPage(emptyPage);
      setActive(undefined);
      setDetail(undefined);
      setDetailError("");
      detailSequence.current++;



    }
    try {
      let result =
        view === "favorites"
          ? await api.GetMarketplaceFavoriteItems(environment)
          : await api.SearchMarketplace({
              ...query,
              cursor: append ? page.cursor : "",
            });
      if (!mounted.current || sequence !== querySequence.current) return;
      if (view === "favorites")
        result = {
          ...result,
          items: result.items.filter(
            (item) =>
              (sourceID === "all" ||
                item.provenance.some((p) => p.source === sourceID)) &&
              (!submitted ||
                `${item.name} ${item.description} ${item.id}`
                  .toLowerCase()
                  .includes(submitted.toLowerCase())) &&
              (!creator ||
                item.provenance.some((p) =>
                  discovery?.sources.some(
                    (s) => s.id === p.source && s.creatorId === creator,
                  ),
                )) &&
              (!type || item.type === type) &&
              (!compatible || item.compatibility === "compatible") &&
              (!price ||
                (price === "free" && item.free) ||
                (price === "paid" && item.prices.length > 0 && !item.free) ||
                (price === "sale" && item.sale)) &&
              (!tag || item.tags.includes(tag)) &&
              (!currency ||
                item.prices.some(
                  (p) =>
                    p.amounts.length === 1 &&
                    p.amounts[0].currency === currency &&
                    (query.maxPrice === undefined ||
                      p.amounts[0].value <= query.maxPrice),
                )),
          ),
        };
      result = {
        ...result,
        items: result.items.filter(
          (item) => view === "collections" || isOffer(item),
        ),
      };
      setPage((old) => ({
        ...result,
        items: append
          ? Array.from(
              new Map(
                [...old.items, ...result.items].map((item) => [
                  marketKey(item),
                  item,
                ]),
              ).values(),
            )
          : result.items,
      }));
      setPage((old) => ({
        ...old,
        items: [...old.items].sort((a, b) =>
          sortLoadedOffers(a, b, sort, currency),
        ),
      }));
      setFacets((old) => ({
        types: Array.from(
          new Set([
            ...old.types,
            ...result.items.map((i) => i.type).filter(Boolean),
          ]),
        ).sort(),
        currencies: Array.from(
          new Set([
            ...old.currencies,
            ...result.items.flatMap((i) =>
              i.prices.flatMap((p) => p.amounts.map((a) => a.currency)),
            ),
          ]),
        ).sort(),
      }));
      const pending = pendingNavigation.current;
      if (pending) {
        pendingNavigation.current = undefined;
        const hit = result.items.find((i) => i.id === pending.item);
        if (hit) void selectItem(hit);
        else {
          try {
            const hit = await api.ResolveMarketplaceReference(pending);
            if (sequence === querySequence.current) void selectItem(hit);
          } catch {
            if (sequence === querySequence.current)
              setNotice(
                "That offer is not in the current online results. Browse its discovery source to resolve it again.",
              );
          }
        }
      }
    } catch (e) {
      if (sequence === querySequence.current) {
        setError(message(e));
        if (!append) setPage(emptyPage);
      }
    } finally {
      if (sequence === querySequence.current) setLoading(false);
    }
  }
  useEffect(() => {
    if (discovery && available) void loadOffers();
    else {
      querySequence.current++;
      setPage(emptyPage);
    }
  }, [queryKey, discovery?.revision, available, navigationRevision]);
  async function selectItem(item: MarketItem) {
    const sequence = ++detailSequence.current;

    setActive(item);
    setDetail(undefined);
    setLoadingDetail(true);
    setDetailError("");
    setDetailTab("overview");
    setImageIndex(0);


    try {
      const ref = reference(item);
      const fresh =
        isOffer(item) || item.type === "3PP_V2.0"
          ? await api.GetMarketplaceItem(ref.source, ref.item, environment)
          : item;
      if (!mounted.current || sequence !== detailSequence.current) return;
      setDetail(fresh);
      if (fresh.type === "3PP_V2.0") {
        const banner = discoveryBanner(fresh);
        setImageIndex(
          Math.max(
            0,
            fresh.images.findIndex((image) => image.url === banner),
          ),
        );
      }
    } catch (e) {
      if (sequence === detailSequence.current) setDetailError(message(e));
    } finally {
      if (sequence === detailSequence.current) setLoadingDetail(false);
    }
  }
  async function action(label: string, run: () => Promise<unknown>) {
    const epoch = sourceSequence.current;
    setBusy(label);
    setError("");
    try {
      await run();
    } catch (e) {
      if (mounted.current && epoch === sourceSequence.current)
        setError(message(e));
    } finally {
      if (mounted.current && epoch === sourceSequence.current) setBusy("");
    }
  }
  async function copy(value: string) {
    await action("Copying", async () => {
      await navigator.clipboard.writeText(value);
      setNotice("Copied to clipboard");
    });
  }
  function hasFavorite(item: MarketItem) {
    const ref = reference(item);
    return favorites.some((f) => favoriteKey(f) === favoriteKey(ref));
  }
  async function toggleFavorite(item: MarketItem) {
    const ref = reference(item),
      enabled = !hasFavorite(item),
      epoch = sourceSequence.current;
    await action("Saving favorite", async () => {
      await api.SetMarketplaceFavorite(
        ref.source,
        ref.item,
        enabled,
        environment,
      );
      if (!mounted.current || epoch !== sourceSequence.current) return;
      setFavorites((old) =>
        enabled
          ? [...old.filter((f) => favoriteKey(f) !== favoriteKey(ref)), ref]
          : old.filter((f) => favoriteKey(f) !== favoriteKey(ref)),
      );
    });
  }
  async function removeUnavailableFavorite(ref: MarketFavorite) {
    const epoch = sourceSequence.current;
    await action("Removing favorite", async () => {
      await api.SetMarketplaceFavorite(
        ref.source,
        ref.item,
        false,
        environment,
      );
      if (epoch === sourceSequence.current)
        setFavorites((old) =>
          old.filter((f) => favoriteKey(f) !== favoriteKey(ref)),
        );
    });
  }
  async function exportItems(format: "json" | "csv") {
    const items =
      detail && isOffer(detail)
          ? [detail]
          : page.items.filter(isOffer).slice(0, 100);
    if (!items.length) return;
    await action("Exporting metadata", async () => {
      const path = await api.ExportMarketplaceItems(
        items.map(reference),
        format,
      );
      if (path) setNotice(`Metadata saved to ${path}`);
    });
  }
  const shownSources = (discovery?.sources || []).filter(
    (s) =>
      (sourceKind === "all" || s.kind === sourceKind) &&
      `${s.name} ${s.creator}`
        .toLowerCase()
        .includes(sourceSearch.toLowerCase()),
  ).sort((a, b) => (playerCounts[b.id] ?? -1) - (playerCounts[a.id] ?? -1) || a.name.localeCompare(b.name));
  const title = source?.name || "All discovered sources";
  const gallery = detail?.images || [];
  const selectedImage = gallery[Math.min(imageIndex, gallery.length - 1)];
  return (
    <div className={`market-workspace${active ? " has-detail" : ""}`}>
      <header className="market-header studio-view-heading">
        <Store size={19} />
        <strong>Marketplace</strong>
        <span className="market-online">
          <i className={available ? "online" : ""} />
          {!online
            ? "Offline"
            : loggedIn
              ? "Online catalog"
              : "Sign-in required"}
        </span>
        <span className="market-header-spacer" />
        <label className="market-environment">
          <span>Catalog</span>
          <select
            aria-label="Marketplace environment"
            value={environment}
            onChange={(e) => {
              clearContent();

              setDiscovery(undefined);
              sourceSequence.current++;
              setCreator("");
              setType("");
              setCurrency("");
              setMaxPrice("");
              setEnvironment(e.target.value as "production" | "staging");
            }}
          >
            <option value="production">Production</option>
            <option value="staging">Staging</option>
          </select>
          <code className="market-title-id">{environment === "production" ? "20CA2" : "EE7B"}</code>
        </label>

        {discovery && <small>Updated {marketDate(discovery.checkedAt)}</small>}
        <button title="Refresh discovery" aria-label="Refresh discovery" className="toolbar-icon-action"
          disabled={!available || loadingSources}
          onClick={() => void refresh()}
        >
          <RefreshCw size={14} className={loadingSources ? "spin" : ""} />
          <span className="toolbar-action-label">Refresh discovery</span></button>
      </header>
      {!available ? (
        <div className="market-unavailable">
          <Globe size={36} />
          <h2>
            {online
              ? "Sign in to browse server offers"
              : "Marketplace requires an internet connection"}
          </h2>
          <p>
            {online
              ? "Use your CraftBox Microsoft account to load the current Featured Server and Experience catalog."
              : "Reconnect, then refresh discovery. Catalog pages and images are not stored for offline browsing."}
          </p>
          {online && <button onClick={onLogin}>Sign in with Microsoft</button>}
        </div>
      ) : (
        <div className="market-columns">
          <aside className="studio-sidebar market-sources">
            <div className="market-source-controls">
              <label className="market-search">
                <Search size={14} />
                <input
                  aria-label="Find discovery source"
                  value={sourceSearch}
                  onChange={(e) => setSourceSearch(e.target.value)}
                  placeholder="Find server or experience"
                />
              </label>
              <select
                aria-label="Discovery source type"
                value={sourceKind}
                onChange={(e) => setSourceKind(e.target.value)}
              >
                <option value="all">Featured servers & experiences</option>
                <option value="featured">Featured servers</option>
                <option value="experience">Experiences</option>
              </select>
              <small>
                {discovery
                  ? `${shownSources.length} discovery entries`
                  : "Loading discovery"}
              </small>
            </div>
            <div className="market-source-list">
              <button
                className={`market-source-all ${sourceID === "all" ? "selected" : ""}`}
                onClick={() => setSourceID("all")}
              >
                <Globe size={18} />
                <span>All discovered sources</span>
              </button>
              {shownSources.map((s) => (
                <button
                  key={s.id}
                  className={`market-source ${s.id === sourceID ? "selected" : ""}`}
                  aria-pressed={s.id === sourceID}
                  onClick={() => setSourceID(s.id)}
                >
                  <div className="market-source-image">
                    <Thumbnail
                      item={s.overview}
                      preferredImage={discoveryBanner(s.overview)}
                    />
                  </div>
                  <span className="market-source-copy">
                    <strong>{s.name}</strong>
                    <small>
                      {s.creator || "Unknown creator"} - {s.kind === "featured" ? "Featured server" : "Experience"}
                    </small>
                    {playerCounts[s.id] !== undefined && <small className="market-source-players">{playerCounts[s.id].toLocaleString("en-US")} players online</small>}
                  </span>
                </button>
              ))}
              {!loadingSources && discovery && !shownSources.length && (
                <p className="market-notice">No matching discovery entries.</p>
              )}
            </div>
            <footer>Discovery sources only</footer>
          </aside>
          <main className="market-main">
            <nav className="market-tabs" aria-label="Marketplace sections">
              {(
                [
                  ["offers", "Offers"],
                  ["collections", "Collections"],
                  ["creator", "More server offers"],
                  ["favorites", `Favorites (${favorites.length})`],
                ] as [View, string][]
              ).map(([id, label]) => (
                <button
                  key={id}
                  aria-current={view === id ? "page" : undefined}
                  className={view === id ? "active" : ""}
                  onClick={() => {
                    setType("");
                    setView(id);
                    setNotice("");
                  }}
                >
                  {label}
                </button>
              ))}
            <div className="market-source-context" aria-label="Current marketplace source">
              <span title="Only offers tagged 3PServerContent from the selected discovery sources">{title}</span>
              {source && (
                <button
                  disabled={!!busy}
                  onClick={() =>
                    void action("Opening Minecraft", () =>
                      api.JoinMarketplaceSource(source.id, environment),
                    )
                  }
                >
                  <Gamepad2 size={15} />
                  Join in Minecraft
                </button>
              )}
            </div>
            </nav>
            {error && (
              <FeedbackFlyout title="Marketplace">
                <TriangleAlert size={15} />
                <span>{error}</span>
                <button aria-label="Dismiss error" onClick={() => setError("")}>
                  <X size={14} />
                </button>
              </FeedbackFlyout>
            )}
            {notice && (
              <div role="status" className="market-notice market-status">
                {notice}
                <button
                  aria-label="Dismiss notice"
                  onClick={() => setNotice("")}
                >
                  <X size={12} />
                </button>
              </div>
            )}
            <>
                <form
                  className="market-filters"
                  onSubmit={(e) => {
                    e.preventDefault();
                    setSubmitted(search.trim());
                    void loadOffers();
                  }}
                >
                  <label className="market-search">
                    <Search size={15} />
                    <input
                      aria-label="Search server offers"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      placeholder="Search server offers"
                    />
                  </label>
                  <select
                    aria-label="Sort offers"
                    value={sort}
                    onChange={(e) => setSort(e.target.value)}
                  >
                    <option value="recent">Newest first</option>
                    <option value="popular">Most rated</option>
                    <option value="rating">Highest rated</option>
                    <option value="name">Name (loaded results)</option>
                    <option value="price-asc" disabled={!currency}>
                      Price: low to high (loaded results)
                    </option>
                    <option value="price-desc" disabled={!currency}>
                      Price: high to low (loaded results)
                    </option>
                  </select>
                  <button
                    type="button"
                    aria-haspopup="dialog"
                    onClick={() => setFiltersOpen(true)}
                  >
                    <SlidersHorizontal size={14} />
                    Filters
                    {[
                      creator,
                      type,
                      price,
                      currency,
                      maxPrice,
                      tag,
                      compatible,
                    ].filter(Boolean).length > 0
                      ? ` (${[creator, type, price, currency, maxPrice, tag, compatible].filter(Boolean).length})`
                      : ""}
                  </button>
                  {!![
                    creator,
                    type,
                    price,
                    currency,
                    maxPrice,
                    tag,
                    compatible,
                  ].filter(Boolean).length && (
                    <button
                      type="button"
                      onClick={() => {
                        setCreator("");
                        setType("");
                        setPrice("");
                        setCurrency("");
                        setMaxPrice("");
                        setTag("");
                        setCompatible(false);
                      }}
                    >
                      Clear filters
                    </button>
                  )}
                </form>
                <div className="market-result-tools">
                  <small>
                    {page.items.length} loaded{" "}
                    {view === "collections" ? "collections" : "offers"}
                    {loading ? " · Loading…" : ""}
                  </small>
                  <span />
                  <button
                    className={layout === "list" ? "active" : ""}
                    aria-label="Offer list"
                    aria-pressed={layout === "list"}
                    onClick={() => setLayout("list")}
                  >
                    <List size={14} />
                  </button>
                  <button
                    className={layout === "grid" ? "active" : ""}
                    aria-label="Offer grid"
                    aria-pressed={layout === "grid"}
                    onClick={() => setLayout("grid")}
                  >
                    <LayoutGrid size={14} />
                  </button>
                  <button
                    disabled={!!busy || !page.items.length}
                    onClick={() => void exportItems("json")}
                  >
                    <Download size={14} />
                    JSON
                  </button>
                  <button
                    disabled={!!busy || !page.items.length}
                    onClick={() => void exportItems("csv")}
                  >
                    CSV
                  </button>
                </div>
                <div className={`market-results collection-items ${layout}${!page.items.length ? " is-empty" : ""}`} data-layout={layout} data-content="offers" aria-busy={loading}>
                  {page.items.map((item) => (
                    <article
                      key={marketKey(item)}
                      className={`market-offer collection-item ${active && marketKey(active) === marketKey(item) ? "selected" : ""}`}
                    >
                      <button
                        className="market-offer-select collection-select"
                        title={item.name}
                        aria-label={`View ${item.name}`}
                        aria-pressed={
                          !!active && marketKey(active) === marketKey(item)
                        }
                        onClick={() => void selectItem(item)}
                      >
                        <div className="market-offer-image collection-media">
                          <Thumbnail item={item} />
                          {item.sale && (
                            <span className="market-sale">Sale</span>
                          )}
                        </div>
                        <div className="market-offer-copy collection-copy">
                          <strong>{item.name}</strong>
                          <span className="market-offer-creator">
                            {item.creator || "Server content"}
                          </span>
                          <span className="market-offer-summary">{marketOfferSummary(item)}</span>
                          <div className="market-offer-meta">
                            <span className="market-offer-price">
                              {isOffer(item)
                                ? marketCardPrice(item)
                                : "Published store / collection"}
                            </span>
                            {item.rating !== null && (
                              <small
                                title={`${item.ratings.toLocaleString("en-US")} ratings`}
                              >
                                {item.rating.toFixed(1)} ★
                              </small>
                            )}
                          </div>
                        </div>
                      </button>
                      <div className="market-offer-actions">
                        {isOffer(item) && (
                          <button
                            aria-label={`${hasFavorite(item) ? "Remove" : "Add"} ${item.name} ${hasFavorite(item) ? "from" : "to"} favorites`}
                            aria-pressed={hasFavorite(item)}
                            className={hasFavorite(item) ? "favorited" : ""}
                            disabled={!!busy}
                            onClick={() => void toggleFavorite(item)}
                          >
                            <Star
                              size={14}
                              fill={hasFavorite(item) ? "currentColor" : "none"}
                            />
                          </button>
                        )}
                        <small>
                          {item.provenance.length > 1
                            ? `${item.provenance.length} discovery sources`
                            : discovery?.sources.find(
                                (s) => s.id === reference(item).source,
                              )?.name}
                        </small>
                      </div>
                    </article>
                  ))}
                  {!page.items.length && !loading && !loadingSources && (
                    <div className="market-empty">
                      <Store size={28} />
                      <h3>
                        {error
                          ? "Catalog unavailable"
                          : view === "collections" ? "No published collections" : "No matching server offers"}
                      </h3>
                      <p>
                        {error
                          ? "Refresh discovery or retry the search."
                          : view === "collections" ? "This source has no matching public collection in the selected environment." : "Only 3PServerContent offers associated with these discovery sources appear here. Filters apply to paged catalog results."}
                      </p>
                      {view === "favorites" &&
                        !loading &&
                        favorites.some(
                          (f) =>
                            !page.items.some(
                              (i) =>
                                i.id === f.item &&
                                i.provenance.some((p) => p.source === f.source),
                            ),
                        ) && (
                          <div className="market-missing-favorites">
                            <h4>Unavailable or filtered favorites</h4>
                            <p>
                              Only saved identifiers are retained. You can
                              remove an unavailable offer without loading it.
                            </p>
                            {favorites
                              .filter(
                                (f) =>
                                  !page.items.some(
                                    (i) =>
                                      i.id === f.item &&
                                      i.provenance.some(
                                        (p) => p.source === f.source,
                                      ),
                                  ),
                              )
                              .map((f) => (
                                <div key={favoriteKey(f)}>
                                  <code>{f.item}</code>
                                  <button
                                    disabled={!!busy}
                                    onClick={() =>
                                      void removeUnavailableFavorite(f)
                                    }
                                  >
                                    Remove favorite
                                  </button>
                                </div>
                              ))}
                          </div>
                        )}
                      {!!page.cursor && (
                        <button onClick={() => void loadOffers(true)}>
                          Continue searching next page
                        </button>
                      )}
                    </div>
                  )}
                  {(loading || loadingSources) && (
                    <div className="market-loading market-loading-state" role="status">
                      <Store size={32} />
                      <LoaderCircle className="spin" size={19} />
                      <span>
                        {loadingSources
                          ? "Loading current discovery sources"
                          : "Loading online catalog"}
                      </span>
                    </div>
                  )}
                </div>
                {!!page.cursor && (
                  <div className="market-load-more">
                    <button
                      disabled={loading}
                      onClick={() => void loadOffers(true)}
                    >
                      {loading ? "Loading…" : "Load more server offers"}
                    </button>
                    <small>
                      Filters and availability may leave a catalog page empty.
                    </small>
                  </div>
                )}
              </>
          </main>
          <aside className="market-detail">
            {!active ? (
              <div className="market-empty">
                <Tags size={27} />
                <h3>Select an offer</h3>
                <p>
                  Inspect official images, prices, availability and the exact
                  discovery association.
                </p>
              </div>
            ) : (
              <>
                <header>
                  <div>
                    <small>{active.type || "Catalog details"}</small>
                    <h3>{active.name}</h3>
                  </div>
                  <button
                    aria-label="Close offer details"
                    onClick={() => {
                      detailSequence.current++;

                      setActive(undefined);
                      setDetail(undefined);


                      setBusy("");
                    }}
                  >
                    <X size={16} />
                  </button>
                </header>
                <nav className="market-tabs" aria-label="Offer detail sections">
                  {["overview", ...(hasMarketPreview(detail) ? ["3d"] : []), "technical"].map((tab) => (
                    <button
                      key={tab}
                      className={detailTab === tab ? "active" : ""}
                      onClick={() => {
                        setDetailTab(tab);
                      }}
                    >
                      {tab === "3d"
                        ? "3D preview"
                        : tab === "technical"
                          ? "Technical details"
                          : "Overview"}
                    </button>
                  ))}
                </nav>
                {loadingDetail && (
                  <div className="market-loading">
                    <LoaderCircle className="spin" size={20} />
                    Refreshing item details
                  </div>
                )}
                {detailError && (
                  <FeedbackFlyout title="Marketplace">
                    {detailError}
                  </FeedbackFlyout>
                )}
                {detail && (
                  <div className="market-detail-body">
                    {detailTab === "overview" ? (
                      <>
                        {selectedImage ? (
                          <div className="market-gallery">
                            <img
                              src={selectedImage.url}
                              alt={`${detail.name} preview ${imageIndex + 1}`}
                              referrerPolicy="no-referrer"
                            />
                            <div>
                              <button
                                aria-label="Previous offer image"
                                disabled={imageIndex <= 0}
                                onClick={() => setImageIndex((i) => i - 1)}
                              >
                                <ArrowLeft size={15} />
                              </button>
                              <small>
                                {imageIndex + 1} / {gallery.length}
                              </small>
                              <button
                                aria-label="Next offer image"
                                disabled={imageIndex >= gallery.length - 1}
                                onClick={() => setImageIndex((i) => i + 1)}
                              >
                                <ArrowRight size={15} />
                              </button>
                            </div>
                          </div>
                        ) : (
                          <div className="market-gallery">
                            <Thumbnail item={detail} />
                          </div>
                        )}
                        <div className="market-detail-price">
                          <strong>
                            {isOffer(detail)
                              ? marketPrice(detail)
                              : "Discovery / collection metadata"}
                          </strong>
                          {detail.sale && (
                            <span className="market-sale-inline">Sale</span>
                          )}
                        </div>
                        <p className="market-description">
                          {detail.description ||
                            "No English description supplied."}
                        </p>
                        <dl className="market-properties">
                          <dt>Creator</dt>
                          <dd
                            className="market-creator-name"
                            title={
                              creatorImages[detail.creatorId]
                                ? `Official creator image · Collection ${creatorImages[detail.creatorId].collectionId}`
                                : undefined
                            }
                          >
                            {creatorImages[detail.creatorId] && (
                              <img
                                src={creatorImages[detail.creatorId].url}
                                alt=""
                                referrerPolicy="no-referrer"
                              />
                            )}
                            {detail.creator || "Not supplied"}
                          </dd>
                          <dt>Rating</dt>
                          <dd>
                            {detail.rating === null
                              ? "Not rated"
                              : `${detail.rating.toFixed(1)} / 5 · ${detail.ratings.toLocaleString("en-US")} ratings`}
                          </dd>
                          <dt>Platforms</dt>
                          <dd>
                            {detail.platforms.join(", ") || "Not supplied"}
                          </dd>
                          <dt>Minimum version</dt>
                          <dd>{detail.minVersion || "Not supplied"}</dd>
                          <dt>Maximum version</dt>
                          <dd>{detail.maxVersion || "Not supplied"}</dd>
                          <dt>Compatibility</dt>
                          <dd>{detail.compatibility}</dd>
                          <dt>Available from</dt>
                          <dd>{marketDate(detail.start)}</dd>
                          <dt>Available until</dt>
                          <dd>{marketDate(detail.end)}</dd>
                          <dt>Last updated</dt>
                          <dd>{marketDate(detail.modified)}</dd>
                        </dl>
                        {detail.references.length > 0 && (
                          <div className="market-contained">
                            <h4>Published item references</h4>
                            {detail.references.map((r) => (
                              <p key={r.id}>
                                <code>{r.id}</code>
                                {r.type && <small>{r.type}</small>}
                              </p>
                            ))}
                          </div>
                        )}
                        <div className="market-detail-actions">
                          {isOffer(detail) && (
                            <button
                              disabled={!!busy}
                              onClick={() =>
                                void action("Opening offer", () =>
                                  api.OpenMarketplaceItem(
                                    reference(detail).source,
                                    detail.id,
                                    environment,
                                  ),
                                )
                              }
                            >
                              <ExternalLink size={14} />
                              Open in Minecraft
                            </button>
                          )}
                          {isOffer(detail) && (
                            <button
                              disabled={!!busy}
                              onClick={() => void toggleFavorite(detail)}
                            >
                              <Star size={14} />
                              {hasFavorite(detail)
                                ? "Remove favorite"
                                : "Favorite"}
                            </button>
                          )}
                          <button onClick={() => void copy(detail.link)}>
                            <Copy size={14} />
                            Copy link
                          </button>
                          <button onClick={() => void copy(detail.id)}>
                            Copy ID
                          </button>
                        </div>
                      </>
                    ) : detailTab === "technical" ? (
                      <>
                        <dl className="market-properties">
                          <dt>Item ID</dt>
                          <dd>
                            <code>{detail.id}</code>
                          </dd>
                          <dt>Title ID</dt>
                          <dd>
                            <code>{detail.titleId}</code>
                          </dd>
                          <dt>Creator entity</dt>
                          <dd>
                            <code>{detail.creatorId || "Not supplied"}</code>
                          </dd>
                          <dt>Tags</dt>
                          <dd>{detail.tags.join(", ") || "None"}</dd>
                          <dt>Alternate IDs</dt>
                          <dd>{detail.ids.join(", ")}</dd>
                        </dl>
                        <h4>Association provenance</h4>
                        {detail.provenance.map((p, i) => (
                          <div className="market-provenance" key={i}>
                            <strong>
                              {discovery?.sources.find((s) => s.id === p.source)
                                ?.name || p.source}
                            </strong>
                            <p>{p.description}</p>
                            <code>{p.id}</code>
                            <small>{p.relation}</small>
                          </div>
                        ))}
                        <div className="market-json-heading">
                          <h4>Catalog JSON</h4>
                          <button
                            onClick={() =>
                              void copy(
                                JSON.stringify(detail.technical, null, 2),
                              )
                            }
                          >
                            <Copy size={13} />
                            Copy JSON
                          </button>
                        </div>
                        <pre className="market-json">
                          {JSON.stringify(detail.technical, null, 2)}
                        </pre>
                      </>
                    ) : detailTab === "3d" && hasMarketPreview(detail) ? (
                      <Suspense fallback={<div className="market-loading">Loading 3D viewer...</div>}>
                        <MarketplaceModel item={detail} />
                      </Suspense>
                    ) : null}
                  </div>
                )}
              </>
            )}
          </aside>
        </div>
      )}
      <Dialog open={filtersOpen} onOpenChange={setFiltersOpen}>
        <DialogContent className="studio-window market-filter-dialog">
          <DialogHeader><DialogTitle>Marketplace filters</DialogTitle><DialogDescription>Refine the current online results.</DialogDescription></DialogHeader>
          <div className="studio-dialog-body">
                    <div
                      className="market-advanced-filters"
                      id="market-advanced-filters"
                    >
                      <label>Content type
                  <select
                    aria-label="Offer content type"
                    value={type}
                    onChange={(e) => setType(e.target.value)}
                  >
                    <option value="">All content types</option>
                    {facets.types.map((v) => (
                      <option key={v} value={v}>
                        {v}
                      </option>
                    ))}
                  </select>
</label>
                      <label>
                        Creator
                        <select
                          aria-label="Offer creator"
                          value={creator}
                          disabled={sourceID !== "all"}
                          onChange={(e) => setCreator(e.target.value)}
                        >
                          <option value="">All discovery creators</option>
                          {Array.from(
                            new Map(
                              (discovery?.sources || [])
                                .filter((s) => s.creatorId)
                                .map((s) => [
                                  s.creatorId,
                                  s.creator || s.creatorId,
                                ]),
                            ).entries(),
                          ).map(([id, name]) => (
                            <option key={id} value={id}>
                              {name}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label>
                        Price
                        <select
                          aria-label="Offer price filter"
                          value={price}
                          onChange={(e) => setPrice(e.target.value)}
                        >
                          <option value="">All prices</option>
                          <option value="free">Free</option>
                          <option value="paid">Paid</option>
                          <option value="sale">On sale</option>
                        </select>
                      </label>
                      <label>
                        Currency
                        <select
                          aria-label="Offer currency"
                          value={currency}
                          onChange={(e) => setCurrency(e.target.value)}
                        >
                          <option value="">All currencies</option>
                          {facets.currencies.map((v) => (
                            <option key={v} value={v}>
                              {currencyLabel(v)}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label>
                        Maximum price
                        <input
                          aria-label="Maximum offer price"
                          type="number"
                          min="0"
                          step="1"
                          value={maxPrice}
                          onChange={(e) => setMaxPrice(e.target.value)}
                          placeholder="Max price"
                          disabled={!currency}
                        />
                      </label>
                      <label>
                        Additional tag
                        <input
                          aria-label="Additional offer tag"
                          value={tag}
                          onChange={(e) => setTag(e.target.value)}
                          placeholder="Additional tag"
                        />
                      </label>
                      <label className="market-checkbox">
                        <input
                          type="checkbox"
                          checked={compatible}
                          onChange={(e) => setCompatible(e.target.checked)}
                        />
                        Compatible with this client
                      </label>
                    </div></div>
          <DialogFooter><Button variant="secondary" onClick={() => { setCreator("");setType("");setPrice("");setCurrency("");setMaxPrice("");setTag("");setCompatible(false); }}>Reset filters</Button><Button onClick={() => setFiltersOpen(false)}>Done</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
