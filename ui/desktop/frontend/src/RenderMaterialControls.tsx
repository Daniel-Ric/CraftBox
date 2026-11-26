import { useState } from "react";
import StudioInput from "./StudioInput";
import {
  blockMaterialLimit,
  blockName,
  type BlockMaterial,
} from "./render-block-materials";
import type { RenderSettings } from "./render-settings";

export default function RenderMaterialControls({
  settings,
  onChange,
  disabled,
}: {
  settings: RenderSettings;
  onChange: (patch: Partial<RenderSettings>) => void;
  disabled: boolean;
}) {
  const [block, setBlock] = useState("");
  const [selected, setSelected] = useState("");
  const materials = settings.blockMaterials ?? [];
  const active = materials.find((material) => material.block === selected);
  const id = block.includes(":") ? block : `minecraft:${block}`;
  const change = (patch: Partial<BlockMaterial>) =>
    onChange({
      blockMaterials: materials.map((material) =>
        material === active ? { ...material, ...patch } : material,
      ),
    });
  return (
    <fieldset data-inspector="detail" disabled={disabled}>
      <legend>Block materials</legend>
      <label className="render-check" data-render-search="clay">
        <StudioInput
          type="checkbox"
          checked={settings.clay}
          onChange={(event) => onChange({ clay: event.target.checked })}
        />{" "}
        Clay preview
      </label>
      {settings.clay && (
        <label className="render-color">
          Clay color
          <StudioInput
            type="color"
            value={settings.clayColor}
            onChange={(event) => onChange({ clayColor: event.target.value })}
          />
        </label>
      )}
      <label className="render-control" data-render-search="block-material">
        Block material override
        <StudioInput
          value={block}
          maxLength={110}
          placeholder="minecraft:gold_block"
          onChange={(event) =>
            setBlock(event.target.value.trim().toLowerCase())
          }
        />
      </label>
      <button
        type="button"
        className="inspector-next"
        disabled={
          !blockName.test(id) ||
          materials.length >= blockMaterialLimit ||
          materials.some((item) => item.block === id)
        }
        onClick={() => {
          onChange({
            blockMaterials: [
              ...materials,
              {
                block: id,
                color: "#ffffff",
                tint: false,
                roughness: 1,
                metalness: 0,
                emission: 0,
              },
            ],
          });
          setSelected(id);
          setBlock("");
        }}
      >
        Add block material
      </button>
      {!!materials.length && (
        <label className="render-control">
          Edit block material
          <select
            value={selected}
            onChange={(event) => setSelected(event.target.value)}
          >
            <option value="">Select block material</option>
            {materials.map((item) => (
              <option key={item.block} value={item.block} translate="no">
                {item.block}
              </option>
            ))}
          </select>
        </label>
      )}
      {active && (
        <>
          <label className="render-check">
            <StudioInput
              type="checkbox"
              checked={active.tint}
              onChange={(event) => change({ tint: event.target.checked })}
            />{" "}
            Tint block texture
          </label>
          {active.tint && (
            <label className="render-color">
              Block tint{" "}
              <StudioInput
                type="color"
                value={active.color}
                onChange={(event) => change({ color: event.target.value })}
              />
            </label>
          )}
          {(
            [
              ["Roughness", "roughness", 1],
              ["Metalness", "metalness", 1],
              ["Emission strength", "emission", 25],
            ] as const
          ).map(([label, key, max]) => (
            <label key={key} className="render-control">
              {label}
              <StudioInput
                type="number"
                min={0}
                max={max}
                increment={0.05}
                value={active[key]}
                onChange={(event) => {
                  const value = Number(event.target.value);
                  if (event.target.value && Number.isFinite(value))
                    change({ [key]: Math.min(max, Math.max(0, value)) });
                }}
              />
            </label>
          ))}
          <button
            type="button"
            onClick={() => {
              onChange({
                blockMaterials: materials.filter((item) => item !== active),
              });
              setSelected("");
            }}
          >
            Remove block material
          </button>
        </>
      )}
    </fieldset>
  );
}
