import Terser from "terser-webpack-plugin";
import path from "path";
import URL from "url";
import fs from "fs";
import crypto from "crypto";
import childProcess from "child_process";

const config = {
    optimization: {
        minimize: true,
        minimizer: [
            new Terser({
                parallel: true,
                terserOptions: {
                    mangle: true,
                    ecma: 5
                }
            })
        ]
    },
    entry: {
        bundle: "./public/index.js"
    },
    output: {
        path: path.resolve(path.dirname(URL.fileURLToPath(import.meta.url)), "./build/public"),
        filename: "[name].js"
    },
    plugins: [{
        apply: (compiler) => {
            compiler.hooks.afterEmit.tap("AfterEmitPlugin", () => {
                console.log("Building go project...");

                // Build go project to build (Targeted for linux amd64)
                childProcess.execSync("go build -o build/koth", {
                    stdio: "inherit",
                    // Set GOOS and GOARCH to linux and amd64 respectively
                    env: {
                        ...process.env,
                        GOOS: "linux",
                        GOARCH: "amd64"
                    }
                });

                // Try to make it executable
                try {
                    fs.chmodSync("build/koth", 0o755);
                } catch (error) {
                    console.error("Failed to make the binary executable:", error);
                }

                console.log("Copying script and env file...");
                if (fs.existsSync("init_script.sh")) {
                    fs.copyFileSync("init_script.sh", "build/init_script.sh");
                } else {
                    fs.copyFileSync("init_script.example.sh", "build/init_script.sh");
                }
                fs.copyFileSync(".env", "build/.env");

                console.log("Copying and processing files...");

                const sourceDir = path.resolve(path.dirname(URL.fileURLToPath(import.meta.url)), "./public");
                const destinationDir = compiler.outputPath;

                // Clear the destination directory of all subdirectories (recursively) but leave the files intact
                fs.readdirSync(destinationDir).forEach((file) => {
                    if (fs.statSync(path.join(destinationDir, file)).isDirectory()) {
                        fs.rmSync(path.join(destinationDir, file), {
                            recursive: true
                        });
                    }
                });

                // Define a function to recursively copy and process files
                function copyAndProcessFiles(srcDir, destDir) {
                    const files = fs.readdirSync(srcDir);

                    files.forEach(file => {
                        const sourcePath = path.join(srcDir, file);
                        const destPath = path.join(destDir, file);

                        if (fs.statSync(sourcePath).isDirectory()) { // Recursively copy and process subdirectories
                            fs.mkdirSync(destPath);
                            copyAndProcessFiles(sourcePath, destPath);
                        } else if (path.extname(file) === ".html" || path.extname(file) === ".css") { // If it's an HTML or CSS file, minimize it and then copy
                            const content = fs.readFileSync(sourcePath, "utf8")
                                .replace(/(\r\n|\n|\r|\t)/gm, "")
                                .replace("index.js", "bundle.js")
                                .replace(/\s+/g, " ");

                            fs.writeFileSync(destPath, content, "utf8");
                        } else if (path.extname(file) !== ".js") { // Copy all other files (except .js files) as they are
                            fs.copyFileSync(sourcePath, destPath);
                        }
                    });
                }

                // Start copying and processing files from the source directory
                copyAndProcessFiles(sourceDir, destinationDir);

                function recursiveRemoveEmptyDirectories(directory) {
                    if (!fs.existsSync(directory)) {
                        return;
                    }

                    if (fs.statSync(directory).isDirectory()) {
                        const files = fs.readdirSync(directory);

                        if (files.length === 0) {
                            fs.rmdirSync(directory);
                            recursiveRemoveEmptyDirectories(path.dirname(directory));
                        } else {
                            files.forEach((file) => {
                                recursiveRemoveEmptyDirectories(path.join(directory, file));
                            });
                        }
                    }
                }

                // Remove empty directories from the destination directory
                recursiveRemoveEmptyDirectories(destinationDir);

                // Generate a hash of the build
                const hash = crypto.createHash("sha256");

                function recursiveHashFiles(directory) {
                    const files = fs.readdirSync(directory);

                    files.forEach((file) => {
                        const filePath = path.join(directory, file);

                        if (fs.statSync(filePath).isDirectory()) {
                            recursiveHashFiles(filePath);
                        } else {
                            hash.update(fs.readFileSync(filePath));
                        }
                    });
                }

                recursiveHashFiles(destinationDir);
                fs.writeFileSync(path.join(destinationDir, "version.txt"), hash.digest("hex"));
            });
        }
    }],
    resolve: {
        extensions: [".js", ".html"]
    },
    mode: "production"
};

export default config;