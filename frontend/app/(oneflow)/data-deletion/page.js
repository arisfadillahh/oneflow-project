import LegalPage from '../legal-pages/LegalPage';
import { legalDocuments } from '../legal-pages/content';

export const metadata = {
  title: 'Penghapusan Data | Oneflow.id',
  description: legalDocuments.dataDeletion.description,
};

export default function DataDeletionPage() {
  return <LegalPage entry={legalDocuments.dataDeletion} currentPath="/data-deletion" />;
}
