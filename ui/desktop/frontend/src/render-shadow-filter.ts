

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
