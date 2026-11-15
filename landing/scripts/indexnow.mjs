// Tells IndexNow (Bing, which ChatGPT search draws on, plus Yandex, Seznam, Naver…) which pages
// changed, so they are crawled again within hours instead of weeks. Run it after a deploy:
//   pnpm indexnow          the pages whose <lastmod> is from the last 2 days
//   pnpm indexnow --all    every page in the sitemap, plus llms.txt and llms-full.txt
// The key is public by design: IndexNow checks it at https://librequenda.app/<key>.txt
const SITE = "https://librequenda.app";
const KEY = "60ac5a310cba1533422fffe41aada07f";
const all = process.argv.includes("--all");

const xml = await (await fetch(`${SITE}/sitemap.xml`)).text();
const since = new Date(Date.now() - 2 * 86400e3).toISOString().slice(0, 10);
const urls = [...xml.matchAll(/<url>([\s\S]*?)<\/url>/g)]
  .map(([, u]) => ({ loc: u.match(/<loc>([^<]+)<\/loc>/)?.[1], lastmod: u.match(/<lastmod>([^<]+)<\/lastmod>/)?.[1] }))
  .filter((u) => u.loc && (all || (u.lastmod && u.lastmod >= since)))
  .map((u) => u.loc);
if (all || urls.length) urls.push(`${SITE}/llms.txt`, `${SITE}/llms-full.txt`);

if (!urls.length) {
  console.log(`IndexNow: no page changed since ${since}; nothing sent (--all sends everything)`);
  process.exit(0);
}
const res = await fetch("https://api.indexnow.org/indexnow", {
  method: "POST",
  headers: { "Content-Type": "application/json; charset=utf-8" },
  body: JSON.stringify({ host: new URL(SITE).host, key: KEY, keyLocation: `${SITE}/${KEY}.txt`, urlList: urls }),
});
// 200 or 202: accepted; 403: the key file is not live yet; 422: a URL is not on this host
console.log(`IndexNow: ${res.status} ${res.statusText} (${urls.length} URLs)`);
if (!res.ok) process.exit(1);
