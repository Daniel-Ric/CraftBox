import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { it, expect, vi } from "vitest";
import AssetLibrary from "./AssetLibrary";
import type { World } from "./bridge";
it("filters worlds and opens the selected world in the cinematic workspace", async () => {
  const open = vi.fn(),
    user = userEvent.setup();
  const worlds = [
    { id: "a", name: "Palast", chunks: 48, entities: 12, state: "finished" },
    { id: "b", name: "Insel", chunks: 2, entities: 0, state: "finished" },
  ] as World[];
  render(
    <AssetLibrary
      page="worlds"
      worlds={worlds}
      packs={[]}
      skins={[]}
      onOpenEditor={open}
      onFolder={() => {}}
      onMap={() => {}}
      onSaveSkin={() => {}}
      onSource={() => {}}
    />,
  );
  await user.click(screen.getByRole("button", { name: /Palast/ }));
  expect(screen.getByText("48 Chunks")).toBeInTheDocument();
  await user.click(
    screen.getByRole("button", { name: "Create camera sequence" }),
  );
  expect(open).toHaveBeenCalledWith(worlds[0], "cinematic");
  await user.type(screen.getByLabelText("Filter results"), "Insel");
  expect(
    screen.queryByRole("button", { name: /Palast/ }),
  ).not.toBeInTheDocument();
});
it("selects a skin and routes its actual PNG to the save action", async () => {
  const save = vi.fn(),
    user = userEvent.setup(),
    skin = { name: "Builder", image: "data:image/png;base64,skin" };
  render(
    <AssetLibrary
      page="skins"
      worlds={[]}
      packs={[]}
      skins={[skin]}
      onOpenEditor={() => {}}
      onFolder={() => {}}
      onMap={() => {}}
      onSaveSkin={save}
      onSource={() => {}}
    />,
  );
  await user.click(screen.getByRole("button", { name: "Builder" }));
  await user.click(
    screen.getByRole("button", { name: "Save skin as PNG" }),
  );
  expect(save).toHaveBeenCalledWith(skin);
});
