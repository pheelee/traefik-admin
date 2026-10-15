import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const args = process.argv.slice(2);
if (args.length > 1 || (args.length === 1 && args[0] !== "--check")) {
  console.error("Usage: node scripts/vendor-assets.mjs [--check]");
  process.exit(2);
}
const check = args[0] === "--check";
const assets = [
  ["vue/dist/vue.min.js", "js/vue.min.js"],
  ["materialize-css/dist/js/materialize.min.js", "js/materialize.min.js"],
  ["materialize-css/dist/css/materialize.min.css", "css/vendor/materialize.min.css"],
];

for (const [source, destination] of assets) {
  const content = await readFile(resolve(root, "node_modules", source));
  const target = resolve(root, "internal/server/webrootSrc", destination);
  if (check) {
    const existing = await readFile(target).catch((error) => {
      if (error.code === "ENOENT") return null;
      throw error;
    });
    if (!existing || !content.equals(existing)) {
      console.error(`Stale or missing asset: ${destination}. Run npm run vendor and commit the result.`);
      process.exitCode = 1;
    }
  } else {
    await mkdir(dirname(target), { recursive: true });
    await writeFile(target, content);
    console.log(`Copied ${destination}`);
  }
}
if (check && !process.exitCode) console.log("Embedded assets match the installed packages.");
