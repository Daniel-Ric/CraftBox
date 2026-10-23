import { useState } from "react";
import { BookOpen, Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { lookCategories, type SceneLook } from "./scene-looks";
import { filterLooks, lookCatalog } from "./look-catalog";
import "./look-library.css";
export default function LookLibrary({
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
    [open, setOpen] = useState(false);
  const filtered = filterLooks(category, query);
  const [learning, setLearning] = useState(false),
    [learningMessage, setLearningMessage] = useState("");
  const learn = async () => {
    setLearning(true);
    setLearningMessage("");
    try {
      await onLearn?.();
      setLearningMessage(
        "Saved as an auto-styling reference. Similar models will consider this look in future.",
      );
    } catch (e) {
      setLearningMessage(String(e));
    } finally {
      setLearning(false);
    }
  };
  const pick = (id: string) => {
    onSelect(id);
    setOpen(false);
  };
  const card = (look: SceneLook) => (
    <button
      key={look.id}
      className={`inspector-look look-${look.id}`}
      disabled={disabled}
      aria-pressed={selected === look.id}
      onClick={() => pick(look.id)}
    >
      <i
        aria-hidden="true"
        style={{
          background: `linear-gradient(180deg,${look.palette.map((c, i) => `${c} ${i * 25}% ${(i + 1) * 25}%`).join(",")})`,
        }}
      />
      <span>
        <strong>{look.name}</strong>
        <small>{look.description}</small>
      </span>
    </button>
  );
  const search = (modal = false) => (
    <label className="look-search">
      <Search size={13} />
      <input
        aria-label={modal ? "Browse library" : "Search looks"}
        type="search"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search scenes or moods …"
      />
    </label>
  );
  return (
    <>
      <legend>Scene looks</legend>
      <p className="inspector-intro">
        Curated scenes for lighting, color, materials and atmosphere.
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
        onClick={() => setOpen(true)}
        disabled={disabled}
      >
        <BookOpen />
        Full library …
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
            ? "Analyzing model features …"
            : "Remember this look for auto styling"}
        </Button>
      )}
      {learningMessage && (
        <p role="status" className="inspector-intro">
          {learningMessage}
        </p>
      )}
      <div className="look-results-count" role="status">
        {filtered.length} looks{query && ` for “${query}”`}
      </div>
      <div className="inspector-looks">{filtered.map(card)}</div>
      {!filtered.length && (
        <p className="inspector-intro">
          No matching look. Select another category or change your search.
        </p>
      )}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="look-library-dialog">
          <DialogHeader>
            <DialogTitle>Scene looks</DialogTitle>
            <DialogDescription>
              Lighting moods for different worlds. Every look remains fully editable.
            </DialogDescription>
          </DialogHeader>
          <div className="look-library-body">
            <nav aria-label="Scene categories">
              <button
                aria-pressed={category === "recommended"}
                onClick={() => setCategory("recommended")}
              >
                Recommended
              </button>
              <button
                aria-pressed={category === "all"}
                onClick={() => setCategory("all")}
              >
                All scenes <small>{lookCatalog.length}</small>
              </button>
              {Object.entries(lookCategories).map(([id, label]) => (
                <button
                  key={id}
                  aria-pressed={category === id}
                  onClick={() => setCategory(id)}
                >
                  {label}
                  <small>
                    {lookCatalog.filter((l) => l.category === id).length}
                  </small>
                </button>
              ))}
            </nav>
            <section>
              <div className="look-library-searchbar">
                {search(true)}
                <span>{filtered.length} Looks</span>
              </div>
              <div className="look-library-grid">{filtered.map(card)}</div>
              {!filtered.length && <p>No looks for this selection.</p>}
            </section>
          </div>
          <DialogFooter>
            <span className="look-library-note">
              Output format and camera are preserved.
            </span>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
