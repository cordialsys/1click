export const getApiHost = () => {
  if (typeof window !== "undefined") {
    // Client-side: use environment variable or same host
    return process.env.NEXT_PUBLIC_API_HOST || window.location.origin;
  }
  // Server-side: use environment variable or empty (relative URLs)
  return process.env.NEXT_PUBLIC_API_HOST || "";
};