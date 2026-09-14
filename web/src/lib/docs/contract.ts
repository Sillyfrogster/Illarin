import type { Doc } from "./read-doc";

export type Schema = {
  $ref?: string;
  type?: string | string[];
  format?: string;
  pattern?: string;
  minLength?: number;
  maxLength?: number;
  minimum?: number;
  enum?: unknown[];
  nullable?: boolean;
  required?: string[];
  properties?: Record<string, Schema>;
  additionalProperties?: boolean | Schema;
  items?: Schema;
  anyOf?: Schema[];
  oneOf?: Schema[];
  allOf?: Schema[];
};

type Operation = {
  requestBody?: {
    content?: Record<string, { schema?: Schema }>;
  };
  responses?: Record<string, { content?: Record<string, { schema?: Schema }> }>;
};

export type Contract = {
  paths: Record<string, Record<string, Operation>>;
  webhooks?: Record<string, Record<string, Operation>>;
  components: { schemas: Record<string, Schema> };
};

export type HttpExample = {
  method: string;
  path: string;
  headers: Record<string, string>;
  body: unknown;
};

const API_PREFIX = "/api";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const REF_PREFIX = "#/components/schemas/";

/** Reads the request line, headers and JSON body of one http fence. */
export function readHttpExample(source: string): HttpExample {
  const [head, ...rest] = source.split("\n\n");
  const [requestLine, ...headerLines] = head.split("\n");
  const [method, target] = requestLine.trim().split(/\s+/);
  const headers: Record<string, string> = {};
  for (const line of headerLines) {
    const at = line.indexOf(":");
    if (at < 0) continue;
    headers[line.slice(0, at).trim().toLowerCase()] = line.slice(at + 1).trim();
  }
  const raw = rest.join("\n\n").trim();
  return {
    method: method.toLowerCase(),
    path: target.startsWith(API_PREFIX)
      ? target.slice(API_PREFIX.length)
      : target,
    headers,
    body: raw === "" ? undefined : JSON.parse(raw),
  };
}

/** Names every example in a doc that the contract does not agree with. */
export function checkDoc(doc: Doc, contract: Contract): string[] {
  const problems: string[] = [];
  for (const block of doc.blocks) {
    if (block.kind === "endpoint") {
      const example = readHttpExample(`${block.method} ${block.path}`);
      problems.push(...checkHttp(example, contract));
      continue;
    }
    if (block.kind !== "code") continue;
    if (block.language === "http") {
      problems.push(...checkHttp(readHttpExample(block.source), contract));
    }
    if (block.language === "json") {
      problems.push(...checkJson(block.label, block.source, contract));
    }
  }
  return problems;
}

function checkHttp(example: HttpExample, contract: Contract): string[] {
  const label = `${example.method.toUpperCase()} ${example.path}`;
  if (!example.path.startsWith("/v1/")) return checkOutbound(example, contract);
  const path = Object.keys(contract.paths).find((template) =>
    matchesTemplate(template, example.path),
  );
  if (!path) return [`${label}: the contract has no such path.`];
  const operation = contract.paths[path][example.method];
  if (!operation) return [`${label}: the contract has no such operation.`];
  if (example.body === undefined) return [];
  const schema = operation.requestBody?.content?.["application/json"]?.schema;
  if (!schema) return [`${label}: the contract takes no JSON body here.`];
  return check(example.body, schema, "body", contract).map(
    (problem) => `${label}: ${problem}`,
  );
}

function checkOutbound(example: HttpExample, contract: Contract): string[] {
  const label = `${example.method.toUpperCase()} ${example.path}`;
  const body = example.body as { type?: unknown } | undefined;
  const event = typeof body?.type === "string" ? body.type : "";
  const operation = contract.webhooks?.[event]?.[example.method];
  if (!operation) return [`${label}: the contract sends no such webhook.`];
  const schema = operation.requestBody?.content?.["application/json"]?.schema;
  if (!schema) return [`${label}: the contract gives that webhook no body.`];
  return check(example.body, schema, "body", contract).map(
    (problem) => `${label}: ${problem}`,
  );
}

function checkJson(
  label: string,
  source: string,
  contract: Contract,
): string[] {
  if (label === "json")
    return ["json: a json example names the schema it fits."];
  const schema = contract.components.schemas[label];
  if (!schema) return [`json ${label}: the contract has no such schema.`];
  return check(JSON.parse(source), schema, "", contract).map(
    (problem) => `json ${label}: ${problem}`,
  );
}

function matchesTemplate(template: string, path: string): boolean {
  const wanted = template.split("/");
  const given = path.split("/");
  if (wanted.length !== given.length) return false;
  return wanted.every((part, at) => part.startsWith("{") || part === given[at]);
}

