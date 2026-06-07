import { Inter } from 'next/font/google';
import './globals.css';
import ClientWrapper from '@/components/ClientWrapper';

const inter = Inter({ subsets: ['latin'] });

export const metadata = {
  title: {
    template: '%s | LedgerPro',
    default: 'LedgerPro Dashboard',
  },
  description: 'FAANG-grade double-entry ledger API and Dashboard.',
  openGraph: {
    title: 'LedgerPro Dashboard',
    description: 'FAANG-grade double-entry ledger API and Dashboard.',
    url: 'https://ledger-pro-api.onrender.com',
    siteName: 'LedgerPro',
    locale: 'en_US',
    type: 'website',
  },
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" className="h-full antialiased dark">
      <body className={`${inter.className} h-full overflow-hidden bg-brand-bg-primary text-brand-text-primary`}>
        <ClientWrapper>{children}</ClientWrapper>
      </body>
    </html>
  );
}
