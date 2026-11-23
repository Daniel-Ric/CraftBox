import { useEffect, useRef, useState } from "react";
import {
  Activity,
  Archive,
  Blocks,
  FolderOpen,
  HardDrive,
  Terminal,
  UserRound,
  UsersRound,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { api, type CraftBoxFolder, type MinecraftFolders } from "./bridge";
import { PackIcon, WorldIcon } from "./studio-icons";

export const craftBoxFolders = [
  { id: "worlds", label: "Captured worlds", icon: WorldIcon },
  { id: "packs", label: "Resource pack archives", icon: PackIcon },
  { id: "asset-library", label: "Asset library", icon: Archive },
  { id: "captures", label: "Captures", icon: FolderOpen },
  { id: "packets", label: "Packet recordings", icon: Activity },
  { id: "structures", label: "Structure sessions", icon: Blocks },
] as const;

const storageProfiles = {
  shared: { label: "Shared storage", icon: UsersRound },
  legacy: { label: "Legacy storage", icon: Archive },
  default: { label: "Default storage", icon: HardDrive },
  manual: { label: "Selected folder", icon: FolderOpen },
  personal: { label: "Local profile", icon: UserRound },
};

type Props = {
  signedIn: boolean;
  onOpenCraftBoxFolder: (folder: CraftBoxFolder) => void;
  onOpenLogFolder: () => void;
  onOpenResultFolder: (path: string) => void;
};

export default function FolderShortcuts(p: Props) {
  const [open, setOpen] = useState(false);
  const [locations, setLocations] = useState<MinecraftFolders[]>();
  const [failed, setFailed] = useState(false);
  const [gamertags, setGamertags] = useState<Record<string, string>>({});
  const folderRequest = useRef<Promise<MinecraftFolders[]> | null>(undefined);
  useEffect(() => {
    if (!open && folderRequest.current === null) return;
    let active = true;
    const request = folderRequest.current ?? api.GetMinecraftFolders();
    folderRequest.current = request;
    void request
      .then(
        (folders) => {
          if (active) {
            setLocations(folders);
            setFailed(false);
          }
        },
        () => {
          if (active) setFailed(true);
        },
      )
      .finally(() => {
        if (folderRequest.current === request) folderRequest.current = null;
      });
    return () => {
      active = false;
    };
  }, [open]);

  const lookupSignature = locations
    ?.flatMap((location) => (location.lookupKey ? [location.lookupKey] : []))
    .sort()
    .join(":");
  useEffect(() => {
    if (!p.signedIn || !lookupSignature) return;
    let active = true;
    void api.ResolveMinecraftGamertags().then(
      (names) => {
        if (active) setGamertags((previous) => ({ ...previous, ...names }));
      },
      () => {},
    );
    return () => {
      active = false;
    };
  }, [p.signedIn, lookupSignature]);
  const groups = [
    { label: "Minecraft for Windows", channel: "stable" },
    { label: "Minecraft Preview", channel: "preview" },
    { label: "Added locations", channel: undefined },
  ]
    .map((group) => ({
      ...group,
      locations:
        locations?.filter((location) => location.channel === group.channel) ??
        [],
    }))
    .filter((group) => group.locations.length);

  function minecraftItems({
    worlds,
    resourcePacks,
    behaviorPacks,
  }: MinecraftFolders) {
    return (
      <>
        {worlds && (
          <DropdownMenuItem onClick={() => p.onOpenResultFolder(worlds)}>
            <WorldIcon /> Worlds
          </DropdownMenuItem>
        )}
        {resourcePacks && (
          <DropdownMenuItem onClick={() => p.onOpenResultFolder(resourcePacks)}>
            <PackIcon /> Resource packs
          </DropdownMenuItem>
        )}
        {behaviorPacks && (
          <DropdownMenuItem onClick={() => p.onOpenResultFolder(behaviorPacks)}>
            <Blocks /> Behavior packs
          </DropdownMenuItem>
        )}
      </>
    );
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        aria-label="Open storage locations"
        data-tooltip="Open storage locations"
      >
        <HardDrive size={19} />
        <span>Saved data</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        side="right"
        sideOffset={8}
        className="sidebar-folder-menu"
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel>CraftBox</DropdownMenuLabel>
          {craftBoxFolders.map(({ id, label, icon: Icon }) => (
            <DropdownMenuItem
              key={id}
              onClick={() => p.onOpenCraftBoxFolder(id)}
            >
              <Icon />
              {label}
            </DropdownMenuItem>
          ))}
          <DropdownMenuItem onClick={p.onOpenLogFolder}>
            <Terminal /> Logs
          </DropdownMenuItem>
        </DropdownMenuGroup>
        {groups.length === 0 ? (
          <DropdownMenuGroup>
            <DropdownMenuLabel>Minecraft</DropdownMenuLabel>
            <DropdownMenuItem disabled>
              {failed
                ? "Could not load Minecraft folders"
                : locations
                  ? "No Minecraft folders found"
                  : "Loading…"}
            </DropdownMenuItem>
          </DropdownMenuGroup>
        ) : (
          groups.map(({ label, locations }) => (
            <DropdownMenuGroup key={label}>
              <DropdownMenuLabel>{label}</DropdownMenuLabel>
              {locations.map((location) => {
                const { profileKind, profileName, profileRef, lookupKey } =
                  location;
                const { label, icon: Icon } = storageProfiles[profileKind];
                const name =
                  (lookupKey && gamertags[lookupKey]) ||
                  profileName ||
                  (profileRef && !profileRef.startsWith("…") ? profileRef : "");
                return (
                  <DropdownMenuSub
                    key={
                      location.worlds ||
                      location.resourcePacks ||
                      location.behaviorPacks
                    }
                  >
                    <DropdownMenuSubTrigger
                      className="sidebar-storage-profile"
                      title={name || undefined}
                    >
                      <Icon />
                      <span className="sidebar-storage-profile-label">
                        {name ? (
                          <span translate="no">{name}</span>
                        ) : (
                          <span>{label}</span>
                        )}
                      </span>
                      {profileRef && profileRef !== name && (
                        <span className="sidebar-storage-profile-ref">
                          {profileRef}
                        </span>
                      )}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="sidebar-folder-menu">
                      {minecraftItems(location)}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>
                );
              })}
            </DropdownMenuGroup>
          ))
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
