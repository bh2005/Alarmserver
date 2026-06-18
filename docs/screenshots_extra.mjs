import puppeteer from 'puppeteer'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dir = dirname(fileURLToPath(import.meta.url))
const outDir = join(__dir, 'screenshots')
const BASE   = 'http://localhost:5173'
const W = 1400, H = 900
const delay  = ms => new Promise(r => setTimeout(r, ms))

const browser = await puppeteer.launch({
  headless: true,
  args: ['--no-sandbox', '--disable-setuid-sandbox', `--window-size=${W},${H}`],
})

// ── Login-Seite Dark Mode ─────────────────────────────────────────────────
{
  const p = await browser.newPage()
  await p.setViewport({ width: W, height: H })
  // Tailwind nutzt .dark-Klasse, nicht prefers-color-scheme
  // → Script-Tag vor dem App-Bundle injizieren
  await p.evaluateOnNewDocument(() => {
    localStorage.setItem('alarmcore_theme', 'dark')
    document.documentElement.classList.add('dark')
  })
  await p.goto(`${BASE}/login`, { waitUntil: 'networkidle0' })
  await delay(600)
  await p.screenshot({ path: join(outDir, '13_dark_login.png') })
  await p.close()
  console.log('✓ 13_dark_login.png')
}

// ── Einloggen und Dark Mode aktivieren ───────────────────────────────────
const page = await browser.newPage()
await page.setViewport({ width: W, height: H })
// Dark Mode VOR dem ersten Laden setzen — Theme-Store liest localStorage beim Init
await page.evaluateOnNewDocument(() => {
  localStorage.setItem('alarmcore_theme', 'dark')
})
await page.goto(`${BASE}/login`, { waitUntil: 'networkidle0' })
await delay(400)
await page.type('input[autocomplete="username"]', 'admin')
await page.type('input[autocomplete="current-password"]', 'admin123')
await page.click('button[type="submit"]')
await page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 10000 }).catch(() => {})
await delay(600)

// Dark Mode via Theme-Toggle (Mond-Icon in der Sidebar, unten links)
// Alternativ: direkt das <html>-Element mit .dark-Klasse versehen
await page.evaluate(() => {
  document.documentElement.classList.add('dark')
  // localStorage setzen damit der Store es als dark erkennt
  localStorage.setItem('theme', 'dark')
})
await delay(300)

// Dashboard Dark
await page.goto(`${BASE}/dashboard`, { waitUntil: 'networkidle0' })
await delay(500)
await page.screenshot({ path: join(outDir, '14_dark_dashboard.png') })
console.log('✓ 14_dark_dashboard.png')

// Eskalationspolitiken Dark
await page.goto(`${BASE}/escalation-policies`, { waitUntil: 'networkidle0' })
await delay(500)
await page.screenshot({ path: join(outDir, '15_dark_eskalation.png') })
console.log('✓ 15_dark_eskalation.png')

await page.close()

// ── /health Endpoint im Browser ──────────────────────────────────────────
{
  const p = await browser.newPage()
  await p.setViewport({ width: W, height: H })

  // Chrome zeigt JSON mit eingebautem Pretty-Printer an
  await p.goto('http://localhost:8080/health', { waitUntil: 'networkidle0' })
  await delay(400)

  // Chrome zeigt rohen JSON-Text — wir formatieren ihn hübsch per JS
  await p.evaluate(() => {
    const pre = document.querySelector('pre')
    if (pre) {
      try {
        const json = JSON.parse(pre.innerText)
        pre.innerText = JSON.stringify(json, null, 2)
        pre.style.cssText = 'font-size:18px;font-family:monospace;padding:32px;line-height:1.6;color:#1e293b;background:#f8fafc'
        document.body.style.background = '#f8fafc'
      } catch {}
    }
  })
  await delay(200)
  await p.screenshot({ path: join(outDir, '16_health_endpoint.png') })
  await p.close()
  console.log('✓ 16_health_endpoint.png')
}

await browser.close()
console.log('\nFertig.')
