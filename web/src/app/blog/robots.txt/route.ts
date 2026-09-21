import { blogAddress } from "@/lib/blog-address";
import { BLOG_HOME } from "@/lib/blog-paths";

export const dynamic = "force-dynamic";

function blogRobots(): string {
  return [
    "User-Agent: *",
    "Allow: /",
    `Sitemap: ${blogAddress(`${BLOG_HOME}/sitemap.xml`)}`,
    "",
  ].join("\n");
}

export function GET(): Response {
  return new Response(blogRobots(), {
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}
