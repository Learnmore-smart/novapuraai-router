/**
 * Writes fully localized docs for every section × language.
 * Code fences / API paths stay technical English; all prose is localized.
 * No self-host / Cloud Run domain-replacement lines.
 *
 * Run: node scripts/docs-i18n/build-all.mjs
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { EN } from './en.mjs'
import { FR } from './fr.mjs'
import { JA } from './ja.mjs'
import { RU } from './ru.mjs'
import { VI } from './vi.mjs'
import { ZHTW } from './zh-TW.mjs'
import { ZH } from './zh.mjs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '../../src/i18n/docs')

const PACKS = {
  en: EN,
  zh: ZH,
  'zh-TW': ZHTW,
  fr: FR,
  ja: JA,
  ru: RU,
  vi: VI,
}

const FORBIDDEN = [
  '自托管时请替换为你的 Cloud Run 或反向代理域名',
  'Self-hosting? Replace the host with your Cloud Run or reverse-proxy domain.',
  'Replace the host with your deployment domain when self-hosting',
  'identifiants techniques',
  '技術識別子のため英語のまま',
  'технические идентификаторы',
  'định danh kỹ thuật',
]

let written = 0
for (const [lang, pack] of Object.entries(PACKS)) {
  for (const [section, body] of Object.entries(pack)) {
    for (const bad of FORBIDDEN) {
      if (body.includes(bad)) {
        throw new Error(`Forbidden phrase in ${section}/${lang}: ${bad}`)
      }
    }
    const dir = path.join(ROOT, section)
    fs.mkdirSync(dir, { recursive: true })
    const text = body.trim() + '\n'
    fs.writeFileSync(path.join(dir, `${lang}.md`), text, 'utf8')
    written++
  }
}

// ensure every EN section has every lang
const sections = Object.keys(EN)
const langs = Object.keys(PACKS)
for (const section of sections) {
  for (const lang of langs) {
    const p = path.join(ROOT, section, `${lang}.md`)
    if (!fs.existsSync(p)) throw new Error(`Missing ${p}`)
  }
}

console.log(
  `Wrote ${written} files for ${sections.length} sections × ${langs.length} langs`
)
