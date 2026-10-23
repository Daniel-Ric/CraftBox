type Completion = {
  remaining: number;
  resolve: () => void;
  reject: (error: Error) => void;
};
export class FrameQueue<T> {
  private tasks: { item: T; completion: Completion }[] = [];
  private commit: (item: T) => void;
  private clock: () => number;
  private budget: number;
  private maxItems: number;
  constructor(
    commit: (item: T) => void,
    clock = () => performance.now(),
    budget = 4,
    maxItems = 2,
  ) {
    this.commit = commit;
    this.clock = clock;
    this.budget = budget;
    this.maxItems = maxItems;
  }
  append(items: T[]): Promise<void> {
    if (!items.length) return Promise.resolve();
    return new Promise((resolve, reject) => {
      const completion = { remaining: items.length, resolve, reject };
      for (const item of items) this.tasks.push({ item, completion });
    });
  }
  flush(): number {
    const start = this.clock();
    let count = 0;
    while (
      this.tasks.length &&
      count < this.maxItems &&
      (count === 0 || this.clock() - start < this.budget)
    ) {
      const task = this.tasks.shift()!;
      try {
        this.commit(task.item);
        task.completion.remaining--;
        if (task.completion.remaining === 0) task.completion.resolve();
      } catch (error) {
        task.completion.reject(
          error instanceof Error ? error : new Error(String(error)),
        );
        this.tasks = this.tasks.filter(
          (item) => item.completion !== task.completion,
        );
      }
      count++;
    }
    return count;
  }
  cancel() {
    const completions = new Set(this.tasks.map((task) => task.completion));
    this.tasks = [];
    for (const completion of completions)
      completion.reject(new Error("3D view closed"));
  }
}
