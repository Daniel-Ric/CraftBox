import * as THREE from "three";

export function nearShadowBounds(
  camera: THREE.OrthographicCamera,
  world: THREE.Box3,
  focus: THREE.Vector3,
  radius: number,
  resolution: number,
) {
  if (world.isEmpty()) return;
  camera.updateMatrixWorld(true);
  const projected = world.clone().applyMatrix4(camera.matrixWorldInverse);
  const center = focus.clone().applyMatrix4(camera.matrixWorldInverse);
  const texel = (2 * radius) / resolution;
  const x = Math.round(center.x / texel) * texel;
  const y = Math.round(center.y / texel) * texel;
  return new THREE.Box3(
    new THREE.Vector3(x - radius, y - radius, projected.min.z - 4),
    new THREE.Vector3(x + radius, y + radius, projected.max.z + 4),
  );
}

export class NearSunShadow {
  light = new THREE.DirectionalLight(0xffffff, 0);
  private footprint = new THREE.Box3();

  constructor() {
    this.light.castShadow = true;
    this.light.visible = false;
    this.light.shadow.autoUpdate = false;
    this.light.shadow.bias = -0.00005;
  }

  update(
    sun: THREE.DirectionalLight,
    view: THREE.PerspectiveCamera | THREE.OrthographicCamera,
    target: THREE.Vector3,
    world: THREE.Box3,
    resolution: number,
    moving: boolean,
    dirty: boolean,
  ) {
    const far = sun.shadow.camera;
    const span = Math.max(far.right - far.left, far.top - far.bottom);
    const radius =
      view instanceof THREE.PerspectiveCamera
        ? THREE.MathUtils.clamp(span / 6, 24, 128)
        : Math.max(
            12,
            (Math.max(view.right - view.left, view.top - view.bottom) /
              view.zoom) *
              0.65,
          );
    const wasVisible = this.light.visible;
    const extendsResolution =
      span / sun.shadow.mapSize.x > (2.4 * radius) / resolution;
    this.light.visible =
      sun.castShadow &&
      resolution >= 1024 &&
      extendsResolution &&
      !world.isEmpty();
    if (
      !this.light.visible ||
      (moving && wasVisible && this.light.shadow.map && !dirty)
    )
      return false;
    const shadow = this.light.shadow;
    const camera = shadow.camera;
    const movedSun =
      !this.light.position.equals(sun.position) ||
      !this.light.target.position.equals(sun.target.position);
    this.light.position.copy(sun.position);
    this.light.target.position.copy(sun.target.position);
    camera.position.copy(sun.position);
    camera.quaternion.copy(far.quaternion);
    const focus =
      view instanceof THREE.PerspectiveCamera
        ? view.getWorldDirection(new THREE.Vector3())
            .multiplyScalar(radius * 0.7)
            .add(view.position)
        : target;
    const bounds = nearShadowBounds(camera, world, focus, radius, resolution)!;
    if (shadow.mapSize.x !== resolution) {
      shadow.mapSize.setScalar(resolution);
      shadow.map?.dispose();
      shadow.map = null;
    }
    if (
      wasVisible &&
      !dirty &&
      !movedSun &&
      shadow.map &&
      bounds.equals(this.footprint)
    )
      return false;
    this.footprint.copy(bounds);
    camera.left = bounds.min.x;
    camera.right = bounds.max.x;
    camera.bottom = bounds.min.y;
    camera.top = bounds.max.y;
    camera.near = Math.max(0.1, -bounds.max.z);
    camera.far = Math.max(camera.near + 1, -bounds.min.z);
    camera.updateProjectionMatrix();
    shadow.normalBias = Math.max(0.005, (radius * 0.15) / resolution);
    shadow.radius = sun.shadow.radius;
    shadow.needsUpdate = true;
    return true;
  }

  dispose() {
    this.light.shadow.dispose();
  }
}

export function nearSunShadowChunk(source: string) {
  return source.replace(
    /directLight\.color \*= ([^;\n]+getShadow\( directionalShadowMap[^;\n]+);/,
    (_, directionalVisibility: string) => `
    {
    float sunVisibility = ${directionalVisibility};
    #if UNROLLED_LOOP_INDEX == 0 && NUM_DIR_LIGHT_SHADOWS > 1 && defined(SHADOWMAP_TYPE_PCF)
      vec3 nearCoord = vDirectionalShadowCoord[1].xyz / vDirectionalShadowCoord[1].w;
      vec2 nearEdge = min(nearCoord.xy, 1.0 - nearCoord.xy);
      float nearWeight = smoothstep(0.02, 0.15, min(nearEdge.x, nearEdge.y));
      DirectionalLightShadow nearShadow = directionalLightShadows[1];
      float visibility = getShadow(directionalShadowMap[1], nearShadow.shadowMapSize, nearShadow.shadowIntensity, nearShadow.shadowBias, nearShadow.shadowRadius, vDirectionalShadowCoord[1]);
      if (receiveShadow && nearCoord.z >= 0.0 && nearCoord.z <= 1.0) {
        sunVisibility = mix(sunVisibility, visibility, nearWeight);
      }
    #endif
    directLight.color *= sunVisibility;
    }`,
  );
}

export function stableShadowChunk(source: string) {
  const start = source.indexOf("float getShadow( sampler2DShadow");
  const end = source.indexOf("#elif defined( SHADOWMAP_TYPE_VSM )", start);
  if (start < 0 || end < 0)
    throw new Error("Unknown Three.js shadow structure");
  const filter = `float getShadow( sampler2DShadow shadowMap, vec2 shadowMapSize, float shadowIntensity, float shadowBias, float shadowRadius, vec4 shadowCoord ) {
  shadowCoord.xyz /= shadowCoord.w;
  shadowCoord.z += shadowBias;
  vec3 dx=dFdx(shadowCoord.xyz),dy=dFdy(shadowCoord.xyz);
  float determinant=dx.x*dy.y-dx.y*dy.x;
  vec2 plane=vec2(0.0);
  if(abs(determinant)>1e-10)plane=vec2(dy.y*dx.z-dx.y*dy.z,dx.x*dy.z-dy.x*dx.z)/determinant;
  float shadow=1.0;
  if(shadowCoord.x>=0.0&&shadowCoord.x<=1.0&&shadowCoord.y>=0.0&&shadowCoord.y<=1.0&&shadowCoord.z<=1.0){
   shadow=0.0;
   for(int y=-1;y<=1;y++)for(int x=-1;x<=1;x++){
    vec2 offset=vec2(float(x),float(y))/shadowMapSize*shadowRadius*0.65;
    float weight=(x==0?2.0:1.0)*(y==0?2.0:1.0);
    shadow+=texture(shadowMap,vec3(shadowCoord.xy+offset,shadowCoord.z+dot(plane,offset)))*weight;
   }
   shadow/=16.0;
  }
  return mix(1.0,shadow,shadowIntensity);
 }
 `;
  return source.slice(0, start) + filter + source.slice(end);
}
