function isImageTarget(target: EventTarget | null): boolean {
  if (!target || typeof target !== 'object' || !('tagName' in target)) {
    return false
  }

  const tagName = (target as { tagName?: unknown }).tagName
  return typeof tagName === 'string' && tagName.toUpperCase() === 'IMG'
}

function preventImageAction(event: Event): void {
  if (isImageTarget(event.target)) {
    event.preventDefault()
  }
}

export function installImageProtection(documentTarget: Document): void {
  documentTarget.addEventListener('contextmenu', preventImageAction, {
    capture: true,
  })
  documentTarget.addEventListener('dragstart', preventImageAction, {
    capture: true,
  })
}
