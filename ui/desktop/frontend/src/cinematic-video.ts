export async function createCinematicVideoEncoder(
  kind: "mp4" | "webm" | "mov" | "mkv",
  width: number,
  height: number,
  fps: number,
  withAudio = false,
  bitrate = 12000000,
) {
  const {
    Output,
    Mp4OutputFormat,
    WebMOutputFormat,
    MovOutputFormat,
    MkvOutputFormat,
    BufferTarget,
    VideoSampleSource,
    AudioSampleSource,
    VideoSample,
    AudioSample,
    Quality,
    canEncodeVideo,
    canEncodeAudio,
  } = await import("mediabunny");
  const videoCodec =
    kind === "mp4" || kind === "mov"
      ? "avc"
      : (await canEncodeVideo("vp9", { width, height }))
        ? "vp9"
        : "vp8";
  if (!(await canEncodeVideo(videoCodec, { width, height })))
    throw new Error(
      `${videoCodec === "avc" ? "H.264" : "VP9/VP8"} encoding is unavailable at this resolution.`,
    );
  const audioCodec = kind === "mp4" || kind === "mov" ? "aac" : "opus";
  if (
    withAudio &&
    !(await canEncodeAudio(audioCodec, {
      numberOfChannels: 2,
      sampleRate: 48000,
    }))
  )
    throw new Error(
      `${audioCodec.toUpperCase()} audio encoding is unavailable in this WebView. Use ${kind === "mp4" ? "WebM" : "MP4"} or mute the audio tracks.`,
    );
  const target = new BufferTarget();
  target.on("write", ({ end }) => {
    if (end > 190 * 1024 * 1024)
      throw new Error(
        "Video export exceeds 190 MiB. Reduce the resolution or duration.",
      );
  });
  const output = new Output({
    format:
      kind === "mp4"
        ? new Mp4OutputFormat({ fastStart: false })
        : kind === "mov"
          ? new MovOutputFormat({ fastStart: false })
          : kind === "mkv"
            ? new MkvOutputFormat()
            : new WebMOutputFormat(),
    target,
  });
  const video = new VideoSampleSource({
    codec: videoCodec,
    quality: new Quality({ bitrate }),
  });
  const audio = withAudio
    ? new AudioSampleSource({
        codec: audioCodec,
        quality: new Quality({ bitrate: 192000 }),
      })
    : undefined;
  output.addVideoTrack(video, { frameRate: fps });
  if (audio)
    output.addAudioTrack(audio, { name: "Cinematic Mix", languageCode: "and" });
  await output.start();
  return {
    add: async (frame: VideoFrame) => {
      const sample = new VideoSample(frame.clone());
      try {
        await video.add(sample);
      } finally {
        sample.close();
      }
    },
    addAudio: async (channels: Float32Array[], startFrame: number) => {
      if (!audio || !channels[0].length) return;
      const data = new Float32Array(channels[0].length * 2);
      data.set(channels[0]);
      data.set(channels[1], channels[0].length);
      const sample = new AudioSample({
        data,
        format: "f32-planar",
        numberOfChannels: 2,
        sampleRate: 48000,
        timestamp: startFrame / 48000,
      });
      try {
        await audio.add(sample);
      } finally {
        sample.close();
      }
    },
    finish: async () => {
      await output.finalize();
      return new Blob([target.buffer!], {
        type:
          kind === "mp4"
            ? "video/mp4"
            : kind === "mov"
              ? "video/quicktime"
              : kind === "mkv"
                ? "video/x-matroska"
                : "video/webm",
      });
    },
    close: async () => {
      if (output.state !== "finalized" && output.state !== "canceled")
        await output.cancel();
    },
  };
}
