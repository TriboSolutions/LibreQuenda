// Every page of the landing as Markdown, for language models and agents (GEO).
// Built from the same texts as the pages (i18n.ts, privacy.ts), so it cannot drift from them.
// Served at <page>/index.md (/index.md, /en/index.md, /en/instalar/index.md…), and by Caddy
// for the page itself when a client asks for Accept: text/markdown.
import { DICT, LANGS, href, type Lang } from "./i18n";
import { PRIVACY, PRIVACY_EMAIL } from "./privacy";
import { INSTALL, REPO, SITE, DEMO, PLAY } from "./site";

export const MD_PAGES = ["/", "/instalar", "/privacidade"] as const;
export type MdPage = (typeof MD_PAGES)[number];

/** Where the Markdown version of a page lives: "/en/instalar" -> "/en/instalar/index.md" */
export function mdPath(pagePath: string): string {
  const p = pagePath.replace(/\/+$/, "");
  return `${p}/index.md`;
}

const list = (items: readonly string[]) => items.map((i) => `- ${i}`).join("\n");
const url = (lang: Lang, path = "/") => new URL(href(lang, path).replace(/\/?$/, "/"), SITE).href;

/** The home page. `level` shifts the headings, to nest it in llms-full.txt */
export function homeMd(lang: Lang, level = 1): string {
  const t = DICT[lang];
  const h = (n: number) => "#".repeat(n + level - 1);
  return `${h(1)} LibreQuenda

> ${t.meta.description}

${t.hero.title1} ${t.hero.title2} ${t.hero.lead}

- ${t.footer.badge}: ${REPO}
- ${t.hero.demo}: ${DEMO}
- Google Play: ${PLAY}
- ${t.nav.install}: ${url(lang, "/instalar")}
- ${t.footer.contest} ${t.footer.contestName} (AGASOL, 2026) · Tribo Solutions

${list([`4 ${t.facts[0]}`, `5 ${t.facts[1]}`, `2 ${t.facts[2]}`, `0 € ${t.facts[3]}`])}

${h(2)} ${t.residents.eyebrow}: ${t.residents.title}

${list(t.residents.items.map((x) => `${x.title}: ${x.text}`))}

${h(2)} ${t.door.eyebrow}: ${t.door.title}

${t.door.lead}

${list(t.door.ways)}

${h(2)} ${t.council.eyebrow}: ${t.council.title}

${list(t.council.tabs)}

${h(3)} ${t.council.askEyebrow}: ${t.council.askTitle}

${t.council.askNote}

${list(t.council.questions)}

${h(3)} ${t.council.mcpTitle}

${t.council.mcpText} \`https://<domain>/api/mcp\` (Streamable HTTP).

${h(3)} ${t.council.widgetTitle}

${t.council.widgetText}

${h(3)} ${t.council.a11yTitle}

${t.council.a11yText}

${h(3)} ${t.council.langTitle}

${t.council.langText}

${h(2)} ${t.app.title}

${t.app.text} ${t.app.iosText}

${h(2)} ${t.free.eyebrow}: ${t.free.title}

${t.trust.stayTitle} ${t.trust.stayText}

${t.trust.cols.map((c) => `${h(3)} ${c.t}\n\n${list(c.items)}`).join("\n\n")}

${h(3)} ${t.free.installTitle}

\`\`\`sh
${INSTALL}
\`\`\`

${h(2)} ${t.faq.title}

${t.faq.items.map((f) => `${h(3)} ${f.q}\n\n${f.a}`).join("\n\n")}
`;
}

export function installMd(lang: Lang, level = 1): string {
  const i = DICT[lang].install;
  const h = (n: number) => "#".repeat(n + level - 1);
  return `${h(1)} ${i.title}

> ${i.description}

${i.h1} ${i.lead}

${h(2)} ${i.needsTitle}

${list(i.needs.map((n) => `${n.t}: ${n.d}`))}

${h(2)} ${i.stepsTitle}

${i.steps.map((s, n) => `${n + 1}. ${s.t}: ${s.d}`).join("\n")}

\`\`\`sh
${INSTALL}
\`\`\`

**${i.warnStrong}** ${i.warn}

${h(2)} ${i.update}

\`\`\`sh
bash install.sh update
\`\`\`

${h(2)} ${i.appTitle}

${i.appText}

\`\`\`sh
bash install.sh app
\`\`\`

${i.docNote} ${i.fullGuide}: ${REPO}/blob/main/docs/self-hosting.md
`;
}

export function privacyMd(lang: Lang): string {
  const p = PRIVACY[lang];
  return `# ${p.title}

> ${p.description}

${p.updated}

${p.intro}

${p.sections.map((s) => `## ${s.title}\n\n${s.body.join("\n\n")}`).join("\n\n")}

${p.contact}: ${PRIVACY_EMAIL}
`;
}

/** Every Markdown page, for getStaticPaths: { md: "en/instalar/index.md", lang, page } */
export function mdRoutes() {
  return LANGS.flatMap((lang) =>
    MD_PAGES.map((page) => ({ md: mdPath(href(lang, page)).slice(1), lang, page })),
  );
}

export function pageMd(lang: Lang, page: MdPage): string {
  if (page === "/instalar") return installMd(lang);
  if (page === "/privacidade") return privacyMd(lang);
  return homeMd(lang);
}
