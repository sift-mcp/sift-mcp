"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const REPO = "sift-mcp/sift-mcp";
const BINARY_DIR = path.join(__dirname, "..", "bin");

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

function getAssetName(version) {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform) {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }
  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}`);
  }

  const ext = process.platform === "win32" ? "zip" : "tar.gz";
  return `sift_${version}_${platform}_${arch}.${ext}`;
}

function httpsGet(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { "User-Agent": "sift-mcp-npm" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          httpsGet(res.headers.location).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode} for ${url}`));
          return;
        }
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function getLatestVersion() {
  const data = await httpsGet(
    `https://api.github.com/repos/${REPO}/releases/latest`
  );
  const release = JSON.parse(data.toString());
  return release.tag_name.replace(/^v/, "");
}

async function downloadAndExtract(version) {
  const assetName = getAssetName(version);
  const downloadUrl = `https://github.com/${REPO}/releases/download/v${version}/${assetName}`;

  console.log(`Downloading sift v${version} for ${process.platform}/${process.arch}...`);

  const data = await httpsGet(downloadUrl);
  const tmpFile = path.join(BINARY_DIR, assetName);

  fs.mkdirSync(BINARY_DIR, { recursive: true });
  fs.writeFileSync(tmpFile, data);

  const binaryName = process.platform === "win32" ? "sift.exe" : "sift";

  if (assetName.endsWith(".zip")) {
    execSync(`unzip -o "${tmpFile}" ${binaryName} -d "${BINARY_DIR}"`, {
      stdio: "pipe",
    });
  } else {
    execSync(`tar xzf "${tmpFile}" -C "${BINARY_DIR}" ${binaryName}`, {
      stdio: "pipe",
    });
  }

  fs.unlinkSync(tmpFile);

  const binaryPath = path.join(BINARY_DIR, binaryName);
  if (process.platform !== "win32") {
    fs.chmodSync(binaryPath, 0o755);
  }

  console.log(`Installed sift to ${binaryPath}`);
}

async function main() {
  try {
    const version = await getLatestVersion();
    await downloadAndExtract(version);
  } catch (err) {
    console.error(`Failed to install sift binary: ${err.message}`);
    console.error("You can build from source: go build -o sift ./cmd/server");
    process.exit(1);
  }
}

main();
