import type { ReactNode } from "react";
import { ExternalLink } from "lucide-react";

export default function WebsiteLink(p: {
  href: string;
  onOpen: (url: string) => void;
  children: ReactNode;
}) {
  return (
    <a
      className="studio-button"
      href={p.href}
      onClick={(event) => {
        event.preventDefault();
        p.onOpen(p.href);
      }}
    >
      <ExternalLink size={14} aria-hidden="true" />
      {p.children}
    </a>
  );
}
