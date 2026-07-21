// Regression repro for the Bun extra-stdio-pipe crash worked around in
// extensions/wt/process.ts.
//
// Bun (confirmed on 1.3.14) backs stdio entries beyond stderr with a
// unix-domain socket it connects to asynchronously. That connect intermittently
// fails with ENOENT ("Failed to connect") and surfaces as a detached
// unhandledRejection that no try/catch can intercept, killing the process. Node
// spawns the same fd-3 pipe reliably.
//
// Usage:
//   bun  plugins/pi/scripts/repro-bun-fd3-crash.mjs   # crashes (exit 42) while buggy
//   node plugins/pi/scripts/repro-bun-fd3-crash.mjs   # clean
//
// It spawns `printf` (no dependency on `wt`) with a fourth stdio pipe, the exact
// shape execWt used before the fix. If Bun ever spawns extra pipes reliably this
// script runs clean under bun too, which is the signal to drop the workaround.

import { spawn } from "node:child_process";

const ITERATIONS = 200;

process.on("unhandledRejection", (reason) => {
  console.error(`CRASH: unhandledRejection at iteration ${globalThis.__iter}:`);
  console.error(reason);
  process.exit(42);
});

function spawnWithFd3() {
  return new Promise((resolve) => {
    const child = spawn("printf", ["hi"], {
      shell: false,
      stdio: ["ignore", "pipe", "pipe", "pipe"],
    });
    child.on("error", () => {});
    child.on("close", (code) => resolve(code));
  });
}

for (let i = 0; i < ITERATIONS; i++) {
  globalThis.__iter = i;
  await spawnWithFd3();
}

console.log(`${ITERATIONS} fd-3 spawns completed with no crash.`);
