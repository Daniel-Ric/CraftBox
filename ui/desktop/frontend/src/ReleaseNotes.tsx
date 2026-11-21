import { createElement, type ReactNode } from "react";

export default function ReleaseNotes({
  notes,
  onOpenURL,
}: {
  notes: string;
  onOpenURL: (url: string) => void;
}) {
  function inline(text: string): ReactNode[] {
    return text
      .split(/(\[[^\]]+\]\([^\s)]+\)|`[^`]+`|\*\*[^*]+\*\*)/g)
      .map((part, index) => {
        const link = part.match(/^\[([^\]]+)\]\(([^\s)]+)\)$/);
        if (link) {
          try {
            const url = new URL(link[2]);
            if (url.protocol === "https:" || url.protocol === "http:")
              return (
                <a
                  key={index}
                  href={url.href}
                  onClick={(event) => {
                    event.preventDefault();
                    onOpenURL(url.href);
                  }}
                >
                  {link[1]}
                </a>
              );
          } catch {}
          return part;
        }
        if (part.startsWith("**") && part.endsWith("**"))
          return <strong key={index}>{part.slice(2, -2)}</strong>;
        if (part.startsWith("`") && part.endsWith("`"))
          return <code key={index}>{part.slice(1, -1)}</code>;
        return part;
      });
  }
  const lines = notes.replace(/\r\n?/g, "\n").trim().split("\n");
  if (/^#{1,6}\s+what['’]s new\s*$/i.test(lines[0])) lines.shift();
  const blocks: ReactNode[] = [];
  for (let index = 0; index < lines.length;) {
    const line = lines[index].trim();
    if (!line) {
      index++;
      continue;
    }
    const heading = line.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      blocks.push(
        createElement(
          `h${Math.min(heading[1].length + 1, 6)}`,
          { key: index },
          inline(heading[2]),
        ),
      );
      index++;
      continue;
    }
    if (/^[-*+]\s+/.test(line)) {
      const start = index,
        items: ReactNode[] = [];
      while (index < lines.length && /^\s*[-*+]\s+/.test(lines[index])) {
        items.push(
          <li key={index}>
            {inline(lines[index].trim().replace(/^[-*+]\s+/, ""))}
          </li>,
        );
        index++;
      }
      blocks.push(<ul key={start}>{items}</ul>);
      continue;
    }
    const start = index,
      paragraph: string[] = [];
    while (
      index < lines.length &&
      lines[index].trim() &&
      !/^\s*(?:#{1,6}\s|[-*+]\s)/.test(lines[index].trim())
    )
      paragraph.push(lines[index++].trim());
    blocks.push(<p key={start}>{inline(paragraph.join(" "))}</p>);
  }
  return <div className="release-notes" lang="en" translate="no">{blocks}</div>;
}
