import { expect, it } from "vitest";
import { parseStudioDocument } from "./studio-document";
import { applyRenderPreset, renderDefaults } from "./render-settings";
import { mapDefaults } from "./map-presentation";
import { createSequence } from "./cinematic";
it("roundtrips scene effects, map markers, camera pose and sequence position", () => {
  const settings = applyRenderPreset(renderDefaults, "cloudGolden");
  const source = {
    format: "craftbox-studio",
    version: 1,
    name: "Insel",
    editor: "cinematic",
    settings,
    mapView: {
      ...mapDefaults,
      enabled: true,
      markers: [
        {
          id: "spawn",
          x: -150,
          z: 32,
          label: "Hub Spawn",
          icon: "★",
          color: "#69b84d",
          background: "#202020",
          scale: 1.2,
        },
      ],
    },
    view: {
      position: [1, 2, 3],
      target: [0, 0, 0],
      fov: 45,
      roll: 0,
      exposure: 1,
      zoom: 2,
    },
    cinematic: {
      sequence: createSequence(),
      time: 3,
      selected: "start",
      preview: true,
      showPath: false,
      zoom: 1,
      snap: true,
      guides: true,
      loop: false,
      speed: 1,
      navigation: "orbit",
      navigationSpeed: 1,
    },
  };
  const result = parseStudioDocument(JSON.stringify(source));
  expect(result.settings).toEqual(settings);
  expect(result.mapView.markers[0].label).toBe("Hub Spawn");
  expect(result.view?.zoom).toBe(2);
  expect(result.cinematic?.time).toBe(3);
  expect(result.cinematic?.sequence.keys.length).toBe(3);
  expect(() =>
    parseStudioDocument(
      JSON.stringify({
        ...source,
        mapView: {
          ...source.mapView,
          markers: [{ ...source.mapView.markers[0], x: 1e100 }],
        },
      }),
    ),
  ).toThrow(/marker/i);
});
