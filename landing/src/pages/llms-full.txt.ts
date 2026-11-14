// /llms-full.txt: everything the landing says, in English plain text, for language models.
// The same Markdown as the pages' .md versions (src/markdown.ts), so it cannot drift from them.
import type { APIRoute } from "astro";
import { homeMd, installMd } from "../markdown";
import { REPO } from "../site";

export const GET: APIRoute = () => {
  // the home page's title and summary, then where everything lives, then the rest
  const [title, summary, ...rest] = homeMd("en").split("\n\n");
  const where = `Website: https://librequenda.app (Galician; also /es/, /ca/, /en/, /pt/)
Source code: ${REPO}
Summary for language models: https://librequenda.app/llms.txt
Each page also has a Markdown version at <page>/index.md, e.g. https://librequenda.app/en/instalar/index.md`;
  const body = [title, summary, where, ...rest].join("\n\n") + "\n" + installMd("en", 2);
  return new Response(body, { headers: { "Content-Type": "text/plain; charset=utf-8" } });
};
