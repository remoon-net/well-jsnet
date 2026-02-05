if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer()
    return await WebAssembly.instantiate(source, importObject)
  }
}

import { Go } from "./gojs/wasm_exec.js"

const _wasmUrl = import.meta.resolve("./gojs/well-jsnet.wasm")

export async function connect(config, wasmUrl = _wasmUrl) {
  const go = new Go()
  const { instance } = await WebAssembly.instantiateStreaming(
    fetch(wasmUrl),
    go.importObject,
  )
  var vpn = {
    config: config,
    connect_result: null,
  }
  go.importObject.vpn = vpn
  go.run(instance)
  const network = await vpn.connect_result
  return network
}
