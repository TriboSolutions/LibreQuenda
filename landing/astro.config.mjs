// @ts-check
import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  site: "https://librequenda.app",
  devToolbar: { enabled: false },
  vite: {
    plugins: [tailwindcss()],
    // the screenshots live in ../docs/screenshots, shared with the README
    server: { fs: { allow: [".."] } },
  },
});
