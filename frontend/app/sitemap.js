const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://oneflow.id';

const publicRoutes = [
  '/',
  '/about',
  '/contact',
  '/pricing',
  '/privacy',
  '/terms',
  '/refund-policy',
  '/data-deletion',
];

export default function sitemap() {
  const now = new Date();

  return publicRoutes.map((route) => ({
    url: new URL(route, siteUrl).toString(),
    lastModified: now,
    changeFrequency: route === '/' ? 'weekly' : 'monthly',
    priority: route === '/' ? 1 : 0.6,
  }));
}
