// /sitemap-index.xml: the sitemap index robots.txt points to. One sitemap, /sitemap.xml, with
// the date of the newest change on the site (src/lastmod.ts).
import type { APIRoute } from "astro";
import { SITE } from "../site";
import { NEWEST } from "../lastmod";

export const GET: APIRoute = ({ site }) => {
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>${new URL("/sitemap.xml", site ?? SITE).href}</loc>
${NEWEST ? `    <lastmod>${NEWEST}</lastmod>\n` : ""}  </sitemap>
</sitemapindex>
`;
  return new Response(xml, { headers: { "Content-Type": "application/xml; charset=utf-8" } });
};
