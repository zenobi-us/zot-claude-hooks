import { createInterface } from "node:readline";
import { homedir } from "node:os";
import { join, resolve } from "node:path";
import { appendFileSync } from "node:fs";
import { access, readFile } from "node:fs/promises";
import { Crust } from "@crustjs/core";

type Json = Record<string, unknown>;

type Hook = {
  event: string;
  command: string;
  matcher: string;
  timeout: number;
  source: string;
};

type HookResult = {
  code: number;
  output: string;
  response?: Json;
};

const NAME = "zot-cluade-hooks";
const VERSION = "0.1.0";
const DEFAULT_TIMEOUT = 10_000;

function trace(direction: "in" | "out", frame: Json): void {
  const path = process.env.ZOT_HOOKS_PROTOCOL_TRACE;
  if (!path) return;
  try {
    appendFileSync(path, `${JSON.stringify({ direction, frame })}\n`);
  } catch (error) {
    console.error(`[${NAME}] cannot write protocol trace: ${error}`);
  }
}

function send(frame: Json): void {
  trace("out", frame);
  process.stdout.write(`${JSON.stringify(frame)}\n`);
}

function log(message: string): void {
  console.error(`[${NAME}] ${message}`);
}

function asObject(value: unknown): Json | undefined {
  return value && typeof value === "object" && !Array.isArray(value)
    ? value as Json
    : undefined;
}

function asArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function asString(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function configPaths(cwd: string): string[] {
  const paths = [
    join(homedir(), ".claude", "settings.json"),
    resolve(cwd, ".claude", "settings.json"),
    resolve(cwd, ".claude", "settings.local.json"),
    process.env.ZOT_USER_CONFIG_DIR
      ? resolve(cwd, process.env.ZOT_USER_CONFIG_DIR, "zot-cluade-hooks.json")
      : undefined,
    resolve(cwd, ".zot", "zot-cluade-hooks.json"),
    resolve(cwd, ".zot", "zot-cluade-hooks.local.json"),
    process.env.ZOT_HOOKS_PATH
      ? resolve(cwd, process.env.ZOT_HOOKS_PATH)
      : undefined,
  ];
  return [...new Set(paths.filter((path): path is string => Boolean(path)))];
}

async function loadHooks(cwd: string): Promise<Hook[]> {
  const hooks: Hook[] = [];
  for (const source of configPaths(cwd)) {
    try {
      await access(source);
    } catch {
      continue;
    }

    let document: Json;
    try {
      document = asObject(JSON.parse(await readFile(source, "utf8"))) ?? {};
    } catch (error) {
      log(`cannot read ${source}: ${error}`);
      continue;
    }

    const definitions = asObject(document.hooks);
    if (!definitions) {
      log(`ignoring ${source}: hooks must be an object`);
      continue;
    }

    for (const [event, groupsValue] of Object.entries(definitions)) {
      for (const groupValue of asArray(groupsValue)) {
        const group = asObject(groupValue);
        if (!group) continue;
        const matcher = asString(group.matcher, ".*") || ".*";
        for (const itemValue of asArray(group.hooks)) {
          const item = asObject(itemValue);
          if (!item || item.type !== "command") {
            // TODO: Add prompt and other hook types when zot defines them.
            continue;
          }
          const command = asString(item.command);
          if (!command) {
            log(`ignoring command without text in ${source}`);
            continue;
          }
          const timeoutValue = Number(item.timeout ?? DEFAULT_TIMEOUT / 1000);
          hooks.push({
            event,
            command,
            matcher,
            timeout: Math.max(100, (Number.isFinite(timeoutValue) ? timeoutValue : 10) * 1000),
            source,
          });
        }
      }
    }
  }
  return hooks;
}

function matches(hook: Hook, toolName: string): boolean {
  try {
    return new RegExp(hook.matcher).test(toolName);
  } catch (error) {
    log(`invalid matcher in ${hook.source}: ${error}`);
    return false;
  }
}

async function runHook(hook: Hook, payload: Json, cwd: string): Promise<HookResult> {
  const shell = process.platform === "win32"
    ? ["cmd.exe", "/d", "/s", "/c", hook.command]
    : ["sh", "-c", hook.command];
  const processHandle = Bun.spawn(shell, {
    cwd,
    stdin: "pipe",
    stdout: "pipe",
    stderr: "pipe",
  });

  processHandle.stdin.write(JSON.stringify(payload));
  processHandle.stdin.end();

  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<HookResult>((resolveResult) => {
    timer = setTimeout(() => {
      processHandle.kill();
      log(`timeout after ${hook.timeout / 1000}s: ${hook.source}`);
      resolveResult({ code: 0, output: "" });
    }, hook.timeout);
  });
  const completed = (async (): Promise<HookResult> => {
    const code = await processHandle.exited;
    const output = (await new Response(processHandle.stdout).text()).trim();
    const errorOutput = (await new Response(processHandle.stderr).text()).trim();
    if (errorOutput) log(`${hook.source}: ${errorOutput}`);
    let response: Json | undefined;
    if (output) {
      try {
        response = asObject(JSON.parse(output));
      } catch {
        // Plain text output does not carry a structured decision.
      }
    }
    return { code, output, response };
  })();

  try {
    return await Promise.race([completed, timeout]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

class HookRunner {
  constructor(
    readonly cwd: string,
    readonly hooks: Hook[],
  ) {}

  forEvent(event: string, toolName = ""): Hook[] {
    return this.hooks.filter((hook) =>
      hook.event === event && (!toolName || matches(hook, toolName))
    );
  }

  async preTool(toolName: string, toolInput: unknown): Promise<{ block: boolean; reason?: string }> {
    const payload: Json = {
      hook_event_name: "PreToolUse",
      cwd: this.cwd,
      tool_name: toolName,
      tool_input: toolInput,
    };
    for (const hook of this.forEvent("PreToolUse", toolName)) {
      const result = await runHook(hook, payload, this.cwd);
      const reason = asString(result.response?.reason, "blocked by hook");
      if (result.code === 2 || result.response?.decision === "block") {
        return { block: true, reason };
      }
      // TODO: Apply updatedInput after zot defines rewrite semantics here.
    }
    return { block: false };
  }

  async event(event: string, payload: Json): Promise<void> {
    for (const hook of this.forEvent(event)) {
      await runHook(hook, payload, this.cwd);
    }
  }
}

async function runProtocol(): Promise<void> {
  send({ type: "hello", name: NAME, version: VERSION, capabilities: ["events"] });
  let runner: HookRunner | undefined;
  const input = createInterface({ input: process.stdin, crlfDelay: Infinity });

  for await (const line of input) {
    let frame: Json;
    try {
      frame = asObject(JSON.parse(line)) ?? {};
      trace("in", frame);
    } catch (error) {
      log(`invalid host frame: ${error}`);
      continue;
    }

    const type = frame.type;
    if (type === "hello_ack") {
      const cwd = asString(frame.cwd, process.cwd());
      runner = new HookRunner(cwd, await loadHooks(cwd));
      send({
        type: "subscribe",
        events: ["session_start", "turn_end", "tool_call", "assistant_message"],
        intercept: ["tool_call"],
      });
      send({ type: "ready" });
      log(`loaded ${runner.hooks.length} hook(s)`);
      continue;
    }

    if (type === "event_intercept" && runner) {
      const event = asString(frame.event);
      const decision = event === "tool_call"
        ? await runner.preTool(asString(frame.tool_name), frame.tool_args)
        : { block: false };
      send({
        type: "event_intercept_response",
        id: asString(frame.id),
        block: decision.block,
        ...(decision.reason ? { reason: decision.reason } : {}),
      });
      continue;
    }

    if (type === "event" && runner) {
      const event = asString(frame.event);
      const mapping: Record<string, string> = {
        session_start: "SessionStart",
        turn_end: "Stop",
        tool_call: "Notification",
        assistant_message: "Notification",
      };
      const hookEvent = mapping[event];
      if (hookEvent) await runner.event(hookEvent, { ...frame, hook_event_name: hookEvent });
      continue;
    }

    if (type === "shutdown") {
      send({ type: "shutdown_ack" });
      input.close();
      return;
    }
  }
}

const app = new Crust(NAME)
  .meta({ description: "Run zot lifecycle hooks" })
  .command(
    "list",
    (command) => command
      .meta({ description: "List discovered hook files" })
      .run(async () => {
        const hooks = await loadHooks(process.cwd());
        for (const hook of hooks) {
          console.log(`${hook.event}\t${hook.matcher}\t${hook.source}\t${hook.command}`);
        }
      }),
  );

if (process.argv.slice(2).length > 0) {
  await app.execute();
} else {
  await runProtocol();
}
