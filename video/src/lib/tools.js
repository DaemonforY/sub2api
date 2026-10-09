// Tool landing pages (/tools/:slug). Each one maps to a gallery category and a create-page preset.
// The data lives in tools.json so the Go server can read it too (page titles, sitemap): the build
// copies it to dist/seo-tools.json (vite.config.js).
import TOOLS from './tools.json'

export { TOOLS }

export const findTool = (slug) => TOOLS.find((t) => t.slug === slug)
