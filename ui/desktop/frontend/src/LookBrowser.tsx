import { useMemo, useState } from "react";
import { notify, notifyFailure } from "./StudioFlyouts";
import { Palette, Search, Star, Sun, Moon } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./components/ui/dialog";
import { Button } from "./components/ui/button";
import { lookCategories, type SceneLook } from "./scene-looks";
import { filterLooks, lookCatalog } from "./look-catalog";
import { renderPresets } from "./render-settings";
import "./look-library.css";
const favouriteKey = "craftbox-look-favourites-v1";
function readFavourites(): string[] {
  try {
    const ids = JSON.parse(localStorage.getItem(favouriteKey) ?? "[]");
    return Array.isArray(ids)
      ? ids
          .filter(
            (id) =>
              typeof id === "string" && lookCatalog.some((l) => l.id === id),
          )
          .slice(0, 200)
      : [];
  } catch {
    return [];
  }
}
function night(look: SceneLook) {
  const p = renderPresets[look.id];
  return (
    p.sky === "night" || (p.timeOfDay ?? 12) >= 20 || (p.timeOfDay ?? 12) < 6
  );
}
export default function LookBrowser({
  selected,
  onSelect,
  disabled,
  onLearn,
}: {
  selected: string;
  onSelect: (id: string) => void;
  disabled: boolean;
  onLearn?: () => Promise<void>;
}) {
  const [category, setCategory] = useState("recommended"),
    [query, setQuery] = useState(""),
    [open, setOpen] = useState(false),
    [inspected, setInspected] = useState(selected || "crisp");
  const [time, setTime] = useState("all"),
    [sort, setSort] = useState("curated"),
    [favourites, setFavourites] = useState(readFavourites),
    [learning, setLearning] = useState(false),
    [message, setMessage] = useState("");
  const filtered = useMemo(
    () =>
      filterLooks(category === "favourites" ? "all" : category, query)
        .filter(
          (l) =>
            (category !== "favourites" || favourites.includes(l.id)) &&
            (time === "all" || (time === "night" ? night(l) : !night(l))),
        )
        .sort((a, b) =>
          sort === "name"
            ? a.name.localeCompare(b.name)
            : sort === "favourite"
              ? Number(favourites.includes(b.id)) -
                Number(favourites.includes(a.id))
              : 0,
        ),
    [category, query, time, sort, favourites],
  );
  const look = lookCatalog.find((l) => l.id === inspected) ?? lookCatalog[0],
    preset = renderPresets[look.id];
  const palette = (l: SceneLook) => (
    <i
      className="look-palette"
      aria-hidden="true"
      style={{ background: `linear-gradient(155deg,${l.palette.join(",")})` }}
    />
  );
  async function learn() {
    setLearning(true);
    setMessage("");
    try {
      await onLearn?.();
      setMessage("Styling reference saved.");
    } catch (e) {
      notifyFailure(e, "Styling reference could not be saved");
    } finally {
      setLearning(false);
    }
  }
  function toggleFavourite() {
    const ids = favourites.includes(look.id)
      ? favourites.filter((id) => id !== look.id)
      : [look.id, ...favourites];
    try {
      localStorage.setItem(favouriteKey, JSON.stringify(ids));
      setFavourites(ids);
    } catch {
      notify({ tone: "warning", title: "Favourites could not be saved", description: "Favourites could not be saved to this profile." });
    }
  }
  const search = (modal = false) => (
    <label className="look-search">
      <Search size={14} />
      <input
        type="search"
        aria-label={modal ? "Browse library" : "Search looks"}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search scenes, seasons or moods..."
      />
    </label>
  );
  const choose = (id: string) => {
    onSelect(id);
    setOpen(false);
  };
  return (
    <>
      <legend>Scene looks</legend>
      <p className="inspector-intro">
        Lighting, atmosphere and color references.
      </p>
      {search()}
      <select
        className="look-category-select"
        aria-label="Look category"
        value={category}
        onChange={(e) => setCategory(e.target.value)}
      >
        <option value="recommended">Recommended</option>
        <option value="all">All {lookCatalog.length} Looks</option>
        <option value="favourites">Favourites</option>
        {Object.entries(lookCategories).map(([id, label]) => (
          <option key={id} value={id}>
            {label}
          </option>
        ))}
      </select>
      <Button
        variant="outline"
        className="look-library-open"
        aria-label="Open look library"
        disabled={disabled}
        onClick={() => {
          setInspected(selected || filtered[0]?.id || "crisp");
          setOpen(true);
        }}
      >
        <Palette />
        Browse {lookCatalog.length} looks...
      </Button>
      {onLearn && selected && (
        <Button
          variant="ghost"
          className="look-library-open"
          disabled={disabled || learning}
          aria-label="Remember current look for auto styling"
          onClick={() => void learn()}
        >
          {learning
            ? "Analyzing model features..."
            : "Remember this look for auto styling"}
        </Button>
      )}
      {message && (
        <p role="status" className="inspector-intro">
          {message}
        </p>
      )}
      <div className="look-results-count" role="status">
        {filtered.length} looks{query && ` for “${query}”`}
      </div>
      <div className="inspector-looks">
        {filtered.slice(0, 12).map((l) => (
          <button
            key={l.id}
            className="inspector-look"
            disabled={disabled}
            aria-pressed={selected === l.id}
            onClick={() => choose(l.id)}
          >
            {palette(l)}
            <span>
              <strong>{l.name}</strong>
              <small>{l.description}</small>
            </span>
          </button>
        ))}
      </div>
      {filtered.length > 12 && (
        <Button
          variant="ghost"
          className="look-library-open"
          onClick={() => setOpen(true)}
        >
          Show all {filtered.length} matches
        </Button>
      )}
      {!filtered.length && (
        <p className="inspector-intro">
          No matching looks. Change the category or search.
        </p>
      )}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="studio-window studio-dialog look-library-dialog">
          <DialogHeader>
            <DialogTitle>Scene looks</DialogTitle>
          </DialogHeader>
          <DialogDescription className="look-library-description">
            Inspect a reference, then apply its lighting and atmosphere. Camera
            and output format stay unchanged.
          </DialogDescription>
          <div className="look-library-body">
            <nav aria-label="Scene categories">
              {[
                ["recommended", "Recommended"],
                ["all", "All scenes"],
                ["favourites", "Favourites"],
                ...Object.entries(lookCategories),
              ].map(([id, label]) => (
                <button
                  key={id}
                  aria-pressed={category === id}
                  onClick={() => setCategory(id)}
                >
                  {label}
                  <small>
                    {id === "all"
                      ? lookCatalog.length
                      : id === "favourites"
                        ? favourites.length
                        : filterLooks(id, "").length}
                  </small>
                </button>
              ))}
            </nav>
            <section aria-label="Scene look results">
              <div className="look-library-searchbar">
                {search(true)}
                <span role="status">{filtered.length} looks</span>
              </div>
              <div className="look-library-filters">
                <select
                  aria-label="Look time filter"
                  value={time}
                  onChange={(e) => setTime(e.target.value)}
                >
                  <option value="all">Any time of day</option>
                  <option value="day">Daylight & twilight</option>
                  <option value="night">Night</option>
                </select>
                <select
                  aria-label="Look sorting"
                  value={sort}
                  onChange={(e) => setSort(e.target.value)}
                >
                  <option value="curated">Curated order</option>
                  <option value="name">Name A–Z</option>
                  <option value="favourite">Favourites first</option>
                </select>
              </div>
              <div className="look-library-grid">
                {filtered.map((l) => (
                  <button
                    key={l.id}
                    className="look-browser-card"
                    aria-label={`Inspect ${l.name}`}
                    aria-pressed={look.id === l.id}
                    onClick={() => setInspected(l.id)}
                  >
                    {palette(l)}
                    <span className="look-card-heading">
                      <strong>{l.name}</strong>
                      {favourites.includes(l.id) && (
                        <Star size={13} aria-label="Favourite" />
                      )}
                    </span>
                    <small>{lookCategories[l.category]}</small>
                    <span>{l.description}</span>
                    {l.id === selected && <em>Currently applied</em>}
                  </button>
                ))}
              </div>
              {!filtered.length && (
                <div className="look-browser-empty">
                  <Search />
                  <strong>No matching scenes</strong>
                  <p>Try fewer search terms or reset the filters.</p>
                  <Button
                    variant="outline"
                    onClick={() => {
                      setCategory("all");
                      setQuery("");
                      setTime("all");
                    }}
                  >
                    Reset filters
                  </Button>
                </div>
              )}
            </section>
            <aside className="look-library-detail" aria-label="Look details">
              {palette(look)}
              <small>{lookCategories[look.category]}</small>
              <h3>{look.name}</h3>
              <p>{look.description}</p>
              <div className="look-detail-tags">
                {look.tags.slice(0, 5).map((tag) => (
                  <span key={tag}>{tag}</span>
                ))}
              </div>
              <dl>
                <dt>Light</dt>
                <dd>
                  {night(look) ? <Moon size={13} /> : <Sun size={13} />}{" "}
                  {night(look) ? "Night" : "Daylight / twilight"}
                </dd>
                <dt>Atmosphere</dt>
                <dd>
                  {preset.atmosphere?.clouds
                    ? "Cloud banks"
                    : preset.atmosphere?.fog
                      ? "Low mist"
                      : "Clear"}
                </dd>
                <dt>Exposure</dt>
                <dd>{(preset.exposure ?? 1).toFixed(2)}</dd>
                <dt>Color intensity</dt>
                <dd>{(preset.saturation ?? 1).toFixed(2)}</dd>
              </dl>
              <p className="look-detail-note">
                Palette illustration. The actual result depends on your world,
                materials and camera.
              </p>
              <Button
                variant="outline"
                aria-label={
                  favourites.includes(look.id)
                    ? "Remove look from favourites"
                    : "Add look to favourites"
                }
                onClick={toggleFavourite}
              >
                <Star
                  size={14}
                  fill={favourites.includes(look.id) ? "currentColor" : "none"}
                />
                {favourites.includes(look.id)
                  ? "Saved favourite"
                  : "Add to favourites"}
              </Button>
            </aside>
          </div>
          <DialogFooter>
            <span className="look-library-note">Inspecting: {look.name}</span>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Close
            </Button>
            <Button disabled={disabled} onClick={() => choose(look.id)}>
              Apply look
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
