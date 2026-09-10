import { afterEach, describe, expect, test } from "bun:test";
import { cpSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { startFakeProvider } from "./support/fake-provider";

const extensionDirectory = resolve(import.meta.dir, "../..");
const useCasesDirectory = join(import.meta.dir, "use-cases");
const zotBinary = process.env.ZOT_BIN ?? "zot";
const temporaryDirectories: string[] = [];

function jsonLines(text: string): Record<string, unknown>[] {
  return text.split("\n").filter(Boolean).map((line) => JSON.parse(line));
}

function readText(path: string): string {
  try {
    return readFileSync(path, "utf8");
  } catch {
    return "";
  }
}

async function runCase(name: string) {
  const sourceProject = join(useCasesDirectory, name, "project");
  const temporaryDirectory = mkdtempSync(join(tmpdir(), "zot-cluade-hooks-e2e-"));
  temporaryDirectories.push(temporaryDirectory);
  const project = join(temporaryDirectory, "project");
  cpSync(sourceProject, project, { recursive: true });

  const provider = await startFakeProvider();
  const actionLog = join(temporaryDirectory, "actions.jsonl");
  const eventLog = join(temporaryDirectory, "zot-events.jsonl");
  const protocolLog = join(temporaryDirectory, "extension-protocol.jsonl");
  const processHandle = Bun.spawn([
    zotBinary,
    "--json",
    "--no-session",
    "--cwd", project,
    "--provider", "openai",
    "--model", "gpt-5",
    "--api-key", "fixture-key",
    "--base-url", provider.url,
    "--max-steps", "3",
    "--ext", extensionDirectory,
    "run the fixture tool",
  ], {
    cwd: project,
    env: {
      ...process.env,
      HOME: join(temporaryDirectory, "home"),
      XDG_CONFIG_HOME: join(temporaryDirectory, "config"),
      XDG_STATE_HOME: join(temporaryDirectory, "state"),
      ZOT_HOOK_TEST_LOG: actionLog,
      ZOT_HOOKS_PROTOCOL_TRACE: protocolLog,
    },
    stdout: "pipe",
    stderr: "pipe",
  });

  const timeout = setTimeout(() => processHandle.kill(), 15_000);
  const [exitCode, stdout, stderr] = await Promise.all([
    processHandle.exited,
    new Response(processHandle.stdout).text(),
    new Response(processHandle.stderr).text(),
  ]);
  clearTimeout(timeout);
  provider.server.stop();
  writeFileSync(eventLog, stdout);

  let actions = "";
  try {
    actions = readFileSync(actionLog, "utf8");
  } catch {
    // A non-matching or invalid fixture is expected not to create a log.
  }

  return {
    actions,
    eventLog: readText(eventLog),
    exitCode,
    protocolLog: readText(protocolLog),
    provider,
    stderr,
    stdout,
  };
}

afterEach(() => {
  for (const directory of temporaryDirectories.splice(0)) rmSync(directory, { recursive: true, force: true });
});

describe("zot-cluade-hooks end-to-end", () => {
  test("allows a matching PreToolUse hook", async () => {
    const result = await runCase("pre-tool-use-allow");
    expect(result.exitCode).toBe(0);
    const action = JSON.parse(result.actions.split("\n")[0]);
    expect(action).toMatchObject({
      action: "hook",
      payload: {
        hook_event_name: "PreToolUse",
        tool_name: "bash",
        tool_input: { command: "printf tool-ran" },
      },
    });
    const events = jsonLines(result.eventLog);
    expect(events.some((event) => event.type === "tool_call")).toBe(true);
    // The Go SDK owns the wire protocol, so protocol frames are not traced by
    // the extension. The observable interception result is asserted below.
    expect(result.stdout).toContain("fixture complete");
  }, 20_000);

  test("blocks on exit status 2", async () => {
    const result = await runCase("pre-tool-use-block-exit");
    expect(result.exitCode).toBe(0);
    expect(result.actions).toContain('"tool_name":"bash"');
    expect(result.stdout).toContain("blocked by hook");
    expect(result.stdout).toContain('"is_error":true');
    expect(result.stdout).not.toContain('"type":"tool_progress"');
  }, 20_000);

  test("blocks on a JSON decision", async () => {
    const result = await runCase("pre-tool-use-block-json");
    expect(result.exitCode).toBe(0);
    expect(result.actions).toContain('"tool_name":"bash"');
    expect(result.stdout).toContain("fixture blocked this tool");
  }, 20_000);
});
