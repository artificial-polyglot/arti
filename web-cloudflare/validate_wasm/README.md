Download TinyGo
https://tinygo.org

npm install -g binaryen

which wasm-opt

export WASMOPT={path to wasm-opt}

tinygo build -o ./web-cloudflare/validate_wasm/validate.wasm -target=wasm ./web-cloudflare/validate_wasm
