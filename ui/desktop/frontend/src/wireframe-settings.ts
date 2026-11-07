export type WireframeSettings = {
  mode?: "edges" | "triangles";
  hideSurface?: boolean;
  thresholdAngle?: number;
  xray?: boolean;
  color?: string;
  width?: number;
  opacity?: number;
  hiddenColor?: string;
  hiddenOpacity?: number;
  dashedHidden?: boolean;
  dashScale?: number;
  vertices?: boolean;
  vertexColor?: string;
  vertexSize?: number;
};
export type WireframeStats = {
  meshes: number;
  segments: number;
  vertices: number;
  skipped: number;
};
