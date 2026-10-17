export type ProductEdition = 'mac' | 'fnos'

export function productEdition(overview: { product?: { edition?: string } } | null): ProductEdition {
  const fromOverview = overview?.product?.edition
  if (fromOverview === 'fnos' || fromOverview === 'mac') return fromOverview
  const fromDocument = document.querySelector('meta[name="opensurge-edition"]')?.getAttribute('content')
  return fromDocument === 'fnos' ? 'fnos' : 'mac'
}

export function productTitle(edition: ProductEdition): string {
  return edition === 'fnos' ? 'OpenSurge for fnOS' : 'OpenSurge for Mac'
}

export function productIcon(edition: ProductEdition): string {
  return edition === 'fnos' ? '/opensurge-fnos-icon.png' : '/opensurge-icon.png'
}

export function productLine(edition: ProductEdition): string {
  return edition === 'fnos' ? 'for fnOS' : 'for Mac'
}
