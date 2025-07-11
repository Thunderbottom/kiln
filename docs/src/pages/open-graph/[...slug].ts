import { getCollection } from 'astro:content'
import { OGImageRoute } from 'astro-og-canvas'

// Get all entries from the `docs` content collection.
const entries = await getCollection('docs')

// Map the entry array to an object with the page ID as key and the
// frontmatter data as value.
const pages = Object.fromEntries(entries.map(({ data, id }) => [id, { data }]))

export const { getStaticPaths, GET } = OGImageRoute({
  // Pass down the documentation pages.
  pages,
  // Define the name of the parameter used in the endpoint path, here `slug`
  // as the file is named `[...slug].ts`.
  param: 'slug',
  // Define a function called for each page to customize the generated image.
  getImageOptions: (_id, page: (typeof pages)[number]) => {
    return {
      // Use the page title and description as the image title and description.
      title: page.data.title,
      description: page.data.description,
      logo: {
        path: "./src/pages/open-graph/images/logo.png",
        size: [300],
      },
      border: { width: 32, side: 'inline-start' },
      padding: 80,
      bgImage: {
        path: "./src/pages/open-graph/images/og-background.png",
      },
      font: {
        title: {
          families: ["Hanken Grotesk"],
          color: [255, 255, 255],
          size: 72,
          weight: "bold",
        },
        description: {
          families: ["Hanken Grotesk"],
          color: [191, 193, 201],
          size: 38,
        },
      },
      fonts: [
        "./src/pages/open-graph/fonts/HankenGrotesk-Regular.ttf",
        "./src/pages/open-graph/fonts/HankenGrotesk-Bold.ttf",
        "./src/pages/open-graph/fonts/HankenGrotesk-Light.ttf",
      ],
    }
  },
})
