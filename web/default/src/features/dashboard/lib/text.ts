/**
 * Get plain text preview (strip HTML tags and Markdown formatting)
 */
export function getPreviewText(
  content: string,
  maxLength: number = 60
): string {
  if (!content) return ''
  const plainText = content
    .replaceAll(/<[^>]*>/g, '') // Remove HTML tags
    .replaceAll(/[#*_]/g, '') // Remove Markdown formatting symbols
    .trim()
  return plainText.length > maxLength
    ? `${plainText.slice(0, maxLength)}...`
    : plainText
}
