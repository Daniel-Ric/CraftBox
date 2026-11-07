import { mergeConfig } from "vite";
import base from "../vite.config.ts";

                                                               
export default mergeConfig(base, {
  server: { host: "127.0.0.1", port: 5204, strictPort: true, hmr: false },
});
