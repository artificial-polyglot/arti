H
Download TinyGo
https://tinygo.org

npm install -g binaryen

which wasm-opt

export WASMOPT={path to wasm-opt}

tinygo build -o ./cmd/wasm/safe.wasm -target=wasm ./cmd/wasm
