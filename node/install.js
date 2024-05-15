// Credits:
// - Bun.js (https://github.com/oven-sh/bun/blob/main/packages/bun-release/src)
const child_process = require("child_process");
const { unzipSync } = require("zlib");
const packageJson = require("./package.json");
const fs = require("fs");

function getPlatform() {}

async function downloadCli(version, platform) {
  const ext = platform.os === "win32" ? ".zip" : ".tar.gz";
  const response = await fetch(
    `https://github.com/flexstack/new-dockerfile/releases/download/v${version}/${platform.bin}${ext}`
  );
  const tgz = await response.arrayBuffer();
  let buffer;

  try {
    buffer = unzipSync(tgz);
  } catch (cause) {
    throw new Error("Invalid gzip data", { cause });
  }

  function str(i, n) {
    return String.fromCharCode(...buffer.subarray(i, i + n)).replace(
      /\0.*$/,
      ""
    );
  }
  let offset = 0;
  const dst = platform.exe;
  while (offset < buffer.length) {
    const name = str(offset, 100).replace("package/", "");
    const size = parseInt(str(offset + 124, 12), 8);
    offset += 512;
    if (!isNaN(size)) {
      write(dst, buffer.subarray(offset, offset + size));
      if (name === platform.exe) {
        try {
          fs.chmodSync(dst, 0o755);
        } catch (error) {}
      }
      offset += (size + 511) & ~511;
    }
  }
}

const fetch = "fetch" in globalThis ? webFetch : nodeFetch;

async function webFetch(url, options) {
  const response = await globalThis.fetch(url, options);
  if (options?.assert !== false && !isOk(response.status)) {
    try {
      await response.text();
    } catch {}
    throw new Error(`${response.status}: ${url}`);
  }
  return response;
}

async function nodeFetch(url, options) {
  const { get } = await import("node:http");
  return new Promise((resolve, reject) => {
    get(url, (response) => {
      const status = response.statusCode ?? 501;
      if (response.headers.location && isRedirect(status)) {
        return nodeFetch(url).then(resolve, reject);
      }
      if (options?.assert !== false && !isOk(status)) {
        return reject(new Error(`${status}: ${url}`));
      }
      const body = [];
      response.on("data", (chunk) => {
        body.push(chunk);
      });
      response.on("end", () => {
        resolve({
          ok: isOk(status),
          status,
          async arrayBuffer() {
            return Buffer.concat(body).buffer;
          },
          async text() {
            return Buffer.concat(body).toString("utf-8");
          },
          async json() {
            const text = Buffer.concat(body).toString("utf-8");
            return JSON.parse(text);
          },
        });
      });
    }).on("error", reject);
  });
}

function isOk(status) {
  return status >= 200 && status <= 204;
}

function isRedirect(status) {
  switch (status) {
    case 301: // Moved Permanently
    case 308: // Permanent Redirect
    case 302: // Found
    case 307: // Temporary Redirect
    case 303: // See Other
      return true;
  }
  return false;
}

const os = process.platform;

const arch =
  os === "darwin" && process.arch === "x64" && isRosetta2()
    ? "arm64"
    : process.arch;

const platforms = [
  {
    os: "darwin",
    arch: "x64",
    bin: "new-dockerfile-darwin-x86_64",
    exe: "bin/new-dockerfile",
  },
  {
    os: "darwin",
    arch: "arm64",
    bin: "new-dockerfile-darwin-arm64",
    exe: "bin/new-dockerfile",
  },
  {
    os: "linux",
    arch: "x64",
    bin: "new-dockerfile-linux-x86_64",
    exe: "bin/new-dockerfile",
  },
  {
    os: "linux",
    arch: "arm64",
    bin: "new-dockerfile-linux-arm64",
    exe: "bin/new-dockerfile",
  },
  {
    os: "win32",
    arch: "x64",
    bin: "new-dockerfile-windows-x86_64",
    exe: "bin/new-dockerfile.exe",
  },
  {
    os: "win32",
    arch: "arm64",
    bin: "new-dockerfile-windows-arm64",
    exe: "bin/new-dockerfile.exe",
  },
];

const supportedPlatforms = platforms.filter(
  (platform) => platform.os === os && platform.arch === arch
);

function isRosetta2() {
  try {
    const { exitCode, stdout } = spawn("sysctl", [
      "-n",
      "sysctl.proc_translated",
    ]);
    return exitCode === 0 && stdout.includes("1");
  } catch (error) {
    return false;
  }
}

function spawn(cmd, args, options = {}) {
  const { status, stdout, stderr } = child_process.spawnSync(cmd, args, {
    stdio: "pipe",
    encoding: "utf-8",
    ...options,
  });
  return {
    exitCode: status ?? 1,
    stdout,
    stderr,
  };
}

function write(dst, content) {
  try {
    fs.writeFileSync(dst, content);
    return;
  } catch (error) {
    // If there is an error, ensure the parent directory
    // exists and try again.
    try {
      fs.mkdirSync(path.dirname(dst), { recursive: true });
    } catch (error) {
      // The directory could have been created already.
    }
    fs.writeFileSync(dst, content);
  }
}

if (supportedPlatforms.length === 0) {
  throw new Error("Unsupported platform: " + os + " " + arch);
}

// Read version from package.json
downloadCli(packageJson.config.bin_version, supportedPlatforms[0]);
