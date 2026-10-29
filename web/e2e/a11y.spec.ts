import AxeBuilder from '@axe-core/playwright'
import { expect, type Page, test } from '@playwright/test'
import QRCode from 'qrcode'
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
  { name: 'home, facilities and map', path: '/', signedIn: false },
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
  { name: 'facility editor, check-in', path: '/lugo/admin/facilities/f1?vista=entrada' },
  { name: 'check-in QR to print', path: '/lugo/admin/facilities/f1/imprimir?que=qr' },
  { name: 'entry code to print', path: '/lugo/admin/facilities/f1/imprimir?que=codigo' },
  { name: 'entry code to print, in braille', path: '/lugo/admin/facilities/f1/imprimir?que=codigo&braille=1' },
  { name: 'check-in at the door, printed QR', path: '/lugo/padel-1/entrada?c=9a1f6c2e-3b4d-4e5f-8a7b-1c2d3e4f5a6b' },
  { name: 'check-in at the door, entry code', path: '/lugo/padel-1/entrada' },
  { name: 'check-in at the door, signed out', path: '/lugo/padel-1/entrada', signedIn: false },
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

    test('flash booking dialog', async ({ page }) => {
      await open(page, '/lugo', { theme })
      await page.getByRole('button', { name: /Ana Pérez/ }).click()
      await page.getByRole('menuitem', { name: /Reserva flash/ }).click()
      await expect(page.getByRole('dialog')).toBeVisible()
      await expect(page.getByRole('img', { name: /QR/ })).toBeVisible()
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })
    })

    test('flash booking dialog, door QR tab with the camera off', async ({ page, context }) => {
      await context.grantPermissions(['geolocation'])
      await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
      await open(page, '/lugo', { theme })
      await page.getByRole('button', { name: /Ana Pérez/ }).click()
      await page.getByRole('menuitem', { name: /Reserva flash/ }).click()
      await page.getByRole('tab', { name: 'QR da porta' }).click()
      await page.getByRole('button', { name: 'Abrir a cámara' }).click()
      await expect(page.getByRole('button', { name: 'Pechar a cámara' })).toBeVisible()
      await page.mouse.move(0, 0)
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })
    })

    test('flash booking dialog, entry code tab', async ({ page, context }) => {
      await context.grantPermissions(['geolocation'])
      await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
      await open(page, '/lugo', { theme })
      await page.getByRole('button', { name: /Ana Pérez/ }).click()
      await page.getByRole('menuitem', { name: /Reserva flash/ }).click()
      await page.getByRole('tab', { name: 'Código' }).click()
      await page.getByRole('button', { name: 'Buscar onde estou' }).click()
      await expect(page.getByRole('radio', { name: /Pista de pádel 1/ })).toBeChecked()
      await page.mouse.move(0, 0)
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })
    })

    // the dialogs on /reservas, each opened from the first upcoming booking
    for (const button of [/^Cancelar$/, /^Convidar$/, /^Cambiar$/, /^Rexistrar entrada$/]) {
      test(`my bookings dialog ${button.source}`, async ({ page }) => {
        await open(page, '/reservas', { theme })
        await page.getByRole('button', { name: button }).first().click()
        await expect(page.getByRole('dialog')).toBeVisible()
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(300)
        await expectAccessible(page, { overlay: true })
      })
    }

    // the bookings calendar (a facility row's Reservas button): book a free turn for someone, open someone's booking
    for (const button of [/^Reservar:/, /^Ana/, /^Carmen/, /^Lucía/]) {
      test(`staff booking dialog ${button.source}`, async ({ page }) => {
        await open(page, '/lugo/admin?vista=reservas', { theme })
        await page.getByRole('button', { name: button }).first().click()
        await expect(page.getByRole('dialog')).toBeVisible()
        await page.waitForTimeout(300)
        await expectAccessible(page, { overlay: true })
      })
    }

    // staff mark Lucía as in: the undo takes the mark button's place, with the focus and a spoken result
    test('staff booking dialog, entry marked', async ({ page }) => {
      await open(page, '/lugo/admin?vista=reservas', { theme })
      await page.getByRole('button', { name: /^Lucía/ }).first().click()
      const dialog = page.getByRole('dialog')
      await dialog.getByRole('button', { name: 'Marcar entrada' }).click()
      await expect(dialog.getByRole('button', { name: 'Desfacer a entrada' })).toBeFocused()
      await expect(dialog.getByText(/^Entrada marcada ás/)).toBeVisible()
      await expect(dialog.getByText('Marcada polo persoal: Xoán Conserxe.')).toBeVisible()
      await page.waitForTimeout(300)
      await expectAccessible(page, { overlay: true })

      await page.keyboard.press('Enter')
      await expect(dialog.getByRole('button', { name: 'Marcar entrada' })).toBeFocused()
      await expect(dialog.getByText('Entrada desfeita.')).toBeVisible()
    })

    test('council settings sheet', async ({ page }) => {
      await open(page, '/lugo/admin', { theme })
      await page.getByRole('button', { name: /axustes|configuraci/i }).click()
      await expect(page.getByRole('dialog')).toBeVisible()
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(400)
      await expectAccessible(page, { overlay: true })
    })

    test('facility editor, check-in with every way on', async ({ page }) => {
      await open(page, '/lugo/admin/facilities/f1?vista=entrada', { theme })
      // turning a way on can show more switches (the location check), so until none is left off
      const off = page.getByRole('switch', { checked: false })
      while ((await off.count()) > 0) await off.first().click()
      await expect(page.getByLabel('Tipo de etiqueta')).toBeVisible()
      await expectAccessible(page)
    })

    test('check-in at the door, the answer and the offer to book', async ({ page, context }) => {
      await context.grantPermissions(['geolocation'])
      await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
      await open(page, '/lugo/padel-1/entrada', { theme })
      await page.getByLabel('Código de entrada').fill('482 913')
      await page.getByRole('button', { name: 'Rexistrar entrada' }).click()
      await expect(page.getByRole('status')).toContainText('Non tes reserva para esta quenda')
      await expect(page.getByRole('button', { name: 'Reservar e entrar' })).toBeVisible()
      await page.mouse.move(0, 0) // the button just clicked would be scanned in its hover colour
      await expectAccessible(page)
    })

    test('platform admin, superadmins tab', async ({ page }) => {
      await open(page, '/admin', { theme })
      await page.getByRole('tab').nth(1).click()
      await expectAccessible(page)
    })

    test('platform admin, tags tab', async ({ page }) => {
      await open(page, '/admin', { theme })
      await page.getByRole('tab').nth(2).click()
      await expect(page.getByRole('heading', { name: 'Deportes' })).toBeVisible()
      await expectAccessible(page)
    })

    test('platform admin, app name and logo tab', async ({ page }) => {
      await open(page, '/admin', { theme })
      await page.getByRole('tab', { name: 'Personalizar a aplicación' }).click()
      await expect(page.getByLabel('Nome da aplicación')).toBeVisible()
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

  test('2.1.1 the check-in tab and its ways can be used with the keyboard alone', async ({ page }) => {
    await open(page, '/lugo/admin/facilities/f1')
    await page.getByRole('tab', { name: 'Datos' }).focus()
    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('tab', { name: 'Entrada' })).toHaveAttribute('aria-selected', 'true')
    await expect(page).toHaveURL(/vista=entrada/)
    const nfc = page.getByRole('switch', { name: 'Etiqueta NFC' })
    await nfc.focus()
    await page.keyboard.press('Space')
    await expect(nfc).toBeChecked()
    await expect(page.getByLabel('Tipo de etiqueta')).toBeVisible()
  })

  test('2.1.1 a flash booking with the entry code works with the keyboard alone', async ({ page, context }) => {
    await context.grantPermissions(['geolocation'])
    await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
    await open(page, '/lugo')
    await page.getByRole('button', { name: /Ana Pérez/ }).click()
    await page.getByRole('menuitem', { name: /Reserva flash/ }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByRole('tab', { name: 'Lector QR' }).focus()
    await page.keyboard.press('ArrowRight')
    await expect(dialog.getByRole('tab', { name: 'Código' })).toHaveAttribute('aria-selected', 'true')
    await dialog.getByRole('button', { name: 'Buscar onde estou' }).focus()
    await page.keyboard.press('Enter')
    await expect(dialog.getByRole('radio', { name: /Pista de pádel 1/ })).toBeChecked()
    await dialog.getByLabel('Código de entrada').focus()
    await page.keyboard.type('482915')
    await page.keyboard.press('Enter')
    await expect(dialog.getByRole('status')).toContainText('Non tes reserva para esta quenda')
  })

  test('the door QR asks for the location first: without it the camera stays closed', async ({ page }) => {
    let cameraAsked = false
    await page.exposeFunction('cameraAsked', () => (cameraAsked = true))
    await page.addInitScript(() => {
      navigator.mediaDevices.getUserMedia = async () => {
        await (window as unknown as { cameraAsked: () => Promise<void> }).cameraAsked()
        throw new DOMException('no', 'NotAllowedError')
      }
    })
    await open(page, '/lugo') // no geolocation permission granted
    await page.getByRole('button', { name: /Ana Pérez/ }).click()
    await page.getByRole('menuitem', { name: /Reserva flash/ }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByRole('tab', { name: 'QR da porta' }).click()
    await dialog.getByRole('button', { name: 'Abrir a cámara' }).click()
    await expect(dialog.getByRole('status')).toContainText('localización')
    await expect(dialog.getByRole('button', { name: 'Pechar a cámara' })).toBeHidden()
    expect(cameraAsked).toBe(false)
  })

  test('a flash booking reads the QR on the door with the phone camera', async ({ page, context }) => {
    // the camera films the door's printed QR: a canvas stream stands in for it
    const door = await QRCode.toDataURL('http://localhost:5179/lugo/padel-1/entrada?c=9a1f6c2e-3b4d-4e5f-8a7b-1c2d3e4f5a6b', { margin: 4, width: 480 })
    await page.addInitScript((src) => {
      navigator.mediaDevices.getUserMedia = async () => {
        const canvas = document.createElement('canvas')
        canvas.width = canvas.height = 480
        const context = canvas.getContext('2d')!
        const image = new Image()
        image.src = src
        await image.decode()
        const draw = () => {
          context.fillStyle = '#fff'
          context.fillRect(0, 0, 480, 480)
          context.drawImage(image, 0, 0)
          requestAnimationFrame(draw)
        }
        draw()
        return canvas.captureStream(10)
      }
    }, door)
    await context.grantPermissions(['geolocation'])
    await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
    await open(page, '/lugo')
    await page.getByRole('button', { name: /Ana Pérez/ }).click()
    await page.getByRole('menuitem', { name: /Reserva flash/ }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByRole('tab', { name: 'QR da porta' }).click()
    const sent = page.waitForRequest((r) => r.method() === 'POST' && r.url().endsWith('/tenants/lugo/facilities/padel-1/entrance'))
    await dialog.getByRole('button', { name: 'Abrir a cámara' }).click()
    const body = (await sent).postDataJSON()
    expect(body).toMatchObject({ qr_token: '9a1f6c2e-3b4d-4e5f-8a7b-1c2d3e4f5a6b', book: true, latitude: 43.0097 })
    await expect(dialog.getByRole('status')).toContainText('Non tes reserva para esta quenda')
    await expect(dialog.getByRole('button', { name: 'Escanear de novo' })).toBeVisible()
  })

  test('2.1.1 a booking can be checked in from my bookings with the keyboard alone', async ({ page, context }) => {
    await context.grantPermissions(['geolocation'])
    await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
    await open(page, '/reservas')
    await expect(page.getByText('Xa podes entrar')).toBeVisible()
    await expect(page.getByText(/^Empeza en \d h/)).toBeVisible()
    await page.getByRole('button', { name: 'Rexistrar entrada' }).first().focus()
    await page.keyboard.press('Enter')
    const dialog = page.getByRole('dialog')
    for (const way of ['Lector QR', 'Código', 'QR da porta', 'Persoal']) {
      await expect(dialog.getByRole('tab', { name: way })).toBeVisible()
    }
    await expect(dialog.getByRole('tab', { name: 'Lector QR' })).toHaveAttribute('aria-selected', 'true')
    await dialog.getByRole('tab', { name: 'Lector QR' }).focus()
    await page.keyboard.press('ArrowRight')
    await expect(dialog.getByRole('tab', { name: 'Código' })).toHaveAttribute('aria-selected', 'true')
    await dialog.getByLabel('Código de entrada').focus()
    await page.keyboard.type('482915')
    await page.keyboard.press('Enter')
    await expect(dialog.getByRole('status')).toContainText('Non tes reserva para esta quenda')
  })

  test('2.1.1 the entry code can be typed and sent with the keyboard alone', async ({ page, context }) => {
    await context.grantPermissions(['geolocation'])
    await context.setGeolocation({ latitude: 43.0097, longitude: -7.5568, accuracy: 10 })
    await open(page, '/lugo/padel-1/entrada')
    await page.getByLabel('Código de entrada').focus()
    await page.keyboard.type('482913')
    await page.keyboard.press('Enter')
    await expect(page.getByRole('status')).toContainText('Non tes reserva para esta quenda')
  })

  test('signed in, the invite link joins the booking by itself, once', async ({ page }) => {
    const joins: string[] = []
    page.on('request', (r) => r.method() === 'POST' && r.url().includes('/join') && joins.push(r.url()))
    await open(page, '/invitacion/tok123')
    await expect(page.getByText(/^Xa estás dentro:/)).toBeVisible()
    expect(joins).toHaveLength(1)
    expect(joins[0]).toContain('/me/invites/tok123/join')
  })

  test('signed out, the invite link waits for a choice', async ({ page }) => {
    const joins: string[] = []
    page.on('request', (r) => r.method() === 'POST' && r.url().includes('/join') && joins.push(r.url()))
    await open(page, '/invitacion/tok123', { signedIn: false })
    await expect(page.getByRole('heading', { name: 'Unirme coa miña conta' })).toBeVisible()
    expect(joins).toHaveLength(0)
  })

  test("the council page's bookings card shows no QR: check-in lives in My bookings", async ({ page }) => {
    await open(page, '/lugo')
    await expect(page.getByRole('heading', { name: 'As túas reservas' })).toBeVisible()
    await expect(page.getByRole('button', { name: /QR/ })).toHaveCount(0)
  })

  test('the QR reader can be switched off and stays set up', async ({ page }) => {
    await open(page, '/lugo/admin/facilities/f1?vista=entrada')
    const reader = page.getByRole('switch', { name: 'QR de reserva ou flash nun lector' })
    await expect(reader).toBeChecked()
    await reader.press('Space')
    await expect(reader).not.toBeChecked()
    await expect(page.getByText('O lector está apagado')).toBeVisible()
    await expect(page.getByText('Entrada principal')).toBeVisible() // the reader is still listed
    await expect(page.getByRole('button', { name: 'Gardar', exact: true })).toBeEnabled()
  })

  test('the facility editor saves the check-in ways and keeps the old QR until asked', async ({ page }) => {
    await open(page, '/lugo/admin/facilities/f1?vista=entrada')
    await expect(page.getByRole('img', { name: 'QR para rexistrar a entrada na instalación' })).toBeVisible()
    const save = page.getByRole('button', { name: 'Gardar', exact: true })
    await expect(save).toBeDisabled() // nothing changed yet
    await page.getByLabel('Distancia ao punto, en metros').fill('150')
    await expect(save).toBeEnabled()
    await page.getByRole('button', { name: 'Novo QR' }).click()
    await expect(page.getByRole('dialog')).toContainText('deixan de funcionar')
    await page.getByRole('button', { name: 'Manter este QR' }).click()
    await expect(page.getByRole('dialog')).toBeHidden()
    await expect(page.getByLabel('Código de entrada: 4 8 2 9 1 5')).toBeVisible()
    await page.getByRole('button', { name: 'Novo código' }).click()
    await expect(page.getByRole('dialog')).toContainText('deixa de funcionar')
    await page.getByRole('button', { name: 'Manter este código' }).click()
    await expect(page.getByRole('dialog')).toBeHidden()
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

  test("an installation's own name replaces LibreQuenda's in the bar and the title, not in the footer", async ({ page }) => {
    const name = 'Reservas de Lugo'
    await open(page, '/lugo/padel-1', { branding: { name, custom_name: name, logo_ref: '', logo_url: '' } })
    await expect(page).toHaveTitle(`Pista de pádel 1 · Concello de Lugo · ${name}`)
    await expect(page.getByRole('link', { name: `${name}, inicio` })).toBeVisible()
    await expect(page.locator('footer')).toContainText('LibreQuenda é software libre')
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
