// Signs in to LibreQuenda through a test identity provider's button, as a person
// would, and prints what the API then knows about them (/api/me) as JSON.
// Runs inside the proxy's network namespace (run.sh): http://localhost is the web,
// http://auth.localhost the login, http://keycloak:8080 the identity provider.
import { chromium } from 'playwright-core'

const [button] = process.argv.slice(2)
const browser = await chromium.launch()
const page = await browser.newPage({ locale: 'gl-ES' })
try {
  await page.goto('http://localhost/')
  await page.getByRole('button', { name: 'Iniciar sesión' }).first().click()
  await page.getByRole('button', { name: button }).click()

  await page.waitForURL(/keycloak:8080/)
  await page.fill('#username', '12345678z')
  await page.fill('#password', 'proba-clave')
  await page.click('#kc-login')

  // the login may offer a second factor: not for this test
  const skip = page.getByRole('button', { name: /omitir|skip/i })
  await Promise.race([
    page.waitForURL(/^http:\/\/localhost\/(?!callback)/, { timeout: 60_000 }),
    skip.waitFor({ timeout: 60_000 }).then(() => skip.click()),
  ])
  await page.waitForURL(/^http:\/\/localhost\/(?!callback)/, { timeout: 60_000 })

  const me = await page.evaluate(async () => {
    const key = Object.keys(localStorage).find((k) => k.startsWith('oidc.user:'))
    const user = JSON.parse(localStorage.getItem(key))
    const res = await fetch('/api/me', { headers: { Authorization: `Bearer ${user.access_token}` } })
    return { status: res.status, sub: user.profile.sub, me: await res.json() }
  })
  console.log(JSON.stringify(me))
} catch (err) {
  console.error(`failed at ${page.url()}: ${err.message}`)
  console.error((await page.locator('body').innerText().catch(() => '')).slice(0, 1500))
  process.exitCode = 1
} finally {
  await browser.close()
}