function check(
  value: unknown,
  schema: Schema,
  path: string,
  contract: Contract,
): string[] {
  const shape = resolve(schema, contract);
  if (shape.anyOf || shape.oneOf) {
    const branches = shape.anyOf ?? shape.oneOf ?? [];
    const fits = branches.some(
      (branch) => check(value, branch, path, contract).length === 0,
    );
    return fits ? [] : [`${path || "it"} fits none of its shapes.`];
  }
  if (shape.allOf) {
    return shape.allOf.flatMap((branch) =>
      check(value, branch, path, contract),
    );
  }
  const types = typesOf(shape);
  if (value === null) {
    return shape.nullable || types.includes("null")
      ? []
      : [`${path || "it"} is not allowed to be null.`];
  }
  if (!Array.isArray(shape.type)) {
    return checkAs(types[0], value, shape, path, contract);
  }
  const type = types.find((listed) => fitsType(value, listed));
  if (!type) return [`${path || "it"} is not a ${types.join(" or ")}.`];
  return checkAs(type, value, shape, path, contract);
}

function typesOf(shape: Schema): string[] {
  if (Array.isArray(shape.type)) return shape.type;
  if (shape.type) return [shape.type];
  return shape.properties ? ["object"] : [];
}

function fitsType(value: unknown, type: string): boolean {
  switch (type) {
    case "object":
      return typeof value === "object" && !Array.isArray(value);
    case "array":
      return Array.isArray(value);
    case "integer":
      return Number.isInteger(value);
    default:
      return typeof value === type;
  }
}

function checkAs(
  type: string | undefined,
  value: unknown,
  shape: Schema,
  path: string,
  contract: Contract,
): string[] {
  switch (type) {
    case "object":
      return checkObject(value, shape, path, contract);
    case "array":
      return checkArray(value, shape, path, contract);
    case "string":
      return checkString(value, shape, path);
    case "integer":
      return Number.isInteger(value)
        ? checkMinimum(value as number, shape, path)
        : [`${path} is not an integer.`];
    case "number":
      return typeof value === "number"
        ? checkMinimum(value, shape, path)
        : [`${path} is not a number.`];
    case "boolean":
      return typeof value === "boolean" ? [] : [`${path} is not a boolean.`];
    case "null":
      return [`${path || "it"} is not null.`];
    default:
      return [];
  }
}

function checkMinimum(value: number, shape: Schema, path: string): string[] {
  return shape.minimum !== undefined && value < shape.minimum
    ? [`${path} is below ${shape.minimum}.`]
    : [];
}

function checkObject(
  value: unknown,
  shape: Schema,
  path: string,
  contract: Contract,
): string[] {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return [`${path || "it"} is not an object.`];
  }
  const fields = value as Record<string, unknown>;
  const problems: string[] = [];
  for (const [key, field] of Object.entries(fields)) {
    const inner = shape.properties?.[key];
    const here = path ? `${path}.${key}` : key;
    if (inner) {
      problems.push(...check(field, inner, here, contract));
      continue;
    }
    if (shape.additionalProperties === false) {
      problems.push(`${here} is not in the contract.`);
    }
  }
  for (const key of shape.required ?? []) {
    if (!(key in fields)) {
      problems.push(`${path ? `${path}.${key}` : key} is required.`);
    }
  }
  return problems;
}

function checkArray(
  value: unknown,
  shape: Schema,
  path: string,
  contract: Contract,
): string[] {
  if (!Array.isArray(value)) return [`${path} is not an array.`];
  if (!shape.items) return [];
  const items = shape.items;
  return value.flatMap((item, at) =>
    check(item, items, `${path}.${at}`, contract),
  );
}

function checkString(value: unknown, shape: Schema, path: string): string[] {
  if (typeof value !== "string") return [`${path} is not a string.`];
  if (shape.enum && !shape.enum.includes(value)) {
    return [`${path} is not one of ${shape.enum.join(", ")}.`];
  }
  if (shape.format === "uuid" && !UUID.test(value)) {
    return [`${path} is not a uuid.`];
  }
  if (shape.format === "date-time" && Number.isNaN(Date.parse(value))) {
    return [`${path} is not a date-time.`];
  }
  const length = [...value].length;
  if (shape.minLength !== undefined && length < shape.minLength) {
    return [`${path} is shorter than ${shape.minLength} characters.`];
  }
  if (shape.maxLength !== undefined && length > shape.maxLength) {
    return [`${path} is longer than ${shape.maxLength} characters.`];
  }
  if (shape.pattern && !new RegExp(shape.pattern, "u").test(value)) {
    return [`${path} does not match ${shape.pattern}.`];
  }
  return [];
}

function resolve(schema: Schema, contract: Contract): Schema {
  if (!schema.$ref) return schema;
  if (!schema.$ref.startsWith(REF_PREFIX)) {
    throw new Error(`Cannot follow ${schema.$ref}.`);
  }
  const name = schema.$ref.slice(REF_PREFIX.length);
  const found = contract.components.schemas[name];
  if (!found) throw new Error(`The contract has no schema ${name}.`);
  return resolve(found, contract);
}
