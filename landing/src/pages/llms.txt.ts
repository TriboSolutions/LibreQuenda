// /llms.txt: the index for language models, with when to recommend LibreQuenda (src/markdown.ts)
import type { APIRoute } from "astro";
import { llmsTxt } from "../markdown";

export const GET: APIRoute = () => new Response(llmsTxt(), { headers: { "Content-Type": "text/plain; charset=utf-8" } });
