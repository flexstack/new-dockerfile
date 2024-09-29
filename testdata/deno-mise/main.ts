import { serve } from "./deps.ts";

const PORT = Deno.env.get("PORT") || "8000";
const s = serve(`0.0.0.0:${PORT}`);
const body = new TextEncoder().encode("Hello World\n");

console.log(`Server started on port ${PORT}`);
for await (const req of s) {
  req.respond({ body });
}

Deno.addSignalListener("SIGINT", () => {
  console.log("\nServer stopped.");
  s.close();
  Deno.exit();
});

Deno.addSignalListener("SIGTERM", () => {
  console.log("\nServer stopped.");
  s.close();
  Deno.exit();
});
