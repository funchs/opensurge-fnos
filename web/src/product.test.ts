// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { productEdition, productIcon, productLine, productTitle } from './product'

afterEach(() => {
  document.head.querySelector('meta[name="opensurge-edition"]')?.remove()
})

describe('productEdition', () => {
  it('defaults to Mac when overview and document are silent', () => {
    expect(productEdition(null)).toBe('mac')
    expect(productTitle('mac')).toBe('OpenSurge for Mac')
    expect(productIcon('mac')).toBe('/opensurge-icon.png')
    expect(productLine('mac')).toBe('for Mac')
  })

  it('uses overview edition when present', () => {
    expect(productEdition({ product: { edition: 'fnos' } })).toBe('fnos')
    expect(productTitle('fnos')).toBe('OpenSurge for fnOS')
    expect(productIcon('fnos')).toBe('/opensurge-fnos-icon.png')
    expect(productLine('fnos')).toBe('for fnOS')
  })

  it('reads the document meta before overview arrives', () => {
    const meta = document.createElement('meta')
    meta.setAttribute('name', 'opensurge-edition')
    meta.setAttribute('content', 'fnos')
    document.head.append(meta)
    expect(productEdition(null)).toBe('fnos')
  })
})
