import LegalPage from '../legal-pages/LegalPage';
import { informationDocuments, informationLinks } from '../legal-pages/content';

export const metadata = {
  title: 'Kontak | Oneflow.id',
  description: informationDocuments.contact.description,
};

export default function ContactPage() {
  return (
    <LegalPage
      entry={informationDocuments.contact}
      currentPath="/contact"
      navigationLinks={informationLinks}
      navigationLabel="Informasi Oneflow"
    />
  );
}
