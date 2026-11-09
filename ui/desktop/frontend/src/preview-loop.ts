type Options = {
  draw: (now: number) => boolean | void;
  visible: () => boolean;
  continuous: () => boolean;
  request?: (callback: FrameRequestCallback) => number;
  cancel?: (id: number) => void;
};
export function createPreviewLoop(options: Options) {
  const request = options.request ?? requestAnimationFrame,
    cancel = options.cancel ?? cancelAnimationFrame;
  let frame: number | undefined,
    disposed = false;
  const schedule = () => {
    if (!disposed && frame === undefined && options.visible())
      frame = request(draw);
  };
  function draw(now: number) {
    frame = undefined;
    if (disposed || !options.visible()) return;
    const changed = options.draw(now);
    if (changed || options.continuous()) schedule();
  }
  return {
    invalidate: schedule,
    dispose() {
      disposed = true;
      if (frame !== undefined) cancel(frame);
      frame = undefined;
    },
  };
}
