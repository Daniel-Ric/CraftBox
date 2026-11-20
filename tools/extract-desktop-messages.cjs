const fs = require("node:fs");
const path = require("node:path");
const ts = require("../ui/desktop/frontend/node_modules/typescript");

const root = path.resolve(__dirname, "..");
const frontend = path.join(root, "ui", "desktop", "frontend", "src");
const messages = new Map();
const attributes = new Set([
  "aria-label", "aria-description", "aria-valuetext", "aria-roledescription",
  "alt", "title", "placeholder", "data-tooltip", "label", "heading", "description",
  "emptyLabel", "emptyTitle", "emptyDescription", "tooltip", "summary",
]);
const nonDisplayAttributes = new Set([
  "className", "id", "key", "htmlFor", "value", "href", "src", "style",
  "aria-controls", "aria-describedby", "aria-labelledby",
]);
const namedText = /^(?:name|label|title|description|detail|tooltip|hint|summary|text|message|reason|queryHint|subtitle|caption|plural|singular|example|heading|tagline|synopsis|unit)$/i;
const entities = { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: "\u00a0", hellip: "…" };

function add(value, source, context, explicit = false) {
  const clean = value.replace(/&(#x[0-9a-f]+|#\d+|\w+);/gi, (token, entity) => {
    if (entity[0] !== "#") return entities[entity] ?? token;
    const code = entity[1].toLowerCase() === "x"
      ? parseInt(entity.slice(2), 16)
      : Number(entity.slice(1));
    return code <= 0x10ffff ? String.fromCodePoint(code) : token;
  }).replace(/\s+/g, " ").trim();
  if (!/[a-zA-ZÀ-ÿ]{2}/.test(clean) || clean.length > 1600 ||
      /^(?:https?:|data:|blob:|file:|wss?:|#[\da-f]{3,8}$)/i.test(clean) ||
      /^(?:[.\w@-]+[./\\:[\]_-]){1,}[\w./\\:@-]+$/.test(clean) ||
      /^(?:[A-Z_]{2,}|[a-z]+[A-Z][\w]*)$/.test(clean) ||
      /^(?:class|function|const|uniform|return|vec[234])\s/.test(clean) ||
      /(?:^|\s)#(?:include|ifdef|define|version)\b/.test(clean) ||
      /\b(?:gl_FragColor|gl_Position|sampler2D)\b[^]*;/.test(clean) ||
      /(?:^|\s)(?:hover|focus|aria-expanded|data-\[\w|group-data-\[\w|peer-disabled)[^\s]*:[^\s]+(?:\s|$)/.test(clean) ||
      /^\.[a-z][\w-]+\s+(?:audio|canvas|img|svg|video)$/.test(clean) ||
      /^(?:text|font|flex|grid|gap|p[xytrbl]?|m[xytrbl]?|bg|size|shadow|border|w|h|rounded|min|max)-\w/.test(clean)) return;
  if (!explicit && !/\s/u.test(clean) && !/^[A-ZÀ-Ý]/u.test(clean)) return;
  const refs = messages.get(clean) ?? [];
  if (refs.length < 5) refs.push([source, context]);
  messages.set(clean, refs);
}

function walk(dir, visit) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory() && !["i18n", "frontend", "node_modules"].includes(entry.name)) walk(file, visit);
    else if (entry.isFile()) visit(file);
  }
}

