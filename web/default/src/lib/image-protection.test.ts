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
