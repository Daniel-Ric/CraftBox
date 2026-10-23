
export const authoredCloudField = `
uniform int aLayerCount;
uniform mat4 aLayerTransform[8];
uniform vec3 aLayerMin[8],aLayerMax[8];
uniform vec4 aLayerParameters[8],aLayerForm[8],aLayerStyle[8],aLayerTint[8],aLayerScale[8];
uniform vec4 aOpening,aOpeningShape,aScene;
uniform vec3 cloudBaseColor;
float cloudMask(vec3 p);
float openingMask(vec3 p){
  if(aOpening.w<=0.)return 1.;
  vec3 local=(p-aScene.xyz)/aScene.w;
  vec2 q=(local.xz-aOpening.xy)/aOpeningShape.xy;
  float angle=atan(q.y,q.x);
  float warp=1.+aOpeningShape.w*(sin(angle*3.+.7)*.12+sin(angle*7.+1.9)*.07);
  float inside=1.-smoothstep(1.-aOpeningShape.z,1.+aOpeningShape.z,length(q)/warp);
  float above=smoothstep(aOpening.z-aOpeningShape.z,aOpening.z+aOpeningShape.z,local.y);
  return 1.-aOpening.w*inside*above;
}
float layerDensity(int i,vec3 p){
  vec3 q=(aLayerTransform[i]*vec4(p,1.)).xyz;
  if(any(greaterThan(abs(q),vec3(1.))))return 0.;
  vec4 params=aLayerParameters[i],form=aLayerForm[i],style=aLayerStyle[i],scale=aLayerScale[i];
  if(params.x<=0.||params.y<=0.||params.z>=1.)return 0.;
  vec3 n=q*vec3(2.8,1.7,2.8)+vec3(form.w*.013,form.w*.021,form.w*.031)-vec3(scale.y,0.,scale.z);
  vec3 warp=(textureLod(aNoise,n*.31,cloudLod).rgb-.5)*form.z*.32;
  vec3 v=q+warp;
  float base=textureLod(aNoise,n*.45,cloudLod).r;
  float billow=textureLod(aNoise,n*.88+vec3(.21,.39,.67),cloudLod).g;
  float detail=textureLod(aNoise,n*mix(1.5,4.,form.x),cloudLod).b;
  float shape;
  if(params.w<.5){
    float threshold=mix(.76,.12,params.y);
    shape=smoothstep(threshold-.08,threshold+.17,base*.55+billow*.45);
    float weather=textureLod(aNoise,vec3(n.x*.32,form.w*.017,n.z*.32),cloudLod).g;
    float bottom=-.65+weather*.15,top=.08+weather*.8;
    shape*=smoothstep(bottom,bottom+.24,v.y)*(1.-smoothstep(top-.28,top,v.y));
    float edge=length(q.xz)+(weather-.5)*.18;
    shape*=1.-smoothstep(.65,1.,edge);
  }else{
    float core=length(v/vec3(.88,.82,.85));
    float crown=length((v-vec3(.35,.25,.03))/vec3(.57,.64,.65));
    float shoulder=length((v+vec3(.32,.03,.23))/vec3(.65,.56,.6));
    float surface=min(core,min(crown,shoulder))+(billow-.5)*.28+(base-.5)*.2;
    float feather=mix(.06,.36,style.x);
    shape=1.-smoothstep(.7-feather,.98,surface+(1.-params.y)*.45);
  }
  shape=max(0.,shape-(1.-detail)*form.y*.38);
  return shape*params.x*(1.-params.z)*scale.x;
}
vec4 authoredMedium(vec3 p,bool groundShadow){
  vec4 result=vec4(0.);
  for(int i=0;i<8;i++){
    if(i>=aLayerCount)break;
    float d=layerDensity(i,p)*(groundShadow?aLayerStyle[i].y:1.);
    result+=vec4(aLayerTint[i].rgb*d,d);
  }
  return result*cloudMask(p);
}
`;
