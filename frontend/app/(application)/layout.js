import "./globals.css";

export const metadata = {
  title: "Oneflow.id | Dashboard",
  description: "Operations dashboard for multifunction AI chatbot, support, sales, and AI policy.",
  icons: {
    icon: [{ url: "/brand/oneflow-icon.png", type: "image/png" }],
    shortcut: "/brand/oneflow-icon.png",
    apple: "/brand/oneflow-icon.png",
  },
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Poppins:wght@600;700&display=swap" rel="stylesheet" />
      </head>
      <body suppressHydrationWarning>{children}</body>
    </html>
  );
}
