export type MarketImage = { url: string; type: string; tag: string };
export function discoveryBanner(item: {
  images: MarketImage[];
}): string | undefined {
  return item.images.find(
    (image) =>
      image.type.toLowerCase() === "banner" ||
      image.tag.toLowerCase() === "banner",
  )?.url;
}
export type MarketPrice = {
  amounts: { currency: string; label?: string; value: number }[];
  units: number;
  duration: number;
};
export type MarketProvenance = {
  source: string;
  relation: string;
  id: string;
  description: string;
};
export type MarketItem = {
  id: string;
  titleId: string;
  name: string;
  description: string;
  creator: string;
  creatorId: string;
  type: string;
  kind: string;
  images: MarketImage[];
  tags: string[];
  platforms: string[];
  ids: string[];
  prices: MarketPrice[];
  free: boolean;
  sale: boolean;
  rating: number | null;
  ratings: number;
  minVersion: string;
  maxVersion: string;
  compatibility: string;
  start: string;
  end: string;
  modified: string;
  references: { id: string; type: string }[];
  link: string;
  technical: Record<string, unknown>;
  provenance: MarketProvenance[];
  preview?: { geometry: unknown; texture: string };
};
export type MarketSource = {
  id: string;
  kind: "featured" | "experience";
  name: string;
  address: string;
  experienceId: string;
  titleId: string;
  storePageId: string;
  creatorId: string;
  creator: string;
  overview: MarketItem;
};
export type MarketDiscovery = {
  revision: string;
  sources: MarketSource[];
  checkedAt: string;
};
export type MarketQuery = {
  creator: string;
  environment: "production" | "staging";
  source: string;
  relation: string;
  search: string;
  cursor: string;
  count: number;
  sort: string;
  type: string;
  tag: string;
  price: string;
  currency: string;
  maxPrice?: number;
  compatible: boolean;
};
export type MarketPage = {
  items: MarketItem[];
  cursor: string;
  warnings: string[];
  checkedAt: string;
  relation: string;
};
export type MarketFavorite = {
  environment?: "production" | "staging";
  source: string;
  item: string;
  relation: string;
};
export function marketKey(item: MarketItem) {
  return `${item.titleId}:${item.id}:${item.provenance[0]?.source || ""}`;
}
export function marketRef(item: MarketItem): MarketFavorite {
  const p = item.provenance[0];
  return {
    source: p?.source || "",
    item: item.id,
    relation: p?.relation || "offers",
  };
}
export function marketPrice(item: MarketItem) {
  if (!item.prices.length) return "Price unavailable";
  return item.prices
    .map(
      (p) =>
        p.amounts
          .map(
            (a) =>
              `${a.value.toLocaleString("en-US")} ${a.label || currencyLabel(a.currency)}`,
          )
          .join(" + ") +
        (p.units > 1 ? ` / ${p.units}` : "") +
        (p.duration ? ` / ${p.duration.toLocaleString("en-US")} seconds` : ""),
    )
    .join(" or ");
}
export function currencyLabel(currency: string) {
  if (currency.toLowerCase() === "ecd19d3c-7635-402c-a185-eb11cb6c6946") return "Minecoins";
  return /^[0-9a-f]{8}-[0-9a-f-]{27}$/i.test(currency) ? "Virtual currency" : currency;
}
export function marketCardPrice(item: MarketItem) {
  if (item.free) return "Free";
  const known = item.prices.find(p => p.amounts.every(a => a.label || currencyLabel(a.currency) !== "Virtual currency"));
  return marketPrice({ ...item, prices: known ? [known] : item.prices.slice(0, 1) });
}
export function marketOfferSummary(item: MarketItem) {
  return item.description.trim().replace(/\s+/g, " ") ||
    (item.references.length ? `${item.references.length} included items` :
      item.minVersion ? `Minecraft ${item.minVersion} or newer` :
      item.sale ? "Currently on sale" : "Server offer");
}
export function hasMarketPreview(item?: MarketItem) {
  if (!item?.preview?.texture) return false;
  const geometry = item.preview.geometry as { "minecraft:geometry"?: { bones?: { cubes?: { size?: unknown }[] }[] }[] } | undefined;
  return geometry?.["minecraft:geometry"]?.some(model => model.bones?.some(bone => bone.cubes?.some(cube =>
    Array.isArray(cube.size) && cube.size.length === 3 && cube.size.every(n => typeof n === "number" && Number.isFinite(n) && n >= 0) && cube.size.filter(n => n > 0).length >= 2,
  ))) ?? false;
}
export function marketDate(value: string) {
  if (!value || value.startsWith("0001-")) return "Not supplied";
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString("en-US");
}
export function openMarketplace(ref: MarketFavorite) {
  sessionStorage.setItem(
    "craftbox:marketplace-navigation",
    JSON.stringify(ref),
  );
  window.dispatchEvent(new CustomEvent("craftbox:open-marketplace"));
}
export function takeMarketplaceNavigation(): MarketFavorite | undefined {
  try {
    const raw = sessionStorage.getItem("craftbox:marketplace-navigation");
    sessionStorage.removeItem("craftbox:marketplace-navigation");
    return raw ? JSON.parse(raw) : undefined;
  } catch {
    return undefined;
  }
}
