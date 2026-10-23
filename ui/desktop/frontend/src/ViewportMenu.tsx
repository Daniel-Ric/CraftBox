import { ChevronDown, View, Check, LockKeyhole } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { perspectives } from "./viewport-views";
export default function ViewportMenu({
  disabled,
  locked,
  onLock,
  onFit,
  onView,
}: {
  disabled: boolean;
  locked: boolean;
  onLock: () => void;
  onFit: () => void;
  onView: (angle: number, tilt: number) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        disabled={disabled}
        className="studio-button viewport-menu-trigger"
      >
        <View size={14} />
        View
        <ChevronDown size={12} />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="application-menu-popup">
        <DropdownMenuItem onClick={onFit}>Fit world to view</DropdownMenuItem>
        <DropdownMenuSeparator />
        {perspectives.map((v) => (
          <DropdownMenuItem key={v.id} onClick={() => onView(v.angle, v.tilt)}>
            {v.label}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onLock}>
          {locked ? <Check size={14} /> : <LockKeyhole size={14} />}Lock tilt
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
