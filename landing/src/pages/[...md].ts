// /index.md, /en/index.md, /instalar/index.md…: each page as Markdown (see src/markdown.ts)
import type { APIRoute } from "astro";
import { mdRoutes, pageMd, type MdPage } from "../markdown";
import type { Lang } from "../i18n";

export function getStaticPaths() {
  return mdRoutes().map(({ md, lang, page }) => ({ params: { md }, props: { lang, page } }));
}

export const GET: APIRoute = ({ props }) =>
  new Response(pageMd(props.lang as Lang, props.page as MdPage), {
    headers: { "Content-Type": "text/markdown; charset=utf-8" },
  });
