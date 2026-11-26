const normalize = (value: string) =>
  value
    .normalize("NFKD")
    .toLowerCase()
    .replace(/\p{M}/gu, "")
    .replaceAll("ß", "ss")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim();

function editDistance(a: string, b: string): number {
  let prior = Array.from({ length: b.length + 1 }, (_, i) => i);
  let earlier = prior;
  for (let i = 1; i <= a.length; i++) {
    const row = [i];
    for (let j = 1; j <= b.length; j++) {
      row[j] = Math.min(
        row[j - 1] + 1,
        prior[j] + 1,
        prior[j - 1] + Number(a[i - 1] !== b[j - 1]),
      );
      if (i > 1 && j > 1 && a[i - 1] === b[j - 2] && a[i - 2] === b[j - 1])
        row[j] = Math.min(row[j], earlier[j - 2] + 1);
    }
    earlier = prior;
    prior = row;
  }
  return prior[b.length];
}

export type RenderSearchEntry = {
  id: string;
  label: string;
  tab: string;
  terms: string[];
  open: () => void;
};

export function findRenderControls(
  query: string,
  entries: RenderSearchEntry[],
) {
  const input = normalize(query.slice(0, 120));
  if (!input) return [];
  const tokens = input.split(" ");
  return entries
    .map((entry) => {
      const label = normalize(entry.label);
      const terms = [
        ...new Set(
          [label, ...entry.terms.map(normalize)].flatMap((value) =>
            value.split(" "),
          ),
        ),
      ];
      let score =
        label === input
          ? 100
          : label.startsWith(input)
            ? 65
            : label.includes(input)
              ? 50
              : 0;
      for (const token of tokens) {
        const best = Math.max(
          0,
          ...terms.map((term) => {
            if (term === token) return 16;
            if (term.startsWith(token)) return 12;
            if (term.includes(token)) return 8;
            if (token.length < 4 || Math.abs(term.length - token.length) > 2)
              return 0;
            const distance = editDistance(token, term);
            return distance <= (token.length >= 7 ? 2 : 1) ? 6 - distance : 0;
          }),
        );
        if (best === 0 && !label.includes(input)) return { entry, score: 0 };
        score += best;
      }
      return { entry, score };
    })
    .filter(({ score }) => score > 0)
    .sort((a, b) => b.score - a.score)
    .map(({ entry }) => entry);
}
