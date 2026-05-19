export async function fetchApi(input: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(input, init)
  if (res.status === 401) {
    window.location.href = '/login'
    return res
  }
  return res
}

export interface Host {
  id: number
  host_id: string
  url: string
  status: string
  sc: string
  title: string
  server: string
  tech: string[]
  ports: { port: string; service: string }[]
  ips: string[]
  cname: string[]
  ctype: string
  triage_status: string
  notes: string
  badges: string[]
}

export interface Hit {
  url: string
  status: string
  sc: string
  size: string
  severity: 'high' | 'medium' | 'low'
}

export interface HostStats {
  total: number
  s200: number
  s403: number
  s500: number
}
