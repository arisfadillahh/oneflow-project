const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://oneflow.id';

export default function robots() {
  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
        disallow: ['/api/', '/dashboard', '/login'],
      },
    ],
    sitemap: new URL('/sitemap.xml', siteUrl).toString(),
  };
}
