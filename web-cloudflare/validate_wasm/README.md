Download TinyGo
https://tinygo.org

npm install -g binaryen

which wasm-opt

export WASMOPT={path to wasm-opt}

tinygo build -no-debug -o ./web-cloudflare/runner/public/validate.wasm -target=wasm ./web-cloudflare/validate_wasm

# -no-debug strips DWARF debug info, which otherwise dominates the file
# size (it was the largest single contributor by far - see below).
# wasm-opt -Oz trims a further ~4% on top of that.
wasm-opt -Oz ./web-cloudflare/runner/public/validate.wasm -o ./web-cloudflare/runner/public/validate.wasm
