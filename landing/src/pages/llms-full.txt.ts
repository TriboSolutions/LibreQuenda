// /llms-full.txt: everything the landing says, in English, in one file (src/markdown.ts)
import type { APIRoute } from "astro";
import { llmsFullTxt } from "../markdown";

export const GET: APIRoute = () => new Response(llmsFullTxt(), { headers: { "Content-Type": "text/plain; charset=utf-8" } });
