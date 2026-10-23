import { expect, it } from "vitest";
import { analyseModel, directModel } from "./auto-direction";
import {
  rememberDirectionExample,
  loadDirectionExamples,
} from "./direction-learning";
it("learns a preferred scene locally and ignores corrupt samples", () => {
  localStorage.clear();
  const model = analyseModel(
    [
      {
        position: [0, 0, 0],
        normal: [0, 1, 0],
        color: [0.2, 0.6, 0.3],
        area: 1,
        mode: "mask",
      },
      {
        position: [100, 70, 100],
        normal: [1, 0, 0],
        color: [0.3, 0.65, 0.25],
        area: 1,
        mode: "mask",
      },
    ],
    { lights: 3, triangles: 1000 },
  );
  rememberDirectionExample(model, "cityNeon");
  expect(loadDirectionExamples()).toHaveLength(1);
  expect(directModel(model, { intent: "auto" }).reference.id).toBe("cityNeon");
  rememberDirectionExample(model, "cloudGolden");
  expect(loadDirectionExamples()).toHaveLength(1);
  expect(directModel(model, { intent: "auto" }).reference.id).toBe(
    "cloudGolden",
  );
  localStorage.setItem(
    "craftbox-direction-examples",
    JSON.stringify([{ lookId: "bad", features: [NaN] }]),
  );
  expect(loadDirectionExamples()).toHaveLength(0);
  localStorage.clear();
});
