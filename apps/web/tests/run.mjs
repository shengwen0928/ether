import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

/**
 * Test runner (Phase 02). No test framework is installed and none is added:
 * - esbuild (already a transitive dependency of Vite) bundles the test
 *   entries to CommonJS in a temporary directory,
 * - node:test runs the .mjs suites against those bundles,
 * - the temporary directory is removed afterwards.
 *
 * Usage from apps/web: `node tests/run.mjs`
 */

const root = fileURLToPath(new URL('..', import.meta.url));
const bundleDir = mkdtempSync(join(tmpdir(), 'ether-web-tests-'));
const esbuildBin = join(root, 'node_modules', '.bin', 'esbuild');

function run(command, args) {
  execFileSync(command, args, { cwd: root, stdio: 'inherit' });
}

let exitCode = 0;
try {
  run(esbuildBin, [
    'tests/entry-logic.ts',
    'tests/entry-components.tsx',
    'tests/entry-api.ts',
    '--bundle',
    '--platform=node',
    '--format=cjs',
    '--jsx=automatic',
    `--outdir=${bundleDir}`,
    '--log-level=warning',
  ]);
  // The suites import the bundles from a fixed relative path; copy them in.
  const staging = join(root, 'tests', '.bundle');
  mkdirSync(staging, { recursive: true });
  copyFileSync(join(bundleDir, 'entry-logic.js'), join(staging, 'entry-logic.cjs'));
  copyFileSync(join(bundleDir, 'entry-components.js'), join(staging, 'entry-components.cjs'));
  copyFileSync(join(bundleDir, 'entry-api.js'), join(staging, 'entry-api.cjs'));
  try {
    run(process.execPath, [
      '--test',
      'tests/logic.test.mjs',
      'tests/components.test.mjs',
      'tests/api.test.mjs',
    ]);
  } finally {
    rmSync(staging, { recursive: true, force: true });
  }
} catch (error) {
  exitCode = typeof error?.status === 'number' ? error.status : 1;
} finally {
  rmSync(bundleDir, { recursive: true, force: true });
}
process.exit(exitCode);
