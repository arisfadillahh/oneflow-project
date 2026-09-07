import LegalPage from '../legal-pages/LegalPage';
import { legalDocuments } from '../legal-pages/content';

export const metadata = {
  title: 'Refund & Pembatalan | Oneflow.id',
  description: legalDocuments.refundPolicy.description,
};

export default function RefundPolicyPage() {
  return <LegalPage entry={legalDocuments.refundPolicy} currentPath="/refund-policy" />;
}
