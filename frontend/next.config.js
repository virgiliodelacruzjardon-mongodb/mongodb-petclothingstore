/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Keep container builds resilient for this demo.
  eslint: { ignoreDuringBuilds: true },
  typescript: { ignoreBuildErrors: true },
  images: {
    remotePatterns: [{ protocol: "https", hostname: "picsum.photos" }],
  },
};

module.exports = nextConfig;
