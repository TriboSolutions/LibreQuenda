// /llms-full.txt: everything the landing says, in English plain text, for language models.
// Built from the same texts as the pages (i18n.ts), so it cannot drift from them.
import type { APIRoute } from "astro";
import { DICT } from "../i18n";
import { INSTALL, REPO } from "../site";

const t = DICT.en;
const list = (items: string[]) => items.map((i) => `- ${i}`).join("\n");

export const GET: APIRoute = () => {
  const i = t.install;
  const body = `# LibreQuenda

> ${t.meta.description}

${t.hero.lead}

${t.footer.tagline} ${t.footer.badge}. ${t.footer.contest} ${t.footer.contestName} (AGASOL, 2026), by Tribo Solutions.

Website: https://librequenda.app (Galician; also /es/, /ca/, /en/, /pt/)
Source code: ${REPO}
Summary for language models: https://librequenda.app/llms.txt

## At a glance

- 4 ${t.facts[0]}
- 5 ${t.facts[1]}
- 2 ${t.facts[2]}
- 0 € ${t.facts[3]}

## ${t.residents.eyebrow}: ${t.residents.title}

${list(t.residents.items.map((x) => `${x.title}: ${x.text}`))}

## ${t.door.eyebrow}: ${t.door.title}

${t.door.lead}

${list(t.door.ways)}

## ${t.council.eyebrow}: ${t.council.title}

Admin sections:

${list(t.council.tabs)}

### ${t.council.askEyebrow}: ${t.council.askTitle}

${t.council.askNote} Example questions:

${list(t.council.questions)}

### ${t.council.mcpTitle}

${t.council.mcpText} Each installation serves an MCP server at https://<installation domain>/api/mcp (Streamable HTTP). An agent can find facilities and free slots, and book, cancel or join public games for the person it helps, under the same rules as the web. The person signs in on the installation's own page with a code; the agent never sees a password.

### ${t.council.widgetTitle}

${t.council.widgetText}

### ${t.council.a11yTitle}

${t.council.a11yText}

### ${t.council.langTitle}

${t.council.langText}

## ${t.app.title}

${t.app.text} ${t.app.iosText}

## ${t.trust.stayTitle}

${t.trust.stayText}

${t.trust.cols.map((c) => `### ${c.t}\n\n${list(c.items)}`).join("\n\n")}

## ${t.free.eyebrow}: ${t.free.title}

## ${i.title}

${i.h1} ${i.lead}

### ${i.needsTitle}

${list(i.needs.map((n) => `${n.t}: ${n.d}`))}

### ${i.stepsTitle}

${i.steps.map((s, n) => `${n + 1}. ${s.t}: ${s.d}`).join("\n")}

\`\`\`sh
${INSTALL}
\`\`\`

${i.warnStrong} ${i.warn}

### ${i.update}

\`\`\`sh
docker compose pull && docker compose up -d --wait
\`\`\`

### ${i.appTitle}

${i.appText}

${i.docNote} ${i.fullGuide}: ${REPO}/blob/main/docs/self-hosting.md
`;
  return new Response(body, { headers: { "Content-Type": "text/plain; charset=utf-8" } });
};
