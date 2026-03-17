#!/usr/bin/env node

"use strict";

const { execFileSync, execSync } = require("child_process");
const path = require("path");
const fs   = require("fs");

const PACKAGE      = require("./package.json");
const EXPECTED_VER = PACKAGE.version;
const NAME         = "agencycli";
const binDir       = path.join(__dirname, "bin");
const ext          = process.platform === "win32" ? ".exe" : "";
const binaryPath   = path.join(binDir, NAME + ext);

// parseVersion splits "1.2.3-beta.1" into { nums: [1,2,3], pre: "beta.1" }
function parseVersion(v) {
  v = v.replace(/^v/, "").trim();
  const [base, ...rest] = v.split("-");
  const nums = base.split(".").map(Number);
  return { nums, pre: rest.join("-") };
}

function isNewerOrEqual(installed, expected) {
  const a = parseVersion(installed);
  const b = parseVersion(expected);
  const len = Math.max(a.nums.length, b.nums.length);
  for (let i = 0; i < len; i++) {
    const av = a.nums[i] || 0;
    const bv = b.nums[i] || 0;
    if (av > bv) return true;
    if (av < bv) return false;
  }
  if (!a.pre && b.pre) return true;
  if (a.pre && !b.pre) return false;
  return a.pre >= b.pre;
}

function needsReinstall() {
  if (!fs.existsSync(binaryPath)) return true;
  try {
    const out = execFileSync(binaryPath, ["version"], { encoding: "utf8", timeout: 5000 });
    if (out.includes(EXPECTED_VER)) return false;
    const match = out.match(/(\d+\.\d+\.\d+[^\s]*)/);
    if (match && isNewerOrEqual(match[1], EXPECTED_VER)) return false;
    return true;
  } catch {
    return true;
  }
}

if (needsReinstall()) {
  console.log(`[agencycli] Binary missing or outdated, installing v${EXPECTED_VER}…`);
  try {
    execSync("node " + JSON.stringify(path.join(__dirname, "install.js")), {
      stdio: "inherit",
      cwd:   __dirname,
    });
  } catch {
    console.error(
      "[agencycli] Auto-install failed.\n" +
      "  Run manually: npm uninstall -g agencycli && npm install -g agencycli"
    );
    process.exit(1);
  }
}

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (err) {
  process.exit(err.status || 1);
}
