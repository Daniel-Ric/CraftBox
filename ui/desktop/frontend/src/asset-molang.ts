type Node = { op: string; value?: string; args: Node[] };
export type MolangContext = Record<string, number>;
const cache = new Map<string, Node>();
const precedence: Record<string, number> = {
  "??": 1,
  "||": 2,
  "&&": 3,
  "==": 4,
  "!=": 4,
  "<": 5,
  ">": 5,
  "<=": 5,
  ">=": 5,
  "+": 6,
  "-": 6,
  "*": 7,
  "/": 7,
  "%": 7,
};
const canonical = (name: string) =>
  name
    .toLowerCase()
    .replace(/^q\./, "query.")
    .replace(/^v\./, "variable.")
    .replace(/^c\./, "context.");
function parse(source: string): Node {
  const saved = cache.get(source);
  if (saved) return saved;
  if (source.length > 2048) throw new Error("Expression exceeds preview limit");
  const tokens =
    source
      .toLowerCase()
      .match(
        /\d*\.?\d+(?:e[+-]?\d+)?|[a-z_][\w.]*|'[^']*'|\?\?|&&|\|\||==|!=|<=|>=|[()+*/%,?:!<>-]/g,
      ) ?? [];
  let i = 0;
  function expression(min = 0): Node {
    const token = tokens[i++];
    if (!token) throw new Error("Incomplete expression");
    let left: Node;
    if (token === "-" || token === "+" || token === "!")
      left = { op: "unary" + token, args: [expression(8)] };
    else if (token === "(") {
      left = expression();
      if (tokens[i++] !== ")") throw new Error("Missing closing parenthesis");
    } else if (tokens[i] === "(") {
      i++;
      const args: Node[] = [];
      if (tokens[i] !== ")")
        do {
          args.push(expression());
          if (tokens[i] !== ",") break;
          i++;
        } while (i < tokens.length);
      if (tokens[i++] !== ")") throw new Error("Missing call parenthesis");
      left = { op: "call", value: token, args };
    } else left = { op: "value", value: token, args: [] };
    while (i < tokens.length) {
      const op = tokens[i];
      const p = precedence[op];
      if (p === undefined || p < min) break;
      i++;
      left = { op, args: [left, expression(p + 1)] };
    }
    if (min === 0 && tokens[i] === "?") {
      i++;
      const yes = expression();
      if (tokens[i++] !== ":") throw new Error("Missing alternative");
      left = { op: "?:", args: [left, yes, expression()] };
    }
    return left;
  }
  const result = expression();
  if (i !== tokens.length) throw new Error("Unsupported expression");
  if (cache.size > 512) cache.clear();
  cache.set(source, result);
  return result;
}
export function molang(
  value: unknown,
  context: MolangContext,
  fallback = 0,
): number {
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "number")
    return Number.isFinite(value) ? value : fallback;
  if (typeof value !== "string") return fallback;
  function run(n: Node): number {
    const a = () => run(n.args[0]),
      b = () => run(n.args[1]);
    if (n.op === "value") {
      const v = n.value!;
      if (v === "true") return 1;
      if (v === "false") return 0;
      if (!isNaN(Number(v))) return Number(v);
      return context[canonical(v)] ?? 0;
    }
    if (n.op === "call") {
      const args = n.args.map(run),
        name = canonical(n.value!);
      const rad = Math.PI / 180;
      const functions: Record<string, (...v: number[]) => number> = {
        "math.sin": (v) => Math.sin(v * rad),
        "math.cos": (v) => Math.cos(v * rad),
        "math.abs": Math.abs,
        "math.floor": Math.floor,
        "math.ceil": Math.ceil,
        "math.round": Math.round,
        "math.sqrt": Math.sqrt,
        "math.pow": Math.pow,
        "math.min": Math.min,
        "math.max": Math.max,
        "math.mod": (a, b) => ((a % b) + b) % b,
        "math.clamp": (v, a, b) => Math.max(a, Math.min(b, v)),
        "math.lerp": (a, b, t) => a + (b - a) * t,
        "math.pi": () => Math.PI,
      };
      return functions[name]?.(...args) ?? context[name] ?? 0;
    }
    switch (n.op) {
      case "unary-":
        return -a();
      case "unary+":
        return a();
      case "unary!":
        return +a() === 0 ? 1 : 0;
      case "+":
        return a() + b();
      case "-":
        return a() - b();
      case "*":
        return a() * b();
      case "/":
        return b() ? a() / b() : 0;
      case "%":
        return a() % b();
      case "<":
        return +(a() < b());
      case ">":
        return +(a() > b());
      case "<=":
        return +(a() <= b());
      case ">=":
        return +(a() >= b());
      case "==":
        return +(a() === b());
      case "!=":
        return +(a() !== b());
      case "&&":
        return a() ? b() : 0;
      case "||":
        return a() || b();
      case "??": {
        const key = n.args[0].value;
        return key && context[canonical(key)] === undefined ? b() : a();
      }
      case "?:":
        return a() ? run(n.args[1]) : run(n.args[2]);
      default:
        return 0;
    }
  }
  try {
    const result = run(parse(value));
    return Number.isFinite(result) ? result : fallback;
  } catch {
    return fallback;
  }
}
export function molangScripts(scripts: unknown, context: MolangContext) {
  for (const script of Array.isArray(scripts) ? scripts : [scripts]) {
    if (typeof script !== "string") continue;
    for (const statement of script.split(";").slice(0, 64)) {
      const match = statement.match(
        /^\s*((?:variable|v)\.[\w]+)\s*=\s*([^=].*)$/,
      );
      if (match) context[canonical(match[1])] = molang(match[2], context);
    }
  }
}
export function previewContext(time: number): MolangContext {
  return {
    "query.anim_time": time,
    "query.life_time": time,
    "query.time_stamp": time * 20,
    "query.is_alive": 1,
    "query.is_item_name_any": 1,
    "query.is_on_ground": 1,
    "variable.entity_scale": 1,
    "query.head_x_rotation": 0,
    "query.head_y_rotation": 0,
    "query.body_y_rotation": 0,
  };
}
