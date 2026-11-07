import FeedbackFlyout from "./FeedbackFlyout";
import { lazy, Suspense, useState } from "react";
import { neutralPose } from "./asset-geometry";
import type { MarketItem } from "./marketplace-model";
import { hasMarketPreview } from "./marketplace-model";
const AssetViewport = lazy(() => import("./AssetViewport"));
export default function MarketplaceModel({item}: {item: MarketItem}) {
  const [rotate, setRotate] = useState(false), [wireframe, setWireframe] = useState(false), [error, setError] = useState("");
  if (!item.preview || !hasMarketPreview(item)) return null;
  return <div className="market-model">
    <div className="market-model-tools"><label><input type="checkbox" checked={rotate} onChange={e=>setRotate(e.target.checked)}/>Turntable</label><label><input type="checkbox" checked={wireframe} onChange={e=>setWireframe(e.target.checked)}/>Wireframe</label></div>
    <Suspense fallback={<div className="market-loading">Loading 3D viewer...</div>}>
      <AssetViewport modelOnly skin={false} geometry={item.preview.geometry} texture={item.preview.texture} pose={neutralPose} walking={false} outer showCape rotate={rotate} wireframe={wireframe} light={0.8} onError={setError}/>
    </Suspense>{error && <FeedbackFlyout title="Model preview">{error}</FeedbackFlyout>}
  </div>;
}
