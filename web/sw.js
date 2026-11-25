/*
  The service worker: the app opens without coverage (at the door of a facility),
  so «My bookings» shows the bookings kept in the browser (src/lib/offline.ts)
  and the booking's QR for the reader. Built into dist/sw.js by vite.config.ts,
  which fills in the build's files and a version.
  - The app's files (hashed, never change): from the cache, kept at install.
  - Pages and /config.json: from the network, the cached copy when it fails.
  - The API, the voice line and storage: never cached; the pages say they are offline.
*/
const VERSION = '__VERSION__'
const PRECACHE = __PRECACHE__
const CACHE = `librequenda-${VERSION}`

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      // the sign-in's settings too, but an install without them yet still works
      .then((cache) => cache.addAll(PRECACHE).then(() => cache.add('/config.json').catch(() => undefined)))
      .then(() => self.skipWaiting()),
  )
})

// the previous builds' files go
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key.startsWith('librequenda-') && key !== CACHE).map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  )
})

/** The network, keeping a copy of what it answered fine; the copy when it fails or a gateway answers for it (5xx). */
async function networkFirst(request, cacheKey) {
  const cache = await caches.open(CACHE)
  let response
  try {
    response = await fetch(request)
  } catch (error) {
    const copy = await cache.match(cacheKey, { ignoreVary: true })
    if (copy) return copy
    throw error
  }
  if (response.ok) await cache.put(cacheKey, response.clone())
  else if (response.status >= 500) return (await cache.match(cacheKey, { ignoreVary: true })) ?? response
  return response
}

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET') return
  const url = new URL(request.url)
  if (url.origin !== self.location.origin) return
  const path = url.pathname
  if (path.startsWith('/api/') || path.startsWith('/voice/') || path.startsWith('/storage/') || path === '/callback') return

  if (request.mode === 'navigate') {
    // every route is the same page (index.html), as the server answers it
    event.respondWith(networkFirst(request, '/'))
  } else if (path === '/config.json') {
    event.respondWith(networkFirst(request, '/config.json'))
  } else if (path.startsWith('/assets/') || PRECACHE.includes(path)) {
    event.respondWith(caches.match(request, { ignoreSearch: true, ignoreVary: true }).then((copy) => copy ?? fetch(request)))
  }
})
