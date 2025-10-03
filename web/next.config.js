/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  trailingSlash: true,
  distDir: 'out',
  basePath: process.env.NODE_ENV === 'production' ? '' : '',
  images: {
    unoptimized: true
  },
  env: {
    NEXT_PUBLIC_API_HOST: process.env.NEXT_PUBLIC_API_HOST || ''
  }
}

module.exports = nextConfig