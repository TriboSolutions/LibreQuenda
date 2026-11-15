// The date each page last changed, from git (scripts/gen-lastmod.mjs -> lastmod.json).
// A page without a date simply goes without one.
import dates from "./lastmod.json";

const LASTMOD: Record<string, string> = dates;

/** "/en/instalar/" -> the date of "/instalar" */
export function lastmod(pathname: string): string | undefined {
  const page = pathname.replace(/^\/(es|ca|en|pt)(?=\/|$)/, "").replace(/\/+$/, "") || "/";
  return LASTMOD[page];
}

/** The newest date of all, for the sitemap index */
export const NEWEST = Object.values(LASTMOD).sort().pop();
