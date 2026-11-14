// Tells IndexNow (Bing, which ChatGPT search draws on, plus Yandex, Seznam, Naver…)
// that the landing changed, so it is crawled again within hours instead of weeks.
// Run it after a deploy: pnpm indexnow
// The key is public by design: IndexNow checks it at https://librequenda.app/<key>.txt
const SITE = "https://librequenda.app";
const KEY = "60ac5a310cba1533422fffe41aada07f";

const sitemap = await (await fetch(`${SITE}/sitemap.xml`)).text();
const urls = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
urls.push(`${SITE}/llms.txt`, `${SITE}/llms-full.txt`);

const res = await fetch("https://api.indexnow.org/indexnow", {
  method: "POST",
  headers: { "Content-Type": "application/json; charset=utf-8" },
  body: JSON.stringify({ host: new URL(SITE).host, key: KEY, keyLocation: `${SITE}/${KEY}.txt`, urlList: urls }),
});
// 200 or 202: accepted; 403: the key file is not live yet
console.log(`IndexNow: ${res.status} ${res.statusText} (${urls.length} URLs)`);
if (!res.ok) process.exit(1);
