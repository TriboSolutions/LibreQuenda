import AxeBuilder from '@axe-core/playwright'
import { expect, type Page, test } from '@playwright/test'
import { mockApp, type MockOptions } from './mocks'

// Every automatable WCAG 2.1 A and AA rule, plus axe's AAA rules (7:1 contrast and the like).
const TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag2aaa', 'best-practice']

// Overlays (dialogs, menus): Radix hides the page behind with aria-hidden and keeps the focus
// trapped inside, so the page's own controls cannot be reached; the overlay sits outside the landmarks.
const OVERLAY_RULES = ['aria-hidden-focus', 'region', 'page-has-heading-one']

async function expectAccessible(page: Page, { overlay = false } = {}) {
  let builder = new AxeBuilder({ page }).withTags(TAGS)
  // the map tiles come from OpenStreetMap
  builder = builder.exclude('.leaflet-tile-pane')
  // the wordmark is a logotype: no contrast requirement (WCAG 1.4.3 and 1.4.6, "Logotypes")
  builder = builder.exclude('[data-logotype]')
  if (overlay) builder = builder.disableRules(OVERLAY_RULES)
  const { violations } = await builder.analyze()
  const report = violations.map((v) => ({
    rule: v.id,
    impact: v.impact,
    help: v.help,
    nodes: v.nodes.map((n) => `${n.target.join(' ')}: ${n.failureSummary?.split('\n').slice(1, 3).join(' ')}`),
  }))
  expect(report, JSON.stringify(report, null, 2)).toEqual([])
}

async function open(page: Page, path: string, options?: MockOptions) {
  await mockApp(page, options)
  await page.goto(path)
  await expect(page.locator('h1').first()).toBeVisible()
  await page.waitForLoadState('networkidle')
  // animations finish before axe reads the colours
  await page.waitForTimeout(350)
}

const pages: { name: string; path: string; signedIn?: boolean }[] = [
  { name: 'council picker', path: '/', signedIn: false },
  { name: 'council page, signed out', path: '/lugo', signedIn: false },
  { name: 'council page, signed in', path: '/lugo' },
  { name: 'facility page', path: '/lugo/padel-1' },
  { name: 'my bookings', path: '/reservas' },
  { name: 'profile', path: '/perfil' },
  { name: 'invite, signed out', path: '/invitacion/tok123', signedIn: false },
  { name: 'invite, signed in', path: '/invitacion/tok123' },
  { name: 'council admin, facilities', path: '/lugo/admin' },
  { name: 'council admin, indicators', path: '/lugo/admin?vista=indicadores' },
  { name: 'council admin, bookings', path: '/lugo/admin?vista=reservas' },
  { name: 'facility editor', path: '/lugo/admin/facilities/f1' },
  { name: 'platform admin', path: '/admin' },
  { name: 'entrance reader', path: '/lector' },
  { name: 'accessibility statement', path: '/accesibilidade', signedIn: false },
]

for (const theme of ['light', 'dark'] as const) {
  test.describe(`${theme} theme`, () => {
    for (const { name, path, signedIn = true } of pages) {
      test(`${name} (${path})`, async ({ page }) => {
        await open(page, path, { signedIn, theme })
        await expectAccessible(page)
      })
    }

    test('booking dialog', async ({ page }) => {
      await open(page, '/lugo/padel-1', { theme })
      await page.locator('[data-free=true]').first().click()
      await expect(page.getByRole('dialog')).toBeVisible()
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })
    })

    test('notifications menu', async ({ page }) => {
      await open(page, '/lugo', { theme })
      await page.getByRole('button', { name: /notifica|aviso/i }).click()
      await expect(page.getByRole('menu')).toBeVisible()
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })
    })

    test('quick entry dialog', async ({ page }) => {
      await open(page, '/lugo', { theme })
      await page.getByRole('button', { name: /Ana Pérez/ }).click()
      await page.getByRole('menuitem', { name: /Entrada rápida/ }).click()
      await expect(page.getByRole('dialog')).toBeVisible()
      await expect(page.getByRole('img', { name: /QR/ })).toBeVisible()
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })
    })

    // the dialogs on /reservas, each opened from the first upcoming booking
    for (const button of [/^Cancelar$/, /QR/, /^Convidar$/, /^Cambiar$/]) {
      test(`my bookings dialog ${button.source}`, async ({ page }) => {
        await open(page, '/reservas', { theme })
        await page.getByRole('button', { name: button }).first().click()
        await expect(page.getByRole('dialog')).toBeVisible()
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(300)
        await expectAccessible(page, { overlay: true })
      })
    }

    // the Reservas tab: book a free turn for someone, cancel someone's booking
    for (const button of [/^Reservar:/, /^Cancelar: Ana/]) {
      test(`staff booking dialog ${button.source}`, async ({ page }) => {
        await open(page, '/lugo/admin?vista=reservas', { theme })
        await page.getByRole('button', { name: button }).click()
        await expect(page.getByRole('dialog')).toBeVisible()
        await page.waitForTimeout(300)
        await expectAccessible(page, { overlay: true })
      })
    }

    test('council settings sheet', async ({ page }) => {
      await open(page, '/lugo/admin', { theme })
      await page.getByRole('button', { name: /axustes|configuraci/i }).click()
      await expect(page.getByRole('dialog')).toBeVisible()
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(400)
      await expectAccessible(page, { overlay: true })
    })

    test('platform admin, superadmins tab', async ({ page }) => {
      await open(page, '/admin', { theme })
      await page.getByRole('tab').nth(1).click()
      await expectAccessible(page)
    })
  })
}

