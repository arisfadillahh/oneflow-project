import Image from 'next/image';
import Link from 'next/link';
import { footerGroups, legalLinks } from './content';

const emailPattern = /([A-Z0-9._%+-]+@oneflow\.id)/gi;
const emailAddress = /^[A-Z0-9._%+-]+@oneflow\.id$/i;

function renderInlineText(text) {
  return text.split(emailPattern).map((part, index) =>
    emailAddress.test(part) ? (
      <a
        className="oneflow-legal-inline-link"
        href={`mailto:${part}`}
        key={`${part}-${index}`}
      >
        {part}
      </a>
    ) : (
      part
    ),
  );
}

export default function LegalPage({
  entry,
  currentPath,
  navigationLinks = legalLinks,
  navigationLabel = 'Halaman legal',
}) {
  return (
    <div className="oneflow-legal-page">
      <header className="oneflow-legal-header">
        <div className="oneflow-legal-nav">
          <Link className="oneflow-legal-brand" href="/" aria-label="Kembali ke Oneflow.id">
            <Image
              src="/brand/oneflow-main-logo.png"
              alt="Oneflow.id"
              width={176}
              height={50}
              priority
            />
          </Link>
          <nav className="oneflow-legal-nav-links" aria-label={navigationLabel}>
            {navigationLinks.map((link) => (
              <Link
                key={link.href}
                className={`oneflow-legal-nav-link${link.href === currentPath ? ' is-active' : ''}`}
                href={link.href}
                aria-current={link.href === currentPath ? 'page' : undefined}
              >
                {link.label}
              </Link>
            ))}
          </nav>
          <Link className="oneflow-legal-cta" href="/#pricing">
            Lihat paket
          </Link>
        </div>
      </header>
      <main className="oneflow-legal-main">
        <section className="oneflow-legal-hero">
          <div className="oneflow-legal-badge">{entry.badge}</div>
          <h1>{entry.title}</h1>
          <p>{entry.description}</p>
          <div className="oneflow-legal-updated">
            Terakhir diperbarui <strong>{entry.updatedAt}</strong>
          </div>
        </section>
        <div className="oneflow-legal-grid">
          <aside className="oneflow-legal-toc">
            <span>Daftar isi</span>
            <nav aria-label="Daftar isi">
              {entry.sections.map((section) => (
                <a href={`#${section.id}`} key={section.id}>
                  {section.title}
                </a>
              ))}
            </nav>
          </aside>
          <article className="oneflow-legal-card">
            {entry.sections.map((section) => (
              <section className="oneflow-legal-section" id={section.id} key={section.id}>
                <h2>{section.title}</h2>
                {section.paragraphs?.map((paragraph) => (
                  <p key={paragraph}>{renderInlineText(paragraph)}</p>
                ))}
                {section.bullets && (
                  <ul>
                    {section.bullets.map((bullet) => (
                      <li key={bullet}>{renderInlineText(bullet)}</li>
                    ))}
                  </ul>
                )}
                {section.action && (
                  <Link className="oneflow-legal-action" href={section.action.href}>
                    {section.action.label}
                    <span aria-hidden="true">-&gt;</span>
                  </Link>
                )}
              </section>
            ))}
          </article>
        </div>
      </main>
      <footer className="oneflow-legal-footer">
        <div className="oneflow-legal-footer-inner">
          <div className="oneflow-legal-footer-brand">
            <Image src="/brand/oneflow-main-logo.png" alt="Oneflow.id" width={160} height={46} />
            <p>AI CS WhatsApp dan workflow bisnis dalam satu workspace.</p>
          </div>
          <div className="oneflow-legal-footer-groups">
            {footerGroups.map((group) => (
              <div className="oneflow-legal-footer-group" key={group.title}>
                <h2>{group.title}</h2>
                <nav className="oneflow-legal-footer-links" aria-label={group.title}>
                  {group.links.map((link) => (
                    <Link href={link.href} key={`${group.title}-${link.label}`}>
                      {link.label}
                    </Link>
                  ))}
                </nav>
              </div>
            ))}
          </div>
        </div>
        <p className="oneflow-legal-copyright">&copy; 2026 PT ONEFLOW TEKNOLOGI NUSANTARA. All rights reserved.</p>
      </footer>
    </div>
  );
}
