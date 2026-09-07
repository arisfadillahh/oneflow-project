import LegalPage from '../legal-pages/LegalPage';
import { informationDocuments, informationLinks } from '../legal-pages/content';

export const metadata = {
  title: 'Tentang Kami | Oneflow.id',
  description: informationDocuments.about.description,
};

export default function AboutPage() {
  return (
    <LegalPage
      entry={informationDocuments.about}
      currentPath="/about"
      navigationLinks={informationLinks}
      navigationLabel="Informasi Oneflow"
    />
  );
}
