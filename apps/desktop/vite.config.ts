import path from "path";
import { fileURLToPath } from "url";

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// `new URL(import.meta.url).pathname` returns `/C:/...` on Windows, breaking
// every path.resolve below. fileURLToPath is the cross-platform form.
const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
      // Explicit alias — pnpm workspace symlinks usually resolve, but this
      // belt-and-suspenders avoids surprises in CI sandboxed environments.
      "@repo/configs": path.resolve(__dirname, "..", "..", "packages", "configs", "src"),
      "react-native": "react-native-web",
      "react-native-svg": "@tamagui/react-native-svg",
    },
  },
  define: {
    __DEV__: JSON.stringify(false),
    "process.env.NODE_ENV": JSON.stringify("production"),
    "process.env": JSON.stringify({}),
  },
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
  },
  build: {
    outDir: "build",
    target: "esnext",
  },
});
