import { defineConfig, searchForWorkspaceRoot } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath, URL } from "node:url";

const localizationRuntime = ["@/i18n/jsx-runtime", "@/i18n/jsx-dev-runtime"];

export default defineConfig({
  plugins: [
    react({ jsxImportSource: "@/i18n" }),
    tailwindcss(),
    {
      name: "craftbox-localization-runtime",
      configResolved(config) {
        for (const options of [config.optimizeDeps, ...Object.values(config.environments).map((environment) => environment.optimizeDeps)]) {
          options.include = options.include?.filter((module) => !localizationRuntime.includes(module));
        }
      },
    },
  ],
  optimizeDeps: {
    exclude: localizationRuntime,
    include: ["react/jsx-runtime", "react/jsx-dev-runtime"],
  },
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  server: {
    host: "127.0.0.1", port: 5173, strictPort: true,
    fs: { allow: [
      searchForWorkspaceRoot(fileURLToPath(new URL(".", import.meta.url))),
      fileURLToPath(new URL("../../../locale/messages", import.meta.url)),
    ] },
  },
});
