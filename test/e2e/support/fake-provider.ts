type ServerHandle = { port: number; stop(): void };

export type FakeProvider = {
  server: ServerHandle;
  url: string;
  requests: unknown[];
};

export async function startFakeProvider(): Promise<FakeProvider> {
  const requests: unknown[] = [];
  let toolCallSent = false;

  const server = Bun.serve({
    port: 0,
    async fetch(request) {
      if (new URL(request.url).pathname !== "/v1/chat/completions") {
        return new Response("not found", { status: 404 });
      }

      requests.push(await request.json());
      const chunks = !toolCallSent
        ? [
            {
              id: "fixture-completion-1",
              choices: [{
                index: 0,
                delta: {
                  role: "assistant",
                  tool_calls: [{
                    index: 0,
                    id: "fixture-tool-call",
                    type: "function",
                    name: "bash",
                    arguments: JSON.stringify({ command: "printf tool-ran" }),
                    function: {
                      name: "bash",
                      arguments: JSON.stringify({ command: "printf tool-ran" }),
                    },
                  }],
                },
                finish_reason: null,
              }],
            },
            {
              id: "fixture-completion-1",
              choices: [{
                index: 0,
                delta: {},
                finish_reason: "tool_calls",
              }],
            },
          ]
        : [
            {
              id: "fixture-completion-2",
              choices: [{
                index: 0,
                delta: { role: "assistant", content: "fixture complete" },
                finish_reason: null,
              }],
            },
            {
              id: "fixture-completion-2",
              choices: [{ index: 0, delta: {}, finish_reason: "stop" }],
            },
          ];
      toolCallSent = true;
      chunks.push({
        id: "fixture-usage",
        choices: [],
        usage: { prompt_tokens: 8, completion_tokens: 2 },
      });
      const body = `${chunks.map((chunk) => `data: ${JSON.stringify({
        object: "chat.completion.chunk",
        created: 0,
        model: "gpt-5",
        ...chunk,
      })}\n\n`).join("")}data: [DONE]\n\n`;
      return new Response(body, {
        headers: { "content-type": "text/event-stream" },
      });
    },
  });

  return {
    server,
    url: `http://127.0.0.1:${server.port}/v1`,
    requests,
  };
}
