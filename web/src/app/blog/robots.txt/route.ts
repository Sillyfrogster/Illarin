import { blogAddress } from "@/lib/blog-address";

export const dynamic = "force-dynamic";

export function blogRobots(): string {
  return [
    "User-Agent: *",
    "Allow: /",
    `Sitemap: ${blogAddress("/sitemap.xml")}`,
    "",
  ].join("\n");
}

export function GET(): Response {
  return new Response(blogRobots(), {
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}
