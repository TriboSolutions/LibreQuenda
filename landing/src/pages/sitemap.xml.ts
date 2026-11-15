// /sitemap.xml: every page in every language, each with its alternates (hreflang) and the
// date of the last commit that changed it (src/lastmod.ts); without a date, no <lastmod>
import type { APIRoute } from "astro";
import { LANGS, DEFAULT_LANG, href } from "../i18n";
import { SITE } from "../site";
import { lastmod } from "../lastmod";

const PAGES = [
  { path: "/", priority: "1.0" },
  { path: "/instalar", priority: "0.8" },
  { path: "/privacidade", priority: "0.3" },
];

export const GET: APIRoute = ({ site }) => {
  // the pages are folders (/en/instalar/), so their URLs end in a slash
  const abs = (p: string) => new URL(p.endsWith("/") ? p : `${p}/`, site ?? SITE).href;
  const urls = PAGES.flatMap(({ path, priority }) =>
    LANGS.map((lang) => {
      const date = lastmod(path);
      const alternates = [
        ...LANGS.map((l) => `    <xhtml:link rel="alternate" hreflang="${l}" href="${abs(href(l, path))}"/>`),
        `    <xhtml:link rel="alternate" hreflang="x-default" href="${abs(href(DEFAULT_LANG, path))}"/>`,
      ].join("\n");
      return `  <url>
    <loc>${abs(href(lang, path))}</loc>
${date ? `    <lastmod>${date}</lastmod>\n` : ""}    <changefreq>weekly</changefreq>
    <priority>${priority}</priority>
${alternates}
  </url>`;
    }),
  );
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
${urls.join("\n")}
</urlset>
`;
  return new Response(xml, { headers: { "Content-Type": "application/xml; charset=utf-8" } });
};
