const go = new Go(); // Defined in wasm_exec.js. Don't forget to add this in your index.html.

const runWasmAdd = async () => {
  // Get the importObject from the go instance.
  const importObject = go.importObject;

  // Instantiate our wasm module
  const wasmModule = await wasmBrowserInstantiate("./safe.wasm", importObject);

  // Allow the wasm_exec go instance, bootstrap and execute our wasm module.
  // main() registers ProcessRequest on the JS global object, then blocks so
  // the instance stays alive to service further calls into it.
  go.run(wasmModule.instance);

  // Stand-in for the JSON string run.arti2.workers.dev's "Save JSON" button
  // produces (the flat dict of form field values).
  const requestJSON = JSON.stringify({ hello: "world" });

  // Call the function Go registered on globalThis. It takes the JSON string
  // and returns a plain JS array of strings, which may be empty.
  const results = ProcessRequest(requestJSON);

  // Set the result onto the body, handling the zero-length case explicitly.
  document.body.textContent = results.length === 0
    ? "Hello World! No results."
    : `Hello World! results: ${results.join(", ")}`;
};
runWasmAdd();