import { copyFile, mkdir, readdir } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const categories = {
  languages: [
    "python", "typescript", "javascript", "c", "csharp", "cplusplus", "java",
    "kotlin", "go", "rust", "php", "ruby", "swift", "dart", "scala",
    "haskell", "elixir", "lua", "r", "perl", "cobol", "fortran", "clojure",
    "fsharp", "ocaml", "julia", "nim", "zig", "bash", "powershell", "groovy",
    "solidity", "assembly", "matlab",
  ],
  frontend: [
    "react", "nextjs", "vuejs", "nuxt", "angular", "svelte", "sveltekit",
    "astro", "remix", "gatsby", "solidjs", "qwik", "lit", "htmx", "alpinejs",
  ],
  backend: [
    "nodejs", "express", "nestjs", "fastapi", "django", "flask", "springboot",
    "laravel", "rails", "symfony", "phoenix", "gin", "dotnet", "bun", "deno",
    "hono", "nextjs", "nuxt", "sveltekit", "astro", "remix", "nginx", "caddy",
  ],
  styling: [
    "css3", "tailwindcss", "sass", "styled-components", "bootstrap", "shadcnui",
    "radixui",
  ],
  databases: [
    "postgresql", "mysql", "mariadb", "sqlite", "mongodb", "redis", "dynamodb",
    "oracle", "sqlserver", "cockroachdb", "cassandra", "neo4j", "influxdb",
    "elasticsearch", "supabase", "firebase", "planetscale", "turso", "neon",
  ],
  "orms-data-access": [
    "prisma", "drizzle", "typeorm", "sequelize", "sqlalchemy", "hibernate",
  ],
  "ai-machine-learning": [
    "tensorflow", "pytorch", "keras", "scikitlearn", "huggingface", "openai",
    "anthropic", "ollama", "langchain", "cuda", "onnx", "jupyter", "pandas",
    "numpy", "matplotlib", "opencv", "spacy", "mlflow", "wandb",
  ],
  "cloud-hosting": [
    "aws", "gcp", "azure", "digitalocean", "cloudflare", "vercel", "netlify",
    "heroku", "railway", "render", "flyio", "hetzner", "firebase", "supabase",
    "planetscale", "neon",
  ],
  "devops-infrastructure": [
    "docker", "kubernetes", "terraform", "ansible", "pulumi", "helm", "nginx",
    "caddy", "jenkins", "githubactions", "gitlabci", "circleci", "argocd",
    "prometheus", "grafana", "datadog", "newrelic", "sentry", "loki", "vault",
  ],
  "version-control": ["git", "github", "gitlab", "bitbucket"],
  "build-package-tools": [
    "vite", "webpack", "esbuild", "rollup", "turbo", "nx", "npm", "pnpm",
    "yarn", "bun", "make", "gradle", "maven", "cargo",
  ],
  testing: [
    "jest", "vitest", "cypress", "playwright", "selenium", "pytest",
    "testing-library", "mocha", "chai", "k6",
  ],
  mobile: [
    "reactnative", "expo", "flutter", "swift", "kotlin", "xamarin", "capacitor",
    "ionic", "android", "xcode",
  ],
  "apis-messaging": [
    "graphql", "websockets", "trpc", "mqtt", "kafka", "rabbitmq", "postman",
    "insomnia",
  ],
  "security-networking": [
    "kalilinux", "wireshark", "metasploit", "openssl", "cloudflare", "vault",
  ],
  "operating-systems": [
    "linux", "archlinux", "ubuntu", "debian", "fedora", "macos", "windows",
    "android", "freebsd",
  ],
  "ides-editors": [
    "vscode", "neovim", "vim", "emacs", "intellij", "webstorm", "pycharm",
    "goland", "xcode", "zed", "cursor", "jupyter",
  ],
  "design-productivity": [
    "figma", "postman", "insomnia", "notion", "jira", "slack", "obsidian",
  ],
  "brands-companies": [
    "google", "meta", "apple", "amazon", "netflix", "linuxfoundation", "mozilla",
    "jetbrains",
  ],
  "blockchain-web3": [
    "ethereum", "solana", "bitcoin", "web3js", "ethersjs", "ipfs", "solidity",
  ],
};

const logosDirectory = path.join(process.cwd(), "public", "logos");
const rootLogos = (await readdir(logosDirectory))
  .filter((entry) => entry.endsWith(".svg"))
  .map((entry) => entry.slice(0, -4));
const categorizedLogos = new Set(Object.values(categories).flat());
const uncategorized = rootLogos.filter((logo) => !categorizedLogos.has(logo));
const missing = [...categorizedLogos].filter((logo) => !rootLogos.includes(logo));

if (uncategorized.length > 0 || missing.length > 0) {
  throw new Error(
    [
      uncategorized.length > 0 && `Uncategorized logos: ${uncategorized.join(", ")}`,
      missing.length > 0 && `Missing source logos: ${missing.join(", ")}`,
    ]
      .filter(Boolean)
      .join("\n"),
  );
}

for (const [category, logos] of Object.entries(categories)) {
  const categoryDirectory = path.join(logosDirectory, category);
  await mkdir(categoryDirectory, { recursive: true });

  await Promise.all(
    logos.map((logo) =>
      copyFile(
        path.join(logosDirectory, `${logo}.svg`),
        path.join(categoryDirectory, `${logo}.svg`),
      ),
    ),
  );
}

console.log(
  `Categorized ${rootLogos.length} logos across ${Object.keys(categories).length} folders.`,
);
