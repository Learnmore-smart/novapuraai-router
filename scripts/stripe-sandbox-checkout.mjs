import fs from 'node:fs/promises'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE ?? 'playwright')

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const artifactPath = path.join(repositoryRoot, '.tmp', 'stripe-sandbox-e2e.json')
const artifact = JSON.parse(await fs.readFile(artifactPath, 'utf8'))
const browser = await chromium.launch({
  executablePath: 'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
  headless: true,
})

async function fillInAnyFrame(page, selectors, value) {
  for (const frame of page.frames()) {
    for (const selector of selectors) {
      const input = frame.locator(selector).first()
      if ((await input.count()) > 0 && (await input.isVisible())) {
        await input.fill(value)
        return true
      }
    }
  }
  return false
}

for (const [index, session] of artifact.sessions.entries()) {
  const page = await browser.newPage({ locale: 'en-CA' })
  await page.goto(session.checkout_url, { waitUntil: 'domcontentloaded', timeout: 60_000 })
  await page.waitForTimeout(3_000)
  const bodyText = await page.locator('body').innerText()
  session.observed_payment_methods = [
    ['card', /\bcard\b/i],
    ['link', /\blink\b/i],
    ['alipay', /\balipay\b/i],
    ['wechat_pay', /wechat\s*pay/i],
    ['apple_pay', /apple\s*pay/i],
    ['google_pay', /google\s*pay/i],
  ].filter(([, pattern]) => pattern.test(bodyText)).map(([name]) => name)

  const email = `stripe-sandbox-${Date.now()}-${index}@example.com`
  await fillInAnyFrame(page, ['input[type="email"]', 'input[name="email"]'], email)
  const cardRadio = page.locator('#payment-method-accordion-item-title-card')
  if ((await cardRadio.count()) > 0) {
    await cardRadio.check({ force: true })
    await page.waitForTimeout(1_000)
  }
  const cardNumberSelectors = ['input[autocomplete="cc-number"]', '#cardNumber', 'input[name="cardNumber"]']
  let cardNumberFilled = await fillInAnyFrame(page, cardNumberSelectors, '4242424242424242')
  if (!cardNumberFilled) {
    const cardChoice = page.getByRole('button', { name: /pay with card/i }).first()
    if ((await cardChoice.count()) > 0 && (await cardChoice.getAttribute('aria-expanded')) === 'false') {
      await cardChoice.evaluate((button) => button.click())
    }
    cardNumberFilled = await fillInAnyFrame(page, cardNumberSelectors, '4242424242424242')
  }
  if (!cardNumberFilled) {
    for (const frame of page.frames()) {
      const inputs = await frame.locator('input').evaluateAll((elements) => elements.map((element) => ({
        autocomplete: element.getAttribute('autocomplete'),
        id: element.id,
        name: element.getAttribute('name'),
        placeholder: element.getAttribute('placeholder'),
        type: element.getAttribute('type'),
      })))
      console.error(JSON.stringify({ frame: frame.url(), inputs }))
    }
    throw new Error(`No card number field for ${session.session_id}`)
  }
  await fillInAnyFrame(page, ['input[autocomplete="cc-exp"]', '#cardExpiry', 'input[name="cardExpiry"]'], '1234')
  await fillInAnyFrame(page, ['input[autocomplete="cc-csc"]', '#cardCvc', 'input[name="cardCvc"]'], '123')
  await fillInAnyFrame(page, ['input[autocomplete="cc-name"]', 'input[name="billingName"]'], 'NovaPuraAI Sandbox')

  const submit = page.locator('button[type="submit"]').last()
  if ((await submit.count()) === 0) throw new Error(`No submit button for ${session.session_id}`)
  await submit.click()
  await page.waitForURL(/example\.com\/stripe-success/, { timeout: 60_000 })
  session.browser_result = 'paid_redirect_observed'
  await page.close()
}

await browser.close()
await fs.writeFile(artifactPath, `${JSON.stringify(artifact, null, 2)}\n`, { mode: 0o600 })
