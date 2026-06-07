/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  compress: true,
  async rewrites() {
    return [
      {
        source: '/v1/:path*',
        // Always proxy to the production backend, even in local development
        destination: 'https://ledger-pro-api.onrender.com/v1/:path*'
      }
    ];
  },
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Content-Security-Policy',
            value: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://ledger-pro-api.onrender.com; img-src 'self' data: https://r2.cloudflarestorage.com; frame-ancestors 'none';"
          }
        ]
      }
    ];
  }
};

export default nextConfig;
