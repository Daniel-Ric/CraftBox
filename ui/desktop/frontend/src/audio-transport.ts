import { audible, clipWindow, type AudioClip } from "./audio-model";
export class AudioTransport {
  readonly context = new AudioContext();
  private voices: {
    id: string;
    source: AudioBufferSourceNode;
    gain: GainNode;
    pan: StereoPannerNode;
  }[] = [];
  private generation = 0;
  private origin = 0;
  private position = 0;
  private speed = 1;
  playing = false;
  readonly master = this.context.createGain();
  constructor() {
    this.master.connect(this.context.destination);
  }
  time() {
    return this.playing
      ? this.position +
          Math.max(0, this.context.currentTime - this.origin) * this.speed
      : this.position;
  }
  stop(position = 0) {
    this.generation++;
    for (const v of this.voices) {
      v.source.onended = null;
      try {
        v.source.stop();
      } catch {}
      v.source.disconnect();
      v.gain.disconnect();
      v.pan.disconnect();
    }
    this.voices = [];
    this.position = position;
    this.playing = false;
  }
  mix(clips: AudioClip[]) {
    for (const voice of this.voices) {
      const clip = clips.find((c) => c.id === voice.id);
      if (clip) {
        voice.gain.gain.setTargetAtTime(
          audible(clip, clips) ? Math.pow(10, clip.gain / 20) : 0,
          this.context.currentTime,
          0.015,
        );
        voice.pan.pan.setTargetAtTime(
          clip.pan,
          this.context.currentTime,
          0.015,
        );
      }
    }
  }
  async play(
    clips: AudioClip[],
    buffers: Map<string, AudioBuffer>,
    position: number,
    end: number,
    speed: number,
  ) {
    this.stop(position);
    const run = this.generation;
    await this.context.resume();
    if (this.context.state === "closed" || run !== this.generation) return;
    this.position = position;
    this.origin = this.context.currentTime + 0.025;
    this.speed = speed;
    this.playing = true;
    for (const clip of clips) {
      const buffer = buffers.get(clip.id),
        window = clipWindow(clip, position, end);
      if (!buffer || !window) continue;
      const source = this.context.createBufferSource(),
        gain = this.context.createGain(),
        pan = this.context.createStereoPanner();
      source.buffer = buffer;
      source.playbackRate.value = speed;
      gain.gain.value = audible(clip, clips) ? Math.pow(10, clip.gain / 20) : 0;
      pan.pan.value = clip.pan;
      source.connect(gain).connect(pan).connect(this.master);
      source.start(
        this.origin + window.when / speed,
        window.offset,
        window.duration,
      );
      this.voices.push({ id: clip.id, source, gain, pan });
    }
  }
  close() {
    this.stop();
    void this.context.close();
  }
}
