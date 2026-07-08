export default {
  async fetch(request, env) {
    // 1. Verify custom authorization header/token to prevent abuse
    const authHeader = request.headers.get("X-WeMall-Proxy-Secret");
    if (!env.PROXY_SECRET || authHeader !== env.PROXY_SECRET) {
      return new Response("Unauthorized", { status: 401 });
    }

    // 2. Parse target URL
    const url = new URL(request.url);
    const targetUrl = `https://developers.ecocash.co.zw${url.pathname}${url.search}`;

    // 3. Clone request headers (filtering out host header) and body
    const headers = new Headers(request.headers);
    headers.delete("host"); // Cloudflare will set correct host for targetUrl
    headers.delete("X-WeMall-Proxy-Secret");
    
    // Spoof User-Agent to bypass WAF bot protection
    headers.set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36");

    const newRequest = new Request(targetUrl, {
      method: request.method,
      headers: headers,
      body: request.method === "POST" || request.method === "PUT" ? request.body : null,
      redirect: "follow"
    });

    // 4. Perform the fetch and return response
    return fetch(newRequest);
  }
}
