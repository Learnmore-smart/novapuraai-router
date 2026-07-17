/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import test from 'node:test'

import { installImageProtection } from './image-protection.ts'

class ImageEventTarget extends EventTarget {
  readonly tagName = 'IMG'
}

test('prevents context menus and dragging when the event target is an image', () => {
  const imageTarget = new ImageEventTarget()
  installImageProtection(imageTarget as unknown as Document)

  for (const eventName of ['contextmenu', 'dragstart']) {
    const event = new Event(eventName, { cancelable: true })
    imageTarget.dispatchEvent(event)
    assert.equal(event.defaultPrevented, true)
  }
})

test('leaves context menus and dragging available for non-image targets', () => {
  const documentTarget = new EventTarget()
  installImageProtection(documentTarget as unknown as Document)

  for (const eventName of ['contextmenu', 'dragstart']) {
    const event = new Event(eventName, { cancelable: true })
    documentTarget.dispatchEvent(event)
    assert.equal(event.defaultPrevented, false)
  }
})
