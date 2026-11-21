import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [react({ jsxImportSource: "@/i18n" })],
  optimizeDeps: { exclude: ["@/i18n/jsx-runtime", "@/i18n/jsx-dev-runtime"] },
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  test: { environment: "jsdom", include: ["tests/**/*.test.tsx"] },
});
