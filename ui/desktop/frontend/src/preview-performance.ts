                                                                                
                                                                             
export class PreviewPerformance {
  private scale = 0.85;
  private frames = 0;
  private average = 16;
  private moving = false;

  update(
    moving: boolean,
    width: number,
    height: number,
    deviceRatio: number,
    frameMs: number,
  ) {
    const native = Math.min(2, Math.max(0.25, deviceRatio || 1));
    if (!moving) {
      this.moving = false;
      this.frames = 0;
	  return Math.min(native,1.5,Math.sqrt(3_200_000/Math.max(1,width*height)));
    }
    if (!this.moving) {
      this.average = 16;
      this.frames = 0;
      this.moving = true;
    }
    if (Number.isFinite(frameMs) && frameMs > 0) {
      this.average += (Math.min(100, frameMs) - this.average) * 0.15;
      this.frames++;
                                                                         
      if (this.frames >= 12 && this.average > 24) {
        this.scale = Math.max(0.65, this.scale - 0.05);
        this.frames = 0;
      } else if (this.frames >= 45 && this.average < 18) {
        this.scale = Math.min(0.85, this.scale + 0.05);
        this.frames = 0;
      }
    }
    const ratio = Math.min(native, 1) * this.scale;
    const pixelLimit = Math.sqrt(1_800_000 / Math.max(1, width * height));
	return Math.max(0.1,Math.min(ratio,pixelLimit));
  }
}
