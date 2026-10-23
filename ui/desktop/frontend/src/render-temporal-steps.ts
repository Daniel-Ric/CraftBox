
export async function renderTemporalSteps(
  steps: Generator<void>,
  signal?: AbortSignal,
  yieldFrame = () => new Promise<void>((resolve) => setTimeout(resolve, 0)),
) {
  signal?.throwIfAborted();
  for (const _step of steps) {
    await yieldFrame();
    signal?.throwIfAborted();
  }
}
