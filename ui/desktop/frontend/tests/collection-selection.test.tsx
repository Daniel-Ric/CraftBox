import { expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach } from "vitest";
import {
  collectionSelection,
  selectionRoots,
} from "../src/collection-selection";
import CollectionToolbar from "../src/CollectionToolbar";
afterEach(cleanup);
const ids = ["a", "b", "c", "d"];
it("does not delete a descendant twice when an ancestor capture is selected", () => {
  const all = [
    { id: "root" },
    { id: "child", parentId: "root" },
    { id: "region", parentId: "child" },
    { id: "other" },
  ];
  expect(
    selectionRoots([all[0], all[2], all[3]], all).map((value) => value.id),
  ).toEqual(["root", "other"]);
});
it("uses single selection normally and toggles via Ctrl or explicit selection mode", () => {
  expect(collectionSelection(ids, ["a"], "a", "b", {}, false).selected).toEqual(
    ["b"],
  );
  expect(
    collectionSelection(ids, ["a"], "a", "b", { ctrlKey: true }, false),
  ).toMatchObject({ selected: ["a", "b"], multiple: true });
  expect(
    collectionSelection(ids, ["a", "b"], "a", "a", { metaKey: true }, false)
      .selected,
  ).toEqual(["b"]);
  expect(collectionSelection(ids, ["a"], "a", "c", {}, true).selected).toEqual([
    "a",
    "c",
  ]);
});
it("selects an ordered Shift range and adds ranges with Ctrl+Shift", () => {
  expect(
    collectionSelection(ids, ["a"], "a", "c", { shiftKey: true }, false)
      .selected,
  ).toEqual(["a", "b", "c"]);
  expect(
    collectionSelection(
      ids,
      ["d"],
      "a",
      "b",
      { shiftKey: true, ctrlKey: true },
      false,
    ).selected,
  ).toEqual(["d", "a", "b"]);
});
it("offers button selection and a single bulk-delete action without checkboxes", () => {
  const mode = vi.fn(),
    selected = vi.fn(),
    remove = vi.fn();
  render(
    <CollectionToolbar
      ids={ids}
      selected={["a", "b"]}
      multiple
      onMultiple={mode}
      onSelected={selected}
      onDelete={remove}
    />,
  );
  expect(screen.queryByRole("checkbox")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Select all" }));
  expect(selected).toHaveBeenCalledWith(ids);
  fireEvent.click(screen.getByRole("button", { name: "Delete selected (2)" }));
  expect(remove).toHaveBeenCalledWith(["a", "b"]);
  expect(
    screen.getAllByRole("button", { name: /Delete selected/ }),
  ).toHaveLength(1);
});
