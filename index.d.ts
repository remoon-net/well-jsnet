export interface Config {
  Key: string
  NAT?: string
  ICE?: string[]
  Peer?: string
  Peers?: Peer[]
  signal?: AbortSignal
  LogLevel?: "DEBUG" | "INFO" | "WARN" | "ERROR"
}

export interface Peer {
  Pubkey: string
  PSK?: string
  Endpoint?: string
  Auto?: number
  Allow?: string
}

interface Network {
  listen(addr: string, handler: Handler): Promise<Server>
  http_proxy(addr: string, handler: HTTPProxyHandler): Promise<Server>
  socks5_proxy(addr: string, handler: HTTPProxyHandler): Promise<Server>
  fetch(req: Request): Promise<Response>
}

interface Handler {
  fetch: (req: Request) => Response
  signal?: AbortSignal
}

interface HTTPProxyHandler {
  signal?: AbortSignal
}

interface Server {
  close(): void
}

export function connect(config: Config): Promise<Network>
