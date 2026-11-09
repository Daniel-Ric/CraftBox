import {
  Boxes,
  Layers3,
  Shirt,
  Camera,
  Clapperboard,
  ScanLine,
  UserRound,
  Server,
  Cloud,
  UsersRound,
  Compass,
  Bookmark,
  Globe,
  type LucideIcon,
  type LucideProps,
} from "lucide-react";

function subjectIcon(role: string, Icon: LucideIcon) {
  function SubjectIcon({ className = "", ...props }: LucideProps) {
    return (
      <Icon
        aria-hidden="true"
        data-icon-role={role}
        className={`studio-icon ${className}`.trim()}
        {...props}
      />
    );
  }
  SubjectIcon.displayName = `${role}Icon`;
  return SubjectIcon;
}

export const WorldIcon = subjectIcon("world", Boxes);
export const PackIcon = subjectIcon("pack", Layers3);
export const SkinIcon = subjectIcon("skin", Shirt);
export const RenderIcon = subjectIcon("render", Camera);
export const CinematicIcon = subjectIcon("cinematic", Clapperboard);
export const CaptureIcon = subjectIcon("capture", ScanLine);
export const AccountIcon = subjectIcon("account", UserRound);

export function workspaceIcon(page: string) {
  switch (page) {
    case "packs":
      return PackIcon;
    case "skins":
      return SkinIcon;
    case "render":
      return RenderIcon;
    case "cinematic":
      return CinematicIcon;
    default:
      return WorldIcon;
  }
}

export function WorkspaceIcon({
  page,
  ...props
}: LucideProps & { page: string }) {
  const Icon = workspaceIcon(page);
  return <Icon {...props} />;
}

export function SourceKindIcon({
  kind,
  className = "",
  ...props
}: LucideProps & { kind: string }) {
  const Icon =
    kind === "minecraft"
      ? Bookmark
      : kind === "featured"
        ? Globe
        : kind === "realms"
          ? Cloud
          : kind === "worlds"
            ? UsersRound
            : kind === "events"
              ? Compass
              : Server;
  return (
    <Icon
      aria-hidden="true"
      className={`studio-icon ${className}`.trim()}
      {...props}
    />
  );
}
