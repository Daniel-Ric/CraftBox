import { createCinematicVideoEncoder } from "./cinematic-video";
export function createWebMEncoder(
  width: number,
  height: number,
  fps: number,
  audio = false,
) {
  return createCinematicVideoEncoder("webm", width, height, fps, audio);
}
