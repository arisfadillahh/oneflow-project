import LegalPage from '../legal-pages/LegalPage';
import { legalDocuments } from '../legal-pages/content';

export const metadata = {
  title: 'Kebijakan Privasi | Oneflow.id',
  description: legalDocuments.privacy.description,
};

export default function PrivacyPage() {
  return <LegalPage entry={legalDocuments.privacy} currentPath="/privacy" />;
}
