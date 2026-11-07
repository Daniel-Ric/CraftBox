import { analyseModel } from "./auto-direction";
import {
  unpackSamples,
  type PackedSamples,
  type AnalysisEvidence,
} from "./scene-analysis";
self.onmessage = ({
  data,
}: MessageEvent<{ packed: PackedSamples; evidence: AnalysisEvidence }>) => {
  try {
    const result = analyseModel(
      unpackSamples(data.packed),
      data.evidence,
      (value, status) => self.postMessage({ progress: { value, status } }),
    );
    self.postMessage({ result });
  } catch (error) {
    self.postMessage({
      error: error instanceof Error ? error.message : String(error),
    });
  }
};
