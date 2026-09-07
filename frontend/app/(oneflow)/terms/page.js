import LegalPage from '../legal-pages/LegalPage';
import { legalDocuments } from '../legal-pages/content';

export const metadata = {
  title: 'Syarat & Ketentuan | Oneflow.id',
  description: legalDocuments.terms.description,
};

export default function TermsPage() {
  return <LegalPage entry={legalDocuments.terms} currentPath="/terms" />;
}