walk(frontend, (file) => {
  if (!/\.[cm]?tsx?$/.test(file) || /\.d\.ts$/.test(file)) return;
  const raw = fs.readFileSync(file, "utf8");
  const ast = ts.createSourceFile(file, raw, ts.ScriptTarget.Latest, true);
  const source = path.relative(frontend, file).replaceAll("\\", "/");
  const context = (node) => ast.getLineAndCharacterOfPosition(node.getStart(ast)).line + 1;
  const record = (node, value, explicit) => add(value, source, context(node), explicit);

  function isVisibleValue(node) {
    const parent = node.parent;
    if (ts.isJsxExpression(parent)) {
      return !ts.isJsxAttribute(parent.parent) || attributes.has(parent.parent.name.getText(ast));
    }
    if (ts.isParenthesizedExpression(parent)) return isVisibleValue(parent);
    if (ts.isConditionalExpression(parent)) {
      return node !== parent.condition && isVisibleValue(parent);
    }
    if (ts.isBinaryExpression(parent)) {
      const operator = parent.operatorToken.kind;
      if (operator === ts.SyntaxKind.BarBarToken || operator === ts.SyntaxKind.QuestionQuestionToken) {
        return isVisibleValue(parent);
      }
      if (operator === ts.SyntaxKind.AmpersandAmpersandToken && node === parent.right) {
        return isVisibleValue(parent);
      }
    }
    return false;
  }

  function inNonDisplayAttribute(node) {
    for (let parent = node.parent; parent; parent = parent.parent) {
      if (ts.isJsxAttribute(parent)) return nonDisplayAttributes.has(parent.name.getText(ast));
      if (ts.isJsxElement(parent) || ts.isJsxSelfClosingElement(parent) || ts.isFunctionLike(parent)) break;
    }
    return false;
  }

  function visit(node) {
    const parent = node.parent;
    if (ts.isJsxText(node)) {
      record(node, node.text, true);
    } else if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      if (ts.isConditionalExpression(parent) && ts.isJsxExpression(parent.parent) &&
        ts.isJsxAttribute(parent.parent.parent) &&
        parent.parent.parent.name.getText(ast) === "translate") return;
      if (ts.isJsxAttribute(parent)) {
        if (attributes.has(parent.name.getText(ast))) record(node, node.text, true);
      } else if (!ts.isImportDeclaration(parent) && !ts.isExportDeclaration(parent) &&
          !ts.isLiteralTypeNode(parent) && !ts.isCaseClause(parent) &&
          !ts.isElementAccessExpression(parent) && !ts.isImportTypeNode(parent) &&
          (!ts.isBinaryExpression(parent) || isVisibleValue(node)) &&
          !(ts.isPropertyAssignment(parent) && node === parent.name)) {
        const property = ts.isPropertyAssignment(parent) ? parent.name.getText(ast).replace(/^['"]|['"]$/g, "") : "";
        const explicit = namedText.test(property) || isVisibleValue(node) || ts.isConditionalExpression(parent);
        if (!property || explicit || /\s/.test(node.text)) record(node, node.text, explicit);
      }
    } else if (ts.isTemplateExpression(node) && !inNonDisplayAttribute(node)) {
      const value = node.head.text + node.templateSpans.map((span, index) => "{" + index + "}" + span.literal.text).join("");
      record(node, value, true);
    }
    if (ts.isJsxElement(node) || ts.isJsxFragment(node)) {
      let group = "";
      let placeholders = 0;
      function flush() {
        if (placeholders) record(node, group, true);
        group = "";
        placeholders = 0;
      }
      for (const child of node.children) {
        if (ts.isJsxText(child)) {
          group += child.text.replace(/\r?\n\s*/g, " ");
        } else if (ts.isJsxExpression(child) && child.expression) {
          const inner = child.expression;
          if (ts.isStringLiteral(inner)) group += inner.text;
          else if (!ts.isJsxElement(inner) && !ts.isJsxFragment(inner) && !ts.isJsxSelfClosingElement(inner))
            group += "{" + placeholders++ + "}";
          else flush();
        } else {
          flush();
        }
      }
      flush();
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
});

for (const dir of ["ui/desktop", "ui/cli", "subcommands", "internal", "utils", "handlers"]) {
  walk(path.join(root, dir), (file) => {
    if (!file.endsWith(".go") || file.endsWith("_test.go")) return;
    const raw = fs.readFileSync(file, "utf8");
    const source = path.relative(root, file).replaceAll("\\", "/");
    const pattern = /(?:\b(?:fmt\.Errorf|errors\.New|locale\.(?:Text|Format)|fmt\.Print(?:f|ln)|log(?:rus)?\.(?:Info|Warn|Error|Infof|Warnf|Errorf)|\w+\.(?:SendMessage|ShowAction|showMenu|commandError))\(\s*|\b(?:Title|DisplayName|Placeholder|Synopsis|Description|Message|CancelButton|ConfirmButton):\s*|\btitle\s*:?=\s*)("(?:\\.|[^"\\])*")/g;
    const patterns = /(?:ingame_|capture_commands|structure_commands)/.test(file)
      ? [pattern, /("(?:\\.|[^"\\])*")/g]
      : [pattern];
    for (const expression of patterns) {
      for (const match of raw.matchAll(expression)) {
        try {
          const decoded = JSON.parse(match[1]);
          const line = raw.slice(0, match.index).split("\n").length;
          if (decoded.startsWith("/") && !decoded.includes(" ")) continue;
          if (decoded.includes("§") || /(?:ingame_|capture_commands|structure_commands)/.test(file)) {
            for (const segment of decoded.split(/§[0-9a-z]|\n/gi))
              if (/\s|^[A-Z]/.test(segment)) add(segment, source, line, true);
          } else {
            add(decoded, source, line, true);
          }
        } catch {}
      }
    }
  });
}

for (const value of [
  "Language", "Application language", "Choose the language used throughout CraftBox.",
  "The language could not be loaded.", "CraftBox project", "Windows", "Application preferences",
]) add(value, "i18n", 0, true);

const ordered = [...messages.entries()].sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0);
const output = JSON.stringify(Object.fromEntries(ordered), null, 2) + "\n";
if (process.argv[2] === "--check") {
  const locales = ["zh-Hans", "es", "de", "fr", "pt", "ja"];
  const signatures = (value) => (value.match(/\{\d+\}|\{\{[^{}]+\}\}|%(?:\[\d+\])?[-+0#.\d]*[bcdeEfFgGopqstTuvwxX%]/g) ?? []).sort().join("\0");
  const known = new Map(ordered.map(([source, locations]) => [source, { signature: signatures(source), locations }]));
  for (const locale of locales) {
    const catalog = JSON.parse(fs.readFileSync(path.join(root, "locale", "messages", locale + ".json"), "utf8"));
    let failures = 0;
    for (const [source, { signature, locations }] of known) {
      const value = catalog[source];
      if (typeof value === "string" && value.trim() && signatures(value) === signature) continue;
      failures++;
      if (failures <= 12) process.stderr.write(locale + " " + locations[0].join(":") + " " + JSON.stringify(source) + "\n");
    }
    if (failures) process.exitCode = 1;
    process.stdout.write(locale + " " + (known.size - failures) + "/" + known.size + " messages checked\n");
  }
} else if (process.argv[2]) {
  fs.writeFileSync(process.argv[2], output);
} else {
  process.stdout.write(output);
}