test.describe('keyboard', () => {
  test('2.4.1 a skip link leads to the main content', async ({ page }) => {
    await open(page, '/lugo')
    await page.keyboard.press('Tab')
    const skip = page.locator(':focus')
    await expect(skip).toHaveAttribute('href', '#main')
    await expect(skip).toBeVisible()
    await page.keyboard.press('Enter')
    await expect(page.locator('main')).toBeFocused()
  })

  test('2.4.7 every focusable control shows a visible focus indicator', async ({ page }) => {
    await open(page, '/lugo/padel-1')
    const missing: string[] = []
    for (let i = 0; i < 40; i++) {
      await page.keyboard.press('Tab')
      const result = await page.evaluate(() => {
        const el = document.activeElement as HTMLElement | null
        if (!el || el === document.body) return null
        const style = getComputedStyle(el)
        const outline = style.outlineStyle !== 'none' && parseFloat(style.outlineWidth) >= 2
        const ring = style.boxShadow !== 'none' && style.boxShadow !== ''
        return { ok: outline || ring, name: `${el.tagName.toLowerCase()} ${el.getAttribute('aria-label') ?? el.textContent?.trim().slice(0, 30)}` }
      })
      if (result && !result.ok) missing.push(result.name)
    }
    expect(missing).toEqual([])
  })

  test('2.1.1 a turn can be booked with the keyboard alone, and focus comes back', async ({ page }) => {
    await open(page, '/lugo/padel-1')
    const slot = page.locator('[data-free=true]').first()
    await slot.focus()
    await page.keyboard.press('Enter')
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.locator(':focus')).toHaveCount(1) // focus moved into the dialog
    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await expect(slot).toBeFocused()
  })

  test('2.4.3 opening a facility from the indicators table keeps the focus on the page', async ({ page }) => {
    await open(page, '/lugo/admin?vista=indicadores')
    await page.getByRole('button', { name: /Pista de pádel 1/ }).focus()
    await page.keyboard.press('Enter')
    await expect(page.locator('#indicators-facility')).toBeFocused()
  })

  test('2.1.2 the account menu opens and closes from the keyboard', async ({ page }) => {
    await open(page, '/lugo')
    const trigger = page.getByRole('button', { name: /Ana Pérez/ })
    await trigger.focus()
    await page.keyboard.press('Enter')
    await expect(page.getByRole('menu')).toBeVisible()
    await page.keyboard.press('ArrowDown')
    await page.keyboard.press('Escape')
    await expect(page.getByRole('menu')).toBeHidden()
    await expect(trigger).toBeFocused()
  })
})

test.describe('structure', () => {
  test('the accessibility statement is linked from every page and lists the contact of each council', async ({ page }) => {
    await open(page, '/lugo/padel-1', { signedIn: false })
    await page.getByRole('contentinfo').getByRole('link', { name: 'Declaración de accesibilidade' }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'Declaración de accesibilidade' })).toBeVisible()
    await expect(page).toHaveTitle('Declaración de accesibilidade · LibreQuenda')
    await expect(page.getByRole('link', { name: /deportes@lugo\.gal/ })).toHaveAttribute('href', 'mailto:deportes@lugo.gal')
    await expect(page.getByRole('link', { name: /982 297 100/ })).toHaveAttribute('href', 'tel:982297100')
    await expect(page.getByText(/aínda non indicou un contacto/)).toBeVisible() // Ourense gave none
    await expect(page.getByText(/^Esta declaración preparouse o \d{1,2} de [a-z]+ de \d{4}\.$/)).toBeVisible()
  })

  test('2.4.2 every page has a title that names it', async ({ page }) => {
    await open(page, '/lugo/padel-1')
    await expect(page).toHaveTitle(/Pista de pádel 1.*LibreQuenda/)
    await page.goto('/reservas')
    await expect(page.locator('main h1')).toBeVisible()
    await expect(page).not.toHaveTitle(/^LibreQuenda$/)
  })

  test('3.1.1 the page language follows the chosen language', async ({ page }) => {
    await open(page, '/lugo', { lang: 'es' })
    await expect(page.locator('html')).toHaveAttribute('lang', 'es')
  })

  test('2.3.3 motion is turned off when the user asks for reduced motion', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await open(page, '/lugo/padel-1')
    await page.locator('[data-free=true]').first().click()
    const duration = await page.getByRole('dialog').evaluate((el) => getComputedStyle(el).animationDuration)
    expect(parseFloat(duration)).toBeLessThanOrEqual(0.01)
  })
})

test.describe('reflow', () => {
  for (const { name, path } of pages.filter((p) => !p.path.includes('admin'))) {
    test(`1.4.10 ${name} fits 320 px without horizontal scrolling @reflow`, async ({ page }) => {
      await open(page, path, { signedIn: !path.includes('invitacion') })
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow).toBeLessThanOrEqual(0)
    })
  }

  test('1.4.12 text spacing can be increased without losing content @reflow', async ({ page }) => {
    await open(page, '/lugo')
    await page.addStyleTag({
      content: '* { line-height: 1.5 !important; letter-spacing: 0.12em !important; word-spacing: 0.16em !important; } p { margin-bottom: 2em !important; }',
    })
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
    expect(overflow).toBeLessThanOrEqual(0)
  })
})
