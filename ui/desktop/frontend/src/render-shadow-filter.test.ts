import { expect, it } from "vitest";
import * as THREE from "three";
import { stableShadowChunk } from "./render-shadow-filter";
it("replaces the stochastic directional PCF filter while retaining other light declarations", () => {
  const code = stableShadowChunk(THREE.ShaderChunk.shadowmap_pars_fragment);
  const directional = code.slice(
    code.indexOf("float getShadow( sampler2DShadow"),
    code.indexOf(
      "#elif defined( SHADOWMAP_TYPE_VSM )",
      code.indexOf("float getShadow( sampler2DShadow"),
    ),
  );
  expect(directional).not.toContain("interleavedGradientNoise(");
  expect(directional).toContain("dFdx");
  expect(code).toContain("struct DirectionalLightShadow");
  expect(code).toContain("SHADOWMAP_TYPE_VSM");
});
