import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Standalone output for a minimal Docker runtime image (see
  // ../docker-compose.yml and Dockerfile) -- bundles only the files
  // next start actually needs instead of the full node_modules tree.
  output: "standalone",
};

export default nextConfig;
