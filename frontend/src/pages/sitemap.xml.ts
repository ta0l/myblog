// src/pages/sitemap.xml.ts
export async function GET({ request }) {
  // 1. 请求 Go 后端，获取所有文章的简略列表（只需 ID 和 更新时间）
  // 假设你给后端加了一个 /api/articles/sitemap 接口，或者直接用现有的列表接口
  const res = await fetch('http://api:8888/api/articles?limit=1000');
  const data = await res.json();
  const articles = data.articles || [];

  // 2. 获取你当前的网站域名 (生产环境下应该是你真实的域名)
  const siteUrl = 'https://你的真实域名.com';

  // 3. 拼接 XML 字符串
  const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
      <url>
        <loc>${siteUrl}/</loc>
        <priority>1.0</priority>
      </url>
      <url>
        <loc>${siteUrl}/explore</loc>
        <priority>0.8</priority>
      </url>
      ${articles.map(article => `
        <url>
          <loc>${siteUrl}/post/${article.id}</loc>
          <lastmod>${article.created_at.split('T')[0]}</lastmod>
          <priority>0.7</priority>
        </url>
      `).join('')}
    </urlset>`;

  // 4. 返回原生的 XML 响应
  return new Response(sitemap, {
    headers: {
      'Content-Type': 'application/xml',
      'Cache-Control': 'public, max-age=3600' // 让 CDN 或浏览器缓存 1 小时
    }
  });
}