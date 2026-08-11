const go = new Go(); // Defined in wasm_exec.js. Don't forget to add this in your index.html.

const runWasmAdd = async () => {
  // Get the importObject from the go instance.
  const importObject = go.importObject;

  // Instantiate our wasm module
  const wasmModule = await wasmBrowserInstantiate("./safe.wasm", importObject);

  // Allow the wasm_exec go instance, bootstrap and execute our wasm module.
  // main() registers SafeVerseNum on the JS global object, then blocks so
  // the instance stays alive to service further calls into it.
  go.run(wasmModule.instance);

  // Call the function Go registered on globalThis, save the result
  const addResult = SafeVerseNum("123t5");

  // Set the result onto the body
  document.body.textContent = `Hello World! addResult: ${addResult}`;
};
runWasmAdd();