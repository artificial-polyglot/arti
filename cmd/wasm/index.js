const go = new Go(); // Defined in wasm_exec.js. Don't forget to add this in your index.html.

const runWasmAdd = async () => {
  // Get the importObject from the go instance.
  const importObject = go.importObject;

  // Instantiate our wasm module
  const wasmModule = await wasmBrowserInstantiate("./safe.wasm", importObject);

  // Allow the wasm_exec go instance, bootstrap and execute our wasm module.
  // main() registers ValidateRequest on the JS global object, then blocks so
  // the instance stays alive to service further calls into it.
  go.run(wasmModule.instance);

  // Stand-in for the JSON string run.arti2.workers.dev's "Save JSON" button
  // produces (the flat dict of form field values).
  const requestJSON = JSON.stringify({ hello: "world" });

  // Call the function Go registered on globalThis. It returns
  // { request: string, errors: string[] } -- request is the validated,
  // normalized Request encoded as JSON; errors is empty on success.
  const result = ValidateRequest(requestJSON);

  document.body.textContent = result.errors.length === 0
    ? `Hello World! request: ${result.request}`
    : `Hello World! errors: ${result.errors.join(", ")}`;
};
runWasmAdd();