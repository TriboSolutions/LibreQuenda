// Writes src/lastmod.json: for each page, the date of the last commit that touched the files
// its content comes from. The sitemap's <lastmod> and the pages' dateModified are then real
// (Google ignores dates that change on every build), and `pnpm indexnow` only sends what changed.
//
// It needs git history: CI runs it before building the image (full checkout), and `pnpm build`
// runs it locally. Inside Docker there is no .git: the committed file is left as it is.
import { execFileSync } from "node:child_process";
import { existsSync, writeFileSync } from "node:fs";

const OUT = "src/lastmod.json";

const date = (files) => {
  try {
    return execFileSync("git", ["log", "-1", "--format=%cs", "--", ...files], { encoding: "utf8" }).trim() || null;
  } catch {
    return null;
  }
};

if (date(["package.json"]) === null) {
  console.log("gen-lastmod: no git history, src/lastmod.json left as it is");
  process.exit(0);
}

// page (without the language prefix) -> the files its content comes from
const map = {
  "/": ["src/components/Home.astro", "src/components/DayStrip.astro", "src/i18n.ts", "src/site.ts", "../docs/screenshots"],
  "/instalar": ["src/components/Install.astro", "src/i18n.ts", "src/site.ts"],
  "/privacidade": ["src/components/Privacy.astro", "src/privacy.ts"],
};

const out = {};
for (const [path, files] of Object.entries(map)) {
  const d = date(files.filter((f) => existsSync(f)));
  if (d) out[path] = d;
}
writeFileSync(OUT, JSON.stringify(out, null, 2) + "\n");
console.log(`gen-lastmod: ${Object.keys(out).length} pages -> ${OUT}`);
